package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "time"

    tea "github.com/charmbracelet/bubbletea"

    "gotcha/internal/platform"
    "gotcha/internal/session"
    "gotcha/internal/tui"
)

func main() {
    var (
        resume   = flag.Bool("resume", false, "Show session selection menu")
        continue_flag = flag.Bool("continue", false, "Continue the most recent session")
    )
    flag.Parse()

    cfg := platform.LoadConfig()
    ctx := context.Background()

    sessionManager := session.NewManager()

    var sessionID string
    var err error

    switch {
    case *resume:
        sessionID, err = runSessionSelector(sessionManager)
        if err != nil {
            fmt.Fprintf(os.Stderr, "error selecting session: %v\n", err)
            os.Exit(1)
        }
        if sessionID == "" {
            // User cancelled selection
            os.Exit(0)
        }

    case *continue_flag:
        sessionID, err = sessionManager.GetLastSession()
        if err != nil {
            fmt.Fprintf(os.Stderr, "error getting last session: %v\n", err)
            os.Exit(1)
        }
        if sessionID == "" {
            // No previous session, create new one
            sessionID, err = sessionManager.CreateNewSession()
            if err != nil {
                fmt.Fprintf(os.Stderr, "error creating session: %v\n", err)
                os.Exit(1)
            }
        }

    default:
        // Create new session
        sessionID, err = sessionManager.CreateNewSession()
        if err != nil {
            fmt.Fprintf(os.Stderr, "error creating session: %v\n", err)
            os.Exit(1)
        }
    }

    // Load session context
    sessionContext, err := sessionManager.LoadSession(sessionID)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error loading session: %v\n", err)
        os.Exit(1)
    }

    // Create root model with session
    m := tui.NewRootModelWithSession(ctx, cfg, sessionID, sessionManager, sessionContext)

    // Enable mouse cell motion for wheel scrolling
    p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithoutSignalHandler())
    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }

    // Ensure graceful shutdown time for background routines
    time.Sleep(50 * time.Millisecond)
}

func runSessionSelector(sessionManager *session.Manager) (string, error) {
    sessions, err := sessionManager.ListSessions()
    if err != nil {
        return "", err
    }

    if len(sessions) == 0 {
        fmt.Println("Nothing to resume from - no existing sessions found.")
        fmt.Println("Use 'gotcha' to start a new session.")
        return "", nil // Return empty string to indicate no session selected
    }

    // Create session selector UI
    selector := tui.NewSessionSelector(sessions)
    p := tea.NewProgram(selector, tea.WithAltScreen(), tea.WithoutSignalHandler())

    result, err := p.Run()
    if err != nil {
        return "", err
    }

    if finalModel, ok := result.(tui.SessionSelector); ok {
        selectedID := finalModel.SelectedSessionID()
        if selectedID == "NEW" {
            return sessionManager.CreateNewSession()
        }
        return selectedID, nil
    }

    return "", fmt.Errorf("unexpected model type")
}
