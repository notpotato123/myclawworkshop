# QA Skill: Review Demo 1 - The Agent Core

Use this skill after completing all steps of Demo 1. Feed these prompts to your coding agent to review and understand the code you've built.

## 1. Architecture visualization

> Generate a mermaid diagram showing the architecture of the agent. Include: the main loop, the tool registry, the individual tools, the LLM client, and the streaming flow. Show how data flows between components.

Review the diagram. Does it match your understanding of the code?

## 2. Explain the agent loop

> Explain the agent loop step by step. What happens from the moment the user types a message to the moment a response appears on screen? Include what happens when the LLM decides to call a tool.

Read the explanation. Can you follow the flow? Is there anything surprising?

## 3. Code review

> Review the codebase for:
> - Error handling: are errors handled consistently? Any places where errors are silently swallowed?
> - Race conditions: is there any shared mutable state accessed from multiple goroutines?
> - Resource leaks: are all readers/connections properly closed?
> - Go idioms: does the code follow Go conventions? (error returns, naming, package structure)
>
> List specific findings with file and line references.

Go through each finding. Do you agree? Fix any real issues before proceeding.

## 4. Improvement ideas

> Suggest three improvements to this codebase. For each: describe the improvement, explain why it matters, and rate its priority (high/medium/low). Don't implement them - just describe them.

Think about which of these you might add later.

## 5. The patterns you just built

> The Tool interface and Registry you implemented are the same core patterns used by Claude Code, zot, Cursor, and every other coding agent harness. Compare your Tool interface with how a production agent harness would implement the same thing. What's identical? What would a production system add on top?

This connects to the broader workshop narrative - the agent core you just built is the shared foundation underneath both claws (always-on personal agents) and harnesses (coding agent frameworks).
