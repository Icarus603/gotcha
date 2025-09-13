package agent

import (
    "context"
    "time"
)

// StartSimulatedResearch kicks off a fake background workflow that emits phased events
// to demonstrate statusline progress. Intended for MVP before real connectors.
func StartSimulatedResearch(bus EventBus, sessionID, query string) {
    go func() {
        ctx := context.Background()
        phases := []Phase{PhaseSearch, PhaseFetch, PhaseExtract, PhaseOutline, PhaseSection, PhaseCompose}
        totals := []int{5, 5, 5, 1, 3, 1}
        // Queue all tasks first
        for i, ph := range phases {
            bus.Publish(ctx, Event{SessionID: sessionID, TaskID: "", Phase: ph, Type: "queued", Progress: Progress{Done: 0, Total: totals[i]}, At: time.Now()})
        }
        // Simulate work
        for i, ph := range phases {
            for d := 1; d <= totals[i]; d++ {
                time.Sleep(250 * time.Millisecond)
                bus.Publish(ctx, Event{SessionID: sessionID, TaskID: "", Phase: ph, Type: "progress", Progress: Progress{Done: d, Total: totals[i]}, At: time.Now(), Meta: map[string]any{"query": query}})
            }
            bus.Publish(ctx, Event{SessionID: sessionID, TaskID: "", Phase: ph, Type: "done", Progress: Progress{Done: totals[i], Total: totals[i]}, At: time.Now()})
        }
    }()
}

