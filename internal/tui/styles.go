package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary     = lipgloss.Color("#0E87B0")
	colorText        = lipgloss.Color("#C7C7C7")
	colorWhite       = lipgloss.Color("#FFFFFF")
	colorGray        = lipgloss.Color("#999999")
	colorPlaceholder = lipgloss.Color("#6C6C6C")
	// Slightly more cyan than #0978B8, just a tiny shift
	colorResearch = lipgloss.Color("#0A82BD")
	colorNotes    = lipgloss.Color("#0A82BD")
	colorWelcome  = lipgloss.Color("#0A82BD")
	// Lighter blue for command menu
	colorLightBlue = lipgloss.Color("#7BB3E0")

	PaneBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 1)

	Title       = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	Muted       = lipgloss.NewStyle().Faint(true)
	Gray        = lipgloss.NewStyle().Foreground(colorGray)
	Placeholder = lipgloss.NewStyle().Foreground(colorPlaceholder)
	PrimaryBold = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	White       = lipgloss.NewStyle().Foreground(colorWhite)

	WelcomeBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorWelcome).
			Padding(0, 1).
			PaddingRight(4)

	Accent        = lipgloss.NewStyle().Foreground(colorPrimary)
	WelcomeAccent = lipgloss.NewStyle().Foreground(colorWelcome)
	WelcomeBold   = lipgloss.NewStyle().Foreground(colorWelcome).Bold(true)
	Text          = lipgloss.NewStyle().Foreground(colorText)
	Strong        = lipgloss.NewStyle().Foreground(colorWhite).Bold(true)

	// Per-pane theming
	ResearchBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4C4C4C")).
			Padding(0, 1)
	ResearchTitle = lipgloss.NewStyle().Bold(true).Foreground(colorResearch)

	NotesBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4C4C4C")).
			Padding(0, 1)
	NotesTitle = lipgloss.NewStyle().Bold(true).Foreground(colorNotes)

	ChatBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4C4C4C")).
			Padding(0, 1)
	ChatTitle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)

	// Command indicator style (matches slash command dropdown)
	CommandIndicator = lipgloss.NewStyle().Foreground(colorLightBlue)
)
