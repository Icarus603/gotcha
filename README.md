# Gotcha

🔍 An intelligent terminal-based research agent powered by AI that helps you find, analyze, and synthesize information through autonomous decision-making.

> **⚠️ Project Status**: Currently in active development

## Features

✨ **Intelligent Research Agent**: Gotcha autonomously decides when to reason, when to search the web, and when to respond based on query complexity

🤖 **Adaptive Intelligence**: Dynamically adjusts research strategies from simple facts to complex multi-faceted analysis

💬 **Session Management**: Persistent conversations with session resumption and intelligent transcript generation

📝 **Smart Note-Taking**: Built-in note-taking with automatic file management

⌨️ **Terminal UI**: Clean, responsive terminal interface built with Bubble Tea framework

🎯 **Autonomous Decision-Making**: No rigid patterns - adapts research approach to each unique query

## What Gotcha Does

- **Research Information**: Find current data, trends, and developments
- **Analyze Topics**: Break down complex subjects with multiple perspectives
- **Compare Options**: Side-by-side analysis of products, services, frameworks
- **Explain Concepts**: Clear explanations of technical and non-technical topics
- **Synthesize Findings**: Combine information from multiple sources into coherent insights

## What Gotcha Doesn't Do

- **No Coding**: Gotcha is a research specialist, not a coding assistant
- **No Code Generation**: Won't write, debug, or create programs
- **No Programming Help**: Redirects coding requests to research alternatives

## Installation

### Prerequisites

- Go 1.22 or later
- Terminal with mouse support (recommended)

### Build from Source

```bash
# Clone the repository
gh repo clone Icarus603/gotcha
cd gotcha

# Build the binary
make build

# Or build manually
go build -o bin/gotcha cmd/gotcha/main.go
```

## Configuration

Gotcha supports OpenAI API integration for enhanced AI capabilities:

1. Copy the example configuration:
```bash
cp configs/config.example.toml config.toml
```

2. Add your OpenAI API key to the configuration file

3. Or set environment variables:
```bash
export OPENAI_API_KEY=your_api_key_here
```

## Usage

### Basic Usage

```bash
# Start a new research session
./bin/gotcha

# Continue your last session
./bin/gotcha -continue

# Resume from session selection menu
./bin/gotcha -resume
```

### Commands

- `/model` - Switch between different reasoning levels (minimal, low, medium, high)
- `/save` - Save current conversation with intelligent summarization
- `/quit` - Exit the application

### Session Management

Gotcha automatically manages your research sessions:
- **Auto-save**: Conversations are saved automatically
- **Session Resume**: Continue where you left off
- **Smart Transcripts**: Generates structured summaries instead of raw conversation dumps
- **Note Integration**: Take notes during research that are automatically saved

## Examples

**Simple Query**:
```
> What is machine learning?
```
*Gotcha responds directly from knowledge*

**Current Information**:
```
> Latest developments in AI 2024
```
*Gotcha searches for recent information then responds*

**Complex Research**:
```
> Compare the top 5 JavaScript frameworks for enterprise applications
```
*Gotcha performs multiple searches and provides comprehensive analysis*

## Project Structure

```
gotcha/
├── cmd/gotcha/          # Application entry point
├── internal/
│   ├── agent/           # Research agent logic
│   ├── app/             # Application services
│   ├── llm/             # LLM integration (OpenAI)
│   ├── platform/        # Platform utilities
│   ├── session/         # Session management
│   ├── storage/         # Data persistence
│   └── tui/             # Terminal UI components
├── configs/             # Configuration templates
├── prompt.md            # AI agent instructions
└── Makefile            # Build automation
```

## Development

### Building

```bash
# Build the application
make build

# Run directly
make run

# Format code
make fmt

# Run tests
make test

# Clean build artifacts
make clean
```

### Architecture

- **Terminal UI**: Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework
- **Session Management**: Persistent storage with JSON-based session files
- **AI Integration**: OpenAI API with streaming support
- **Research Engine**: Autonomous decision-making for research strategies
- **Phase Management**: Clean separation between reasoning, searching, and responding

## Contributing

This project is currently in active development. Contributions, issues, and feature requests are welcome!

## License

[Add your license information here]

## Roadmap

- [ ] Web search integration
- [ ] Multiple LLM provider support
- [ ] Export capabilities (Markdown, PDF)
- [ ] Plugin system
- [ ] Advanced filtering and search within sessions
- [ ] Team collaboration features

---

**Built with ❤️ using Go and Bubble Tea**