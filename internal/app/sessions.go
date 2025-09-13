package app

import (
    "context"
    "fmt"
    "path/filepath"
    "time"
    "gotcha/internal/platform"
    "gotcha/internal/storage"
)

type Service struct{
    db    *storage.DB
    paths platform.Paths
}

func NewService(db *storage.DB, paths platform.Paths) *Service { return &Service{db: db, paths: paths} }

// CreateOrOpenSession persists a session row and ensures its directory.
func (s *Service) CreateOrOpenSession(ctx context.Context, id, title, query string) (string, error) {
    if _, err := s.paths.EnsureSession(id); err != nil { return "", err }
    err := s.db.Update(ctx, func(u storage.UnitOfWork) error {
        return u.Sessions().Upsert(id, title, query)
    })
    if err != nil { return "", err }
    return id, nil
}

// SaveNote appends to notes.md and inserts into the DB.
func (s *Service) SaveNote(ctx context.Context, sessionID, text string) error {
    if text == "" { return nil }
    // Append to file
    notesPath := s.paths.SessionNotesPath(sessionID)
    if err := s.paths.Ensure(); err != nil { return err }
    if _, err := s.paths.EnsureSession(sessionID); err != nil { return err }
    line := fmt.Sprintf("- [%s] %s\n", time.Now().Format(time.RFC3339), text)
    if err := platform.AppendFile(notesPath, []byte(line)); err != nil { return err }
    // Insert DB
    return s.db.Update(ctx, func(u storage.UnitOfWork) error {
        return u.Notes().Insert(sessionID, text)
    })
}

func (s *Service) ReportPath(sessionID string) string { return s.paths.SessionReportPath(sessionID) }
func (s *Service) NotesPath(sessionID string) string { return s.paths.SessionNotesPath(sessionID) }
func (s *Service) DBPath() string { return filepath.Clean(s.paths.DBPath()) }
