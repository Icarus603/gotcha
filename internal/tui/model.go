package tui

import (
	"context"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gotcha/internal/agent"
	"gotcha/internal/app"
	"gotcha/internal/llm"
	"gotcha/internal/platform"
	"gotcha/internal/session"
	"gotcha/internal/storage"
)

type RootModel struct {
	ctx context.Context

	cfg platform.Config
	bus agent.EventBus
	app *app.Service

	// Session management
	sessionID      string
	sessionManager *session.Manager
	sessionContext session.Context

	focus int // 0=input,1=notes

	welcome WelcomePane
	input   InputPane
	notes   NotesPane
	status  StatusPane

	vp            viewport.Model
	mouseEnabled  bool
	selecting     bool
	selectAt      time.Time
	width, height int
	cancelSub     func()
}

func NewRootModel(ctx context.Context, cfg platform.Config) RootModel {
	// Default behavior for backward compatibility - create a default session
	sessionManager := session.NewManager()
	sessionID, _ := sessionManager.CreateNewSession()
	sessionContext, _ := sessionManager.LoadSession(sessionID)

	return NewRootModelWithSession(ctx, cfg, sessionID, sessionManager, sessionContext)
}

func NewRootModelWithSession(ctx context.Context, cfg platform.Config, sessionID string, sessionManager *session.Manager, sessionContext session.Context) RootModel {
	bus := agent.NewMemoryBus(64)
	// Storage
	db, _ := storage.Open(cfg.Paths.DBPath())
	_ = storage.Migrate(db)
	service := app.NewService(db, cfg.Paths)

	// Ensure session exists in app service
	_, _ = service.CreateOrOpenSession(ctx, sessionID, "Session", "")

	// LLM client
	llmClient := llmClientFrom(cfg)

	rm := RootModel{
		ctx:            ctx,
		cfg:            cfg,
		bus:            bus,
		app:            service,
		sessionID:      sessionID,
		sessionManager: sessionManager,
		sessionContext: sessionContext,
		welcome:        NewWelcomePane(mustCwd()),
		input:          NewInputPaneWithSessionAndLLM(bus, sessionID, llmClient),
		notes:          NewNotesPaneWithSession(bus, service, sessionID),
		status:         NewStatusPane(),
	}

	// Restore conversation context if exists
	if len(sessionContext.Conversations) > 0 {
		rm.input.RestoreConversation(sessionContext.Conversations)
	}

	// Set note counter
	rm.notes.SetNoteCount(sessionContext.NoteCount)

	// load prompt.md if present
	if b, err := os.ReadFile("prompt.md"); err == nil {
		rm.input.SetSystemPrompt(string(b))
	}
	rm.vp = viewport.New(0, 0)
	rm.mouseEnabled = true
	return rm
}

func llmClientFrom(cfg platform.Config) llm.Client {
	if cfg.LLM.Provider == "openai" && cfg.LLM.APIKey != "" {
		return llm.NewOpenAI(cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model, cfg.ProxyURL)
	}
	return nil
}

type EventMsg struct{ E agent.Event }

func (m RootModel) Init() tea.Cmd {
	subCmd := (&m).subscribeCmd()
	// Start with mouse enabled for page-level scrolling
	enableMouse := func() tea.Msg { return tea.EnableMouseCellMotion() }
	return tea.Batch(m.input.Init(), m.notes.Init(), subCmd, enableMouse)
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Let viewport process messages first (mouse wheel scrolling, etc.)
	var cmds []tea.Cmd
	var cmd tea.Cmd
	wasBottom := m.vp.AtBottom()
	m.vp, cmd = m.vp.Update(msg)
	cmds = append(cmds, cmd)
	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Auto-detect selection: disable mouse reporting on left press to allow native selection.
		switch msg.Type {
		case tea.MouseLeft:
			if m.mouseEnabled {
				m.mouseEnabled = false
				m.selecting = true
				m.selectAt = now()
				cmds = append(cmds, func() tea.Msg { return tea.DisableMouse() })
				// auto re-enable shortly after to restore wheel scrolling
				cmds = append(cmds, m.autoReenableMouseCmd(1500*time.Millisecond))
			}
		}
		// Do NOT forward mouse to child panes so caret won't move.
		// For mouse events (especially scroll), don't force stick to bottom
		m.updateViewportContent(false)
		return m, tea.Batch(cmds...)
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.focus = (m.focus + 1) % 2
			// Removed F2 toggle; auto detection handles selection vs scroll.
		}
	case autoMouseMsg:
		if !m.mouseEnabled {
			m.mouseEnabled = true
			m.selecting = false
			cmds = append(cmds, func() tea.Msg { return tea.EnableMouseCellMotion() })
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.recalcLayout()
		m.vp.Width = m.width
		if m.vp.Width < 1 {
			m.vp.Width = 1
		}
		m.vp.Height = m.height
		if m.vp.Height < 1 {
			m.vp.Height = 1
		}
		m.updateViewportContent(wasBottom)
	case EventMsg:
		// We no longer show phase counts; ignore agent events for statusline.
		return m, tea.Batch((&m).subscribeCmd(), m.saveSessionCmd())
	case SessionSaveMsg:
		// Session has been saved successfully - no action needed
		return m, nil
	case ChatDoneMsg:
		// Let InputPane handle the message first to stop blinking
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		// Then save session context
		return m, tea.Batch(cmd, m.saveSessionCmd())
	case UserMessageMsg:
		// User sent a message - save session context
		return m, m.saveSessionCmd()
	case SaveSessionMsg:
		// Manual save session command
		return m, m.saveSessionCmd()
	case SaveTranscriptMsg:
		// Manual save transcript command
		return m, m.saveTranscriptCmd()
	}

	// Set focus before updating components so they know their focus state
	if m.focus == 0 {
		m.input.SetFocused(true)
		m.notes.SetFocused(false)
	} else {
		m.input.SetFocused(false)
		m.notes.SetFocused(true)
	}

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)
	m.notes, cmd = m.notes.Update(msg)
	cmds = append(cmds, cmd)
	m.status, cmd = m.status.Update(msg)
	cmds = append(cmds, cmd)

	// Recalculate layout after content changes for dynamic height growth
	m.recalcLayout()
	m.updateViewportContent(wasBottom)
	return m, tea.Batch(cmds...)
}

