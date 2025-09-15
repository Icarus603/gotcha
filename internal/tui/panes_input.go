package tui

import (
    "context"
    "strings"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/lipgloss"
    "gotcha/internal/agent"
    "gotcha/internal/llm"
    "gotcha/internal/session"
    "time"
    "encoding/json"
    "math/rand"
)

type InputPane struct {
    ta       textarea.Model
    focused  bool
    bus      agent.EventBus
    sessionID string
    client   llm.Client
    output   string
    working  bool
    errText  string

    // inline transcript above the textarea
    convo   []chatMsg
    // streaming state
    streamCh    chan string
    streamErrCh chan error
    assistantIdx int
    streaming    bool
    blinkOn      bool

    // tool streaming state (e.g., web_search)
    toolActive   bool
    toolBuffer   string
    toolIdx      int

    // system prompt override
    sysPrompt string
    // reasoning summary streaming
    reasonActive bool
    reasonIdx    int

    // buffer to hold assistant text until completion (non-stream display)
    pendingAssistant string

    // working indicator
    indicatorFrame int
    
    // command system
    showCommands    bool
    commandFilter   string
    selectedCmd     int
    commandMode     string // "" | "model_selection"
    selectedModel   string
    modelOptions    []ModelOption
    selectedOption  int
}

func NewInputPane(bus agent.EventBus) InputPane { // deprecated constructor kept for compat
    ta := textarea.New()
    ta.Placeholder = "Ask your agent to look into something. Enter submit. Shift+Enter newline."
    ta.Prompt = ""
    ta.ShowLineNumbers = false
    // Remove cursor line background to match app background
    f, b := textarea.DefaultStyles()
    f.CursorLine = lipgloss.NewStyle()
    b.CursorLine = lipgloss.NewStyle()
    f.Placeholder = Placeholder
    b.Placeholder = Placeholder
    ta.FocusedStyle = f
    ta.BlurredStyle = b
    ta.Focus()
    return InputPane{ta: ta, bus: bus}
}

func NewInputPaneWithSession(bus agent.EventBus, sessionID string) InputPane {
    p := NewInputPane(bus)
    p.sessionID = sessionID
    return p
}

func NewInputPaneWithSessionAndLLM(bus agent.EventBus, sessionID string, client llm.Client) InputPane {
    p := NewInputPaneWithSession(bus, sessionID)
    p.client = client
    return p
}

func (p *InputPane) SetSystemPrompt(s string) { p.sysPrompt = strings.TrimSpace(s) }

func (p InputPane) Init() tea.Cmd { return textarea.Blink }

func (p *InputPane) SetFocused(f bool) {
    p.focused = f
    if f { p.ta.Focus() } else { p.ta.Blur() }
}

func (p *InputPane) SetSize(paneWidth, paneHeight int) {
    innerW := paneWidth - 4 // 2 border + 2 padding
    if innerW < 10 { innerW = 10 }
    innerH := paneHeight - 2
    // Keep content area at least 1 line tall
    if innerH < 1 { innerH = 1 }
    p.ta.SetWidth(innerW)
    p.ta.SetHeight(innerH)
}

type llmDoneMsg struct{ text string; err error }

