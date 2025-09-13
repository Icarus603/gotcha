package storage

// File-backed fallback repositories (no-op). Persistence happens in files.
type SessionsRepo struct{}

func (r SessionsRepo) Upsert(id, title, query string) error { return nil }
func (r SessionsRepo) Exists(id string) (bool, error) { return true, nil }

type NotesRepo struct{}

func (r NotesRepo) Insert(sessionID, text string) error { return nil }
