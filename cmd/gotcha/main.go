package main

import (
    "context"
    "fmt"
    "os"
    "time"

    tea "github.com/charmbracelet/bubbletea"

    "gotcha/internal/platform"
    "gotcha/internal/tui"
)

func main() {
    cfg := platform.LoadConfig()
    ctx := context.Background()

    m := tui.NewRootModel(ctx, cfg)
    // Enable mouse cell motion for wheel scrolling; F2 toggles to allow native selection
    p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
    // Ensure graceful shutdown time for background routines (none yet).
    time.Sleep(50 * time.Millisecond)
}
