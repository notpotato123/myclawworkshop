# Workshop Design Document - GopherCon Europe 2026

## Go Faster with Agents: Build Your Own Claw in Go

**Instructor:** Daniel Mahlow, Contiamo
**Duration:** 7 hours (including breaks)
**Expected participants:** ~50
**Teaching assistants:** Possibly 1 (TBD)

---

## 1. Workshop Philosophy

### Core thesis
Teaching Go developers to be effective with coding agents by having them build a claw (always-on autonomous AI agent) from scratch in Go - using coding agents to do it. Building an AI agent with the help of AI agents.

### Narrative arc: Agent -> Claw -> Harness
The workshop traces a progression through three layers of the same patterns:
- **Agent**: tool interface, reasoning loop, streaming (Demo 1). These are the same primitives inside Claude Code, zot, Cursor, and every agent harness.
- **Claw**: add memory, scheduling, web UI, and the agent becomes always-on and autonomous (Demo 2). This is where OpenClaw, NanoClaw, and Microsoft Scout live.
- **Harness**: add intercept hooks, guard rails, and credential isolation, and the claw becomes an orchestration layer (introduced via framing in Demo 2, experienced through game mechanics in Demo 3).

Participants build the agent and claw. They encounter harness patterns through the maze heist game, where some doors require implementing guard hooks to progress.

### What participants actually learn
The claw is the artifact, but the real skills are:
- How to direct coding agents effectively with Go
- How to review, QA, and understand agent-generated code
- How to structure prompts/skills for incremental, reviewable output
- Go concurrency patterns applied to real agent architectures
- The A2A protocol for inter-agent communication
- Security patterns for agent systems (sandboxing, credential isolation, intercept hooks)

### Methodology: Skills + Checkpoints + Review Gates
Instead of traditional starter code with TODOs, participants receive:
- **Skills:** Markdown files they feed to their coding agent, broken into sub-steps
- **Checkpoints:** Full source code at each stage, so anyone who falls behind can catch up
- **Review gates:** After each sub-step, participants QA the output with their agent (visualize as mermaid, explain the code, review for issues, run tests)

The rhythm per sub-step:
1. Instructor walks through the skill on screen, explains what it does and why
2. Participants feed the skill to their coding agent
3. Agent implements, participant reviews
4. Participant runs a validation skill to verify the checkpoint
5. Participant does QA with the agent: "visualize this as a mermaid diagram", "explain what this code does", "review this for issues"
6. Move to next sub-step

---

## 2. Infrastructure

### LiteLLM Proxy
- Hosted on a VPS (not local to venue)
- OpenAI-compatible endpoint
- Serves only the claws' LLM calls (not coding agents)
- Participants point their claw at this endpoint
- Ensures uniform model/capability across all participant claws
- Controls cost (single billing, rate limits if needed)

### Coding Agents
- BYO: participants bring their own coding agent setup
- Recommended: Claude Code (preferred), Gemini CLI, Codex CLI, Cursor
- Must be agentic (not just autocomplete)
- Participants responsible for their own API keys/subscriptions for the coding agent

### Repository Structure
- Workshop repo cloned by all participants before the session
- Contains: skills, checkpoints, game client tools, validation skills
- Each demo has its own directory with numbered sub-steps

---

## 3. Demo Structure

### Demo 1: The Agent Core (~2 hours)

**Goal:** Build a working CLI agent that can reason, use tools, and stream responses.

**Sub-steps:**
1. **Setup + loop + tools:** Combined step. Project setup, basic agent reasoning loop using `openai-go` SDK, define the `Tool` interface, implement `read_file` and `list_directory`, wire tool dispatch into the loop. This combines what were previously separate setup, loop, and tool steps to save time on scaffolding.
2. **Streaming:** Replace batch responses with streaming. Goroutine reads SSE stream, channel delivers tokens to the display loop.
3. **More tools:** Add `write_file`, `run_command`. Agent can now do useful things.

**Review gates after each sub-step:**
- Run `go test ./...`
- Ask agent to visualize the architecture as a mermaid diagram
- Ask agent to explain the tool dispatch flow
- Ask agent to review for error handling issues

**Checkpoint:** A working CLI agent that streams responses and can use 4 tools.