func (p InputPane) Update(msg tea.Msg) (InputPane, tea.Cmd) {
    switch m := msg.(type) {
    case llmDoneMsg:
        p.working = false
        if m.err != nil { p.errText = m.err.Error() } else { p.output = m.text }
        return p, nil
    case ChatDeltaMsg:
        if strings.HasPrefix(m.Delta, "\x00WEBSEARCH:") {
            payload := strings.TrimPrefix(m.Delta, "\x00WEBSEARCH:")
            q := extractQuery(payload)
            if !p.toolActive {
                p.toolActive = true
                if q == "" { p.toolBuffer = "Searching…" } else { p.toolBuffer = q }
                p.convo = append(p.convo, chatMsg{Role: "tool", Text: p.toolBuffer})
                p.toolIdx = len(p.convo) - 1
            } else {
                if q != "" { p.toolBuffer = q }
                if p.toolIdx >= 0 && p.toolIdx < len(p.convo) {
                    if p.toolBuffer == "" { p.convo[p.toolIdx].Text = "Searching…" } else { p.convo[p.toolIdx].Text = p.toolBuffer }
                }
            }
        } else if strings.HasPrefix(m.Delta, "\x00REASON:") {
            content := strings.TrimPrefix(m.Delta, "\x00REASON:")
            if !p.reasonActive {
                // First reasoning output - create reason message
                p.reasonActive = true
                p.convo = append(p.convo, chatMsg{Role: "reason", Text: ""})
                p.reasonIdx = len(p.convo) - 1
            }
            if p.reasonIdx >= 0 && p.reasonIdx < len(p.convo) {
                p.convo[p.reasonIdx].Text += content
            }
        } else {
            // Regular text output - create assistant message only when needed
            p.pendingAssistant += m.Delta
            if p.assistantIdx < 0 {
                // First text output - create assistant message
                p.appendAssistant("")
            }
            if p.assistantIdx >= 0 && p.assistantIdx < len(p.convo) {
                p.convo[p.assistantIdx].Text = p.pendingAssistant
            }
        }
        // keep listening for more deltas
        return p, p.subscribeStreamCmd(p.streamCh, p.streamErrCh)
    case ChatErrMsg:
        if p.assistantIdx >= 0 && p.assistantIdx < len(p.convo) {
            p.convo[p.assistantIdx].Text += "\n(error) " + m.Err
        }
        p.streaming = false
        p.streamCh, p.streamErrCh = nil, nil
        p.toolActive = false
        return p, nil
    case ChatDoneMsg:
        p.streaming = false
        p.streamCh, p.streamErrCh = nil, nil
        // Final update of assistant message if it exists
        if p.assistantIdx >= 0 && p.assistantIdx < len(p.convo) {
            p.convo[p.assistantIdx].Text = p.pendingAssistant
        }
        // Reset streaming state
        p.reasonActive = false
        p.toolActive = false
        p.pendingAssistant = ""
        // Reset indices so old messages don't blink on new streams
        p.assistantIdx = -1
        p.toolIdx = -1
        p.reasonIdx = -1
        return p, nil
    case blinkMsg:
        if p.streaming {
            p.blinkOn = !p.blinkOn
            p.indicatorFrame++ // Advance indicator animation 
            return p, p.blinkCmd()
        }
        return p, nil
    case tea.KeyMsg:
        if !p.focused { break }

        // Handle command system first
        if p.showCommands || p.commandMode != "" {
            newP, cmd := p.handleCommandKeys(m)
            return *newP, cmd
        }

        // Shift+Enter inserts newline; also support alt+enter/ctrl+j as fallbacks.
        if m.String() == "shift+enter" || m.String() == "alt+enter" || m.String() == "ctrl+j" {
            p.ta.SetValue(p.ta.Value() + "\n")
            return p, nil
        }


        if m.String() == "enter" {
            query := p.ta.Value()
            if query == "" { return p, nil }
            // Chat streaming (default)
            p.appendUser(query)
            p.pendingAssistant = ""
            // Reset all streaming state
            p.reasonActive = false
            p.toolActive = false
            p.assistantIdx = -1  // No assistant message yet
            p.ta.SetValue("")
            if p.client == nil {
                p.appendAssistant("(No LLM configured)")
                return p, nil
            }
            cmd := p.startChatStreamCmd(query)
            return p, cmd
        }
    // Mouse wheel scrolling is handled by the page-level viewport
    }
    var cmd tea.Cmd
    p.ta, cmd = p.ta.Update(msg)

    // Check for command trigger after every text update
    value := p.ta.Value()
    if strings.HasPrefix(value, "/") {
        p.showCommands = true
        p.commandFilter = value
        p.selectedCmd = 0
    } else {
        // Clear command mode if not typing a command
        p.showCommands = false
        p.commandMode = ""
    }

    return p, cmd
}

func (p InputPane) View() string {
    return p.ta.View()
}

// Get command dropdown view (called from root model)
func (p InputPane) CommandDropdownView() string {
    if p.showCommands || p.commandMode != "" {
        return p.renderCommandDropdown()
    }
    return ""
}

// Get indicator view (called from root model)
func (p InputPane) IndicatorView() string {
    return p.renderIndicator()
}

// Render command dropdown menu
func (p InputPane) renderCommandDropdown() string {
    if p.commandMode == "model_selection" {
        return p.renderModelSelection()
    } else if p.showCommands {
        return p.renderCommandList()
    }
    return ""
}

