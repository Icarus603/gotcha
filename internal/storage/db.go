package storage

import "context"

// DB is a placeholder for a real database. In environments without module access,
// we fall back to a no-op transaction wrapper; file persistence is handled elsewhere.
type DB struct{}

func Open(path string) (*DB, error) { return &DB{}, nil }
func (db *DB) Close() error { return nil }

type UnitOfWork interface {
    Sessions() SessionsRepo
    Notes() NotesRepo
    Commit() error
    Rollback() error
}

type noopUOW struct{}

func (n *noopUOW) Sessions() SessionsRepo { return SessionsRepo{} }
func (n *noopUOW) Notes() NotesRepo { return NotesRepo{} }
func (n *noopUOW) Commit() error { return nil }
func (n *noopUOW) Rollback() error { return nil }

func (db *DB) Update(ctx context.Context, fn func(UnitOfWork) error) error {
    u := &noopUOW{}
    return fn(u)
}

func (db *DB) View(ctx context.Context, fn func(UnitOfWork) error) error {
    u := &noopUOW{}
    return fn(u)
}