**Slide moment - harness connection:** "You just built the same core that powers Claude Code, zot, and every other agent harness. Same Tool interface, same agent loop, same streaming pattern." Show a side-by-side of the workshop's Tool interface and zot's. Mention that sorting tool specs alphabetically improves prompt cache hit rates (a pattern from zot).

**Go concepts highlighted:**
- Interfaces (Tool interface)
- Goroutines and channels (streaming)
- JSON struct tags (tool schemas)
- Context propagation

### Demo 2: From Agent to Claw (~2 hours)

**Goal:** Transform the CLI agent into a claw with memory, scheduling, and a web UI.

**Sub-steps:**
1. **Persistent memory:** Markdown-based memory system on disk. Agent can store and recall facts across sessions. System prompt includes relevant memories.
2. **Autonomous scheduling:** Background goroutine with a task scheduler. Agent can schedule future actions ("remind me in 30 minutes", "check this URL every hour"). Cron-like execution.
3. **Web UI:** Minimal chat interface - single `index.html` served by the Go binary. Text input, send button, scrolling message area. WebSocket connection for streaming responses. Replace CLI interaction with web-based interaction.
4. **Security + harness discussion (no code):** Security and harness framing only - no coding. Coding from this step (system prompt, config, slog) has been rolled into step 3. Cover: `run_command` sandboxing in production, credential isolation patterns (nono's phantom token, Infisical Agent Vault's MITM proxy), and the agent-claw-harness triptych.

**Slide moment - the triptych:** "Agent has tools and a loop. Harness adds intercept hooks, skill loading, and extensions. Claw adds memory, scheduling, always-on. Same core, different surface." Show three columns. Mention the three guard hooks (before-tool, before-turn, before-message) as "what you'd add to turn this into a harness."

**Security topics to cover:**
- Tool-level: nono (Landlock/Seatbelt kernel sandboxing, Go bindings available)
- Credential-level: phantom token pattern (agent gets a useless session token, proxy swaps in real key at network boundary), Agent Vault (TLS-intercepting forward proxy, written in Go)
- Container-level: gVisor for multi-tenant, Firecracker microVMs for strongest isolation
- The OpenClaw security crisis as a cautionary tale (48k exposed nodes, malicious skills)

**Review gates after each sub-step:**
- Verify memory persists across restarts
- Verify scheduled tasks fire correctly
- Verify web UI connects and streams
- Ask agent to review the concurrency model for race conditions

**Checkpoint:** A claw with web UI, persistent memory, and autonomous scheduling - running as a single Go binary.

**Go concepts highlighted:**
- File I/O and embed.FS
- time.Ticker, cron patterns
- net/http, WebSocket (gorilla/websocket or nhooyr.io/websocket)
- sync patterns, graceful shutdown with context cancellation
- errgroup for concurrent subsystems

### Demo 3: A2A + The Maze Heist (~1.5-2 hours)

**Goal:** Wire up A2A protocol, connect claws to peers, then play the game.

**Sub-steps:**
1. **A2A server:** Add an A2A endpoint to the claw (manual JSON-RPC implementation). Publish an Agent Card at `/.well-known/agent-card.json`. Handle incoming messages from peers.
2. **A2A client:** Discover peers by resolving their Agent Cards. Send messages to other claws. Implement basic request/response patterns.
3. **Connectivity test:** Verify claws can find and message each other on the network. Instructor runs a discovery check from the game server.
4. **Join the game:** Feed the join_game skill. Point claw at the game server. Claw joins the maze but sits stationary. Then 4 rounds of incremental capability building: Round 1 (move), Round 2 (look), Round 3 (doors + coordinate via use_ability + broadcast), Round 4 (jewel convergence). Each round: instructor shows a slide, participants prompt their coding agent, rebuild, rejoin. Hint files at hints/01-04 for anyone stuck.

**Checkpoint:** A networked claw that can communicate with peers and participate in the maze heist.

**Go concepts highlighted:**
- HTTP server multiplexing (agent server + A2A server + web UI on same binary)
- JSON-RPC handling
- Network discovery patterns
- Concurrent connections to multiple peers

---

## 4. The Maze Heist Game

### Concept
A large maze displayed on the big screen, covered in fog of war. A crown jewel is hidden deep inside, behind multiple locked doors. Each participant's claw controls an explorer. Claws must explore, share map data, coordinate roles, and solve challenges to reach the treasure.

