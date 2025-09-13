# Gotcha 🎯

> A terminal-native AI-powered research assistant and note-taking companion

Gotcha is a sophisticated Terminal User Interface (TUI) application built with Go that combines the power of Large Language Models with an intuitive note-taking system. It provides a seamless research experience directly in your terminal, with session management, real-time AI assistance, and organized markdown-based output.

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=flat&logo=go&logoColor=white)
![Terminal](https://img.shields.io/badge/Terminal-black?style=flat&logo=gnome-terminal&logoColor=white)
![OpenAI](https://img.shields.io/badge/OpenAI-412991.svg?style=flat&logo=openai&logoColor=white)

## ✨ Features

### 🔬 AI-Powered Research
- **Intelligent Research Agent**: Generates structured research plans and detailed compositions
- **LLM Integration**: Native OpenAI API support with GPT-5 and reasoning models
- **Real-time Streaming**: Token-by-token response streaming for immediate feedback
- **Configurable Reasoning**: Choose between minimal, low, medium, and high reasoning effort levels

### 📝 Smart Note-Taking
- **Session-Based Notes**: Organize notes by research sessions with automatic timestamping
- **Markdown Output**: Generate structured reports and notes in markdown format
- **Persistent Storage**: Local file-based storage in workspace-aware directory structure

### 💻 Advanced Terminal Interface
- **Modern TUI**: Built with Bubble Tea framework for smooth terminal interactions
- **Dynamic Layout**: Content-aware pane sizing that adapts to your content
- **Smart Mouse Support**: Intelligent switching between scrolling and text selection
- **Responsive Design**: Adapts seamlessly to terminal size changes

### ⚡ Slash Command System
- `/model` - Select AI reasoning level (minimal → high)
- `/research` - Trigger structured research workflow
- `/quit` - Exit the application gracefully
- Auto-complete with fuzzy search and keyboard navigation

## 🏗️ Architecture

Gotcha follows clean architecture principles with clear separation of concerns:

```
┌─────────────────────────────────────┐
│            TUI Layer               │
│  ┌──────────┐ ┌──────────┐ ┌──────┐│
│  │RootModel │ │InputPane │ │Notes ││
│  │(viewport)│ │(research)│ │(session)││
│  └──────────┘ └──────────┘ └──────┘│
└─────────────────────────────────────┘
                    │
┌─────────────────────────────────────┐
│          Business Layer            │
│  ┌──────────┐ ┌──────────┐ ┌──────┐│
│  │Researcher│ │EventBus  │ │AppSvc││
│  │(AI agent)│ │(pub/sub) │ │(mgmt)││
│  └──────────┘ └──────────┘ └──────┘│
└─────────────────────────────────────┘
                    │
┌─────────────────────────────────────┐
│       Infrastructure Layer         │
│  ┌──────────┐ ┌──────────┐ ┌──────┐│
│  │OpenAI    │ │File      │ │Config││
│  │Client    │ │Storage   │ │Mgmt  ││
│  └──────────┘ └──────────┘ └──────┘│
└─────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Go 1.22 or later
- OpenAI API key
- Terminal with true color support (recommended)

### Installation

1. **Clone and build:**
   ```bash
   git clone https://github.com/yourusername/gotcha.git
   cd gotcha
   go mod tidy
   go build ./cmd/gotcha
   ```

2. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your OpenAI API key
   ```

3. **Run:**
   ```bash
   ./gotcha
   ```

### Configuration

Create a `.env` file in your project directory:

```bash
# LLM Configuration
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-your-api-key-here
OPENAI_BASE_URL=https://api.openai.com
LLM_MODEL=gpt-5-mini-2025-08-07

# Optional: Proxy Configuration
PROXY_URL=http://127.0.0.1:7890
HTTP_PROXY=http://127.0.0.1:7890

# Optional: Generation Parameters
LLM_MAX_TOKENS=1500
LLM_TEMPERATURE=0.2
```

### System Prompt (Optional)

Create a `prompt.md` file in your working directory to customize the AI behavior:

```markdown
You are a research assistant helping users find information and analyze complex topics through a terminal interface.

Role & Capabilities
- Research agent: Your primary strength is finding, synthesizing, and analyzing information
- You have access to web search and reasoning capabilities
- Help users discover insights, compare options, and understand complex topics

Response Style
- Keep replies concise and high-signal
- Use bullet points and short paragraphs
- Include citations when presenting web-sourced information
- Be direct and helpful
```

## 📖 Usage Guide

### Basic Navigation

- **Enter**: Submit research queries or save notes
- **Shift+Enter**: Add new lines in text areas
- **Tab** or **Ctrl+S**: Switch between Research and Notes panes
- **Ctrl+C** or **Q**: Quit application

### Research Workflow

1. **Start a query**: Type your research question in the input pane
2. **Choose reasoning level**: Use `/model` command to select AI effort level
3. **Trigger research**: Use `/research` prefix or submit directly for chat
4. **Review results**: AI generates structured research with planning and composition phases
5. **Take notes**: Switch to Notes pane to add your own observations

### Command System

#### `/model` Command
Select the reasoning effort level for AI responses:
- **Minimal**: Quick, direct responses
- **Low**: Basic reasoning with simple explanations
- **Medium**: Balanced analysis with moderate depth
- **High**: Deep reasoning with comprehensive analysis

#### `/research` Command
Triggers the structured research workflow:
- Generates detailed research plan
- Executes research in phases
- Produces comprehensive markdown reports
- Saves results to session directory

### File Organization

Gotcha creates a `.gotcha` directory structure:

```
.gotcha/
├── sessions/
│   └── dev-session/           # Default session
│       ├── notes.md          # Your timestamped notes
│       └── report.md         # Generated research reports
└── gotcha.sqlite            # Future database support
```

## 🔧 Advanced Features

### Session Management

Sessions organize your research work:
- **Automatic creation**: Default "dev-session" created on first run
- **Persistent notes**: Notes are saved with timestamps
- **Report generation**: Research results automatically saved
- **Cross-session**: Resume work across application restarts

### Event-Driven Architecture

Gotcha uses an event bus system for real-time coordination:
- Research progress updates
- UI state synchronization
- Decoupled component communication

### Smart UI Behaviors

- **Dynamic sizing**: Panes grow with content, no arbitrary limits
- **Smart scrolling**: Auto-detection of selection vs scroll intent
- **Mouse integration**: Seamless wheel scrolling with selection support
- **Focus management**: Intuitive tab navigation between panes

## 🛠️ Development

### Build Commands

```bash
# Development build
go build ./cmd/gotcha

# With custom cache (sandbox environments)
GOCACHE=$PWD/.gocache go build ./cmd/gotcha

# Clean build
go clean -cache ./cmd/gotcha
go build ./cmd/gotcha
```

### Project Structure

```
├── cmd/gotcha/           # Application entry point
├── internal/
│   ├── tui/             # Terminal UI components
│   │   ├── model.go     # Root application model
│   │   ├── panes_*.go   # UI pane implementations
│   │   └── styles.go    # Visual styling
│   ├── app/             # Business logic
│   ├── agent/           # AI research agents
│   ├── llm/             # LLM client implementations
│   ├── platform/        # Configuration & paths
│   └── storage/         # Data persistence
├── prompt.md            # System prompt (optional)
├── .env                 # Configuration
└── go.mod
```

### Key Dependencies

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**: TUI framework
- **[Bubbles](https://github.com/charmbracelet/bubbles)**: UI components
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)**: Terminal styling

## 🤝 Contributing

Contributions are welcome! This project follows clean architecture principles and maintains high code quality standards.

### Development Guidelines

1. **Architecture**: Follow the established layered architecture
2. **Testing**: Add tests for new functionality
3. **Documentation**: Update README for significant changes
4. **Code Style**: Use `go fmt` and follow Go best practices

### Extension Points

- **LLM Providers**: Add support for other AI providers
- **Storage Backends**: Implement database persistence
- **UI Components**: Add new panes or interface elements
- **Research Phases**: Extend the research workflow
- **Commands**: Add new slash commands

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Excellent TUI framework
- [OpenAI](https://openai.com) - AI capabilities
- The Go community for excellent tooling and libraries

---

<p align="center">
  <i>Built with ❤️ using Go and Bubble Tea</i>
</p>