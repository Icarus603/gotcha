package agent

import (
    "context"
    "time"
)

type Phase string

const (
    PhaseSearch  Phase = "search"
    PhaseFetch   Phase = "fetch"
    PhaseExtract Phase = "extract"
    PhaseOutline Phase = "outline"
    PhaseSection Phase = "section"
    PhaseCompose Phase = "compose"
)

type Progress struct {
    Done, Total int
    Elapsed      time.Duration
}

// Event is emitted by background workers and consumed by the TUI.
type Event struct {
    SessionID string
    TaskID    string
    Phase     Phase
    Type      string            // queued|started|progress|done|error
    Progress  Progress          // optional
    Meta      map[string]any    // url, title, model, cost tokens, etc.
    Err       string            // user-safe error
    At        time.Time
}

// EventBus is a minimal pub/sub for UI updates.
type EventBus interface {
    Publish(ctx context.Context, e Event)
    Subscribe(ctx context.Context, sessionID string) (<-chan Event, func())
}

// memoryBus is a minimal in-memory implementation suitable for MVP/testing.
type memoryBus struct{
    ch chan Event
}

func NewMemoryBus(buffer int) EventBus {
    return &memoryBus{ch: make(chan Event, buffer)}
}

func (b *memoryBus) Publish(_ context.Context, e Event) { b.ch <- e }
func (b *memoryBus) Subscribe(_ context.Context, _ string) (<-chan Event, func()) {
    return b.ch, func() {}
}

