// Package msgs defines the Message type shared between the agent and tools
// packages to avoid import cycles.
package msgs

// Message is a unit of work for the agent loop.
type Message struct {
	Content string
	Source  string           // "cli", "web", "scheduler", "a2a", "inbox"
	ReplyTo func(string)     // called with each response text chunk
	Done    func()           // called once the full response is complete
	OnTool  func(name, status string)
}
