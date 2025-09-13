package llm

import "context"

type StreamHandler func(delta string)

type Request struct {
    System      string
    Prompt      string
    MaxTokens   int
    Temperature float64
    Stop        []string
    Model       string
    // Optional tool usage (Responses API)
    Tools       []map[string]any
    ToolChoice  string
    Include     []string
    ReasoningEffort  string // low|medium|high
    ReasoningSummary string // e.g., auto|concise|detailed
}

type Response struct {
    Text             string
    PromptTokens     int
    CompletionTokens int
}

type Client interface {
    Name() string
    Complete(ctx context.Context, req Request, onToken StreamHandler) (Response, error)
}
