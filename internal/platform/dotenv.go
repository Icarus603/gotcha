package platform

import (
    "bufio"
    "os"
    "strings"
)

// LoadDotEnv reads KEY=VALUE lines from a file and sets os.Environ.
// Lines beginning with # are comments; surrounding quotes are trimmed.
func LoadDotEnv(path string) error {
    f, err := os.Open(path)
    if err != nil { return err }
    defer f.Close()
    s := bufio.NewScanner(f)
    for s.Scan() {
        line := strings.TrimSpace(s.Text())
        if line == "" || strings.HasPrefix(line, "#") { continue }
        if i := strings.IndexByte(line, '='); i > 0 {
            k := strings.TrimSpace(line[:i])
            v := strings.TrimSpace(line[i+1:])
            v = strings.Trim(v, "\"'")
            _ = os.Setenv(k, v)
        }
    }
    return nil
}

