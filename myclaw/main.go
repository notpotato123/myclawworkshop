package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"myclaw/a2a"
	"myclaw/agent"
	"myclaw/config"
	"myclaw/game"
	"myclaw/memory"
	"myclaw/scheduler"
	"myclaw/tools"
	"myclaw/web"
)

// memoryTokenBudget caps injected memories at ~2000 tokens (≈8000 chars).
const memoryTokenBudget = 8000

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cfg.Log()

	opts := []option.RequestOption{option.WithAPIKey(cfg.APIKey)}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	client := openai.NewClient(opts...)

	memStore, err := memory.NewStore(cfg.MemoryDir)
	if err != nil {
		slog.Error("failed to create memory store", "err", err)
		os.Exit(1)
	}

	// msgCh is the single channel all input sources feed into the agent loop.
	msgCh := make(chan agent.Message, 16)

	// hub broadcasts agent output to all connected WebSocket clients.
	hub := web.NewHub()
	srv := web.NewServer(hub, msgCh, cfg.Port)
	srv.SetA2ASender(agent.MakeSendFn(msgCh))

	sched, err := scheduler.New(cfg.TasksFile, func(description string) {
		slog.Info("scheduled task fired", "description", description)
		hub.Broadcast(web.SystemMsg("Scheduled task: " + description))
		msgCh <- agent.Message{
			Content: description,
			Source:  "scheduler",
			ReplyTo: func(text string) {
				fmt.Print(text)
				hub.Broadcast(web.ChunkMsg(text))
			},
			Done: func() {
				fmt.Println()
				hub.Broadcast(web.DoneMsg())
			},
			OnTool: func(name, status string) {
				slog.Info("tool call (scheduler)", "tool", name, "status", status)
				fmt.Fprintf(os.Stderr, "[tool %s: %s]\n", name, status)
				hub.Broadcast(web.ToolMsg(name, status))
			},
		}
	})
	if err != nil {
		slog.Error("failed to create scheduler", "err", err)
		os.Exit(1)
	}

	peerRegistry := a2a.NewRegistry()
	gameState := &game.State{}

	// ctx is cancelled on Ctrl+C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	registry := tools.NewRegistry()
	for _, t := range []tools.Tool{
		tools.ReadFile{},
		tools.ListDirectory{},
		tools.WriteFile{},
		tools.RunCommand{},
		tools.Remember{Store: memStore},
		tools.Recall{Store: memStore},
		tools.Schedule{Sched: sched},
		tools.DiscoverPeer{Registry: peerRegistry},
		tools.AskPeer{Registry: peerRegistry},
		tools.Broadcast{Registry: peerRegistry},
		tools.FindPeerWithSkill{Registry: peerRegistry},
		tools.JoinGame{State: gameState, Registry: peerRegistry, MsgCh: msgCh, AppCtx: ctx},
	} {
		if err := registry.Register(t); err != nil {
			slog.Error("failed to register tool", "tool", t.Name(), "err", err)
			os.Exit(1)
		}
	}

	prompt := web.SystemPrompt()
	if memories := memStore.Dump(memoryTokenBudget); memories != "" {
		prompt += "\n\n## Memories\nThe following information was saved from previous sessions:\n" + memories
	}

	go sched.Run(ctx)
	go agent.StartCLIInput(ctx, msgCh)
	go func() {
		if err := srv.Start(cfg.Port); err != nil {
			slog.Error("web server error", "err", err)
		}
	}()

	slog.Info("agent ready")
	fmt.Println("Agent ready. Type 'exit' or press Ctrl+C to quit.")

	if err := agent.RunAgent(ctx, &client, cfg.Model, prompt, registry, msgCh); err != nil {
		slog.Error("agent error", "err", err)
		os.Exit(1)
	}

	// Graceful shutdown: give in-flight work 5 s to drain then close WebSockets.
	slog.Info("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx)
	slog.Info("shutdown complete")
}
