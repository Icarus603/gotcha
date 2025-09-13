package tui

// NewTaskMsg is emitted when a new research task has been created from input.
type NewTaskMsg struct{ Title string }

// Chat streaming messages from InputPane's LLM call.
type ChatDeltaMsg struct{ Delta string }
type ChatDoneMsg struct{}
type ChatErrMsg struct{ Err string }