func (m RootModel) View() string { return m.vp.View() }

// Helpers
func now() time.Time { return time.Now() }

func (m *RootModel) subscribeCmd() tea.Cmd {
	ch, cancel := m.bus.Subscribe(m.ctx, "")
	// store cancel once
	if m.cancelSub == nil {
		m.cancelSub = cancel
	}
	return func() tea.Msg {
		e, ok := <-ch
		if !ok {
			return nil
		}
		return EventMsg{E: e}
	}
}

func mustCwd() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return ""
}

// recalcLayout sets pane sizes based on terminal size and content lines.
func (m *RootModel) recalcLayout() {
	// No page-level height cap: panes grow with content; the outer viewport scrolls.
	const frame = 2 // border + padding top/bottom

	// Research input height grows with content; initial 1 line; no upper cap
	inContent := m.input.ContentLines()
	if inContent < 1 {
		inContent = 1
	}
	notesContent := m.notes.ContentLines()
	if notesContent < 10 {
		notesContent = 10
	}

	inH := inContent + frame
	notesH := notesContent + frame

	paneW := m.width - 2
	if paneW < 10 {
		paneW = m.width - 1
	}
	m.input.SetSize(paneW, inH)
	m.notes.SetSize(paneW, notesH)
}

func (m *RootModel) updateViewportContent(stickBottom bool) {
	// Compose the full page content into the viewport
	paneW := m.width - 2
	if paneW < 1 {
		paneW = m.width - 1
	}
	welcome := m.welcome.View()
	status := m.status.View()
	transcript := m.input.TranscriptViewWithWidth(paneW)
	indicator := m.input.IndicatorView()
	inputV := ResearchBorder.Copy().Width(paneW).Render(m.input.View())
	commandDropdown := m.input.CommandDropdownView()
	notesV := NotesBorder.Copy().Width(paneW).Render(m.notes.View())
	var content string
	if transcript != "" {
		if indicator != "" && commandDropdown != "" {
			content = lipgloss.JoinVertical(lipgloss.Left, welcome, status, transcript, indicator, inputV, commandDropdown, notesV)
		} else if indicator != "" {
			content = lipgloss.JoinVertical(lipgloss.Left, welcome, status, transcript, indicator, inputV, notesV)
		} else if commandDropdown != "" {
			content = lipgloss.JoinVertical(lipgloss.Left, welcome, status, transcript, inputV, commandDropdown, notesV)
		} else {
			content = lipgloss.JoinVertical(lipgloss.Left, welcome, status, transcript, inputV, notesV)
		}
		m.vp.SetContent(content)
		if stickBottom {
			m.vp.GotoBottom()
		}
		return
	}
	if indicator != "" && commandDropdown != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, welcome, status, indicator, inputV, commandDropdown, notesV)
	} else if indicator != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, welcome, status, indicator, inputV, notesV)
	} else if commandDropdown != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, welcome, status, inputV, commandDropdown, notesV)
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left, welcome, status, inputV, notesV)
	}
	m.vp.SetContent(content)
	if stickBottom {
		m.vp.GotoBottom()
	}
}

// Auto re-enable mouse after a timeout
type autoMouseMsg struct{}
type SessionSaveMsg struct{}

func (m RootModel) autoReenableMouseCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return autoMouseMsg{} })
}

// saveSessionCmd periodically saves the session context
func (m *RootModel) saveSessionCmd() tea.Cmd {
	return func() tea.Msg {
		// Update session context with current conversation and note count
		conversations := make([]session.ChatMsg, len(m.input.GetConversation()))
		for i, conv := range m.input.GetConversation() {
			conversations[i] = session.ChatMsg{Role: conv.Role, Text: conv.Text}
		}

		m.sessionContext.Conversations = conversations
		m.sessionContext.NoteCount = m.notes.GetNoteCount()

		// Save session context
		if err := m.sessionManager.SaveSession(m.sessionID, m.sessionContext); err != nil {
			// Could log error or handle it, for now just continue
		}

		return SessionSaveMsg{}
	}
}

// saveTranscriptCmd saves the conversation to a transcript.md file
func (m *RootModel) saveTranscriptCmd() tea.Cmd {
	return func() tea.Msg {
		// Update session context with current conversation
		conversations := make([]session.ChatMsg, len(m.input.GetConversation()))
		for i, conv := range m.input.GetConversation() {
			conversations[i] = session.ChatMsg{Role: conv.Role, Text: conv.Text}
		}

		m.sessionContext.Conversations = conversations
		m.sessionContext.NoteCount = m.notes.GetNoteCount()

		// Save transcript
		if err := m.sessionManager.SaveTranscript(m.sessionID, m.sessionContext); err != nil {
			// Could show error to user, for now just continue
			return SessionSaveMsg{}
		}

		// Update last save index
		if err := m.sessionManager.UpdateLastSaveIndex(m.sessionID, m.sessionContext); err != nil {
			// Could show error to user, for now just continue
		}

		return SessionSaveMsg{}
	}
}
