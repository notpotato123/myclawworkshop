# Go Faster with Agents: Build Your Own Claw in Go

Daniel Mahlow, Managing Partner and Generative AI Lead, Contiamo


## Workshop Description

Claws - always-on, autonomous AI agents like OpenClaw and NanoClaw - are changing how people interact with AI. Unlike coding agents that wait for your commands, a claw runs continuously, remembers context across sessions, schedules its own tasks, and acts on your behalf. This hands-on workshop walks you through building your own claw from scratch in Go, covering everything from the core reasoning loop to inter-agent communication.

Starting from an empty main.go, we'll build a fully functional claw step by step: the agent reasoning loop, tool registration and execution, streaming responses, persistent memory, autonomous scheduling, and a web UI to interact with your agent. We'll use coding agents throughout the process - building an AI agent with the help of AI agents - while reviewing and understanding every layer of what gets produced.

Go's concurrency model maps directly to claw architecture. Goroutines handle parallel tool execution and background scheduling naturally. Channels deliver streaming LLM responses. The type system keeps tool schemas honest. And a single binary deployment means your claw runs anywhere - on your laptop, a Raspberry Pi, or a VPS - without runtime dependencies.

The workshop progresses through carefully designed demonstrations, each building on the previous. Along the way you'll learn to manage context windows, apply conversation compaction strategies, and evaluate security boundaries for agent systems. The final demonstration connects all participant-built claws via Google's A2A (Agent-to-Agent) protocol, where your agents discover peers on the network and begin collaborating autonomously - delegating tasks, sharing results, and solving problems together across the room.

While you're building a personal agent, the patterns transfer directly to production use cases: internal knowledge assistants, automated ops tools, LLM-powered APIs, and service-to-service coordination. The agent loop, tool system, and concurrency patterns you implement are the same foundations your team will need for any Go-based LLM integration.


## What You Will Learn

**Agent Fundamentals**

- Master the agent reasoning loop from first principles: prompt assembly, LLM inference, tool dispatch, and result integration.
- Implement a flexible tool registration system using Go interfaces and JSON schema generation.
- Handle streaming LLM responses using goroutines and channels.
- Understand context window management and conversation compaction strategies.

**Building a Claw**

- Build persistent memory so your claw remembers across sessions and restarts.
- Create autonomous scheduling - your claw acts without being asked.
- Develop a web UI for real-time interaction with your agent.
- Apply Go concurrency patterns (errgroup, fan-out/fan-in) to parallel tool execution.
- Evaluate architecture decisions for agent systems: sandboxing, permissions, and security boundaries.

**Multi-Agent Networking**

- Implement the A2A (Agent-to-Agent) protocol for agent discovery and communication.
- Connect your claw to peers for collaborative, multi-agent workflows across the network.


## Prerequisites

- Basic knowledge of Go programming.
- Familiarity with command-line interfaces.
- A conceptual understanding of APIs and server-client interactions (agent and protocol concepts will be introduced).
- Access to Claude, GPT-4o, Gemini, or similar LLM API (or equivalent; instructions provided before the workshop).
- A working Go development environment (Go 1.22+ recommended).
- Experience with generative AI coding tools or agents is beneficial but not mandatory.


## Recommended Agentic Coding Environments (Optional)

To enhance your workshop experience, you might consider exploring one of the following agentic coding environments. While not strictly required, having one set up can provide a more integrated experience.

- Claude Code (Preferred)
- Gemini CLI
- Codex CLI
- Cursor

Please ensure you have access to and are familiar with your chosen AI assistant's API or interface.


## Preparation

- Please clone the workshop repository (link to be provided) before the session.
- Ensure your Go environment and chosen AI assistant (API access or tool) are set up and working.
- Optional: Install Ollama for local model fallback.


## Bio

Daniel Mahlow is managing partner and generative AI lead at Contiamo, a Berlin-based consultancy. He has worked on various software and data engineering projects and since 2020 has been driving generative AI projects from prototype to large-scale production. He is a generalist, a builder and enjoys diving into new technology.
