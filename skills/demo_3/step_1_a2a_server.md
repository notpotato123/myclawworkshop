# Skill: A2A Server

> **Pacing:** Feed this skill to your agent ONE step at a time. After each "Stop here" marker, wait for the instructor before continuing to the next step.

## Context
Our claw runs standalone. To participate in the maze heist, it needs to speak the A2A (Agent-to-Agent) protocol - both as a server (so other claws and the game server can talk to it) and as a client (so it can talk to peers). We'll implement A2A manually using JSON-RPC 2.0 over HTTP - this is more educational than using an SDK because you see the raw protocol.

## Step 1: Define the A2A types

Create an `a2a` package with the core types. No external dependencies - this is just Go structs and `net/http`:

- `AgentCard` - describes an agent's identity and capabilities
- `Task` - a unit of work with a lifecycle
- `Message` - a single-turn communication
- `Part` - the smallest content unit (text, file, data)

### Acceptance criteria
- [ ] The SDK is installed and importable
- [ ] You can explain in one sentence each: AgentCard, Task, Message, Part

### Stop here
Make sure you understand the core concepts before writing code.

## Step 2: Agent Card

Create an Agent Card for your claw:

- Name: your claw's name (from the system prompt)
- Description: what your claw can do
- URL: `http://localhost:{port}` (the claw's HTTP address)
- Skills: list the claw's capabilities (file operations, command execution, memory, scheduling)
- Serve the Agent Card as JSON at `/.well-known/agent-card.json` on your existing HTTP server

### Acceptance criteria
- [ ] `GET http://localhost:8080/.well-known/agent-card.json` returns a valid Agent Card
- [ ] The Agent Card includes the claw's name, description, and skills
- [ ] The URL in the card matches the claw's actual address

### Stop here
Test with `curl http://localhost:8080/.well-known/agent-card.json | jq .` - verify the card is valid JSON with all required fields.

## Step 3: A2A message handler

Add an A2A server endpoint to handle incoming messages from other agents:

- Use `a2asrv.NewHandler()` from the SDK to create a handler
- Register it on your HTTP server (typically at `/a2a` or as the SDK recommends)
- When a message arrives from another agent:
  1. Extract the text content from the message parts
  2. Feed it into your agent loop as if a user sent it
  3. Collect the agent's response
  4. Return it as an A2A response with text parts

For now, treat incoming A2A messages the same as user messages - the agent processes them and responds.

### Acceptance criteria
- [ ] The A2A endpoint is registered on the HTTP server
- [ ] Incoming A2A messages are processed by the agent
- [ ] Responses are returned in proper A2A format
- [ ] The existing web UI and CLI still work (A2A is an additional interface, not a replacement)

### Stop here
You'll test this with a peer in the next step. For now, verify the endpoint starts without errors.