// Render command list dropdown
func (p InputPane) renderCommandList() string {
    commands := p.getFilteredCommands()
    if len(commands) == 0 {
        return ""
    }

    var rows []string
    for i, cmd := range commands {
        var row string
        if i == p.selectedCmd {
            // Selected: light blue text, bold name, dim description
            nameStyle := lipgloss.NewStyle().Foreground(colorLightBlue).Bold(true)
            descStyle := lipgloss.NewStyle().Foreground(colorLightBlue).Faint(true)
            row = nameStyle.Render(cmd.Name) + strings.Repeat(" ", 4) + descStyle.Render(cmd.Description)
        } else {
            // Unselected: bold name, normal description with more spacing
            nameStyle := lipgloss.NewStyle().Foreground(colorText).Bold(true)
            descStyle := lipgloss.NewStyle().Foreground(colorGray)
            row = nameStyle.Render(cmd.Name) + strings.Repeat(" ", 4) + descStyle.Render(cmd.Description)
        }
        rows = append(rows, row)
    }

    content := strings.Join(rows, "\n")
    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("#4C4C4C")).
        Padding(0, 1).
        Render(content)
}

// Render model selection dropdown
func (p InputPane) renderModelSelection() string {
    var rows []string
    for i, option := range p.modelOptions {
        var row string
        if i == p.selectedOption {
            // Selected: light blue text, bold name, dim description
            nameStyle := lipgloss.NewStyle().Foreground(colorLightBlue).Bold(true)
            descStyle := lipgloss.NewStyle().Foreground(colorLightBlue).Faint(true)
            row = nameStyle.Render(option.Name) + strings.Repeat(" ", 4) + descStyle.Render(option.Description)
        } else {
            // Unselected: bold name, normal description with more spacing
            nameStyle := lipgloss.NewStyle().Foreground(colorText).Bold(true)
            descStyle := lipgloss.NewStyle().Foreground(colorGray)
            row = nameStyle.Render(option.Name) + strings.Repeat(" ", 4) + descStyle.Render(option.Description)
        }
        rows = append(rows, row)
    }

    content := strings.Join(rows, "\n")
    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("#4C4C4C")).
        Padding(0, 1).
        Render(content)
}

// ContentLines returns the number of lines in the textarea value.
func (p InputPane) ContentLines() int {
    v := p.ta.Value()
    c := 1
    for i := 0; i < len(v); i++ { if v[i] == '\n' { c++ } }
    return c
}


// Streaming helpers
type chatMsg struct{ Role, Text string }

// Command system types
type Command struct {
    Name        string
    Description string
    Handler     func(*InputPane) tea.Cmd
}

type ModelOption struct {
    Name        string
    Description string
    Effort      string
}

func (p *InputPane) appendUser(t string) { p.convo = append(p.convo, chatMsg{Role: "user", Text: strings.TrimSpace(t)}) }
func (p *InputPane) appendAssistant(t string) { p.convo = append(p.convo, chatMsg{Role: "assistant", Text: t}); p.assistantIdx = len(p.convo)-1 }

// Initialize available commands
func (p *InputPane) getCommands() []Command {
    return []Command{
        {
            Name:        "/model",
            Description: "Select reasoning model level",
            Handler:     nil, // handled in handleCommandKeys
        },
        {
            Name:        "/save",
            Description: "Save current session",
            Handler:     nil, // handled in handleCommandKeys
        },
        {
            Name:        "/quit",
            Description: "Exit the program",
            Handler:     nil, // handled in handleCommandKeys
        },
    }
}

// Initialize model options
func (p *InputPane) getModelOptions() []ModelOption {
    return []ModelOption{
        {"gpt-5-mini minimal", "fastest responses with limited reasoning", ""},
        {"gpt-5-mini low", "balances speed with some reasoning", "low"},
        {"gpt-5-mini medium", "provides a solid balance of reasoning depth and latency", "medium"},
        {"gpt-5-mini high", "maximizes reasoning depth for complex or ambiguous problems", "high"},
    }
}

// Handle /model command
func (p *InputPane) handleModelCommand() tea.Cmd {
    p.commandMode = "model_selection"
    p.modelOptions = p.getModelOptions()
    p.selectedOption = 0
    // Find current selection
    for i, opt := range p.modelOptions {
        if opt.Effort == p.selectedModel {
            p.selectedOption = i
            break
        }
    }
    return nil
}

// Handle /save command
func (p *InputPane) handleSaveCommand() tea.Cmd {
    return func() tea.Msg {
        return SaveTranscriptMsg{}
    }
}