### Self-running game design
The game server runs autonomously by default. The instructor's role during the game is commentator, not operator. Key automation features:
- **Auto-broadcast coordination hints**: when a door reaches 50%+ presence, the server broadcasts to all claws (e.g., "Door-5 at (12,8) needs 1 more lockpick! 2/3 present")
- **Auto-difficulty scaling**: if fewer than 3 doors open after 10 min, reduce outer door requirements; if fewer than 5 after 20 min, reduce middle doors
- **Auto-endgame timer**: convergence at 35 min, full reveal at 45 min (configurable via `GAME_CONVERGE_MINUTES`, `GAME_REVEAL_MINUTES`)
- **Auto-camera director**: visualization auto-pans between interesting events; click to override, drifts back after 10s of no interaction
- **GM panel as override**: simplified to status dashboard + broadcast input + three big buttons (harness challenge, force converge, force reveal)

### Game mechanics

**Exploration:**
- Claws interact with the game server via tool calls: `move(direction)`, `look()`, `use_ability(target)`
- `look()` returns visible surroundings: walls, paths, locked doors, other explorers
- Fog of war clears around each explorer as they move
- Maze is ~40x40 grid, large enough for 50 explorers to spread meaningfully

**Roles and locked doors:**
- On joining, each claw receives a role: lockpick, hacker, demolitions, analyst, scout
- ~10 claws per role type across 50 participants
- Key passages are blocked by doors requiring specific roles
- Some doors require two roles simultaneously
- Claws must discover peers via A2A and coordinate to open doors

**Map sharing via A2A:**
- Claws share explored map fragments with peers
- Shared data clears fog for the receiving claw
- Encourages A2A communication - more sharing = faster progress

**Human-in-the-loop challenges (deep doors):**
- Certain locks require a key string that the claw can't generate
- The claw presents a Go coding challenge to its human via the web UI
- The human writes Go code, runs it, gets the key string
- Human pastes the key back to the claw, which uses it to unlock the door
- Pre-configured challenges: Fibonacci, string reversal, SHA256 prefix, palindrome check, prime sum
- Challenges get harder deeper in the maze
- Keeps participants actively engaged during the game phase

**Harness challenge doors (deepest, GM-triggered, optional):**
- The instructor triggers these only if the game is going well and there's time
- Challenges require participants to implement a harness pattern in their claw:
  - Before-tool guard: block `run_command` calls containing dangerous patterns
  - Output masking: redact strings matching a secret pattern from tool results
  - Rate limiter: add a cooldown between consecutive tool calls
  - Skill loader: load a markdown file provided by the GM and follow its instructions
- Flow: GM broadcasts challenge to specific explorers, participants direct their coding agent to implement it, rebuild, reconnect, game server verifies via A2A test payload
- Reinforces the harness concept through gameplay, not slides

**The crown jewel:**
- Located behind multiple locked doors deep in the maze
- Reaching it requires coordination across multiple roles and human challenges
- When found: full maze reveal on big screen, all explorer paths traced, showing how the collective effort came together

### Why multiple claws are better than one
1. Parallel exploration: 50 explorers clear fog 50x faster
2. Role distribution: no single claw has all role types
3. Map sharing: communicating claws find optimal paths faster
4. Multi-role doors: some barriers need simultaneous cooperation
5. Human parallelism: 50 humans solving challenges simultaneously

### Visualization (big screen)
- Top-down maze view with fog of war
- Colored dots for each explorer, with role icon
- Locked doors glow in their required role color
- A2A messages visualized as pulses/arcs between dots
- Side panel: live feed of A2A messages and game events
- Door-opening animations
- Crown jewel reveal when found
- Explorer path traces on full reveal
- Auto-camera director with smooth lerp transitions between events
- "AUTO" / "MANUAL" indicator in corner; click to override, auto-resumes after 10s

### Technical implementation
**Game server (pre-built, self-running):**
- Go binary
- A2A endpoint for claw registration and peer discovery
- Polling inbox model for peer messaging: the game server queues A2A messages per explorer. Claws poll GET /api/inbox?explorer_id=X (outbound-only HTTP) and POST responses back. No inbound connectivity to participant machines is ever required - works through client isolation, NAT, and firewalls.
- HTTP API for game actions (move, look, use_ability, submit_key)
- WebSocket endpoint pushing state to the visualization
- Static HTML/JS/Canvas for the big screen visualization
- Pre-generated maze with randomized layout per session
- Pre-configured human challenges on deep doors
- Auto-broadcast, auto-difficulty, auto-endgame, auto-camera
- GM override panel at `/gm`

