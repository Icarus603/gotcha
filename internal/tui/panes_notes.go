package tui

import (
    "fmt"
    "os"
    "path/filepath"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/lipgloss"

    "gotcha/internal/agent"
    "gotcha/internal/app"
)

type NotesPane struct {
    ta        textarea.Model
    focused   bool
    bus       agent.EventBus
    svc       *app.Service
    sessionID string
    noteCount int // Counter for generating sequential note files
}

type NoteSaveResultMsg struct{ Success, Error string }

func NewNotesPane(bus agent.EventBus) NotesPane {
    ta := textarea.New()
    ta.Placeholder = "Take notes here. Enter saves. Shift+Enter newline."
    ta.ShowLineNumbers = false
    ta.Prompt = ""
    // Remove cursor line background to match app background
    f, b := textarea.DefaultStyles()
    f.CursorLine = lipgloss.NewStyle()
    b.CursorLine = lipgloss.NewStyle()
    f.Placeholder = Placeholder
    b.Placeholder = Placeholder
    ta.FocusedStyle = f
    ta.BlurredStyle = b
    return NotesPane{ta: ta, bus: bus}
}

func NewNotesPaneWithSession(bus agent.EventBus, svc *app.Service, sessionID string) NotesPane {
    p := NewNotesPane(bus)
    p.svc = svc
    p.sessionID = sessionID
    return p
}

func (p NotesPane) Init() tea.Cmd { return textarea.Blink }

func (p *NotesPane) SetFocused(f bool) {
    p.focused = f
    if f { p.ta.Focus() } else { p.ta.Blur() }
}

func (p *NotesPane) SetSize(paneWidth, paneHeight int) {
    innerW := paneWidth - 4
    if innerW < 10 { innerW = 10 }
    innerH := paneHeight - 2
    if innerH < 3 { innerH = 3 }
    p.ta.SetWidth(innerW)
    p.ta.SetHeight(innerH)
}

func (p NotesPane) Update(msg tea.Msg) (NotesPane, tea.Cmd) {
    switch m := msg.(type) {
    case NoteSaveResultMsg:
        // Clear textarea after successful save
        if m.Success != "" {
            p.ta.SetValue("")
        }
        return p, nil
    case tea.KeyMsg:
        if !p.focused { break }
        // Enter saves; Shift+Enter inserts newline.
        if m.String() == "enter" {
            val := p.ta.Value()
            if val != "" {
                return p, p.saveNoteAsFile(val)
            }
            return p, nil
        }
        if m.String() == "shift+enter" || m.String() == "alt+enter" || m.String() == "ctrl+j" {
            p.ta.SetValue(p.ta.Value() + "\n")
            return p, nil
        }
    // Mouse wheel scrolling is handled by the page-level viewport
    }
    var cmd tea.Cmd
    p.ta, cmd = p.ta.Update(msg)
    return p, cmd
}

func (p NotesPane) View() string {
    title := NotesTitle.Render("Notes")
    return title + "\n" + p.ta.View()
}

func (p NotesPane) sessionIDOrDefault() string {
    if p.sessionID != "" { return p.sessionID }
    return "dev-session"
}

// ContentLines returns the number of lines in the notes textarea value.
func (p NotesPane) ContentLines() int {
    v := p.ta.Value()
    if v == "" { return 1 }
    c := 1
    for i := 0; i < len(v); i++ { if v[i] == '\n' { c++ } }
    return c
}

// SetNoteCount sets the current note counter (for session restoration)
func (p *NotesPane) SetNoteCount(count int) {
    p.noteCount = count
}

// GetNoteCount returns the current note counter
func (p *NotesPane) GetNoteCount() int {
    return p.noteCount
}

// saveNoteAsFile saves the note content as a separate markdown file
func (p *NotesPane) saveNoteAsFile(content string) tea.Cmd {
    p.noteCount++
    sessionID := p.sessionIDOrDefault()

    return func() tea.Msg {
        // Get current working directory and create .gotcha path
        cwd, err := os.Getwd()
        if err != nil {
            return NoteSaveResultMsg{Error: fmt.Sprintf("Failed to get working directory: %v", err)}
        }

        sessionDir := filepath.Join(cwd, ".gotcha", "sessions", sessionID)
        if err := os.MkdirAll(sessionDir, 0o755); err != nil {
            return NoteSaveResultMsg{Error: fmt.Sprintf("Failed to create session directory: %v", err)}
        }

        // Generate filename with sequential numbering
        filename := fmt.Sprintf("note_%d.md", p.noteCount)
        notePath := filepath.Join(sessionDir, filename)

        // Create note content with timestamp header
        noteContent := fmt.Sprintf("# Note %d\n\n**Created:** %s\n\n---\n\n%s\n",
            p.noteCount,
            time.Now().Format(time.RFC3339),
            content)

        if err := os.WriteFile(notePath, []byte(noteContent), 0o644); err != nil {
            return NoteSaveResultMsg{Error: fmt.Sprintf("Failed to save note: %v", err)}
        }

        return NoteSaveResultMsg{Success: fmt.Sprintf("Note saved to %s", filename)}
    }
}