// Get current reasoning effort setting
func (p *InputPane) getReasoningEffort() string {
    if p.selectedModel == "" {
        return "low" // default
    }
    return p.selectedModel
}

// Handle keyboard input in command mode
func (p *InputPane) handleCommandKeys(msg tea.KeyMsg) (*InputPane, tea.Cmd) {
    switch msg.String() {
    case "esc":
        p.showCommands = false
        p.commandMode = ""
        return p, nil
    case "up":
        if p.commandMode == "model_selection" {
            if p.selectedOption > 0 {
                p.selectedOption--
            }
        } else if p.showCommands {
            if p.selectedCmd > 0 {
                p.selectedCmd--
            }
        }
        return p, nil
    case "down":
        if p.commandMode == "model_selection" {
            if p.selectedOption < len(p.modelOptions)-1 {
                p.selectedOption++
            }
        } else if p.showCommands {
            commands := p.getFilteredCommands()
            if p.selectedCmd < len(commands)-1 {
                p.selectedCmd++
            }
        }
        return p, nil
    case "enter":
        if p.commandMode == "model_selection" {
            // Select model option
            if p.selectedOption < len(p.modelOptions) {
                p.selectedModel = p.modelOptions[p.selectedOption].Effort
                p.commandMode = ""
                p.showCommands = false
                p.ta.SetValue("")
            }
            return p, nil
        } else if p.showCommands {
            // Execute command
            commands := p.getFilteredCommands()
            if p.selectedCmd < len(commands) {
                cmd := commands[p.selectedCmd]
                switch cmd.Name {
                case "/model":
                    p.showCommands = false
                    p.ta.SetValue("")
                    return p, p.handleModelCommand()
                case "/save":
                    p.showCommands = false
                    p.ta.SetValue("")
                    return p, p.handleSaveCommand()
                case "/quit":
                    return p, tea.Quit
                }
            }
            return p, nil
        }
    default:
        // Update filter if typing
        if p.showCommands && p.commandMode == "" {
            var cmd tea.Cmd
            p.ta, cmd = p.ta.Update(msg)
            newValue := p.ta.Value()
            // Check if we still have a command prefix
            if strings.HasPrefix(newValue, "/") {
                p.commandFilter = newValue
                p.selectedCmd = 0
            } else {
                // Clear command mode if no longer typing a command
                p.showCommands = false
                p.commandFilter = ""
                p.commandMode = ""
            }
            return p, cmd
        }
    }
    return p, nil
}

// Get filtered commands based on current input
func (p *InputPane) getFilteredCommands() []Command {
    commands := p.getCommands()
    if p.commandFilter == "" || p.commandFilter == "/" {
        return commands
    }

    var filtered []Command
    filter := strings.ToLower(p.commandFilter)
    for _, cmd := range commands {
        if strings.Contains(strings.ToLower(cmd.Name), filter) ||
           strings.Contains(strings.ToLower(cmd.Description), filter) {
            filtered = append(filtered, cmd)
        }
    }
    return filtered
}

func (p *InputPane) startChatStreamCmd(user string) tea.Cmd {
    sys := p.systemPrompt()
    
    // Build conversation history (only user/assistant messages)
    var history []llm.ConversationMessage
    for _, msg := range p.convo {
        if msg.Role == "user" || msg.Role == "assistant" {
            history = append(history, llm.ConversationMessage{Role: msg.Role, Text: msg.Text})
        }
    }
    
    req := llm.Request{System: sys, Prompt: user, ConversationHistory: history, MaxTokens: 600}
    // Allow model to use web_search tool automatically
    req.Tools = []map[string]any{{"type": "web_search"}}
    req.ToolChoice = "auto"
    req.Include = []string{"web_search_call.action.sources"}
    // Enable reasoning summary to show thinking process
    req.ReasoningSummary = "auto"
    // Set reasoning effort based on selected model
    req.ReasoningEffort = p.getReasoningEffort()
    ch := make(chan string, 128)
    errCh := make(chan error, 1)
    p.streamCh, p.streamErrCh = ch, errCh
    p.streaming = true
    client := p.client
    go func() {
        _, err := client.Complete(context.Background(), req, func(delta string) { ch <- delta })
        if err != nil { errCh <- err }
        close(ch)
    }()
    return tea.Batch(p.subscribeStreamCmd(ch, errCh), p.blinkCmd())
}

