# Repository Guidelines

## Project Structure & Module Organization
- `cmd/gotcha/` holds the entrypoint; start here when tracing program flow.
- `internal/` contains feature packages: for example `tui/` for Bubble Tea UI, `agent/` for research orchestration, `session/` for persistence helpers, and `llm/` for provider integration.
- Runtime assets live under `.gotcha/` (session state) and `configs/` (sample TOML config). Avoid hand-editing `.gotcha/`—it is managed by the app.

## Build, Test, and Development Commands
- `make build` or `go build ./cmd/gotcha` produces the CLI binary in `bin/`.
- `go install ./cmd/gotcha` installs a native binary to `$(go env GOPATH)/bin`; ensure `~/go/bin` is on your `PATH`.
- `make run` builds and launches the TUI.
- `make test` (or `go test ./...`) runs the Go test suite.

## Coding Style & Naming Conventions
- Follow standard Go formatting: run `gofmt` or `go fmt ./...` before submitting.
- Use camelCase for private identifiers and PascalCase for exported names. Keep filenames lowercase with underscores only when conventional (e.g., `manager.go`).
- UI strings should stay in `internal/tui` to keep logic and presentation separated.

## Testing Guidelines
- Unit tests live alongside code in `*_test.go` files. Prefer table-driven tests and explicit error messages.
- New features should include coverage or explain why they are untestable. Run `go test ./internal/...` to narrow scope while iterating.

## Commit & Pull Request Guidelines
- Use imperative, concise commit messages (e.g., `Fix session selector cursor wrap`). Squash noisy fix-ups before opening a PR.
- PRs should summarize motivation, list key changes, and call out testing performed (`go test`, manual TUI checks). Link related issues and include screenshots/gifs for UI changes when feasible.

## Security & Configuration Notes
- Place secrets in environment variables (`OPENAI_API_KEY`) or `.env` (not tracked). Do not commit personal `.gotcha/` session data.
- Respect proxy settings via `PROXY_URL` or standard `HTTPS_PROXY` variables when testing networked features.
