package tui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// StatusPane renders only the current research task titles on one subtle line.
type StatusPane struct { tasks []string }

func NewStatusPane() StatusPane { return StatusPane{tasks: []string{}} }

func (p StatusPane) Init() tea.Cmd { return nil }

func (p StatusPane) Update(msg tea.Msg) (StatusPane, tea.Cmd) {
    switch m := msg.(type) {
    case NewTaskMsg:
        if m.Title != "" { p.tasks = append([]string{m.Title}, p.tasks...) }
    }
    return p, nil
}

func (p StatusPane) View() string {
    if len(p.tasks) == 0 { return "" }
    max := 3
    if len(p.tasks) < max { max = len(p.tasks) }
    line := p.tasks[0]
    for i := 1; i < max; i++ { line += " • " + p.tasks[i] }
    return lipgloss.NewStyle().Foreground(Gray.GetForeground()).Render(line)
}