**Game client (round-based, built incrementally during the game):**
- Skill file contains only join_game with inbox poller + peer refresh
- Round 1: participants add move tool + auto-explore loop by prompting their coding agent
- Round 2: participants add look tool and update auto-explore to look before moving
- Round 3: participants add use_ability tool and broadcast door locations for coordination
- Round 4: participants update auto-explore instruction to converge on the jewel
- Hint files (hints/01-04) provide fallback implementations for each round

---

## 5. Realistic Timeline

7 hours gross, ~4h 45min working time after breaks and overhead.

| Time | Block | Notes |
|---|---|---|
| 9:00-9:20 | Intro slides, setup verification | Verify proxy, coding agent, Go version |
| 9:20-10:40 | Demo 1: Agent Core (3 steps) | Steps 1+2 combined, then streaming, then more tools |
| 10:40-11:00 | Coffee break | |
| 11:00-12:15 | Demo 2: Agent to Claw (4 steps) | Memory, scheduling, web UI, security/harness framing |
| 12:15-13:15 | Lunch | |
| 13:15-14:15 | Demo 3 steps 1-2: A2A server + client | |
| 14:15-14:40 | Connectivity test + debugging | Budget 25 min; primarily "verify you can reach the game server" - no peer-to-peer reachability needed |
| 14:45-15:00 | Coffee break + checkpoint catch-up | "Copy checkpoint 10 if needed" |
| 15:00-15:25 | Game Rounds 0-1 | Join (stationary dots), then Move (dots bumble) |
| 15:25-15:51 | Game Rounds 2-3 | Look (smarter movement), Doors + Coordinate (A2A payoff) |
| 15:51-16:05 | Round 4 + Endgame + Wrap-up | Jewel convergence, reveal, wrap-up slides |

### Critical mitigations
- **Checkpoint announcements**: Time-box each demo. At each transition, announce: "If you're not at checkpoint N, copy it now." Normalize this at the start.
- **Pre-built game-ready binary**: Build checkpoint 10 for macOS arm64/amd64 and Linux amd64 before the workshop. Participants who had trouble can still play.
- **WiFi backup**: Bring a dedicated travel router. Test the night before with multiple devices. Conference WiFi with client isolation will kill A2A.
- **Game difficulty**: Start with reduced door requirements (2 instead of 3 for outer doors). Smaller maze (`GAME_MAZE_SIZE=25`) is an option if time is tight.

## 6. Skill File Format

Each skill is a markdown file with clear sub-steps, designed to be fed to a coding agent incrementally.

```
# Skill: [Name]

## Context
[What this skill builds on, what the current state of the code is]

## Step 1: [Name]
[Clear instructions for the coding agent]

### Acceptance criteria
- [ ] [Specific testable outcome]
- [ ] [Specific testable outcome]

### Stop here
Verify the acceptance criteria before proceeding. Run `go test ./...` and confirm all tests pass.

## Step 2: [Name]
...
```

After implementation, a separate QA skill is used:
```
# QA Skill: Review [Feature]

1. Generate a mermaid diagram of the current architecture
2. Explain how [specific component] works in 3-4 sentences
3. Review the code for: error handling, race conditions, resource leaks
4. Suggest one improvement (but don't implement it)
```

Validation skills run automated checks:
```
# Validation Skill: Checkpoint [N]

Run the following checks and report results:
1. `go build ./...` succeeds
2. `go test ./...` passes
3. `go vet ./...` has no warnings
4. The Tool interface is defined with methods: Name(), Description(), Schema(), Execute()
5. At least 4 tools are registered
6. [Feature-specific checks]
```

---

## 7. Checkpoint Structure

Each checkpoint is a complete, buildable Go project representing the expected state at that point.

