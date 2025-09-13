package tui

import (
    "fmt"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/lipgloss"
    "time"
    "gotcha/internal/agent"
    "gotcha/internal/app"
    "context"
)

type NotesPane struct {
    ta      textarea.Model
    focused bool
    saved   []string
    bus     agent.EventBus
    svc     *app.Service
    sessionID string
}

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
    case tea.KeyMsg:
        if !p.focused { break }
        // Enter saves; Shift+Enter inserts newline.
        if m.String() == "enter" {
            val := p.ta.Value()
            if val != "" {
                p.saved = append(p.saved, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), val))
                if p.svc != nil {
                    _ = p.svc.SaveNote(context.Background(), p.sessionIDOrDefault(), val)
                }
                p.ta.SetValue("")
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
    list := ""
    for i := len(p.saved) - 1; i >= 0 && i >= len(p.saved)-3; i-- {
        list += "\n• " + p.saved[i]
    }
    return title + "\n" + p.ta.View() + list
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
