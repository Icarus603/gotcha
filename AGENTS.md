# Repository Guidelines

This repo contains a terminal‑native research and note‑taking TUI written in Go (Bubble Tea). Use this guide to navigate the codebase, build locally, and contribute safely.

## Project Structure & Module Organization
- `cmd/gotcha/` — CLI entrypoint.
- `internal/tui/` — Bubble Tea models and panes (`model.go`, `panes_*.go`, `styles.go`).
- `internal/agent/` — events and (simulated) background research tasks.
- `internal/app/` — session/notes services.
- `internal/platform/` — config, paths, filesystem helpers.
- `internal/storage/` — persistence layer (file‑backed fallback; SQLite later).
- `internal/llm/` — LLM provider abstraction (OpenAI adapter to be added).
- `configs/` — example configuration (e.g., `config.example.toml`).

## Build, Test, and Development Commands
- `make tidy` — sync Go modules.
- `make build` — build `bin/gotcha`.
- `GOCACHE=$PWD/.gocache make run` — build and run the TUI (compatible with sandboxed envs).
- `make test` — run unit tests when present (`*_test.go`).
- `make fmt` — format sources (`go fmt ./...`).

## Coding Style & Naming Conventions
- Go 1.22+. Rely on `gofmt`/`go fmt` for formatting.
- Packages: short lowercase (`tui`, `agent`, `compose`), no stutter.
- Files: feature‑oriented names (e.g., `panes_notes.go`, `events.go`).
- Avoid exporting symbols unless needed across packages; prefer dependency injection via interfaces.
- Use `context.Context` for all I/O or long‑running operations.

## Testing Guidelines
- Unit tests colocated: `*_test.go` next to code; golden fixtures under `testdata/`.
- Prefer deterministic tests (seeded time/IDs). Avoid network in unit tests; mock providers.
- Run: `go test ./...`. For integration tests, place under `test/integration` (optional) and guard with build tags.

## Commit & Pull Request Guidelines
- Commits: concise, imperative subject (≤72 chars), body explaining rationale and user impact.
- PRs: include description, screenshots/asciinema for TUI changes, and checklist of tested flows.
- Link related issues; note any config/env changes (e.g., new vars).

## Security & Configuration Tips
- Never commit secrets. Set `OPENAI_API_KEY` in your shell or `.env` (loaded on startup). Example:
  - `export OPENAI_API_KEY=...`
- Local data paths default to workspace `.gotcha/`; safe to remove with `make clean` (does not delete `.gotcha`).

## Agent‑Specific Notes
- Keep UI responsive: do not block the Bubble Tea update loop; emit progress via events.
- When adding tools/connectors, log action inputs/outputs to the event bus (no chain‑of‑thought, no raw HTML in logs).
