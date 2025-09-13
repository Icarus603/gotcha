package tui

import (
    "fmt"
    "strings"
    "github.com/charmbracelet/lipgloss"
    "github.com/mattn/go-runewidth"
)

type WelcomePane struct {
    cwd string
}

func NewWelcomePane(cwd string) WelcomePane { return WelcomePane{cwd: cwd} }

func (p WelcomePane) Init() func() any { return nil }
func (p WelcomePane) Update(msg any) (WelcomePane, func() any) { return p, nil }

func (p WelcomePane) View() string {
    // Title with leading bullet
    bullet := "◉ "
    title := WelcomeAccent.Render(bullet) + Text.Render("Welcome to ") + Strong.Render("Gotcha") + Text.Render("!")
    // Compute indent equal to visual width of the bullet, align sublines under the title text
    indent := strings.Repeat(" ", runewidth.StringWidth(bullet))
    sub := indent + Gray.Render("your ") + WelcomeBold.Render("Note-taking Copilot")
    cwd := indent + Gray.Render(fmt.Sprintf("cwd: %s", p.cwd))
    // Let the box size to content width naturally (no forced width)
    content := lipgloss.JoinVertical(lipgloss.Left, title, "", sub, "", cwd)
    return WelcomeBox.Render(content)
}
