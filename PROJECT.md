# Claw — Project Summary

An autonomous AI agent built in Go as a workshop project. It exposes a chat UI over HTTP/WebSocket, persists memories across sessions, and can schedule tasks to run in the future. Everything lives in `myclaw/`.

## Repository

- **Repo:** `notpotato123/myclawworkshop`
- **Active branch:** `claude/tender-hamilton-h1166k`
- **PR #1** tracks all workshop work

---

## Architecture

```
main.go
├── config.Load()          — env/flag config
├── memory.NewStore()      — persistent memory directory
├── scheduler.New()        — task scheduler with JSON persistence
├── web.NewHub()           — WebSocket broadcast hub
├── web.NewServer()        — HTTP + WS server
├── tools.NewRegistry()    — registers all tools
└── agent.RunAgent()       — main agent loop (reads from msgCh)
```

All input sources (CLI stdin, WebSocket, scheduler callbacks) funnel into a single `chan agent.Message`. Each `Message` carries `ReplyTo`, `Done`, and `OnTool` callbacks so sources are handled uniformly.

---

## Package Tour

### `myclaw/agent`
- `Message` — unit of work: `Content`, `Source`, `ReplyTo func(string)`, `Done func()`, `OnTool func(name, status string)`
- `RunAgent(ctx, client, model, systemPrompt, registry, msgCh)` — loops on `msgCh`, calls `agentTurn` per message
- `agentTurn` — streaming OpenAI call → handles tool calls in a loop until no more tool calls
- `StartCLIInput(ctx, ch)` — reads stdin, blocks per-message on a `doneCh` until `Done()` fires (keeps prompt timing clean)

### `myclaw/tools`
| Tool | File | Description |
|---|---|---|
| `read_file` | `read_file.go` | Read a file from disk |
| `list_directory` | `list_directory.go` | List directory contents |
| `write_file` | `write_file.go` | Write/overwrite a file |
| `run_command` | `run_command.go` | Run a shell command |
| `remember` | `remember.go` | Save a memory to disk |
| `recall` | `recall.go` | Search saved memories |
| `schedule` | `schedule.go` | Create/list/cancel scheduled tasks |

All tools implement the `Tool` interface: `Name() string`, `Description() string`, `Schema() map[string]any`, `Execute(ctx, json.RawMessage) (string, error)`.

`Registry` (`registry.go`) maps names → tools; `buildToolDefs` in `agent.go` converts them to OpenAI function-calling format.

#### Schedule tool actions
- `schedule` — requires `description` + `delay` (Go duration string, e.g. `"1h"`); optional `recurring bool`
- `list` — lists pending tasks with IDs and fire times
- `cancel` — cancels by `id` OR by `description` (case-insensitive substring match)

### `myclaw/memory`
- Files stored in `MemoryDir` (default `~/.myclaw/memories/`) as markdown with YAML frontmatter
- `Save`, `Load`, `List`, `Search` (substring), `Dump(tokenBudget)` — `Dump` is used at startup to inject memories into the system prompt

### `myclaw/scheduler`
- `Task` — `ID`, `Description`, `ExecuteAt`, `Recurring`, `Interval`
- Persists tasks to `TasksFile` (default `~/.myclaw/tasks.json`) via atomic tmp+rename writes
- `Run(ctx)` — 1-second tick loop, fires due tasks, re-schedules recurring ones
- Releases the mutex before invoking callbacks to prevent deadlock when callbacks call `Add`/`Remove`

### `myclaw/web`
- `Hub` — RWMutex-protected client set; read lock held for entire `Broadcast`; `CloseAll` for graceful shutdown
- `ws.go` — single writer goroutine per connection; `sync.Once` prevents double-close of `sendCh`
- `server.go` — `embed.FS` for `static/` and `system_prompt.md`; `Start(port)` / `Shutdown(ctx)`
- WebSocket message types: `chunk`, `done`, `tool_call`, `tool_result`, `system`
- `index.html` — vanilla JS chat UI with hand-rolled markdown renderer, collapsible tool sections, auto-reconnect

### `myclaw/config`
- Sources: env vars (`OPENAI_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_MODEL`, `PORT`) + defaults
- `MemoryDir` defaults to `~/.myclaw/memories`, `TasksFile` to `~/.myclaw/tasks.json`
- `Log()` redacts the API key (first 4 + `****` + last 4)

---

## Key Design Decisions

- **Single `msgCh` fan-in** — CLI, WebSocket, and scheduler all produce `agent.Message` values; the agent loop doesn't know or care about the source.
- **Streaming first** — every response streams via SSE; `ReplyTo` is called per chunk so both the terminal and WebSocket clients see text as it arrives.
- **`sync.Once` double-close guard** — `CloseAll()` and the WS reader's defer both try to close `sendCh`; `Once` makes it safe.
- **Atomic JSON writes** — scheduler writes to a `.tmp` file then renames to prevent corruption on crash.
- **Memory injection** — `memStore.Dump(8000)` is called once at startup and appended to the system prompt so the LLM has context from past sessions.

---

## Running

```bash
cd myclaw
export OPENAI_API_KEY=sk-...
go run .
# Web UI: http://localhost:8080
# CLI: type in the terminal
```

---

## Tests

```bash
cd myclaw
go test ./...
```

Packages with tests: `memory`, `scheduler`, `tools` (memory tools + schedule tool).

---

## Files

```
myclaw/
├── main.go
├── go.mod
├── agent/
│   ├── agent.go
│   └── streaming.go
├── config/
│   └── config.go
├── memory/
│   ├── memory.go
│   └── memory_test.go
├── scheduler/
│   ├── scheduler.go
│   └── scheduler_test.go
├── tools/
│   ├── registry.go
│   ├── read_file.go
│   ├── list_directory.go
│   ├── write_file.go
│   ├── run_command.go
│   ├── remember.go
│   ├── recall.go
│   ├── schedule.go
│   ├── memory_tools_test.go
│   └── schedule_test.go
└── web/
    ├── hub.go
    ├── server.go
    ├── ws.go
    ├── system_prompt.md
    └── static/
        └── index.html
```
