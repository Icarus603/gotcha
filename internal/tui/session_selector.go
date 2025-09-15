package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gotcha/internal/session"
)

type SessionSelector struct {
	sessions        []session.SessionInfo
	cursor          int
	selectedSession string
	cancelled       bool
}

func NewSessionSelector(sessions []session.SessionInfo) SessionSelector {
	return SessionSelector{
		sessions: sessions,
		cursor:   0,
	}
}

func (m SessionSelector) Init() tea.Cmd {
	return nil
}

func (m SessionSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter", " ":
			if len(m.sessions) > 0 && m.cursor >= 0 && m.cursor < len(m.sessions) {
				m.selectedSession = m.sessions[m.cursor].ID
			}
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}
		case "n":
			// Create new session by returning empty session ID
			m.selectedSession = "NEW"
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m SessionSelector) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0A82BD")).
		Bold(true).
		Padding(1, 0)

	b.WriteString(titleStyle.Render("📚 Gotcha Session Manager"))
	b.WriteString("\n\n")

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Italic(true)

	instructions := "Use ↑/↓ to navigate • Enter to select • 'n' for new session • 'q' to quit"
	b.WriteString(instructionStyle.Render(instructions))
	b.WriteString("\n\n")

	if len(m.sessions) == 0 {
		noSessionsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#999999")).
			Italic(true)

		b.WriteString(noSessionsStyle.Render("No existing sessions found."))
		b.WriteString("\n\n")
		b.WriteString("Press 'n' to create a new session or 'q' to quit.")
		return b.String()
	}

	// Sessions list
	for i, sess := range m.sessions {
		var sessionLine string

		// Format creation time
		timeStr := sess.CreatedAt.Format("2006-01-02 15:04")

		// Calculate session age
		age := time.Since(sess.CreatedAt)
		var ageStr string
		if age < time.Hour {
			ageStr = fmt.Sprintf("%.0fm ago", age.Minutes())
		} else if age < 24*time.Hour {
			ageStr = fmt.Sprintf("%.0fh ago", age.Hours())
		} else {
			days := int(age.Hours() / 24)
			ageStr = fmt.Sprintf("%dd ago", days)
		}

		// Session info
		title := sess.Title
		if title == "" {
			title = "Untitled Session"
		}

		sessionInfo := fmt.Sprintf("%s  •  %s  •  %s", sess.ID, timeStr, ageStr)

		if i == m.cursor {
			// Selected session
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0A82BD")).
				Background(lipgloss.Color("#2D2D2D")).
				Bold(true).
				Padding(0, 1)

			sessionLine = selectedStyle.Render("► " + sessionInfo)
		} else {
			// Unselected session
			normalStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#C7C7C7")).
				Padding(0, 1)

			sessionLine = normalStyle.Render("  " + sessionInfo)
		}

		b.WriteString(sessionLine)
		b.WriteString("\n")
	}

	// Footer options
	b.WriteString("\n")
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Italic(true)

	footer := "💡 Tip: Press 'n' to create a new session"
	b.WriteString(footerStyle.Render(footer))

	return b.String()
}

func (m SessionSelector) SelectedSessionID() string {
	if m.cancelled {
		return ""
	}
	if m.selectedSession == "NEW" {
		return "NEW"
	}
	return m.selectedSession
}