func (p InputPane) subscribeStreamCmd(ch <-chan string, errCh <-chan error) tea.Cmd {
    return func() tea.Msg {
        select {
        case d, ok := <-ch:
            if !ok { return ChatDoneMsg{} }
            return ChatDeltaMsg{Delta: d}
        case err := <-errCh:
            if err != nil { return ChatErrMsg{Err: err.Error()} }
            return ChatDoneMsg{}
        }
    }
}

// blink ticker for streaming indicator
type blinkMsg struct{}
func (p InputPane) blinkCmd() tea.Cmd {
    if !p.streaming { return nil }
    return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return blinkMsg{} })
}


func (p InputPane) TranscriptLines() int {
    lines := 0
    for _, m := range p.convo {
        // prefix line and trailing blank
        l := 1
        for i := 0; i < len(m.Text); i++ { if m.Text[i] == '\n' { l++ } }
        lines += l + 1
    }
    return lines
}

// TranscriptViewWithWidth renders the transcript with hanging indents so
// text columns are left-aligned and not colliding with the prefix symbols.
func (p InputPane) TranscriptViewWithWidth(width int) string {
    if len(p.convo) == 0 { return "" }
    rows := make([]string, 0, len(p.convo)*2)
    for _, m := range p.convo {
        switch m.Role {
        case "user":
            prefix := Gray.Render("> ")
            contentW := width - lipgloss.Width(prefix)
            if contentW < 4 { contentW = width }
            row := lipgloss.JoinHorizontal(lipgloss.Top,
                prefix,
                lipgloss.NewStyle().Width(contentW).Render(Gray.Render(m.Text)),
            )
            rows = append(rows, row)
        case "assistant":
            dot := "⏺"
            prefix := White.Render(dot+" ")
            contentW := width - lipgloss.Width(prefix)
            if contentW < 4 { contentW = width }
            row := lipgloss.JoinHorizontal(lipgloss.Top,
                prefix,
                lipgloss.NewStyle().Width(contentW).Render(Text.Render(m.Text)),
            )
            rows = append(rows, row)
        case "tool":
            dot := "⏺"
            prefix := lipgloss.NewStyle().Foreground(colorPrimary).Render(dot+" ")
            contentW := width - lipgloss.Width(prefix)
            label := Strong.Render("Web Search")
            var body string
            if m.Text != "" { body = Text.Render("(") + Gray.Render(m.Text) + Text.Render(")") }
            row := lipgloss.JoinHorizontal(lipgloss.Top,
                prefix,
                lipgloss.NewStyle().Width(contentW).Render(label+body),
            )
            rows = append(rows, row)
        case "reason":
            // Thinking header with static dot
            dot := "⏺"
            prefix := Gray.Render(dot+" ")
            contentW := width - lipgloss.Width(prefix)
            if contentW < 4 { contentW = width }

            // Always show "Thinking..." header
            header := lipgloss.NewStyle().Width(contentW).Foreground(colorGray).Italic(true).Render("Thinking…")
            rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, prefix, header))

            // Show actual thinking content if available
            if strings.TrimSpace(m.Text) != "" {
                // Empty prefix for continuation
                emptyPrefix := strings.Repeat(" ", lipgloss.Width(prefix))
                body := lipgloss.NewStyle().Width(contentW).Foreground(colorGray).Italic(true).Render(m.Text)
                rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, emptyPrefix, body))
            }
        }
        rows = append(rows, "")
    }
    if len(rows) > 0 { rows = rows[:len(rows)-1] }
    return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (p InputPane) systemPrompt() string {
    if p.sysPrompt != "" {
        return p.sysPrompt
    }
    // Fallback if prompt.md wasn't loaded
    return `You are an AI assistant helping users through a terminal interface.

Role & Capabilities
- AI assistant: Your primary strength is helping with various tasks, answering questions, and providing information
- You have access to web search and reasoning capabilities
- Help users discover insights, compare options, and understand complex topics

When to Search the Web
- User asks about current events, recent developments, or time-sensitive information
- Need to verify facts, statistics, or specific claims
- User wants to compare products, services, or options
- Questions about trends, news, or evolving topics
- User explicitly asks you to search or "look up" something

When to Think/Reason
- User explicitly requests thinking (think, think hard, analyze, etc.)
- Complex problems requiring multi-step analysis
- Comparing multiple options or trade-offs
- Breaking down complex concepts or strategies
- Working through logical problems or decision trees

Do NOT think for:
- Simple greetings (hi, hello, thanks)
- Basic factual questions you can answer directly
- Straightforward requests that don't need analysis

Response Style
- Keep replies concise and high-signal
- Use bullet points and short paragraphs
- Include citations when presenting web-sourced information
- Be direct and helpful

Critical: When thinking, write natural thoughts without any special formatting, stars, or meta-commentary like "**Responding to...**". Just think naturally about the problem.`
}

