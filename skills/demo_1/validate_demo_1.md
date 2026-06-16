# Validation Skill: Demo 1 Checkpoint

This validates the end state of Demo 1 (3 steps: agent loop + tools, streaming, more tools). Your code should match checkpoint_4_full_agent.

Run the following checks and report results as a checklist. For any failures, explain what's wrong and suggest a fix.

## Checks

1. `go build ./...` succeeds with no errors
2. `go vet ./...` reports no warnings
3. `go run -race .` starts without immediate race detector warnings (start and immediately Ctrl+C)
4. A `Tool` interface exists with methods: `Name() string`, `Description() string`, `Schema() map[string]any`, `Execute(ctx context.Context, params json.RawMessage) (string, error)`
5. A tool registry exists that can register and look up tools by name
6. At least 4 tools are registered: read_file, list_directory, write_file, run_command
7. Each tool has a non-empty description
8. Each tool has a valid JSON schema for its parameters
9. The agent loop sends tools to the LLM in the correct format
10. The agent loop handles tool call responses and feeds results back to the LLM
11. Streaming is implemented (responses appear incrementally, not all at once)
12. Context cancellation is wired up (Ctrl+C triggers graceful shutdown)
13. No hardcoded API keys or URLs in the source code (should come from environment variables)
14. The write_file tool rejects path traversal attempts (paths containing `..` that escape the working directory)

## Report format

```
Demo 1 Validation Results
========================
[PASS] 1. go build succeeds
[PASS] 2. go vet clean
[FAIL] 3. Race detector - found race in streaming.go:42
...

X/14 checks passed
```

For any FAIL, provide the fix.
