package storage

// Migrate is a no-op in the file-backed fallback.
func Migrate(db *DB) error { return nil }
