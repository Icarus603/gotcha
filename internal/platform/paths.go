package platform

import (
    "fmt"
    "os"
    "path/filepath"
)

// Paths computes on-disk locations. For sandbox friendliness, default to a
// workspace-local .gotcha directory.
type Paths struct {
    Base string // base data dir (e.g., .gotcha)
}

func DefaultPaths() Paths {
    return Paths{Base: ".gotcha"}
}

func (p Paths) Ensure() error {
    return os.MkdirAll(p.Base, 0o755)
}

func (p Paths) SessionsDir() string { return filepath.Join(p.Base, "sessions") }
func (p Paths) SessionDir(id string) string { return filepath.Join(p.SessionsDir(), id) }
func (p Paths) SessionNotesPath(id string) string { return filepath.Join(p.SessionDir(id), "notes.md") }
func (p Paths) SessionReportPath(id string) string { return filepath.Join(p.SessionDir(id), "report.md") }
func (p Paths) DBPath() string { return filepath.Join(p.Base, "gotcha.sqlite") }

func (p Paths) EnsureSession(id string) (string, error) {
    dir := p.SessionDir(id)
    if err := os.MkdirAll(dir, 0o755); err != nil { return "", fmt.Errorf("mkdir session: %w", err) }
    return dir, nil
}