```
checkpoints/
  checkpoint_1_agent_loop/     # Mid-step catch-up: loop works, no tools yet
  checkpoint_2_tools/          # After demo 1, step 1: loop + tool interface + 2 tools
  checkpoint_3_streaming/      # After demo 1, step 2: streaming responses
  checkpoint_4_full_agent/     # After demo 1, step 3 (end of demo 1): all 4 tools
  checkpoint_5_memory/         # After demo 2, step 1: persistent memory
  checkpoint_6_scheduling/     # After demo 2, step 2: background scheduler
  checkpoint_7_web_ui/         # After demo 2, step 3: web UI
  checkpoint_8_claw/           # After demo 2, step 3 complete (incl. system prompt + config). Step 4 is discussion-only.
  checkpoint_9_a2a/            # After demo 3, steps 1-2: A2A server + client
  checkpoint_10_game_ready/    # After demo 3, steps 3-4: game client tools
```

Note: Demo 1 has 3 steps but 4 checkpoints. Checkpoint 1 is a mid-step snapshot useful if someone's coding agent gets the loop working but struggles with tools. Demo 2 step 3 now includes system prompt, config, and logging (rolled in from old step 4). Demo 2 step 4 is discussion-only (security/harness framing).

Participants who fall behind can copy a checkpoint and continue from there.

---

## 8. Go SDK Choice

The claw uses `openai-go` (github.com/openai/openai-go) pointed at the litellm proxy. Reasons:
- OpenAI-compatible API is the lingua franca (litellm, Ollama, vLLM all support it)
- Official SDK with good streaming support
- Tool/function calling support built in
- Participants who want to swap providers later just change the base URL

For A2A: manual implementation (JSON-RPC 2.0 over HTTP). More educational than using an SDK - participants see the raw protocol.

---

## 9. Slide Design

Reference: http://symbiotic.ctmo.io (Contiamo's existing Reveal.js deck)

### Framework and tooling
- Reveal.js
- Single HTML file with embedded CSS/JS
- Hash-based navigation, keyboard controls, overview mode

### Visual style
- Dark theme: #111827 (charcoal) background, #E20074 (magenta) accent
- Fonts: Inter (body), JetBrains Mono (code)
- Minimal: one concept per slide, ~70% whitespace, max 3-4 lines of text
- Terminal windows with OS chrome (red/yellow/green dots) for code/command examples
- Glow effects around accent colors
- Tags with subtle backgrounds for categorization

### Content approach
- Slides are anchors for the instructor's talking, not a transcript
- Progressive reveal via fragment animations (fade-up)
- Terminal/editor windows as primary visual elements
- Comparison layouts with flex columns and vertical dividers
- No corporate polish, no flowcharts - engineer-to-engineer tone

### Slide sections needed
1. **Intro:** Who am I, Contiamo, what we'll build today
2. **Claws overview:** What are claws, how they differ from coding agents, the ecosystem (OpenClaw, NanoClaw, GoClaw)
3. **Why Go:** Concurrency model, single binary, type safety - with terminal examples
4. **Setup:** Verify litellm proxy, verify coding agent, clone repo
5. **Demo 1 intro:** The agent loop - architecture diagram, what we'll build
6. **Demo 2 intro:** From agent to claw - what makes it a claw
7. **Demo 3 intro:** A2A protocol, the maze heist
8. **Wrap-up:** What you built, where to go next, resources

Each demo intro is a brief slide section (3-5 slides) before participants start working with skills.

---

## 10. Open Design Questions

- ~~How many human-in-the-loop challenges in the maze?~~ **Resolved**: 5 pre-configured challenges on deep doors (fib-20, reverse-str, sha256-prefix, palindrome, sum-primes)
- Should Go challenges be solvable with coding agents, or designed to need human reasoning?
- ~~Maze time limit?~~ **Resolved**: Auto-converge at 35 min, auto-reveal at 45 min (configurable)
- ~~Should claws see each other on their local `look()` response?~~ **Resolved**: Yes, within radius 3
- How to handle claws that go offline mid-game? (explorer freezes in place, rejoins where it left off)
- ~~Exact maze dimensions and door placement~~ **Resolved**: 40x40 default, configurable via `GAME_MAZE_SIZE`, 10 doors in 3 tiers
- Whether WebSocket streaming in the web UI works reliably zero-shot from coding agents (needs testing in reference implementation)
- Harness challenge door verification: how does the game server test that a participant's guard hook actually works? (likely: POST a test payload to their A2A endpoint, check response)
- Pre-built binary distribution: should checkpoint 10 binaries be in the repo, a GitHub release, or built on-site?
- Travel router model and network setup for WiFi backup
