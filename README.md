# Go Faster with Agents: Build Your Own Claw in Go

GopherCon Europe 2026 Workshop

## What is this?

A 7-hour hands-on workshop where you build a claw (always-on autonomous AI agent) from scratch in Go, using coding agents to help with implementation. The workshop culminates in a networked maze heist game where all participant-built claws collaborate via the A2A protocol.

## Prerequisites

- Go 1.22+ installed (`go version`)
- A coding agent ready (Claude Code, Gemini CLI, Codex CLI, or Cursor)
- This repo cloned

## Repository Structure

```
skills/          Markdown skill files fed to coding agents during demos
  demo_1/        The Agent Core - reasoning loop, tools, streaming
  demo_2/        From Agent to Claw - memory, scheduling, web UI
  demo_3/        A2A networking - server, client, connectivity
checkpoints/     Full source code at each stage (for catch-up)
```

## For Participants

1. Clone this repo before the workshop
2. Verify your Go environment: `go version`
3. Have your coding agent ready
4. LLM API endpoint will be provided at the workshop

## Workshop Flow

The workshop has three demos:

**Demo 1: The Agent Core** - Build a working CLI agent with tools and streaming
**Demo 2: From Agent to Claw** - Add memory, scheduling, and a web UI
**Demo 3: A2A + The Maze Heist** - Network your claws, then play the game

Each demo uses skill files you feed to your coding agent. Checkpoints let you catch up at any point.
