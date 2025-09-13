package platform

import (
    "fmt"
    "os"
)

func AppendFile(path string, data []byte) error {
    f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
    if err != nil { return fmt.Errorf("open %s: %w", path, err) }
    defer f.Close()
    if _, err := f.Write(data); err != nil { return fmt.Errorf("write %s: %w", path, err) }
    return nil
}

// WriteFileAtomic writes a file atomically by writing to temp file and renaming.
func WriteFileAtomic(path string, data []byte) error {
    dir := os.DirFS(".")
    _ = dir // avoid unused in some envs; we use os functions below
    tmp := path + ".tmp"
    f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
    if err != nil { return fmt.Errorf("open tmp %s: %w", tmp, err) }
    if _, err := f.Write(data); err != nil { _ = f.Close(); return fmt.Errorf("write tmp %s: %w", tmp, err) }
    if err := f.Close(); err != nil { return fmt.Errorf("close tmp %s: %w", tmp, err) }
    if err := os.Rename(tmp, path); err != nil { return fmt.Errorf("rename %s -> %s: %w", tmp, path, err) }
    return nil
}
