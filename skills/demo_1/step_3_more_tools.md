# Skill: More Tools

> **Pacing:** Feed this skill to your agent ONE step at a time. After each "Stop here" marker, wait for the instructor before continuing to the next step.

## Context
Our agent can stream, use tools, and maintain conversation. But it only has read_file and list_directory - it can observe but not act. A real agent needs to be able to change things. Let's add write_file and run_command.

## Step 1: write_file tool

Create a write_file tool:
- Parameters: `{"path": "string", "content": "string"}`
- Writes content to the given path, creating directories as needed
- Returns confirmation with the number of bytes written
- Safety: refuse to write outside the current working directory (no path traversal via `..`)

### Acceptance criteria
- [ ] Tool writes files correctly
- [ ] Creates parent directories if they don't exist
- [ ] Rejects paths that escape the working directory
- [ ] Returns a clear confirmation message

### Stop here
Test: Ask your agent to "create a file called hello.txt with the contents 'Hello from my claw!'". Verify the file exists.

## Step 2: run_command tool

Create a run_command tool:
- Parameters: `{"command": "string"}` (the shell command to run)
- Executes the command using `exec.CommandContext` with the agent's context
- Captures both stdout and stderr
- Has a timeout to prevent hanging (you decide what's reasonable for a personal agent)
- Returns the output and the exit code in a format that helps the LLM understand what happened

Security considerations:
- This is a powerful tool. For the workshop, it's fine to allow all commands.
- In production you'd want sandboxing, allowlists, etc. (we'll discuss this in the review)

### Acceptance criteria
- [ ] Commands execute and output is returned to the LLM
- [ ] Both stdout and stderr are captured
- [ ] Long-running commands are killed after the timeout
- [ ] The exit code is included in the result
- [ ] The command runs in the current working directory

### Stop here
Test: Ask your agent to "run go version". Then ask it to "run the tests" (it should run `go test ./...`). Verify the output is correct.

## Step 3: Test the full toolkit

Now test all four tools together in a real workflow. Ask your agent to:

1. "List the files in this directory"
2. "Read the main.go file and tell me what it does"
3. "Create a file called AGENTS.md with a short description of this project"
4. "Run `cat AGENTS.md` to verify the file was created correctly"

This tests the full loop: observe (list, read) -> act (write) -> verify (run command).

### Acceptance criteria
- [ ] All four tools work together in a multi-step conversation
- [ ] The agent can chain tools to accomplish multi-step tasks
- [ ] The agent makes reasonable decisions about which tool to use

### Stop here
Clean up any test files created. This is the end of Demo 1 - you have a working agent with a tool system, streaming, and four tools. Run the validation skill next.
