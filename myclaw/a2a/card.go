package a2a

import "fmt"

// NewClawCard builds the AgentCard for the Claw agent.
func NewClawCard(port string) AgentCard {
	return AgentCard{
		Name:        "Claw",
		Description: "An autonomous AI assistant with persistent memory, task scheduling, and local system access.",
		URL:         fmt.Sprintf("http://localhost:%s", port),
		Version:     "0.1.0",
		Skills: []string{
			"read_file",
			"write_file",
			"list_directory",
			"run_command",
			"remember",
			"recall",
			"schedule",
			"discover_peer",
			"ask_peer",
			"broadcast",
			"find_peer_with_skill",
		},
	}
}