// Best-effort extraction of a query string from streaming tool payload.
// The payload may be a JSON object; we try to pull `query` if present,
// otherwise fallback to raw text.
func extractQuery(payload string) string {
    payload = strings.TrimSpace(payload)
    if payload == "" { return "" }
    // Best effort: parse JSON and search for "query" key anywhere.
    var any map[string]any
    if payload != "" && payload[0] == '{' {
        if err := json.Unmarshal([]byte(payload), &any); err == nil {
            if q := findQuery(any); q != "" { return q }
        }
    }
    // Fallback simple scan
    if idx := strings.Index(payload, "\"query\""); idx >= 0 {
        s := payload[idx+7:]
        if qstart := strings.Index(s, ":"); qstart >= 0 {
            s = strings.TrimSpace(s[qstart+1:])
            if len(s) > 0 && s[0] == '"' {
                s = s[1:]
                if j := strings.IndexByte(s, '"'); j >= 0 { return s[:j] }
            }
        }
    }
    return "Searching…"
}

func findQuery(v any) string {
    switch t := v.(type) {
    case map[string]any:
        for k, vv := range t {
            if k == "query" {
                if s, ok := vv.(string); ok { return s }
            }
            if q := findQuery(vv); q != "" { return q }
        }
    case []any:
        for _, it := range t { if q := findQuery(it); q != "" { return q } }
    }
    return ""
}



// RestoreConversation restores a previous conversation from session data
func (p *InputPane) RestoreConversation(conversations []session.ChatMsg) {
    p.convo = make([]chatMsg, len(conversations))
    for i, conv := range conversations {
        p.convo[i] = chatMsg{Role: conv.Role, Text: conv.Text}
    }
}

// GetConversation returns the current conversation for session saving
func (p *InputPane) GetConversation() []chatMsg {
    return p.convo
}

// Working indicator functions
func (p *InputPane) getIndicatorBall() string {
    frames := []string{
        "[●    ]",
        "[ ●   ]",
        "[  ●  ]",
        "[   ● ]",
        "[    ●]",
        "[   ● ]",
        "[  ●  ]",
        "[ ●   ]",
    }
    return frames[p.indicatorFrame%len(frames)]
}

func (p *InputPane) getIndicatorText() string {
    texts := []string{
        "Spelunking…", "Archaeologizing…", "Cryptohunting…", "Questifying…", "Mystifying…",
        "Enigmatizing…", "Riddling…", "Puzzlefying…", "Codebreaking…", "Secretizing…",
        "Whispertracking…", "Shadowdancing…", "Timebending…", "Mindreading…", "Truthseeking…",
        "Dreamweaving…", "Thoughtdigging…", "Memorysifting…", "Ideachasing…", "Wisdomhunting…",
        "Neuronspinning…", "Brainstitching…", "Synapsehopping…", "Cognitiondiving…", "Insightmining…",
        "Galaxysurfing…", "Starwhispering…", "Cosmosprobing…", "Universesearching…", "Dimensionhopping…",
        "Algorithmdancing…", "Databending…", "Logictwisting…", "Patternweaving…", "Codewhispering…",
        "Infinitywalking…", "Paradoxsolving…", "Quantumleaping…", "Realityhacking…", "Matrixdiving…",
        "Treasuremystifying…", "Artifactwhispering…", "Relicchasing…", "Fossilmining…", "Antiquehunting…",
        "Soulreading…", "Spirittracking…", "Essencehunting…", "Vibrationfeeling…", "Energysurfing…",
        "Moonwalking…", "Starhopping…", "Cloudweaving…", "Rainbowchasing…", "Thunderlistening…",
        "Magicspelling…", "Wizardwhistling…", "Spellcrafting…", "Enchantweaving…", "Potionbrewing…",
    }
    // Pick random word each time
    return texts[rand.Intn(len(texts))]
}

func (p *InputPane) renderIndicator() string {
    if !p.streaming {
        return ""
    }
    ball := p.getIndicatorBall()
    text := p.getIndicatorText()
    return WelcomeAccent.Render(ball + " " + text)
}
