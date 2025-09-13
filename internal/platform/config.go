package platform

import (
    "os"
    "strconv"
)

// LLMConfig captures model provider settings.
type LLMConfig struct {
    Provider    string
    Model       string
    APIKey      string
    BaseURL     string
    MaxTokens   int
    Temperature float64
}

// Config holds runtime configuration.
type Config struct {
    AppName string
    // TUI toggles
    ShowSources bool
    Paths Paths
    LLM  LLMConfig
    ProxyURL string
}

func LoadConfig() Config {
    // Load .env if present (no external deps)
    _ = LoadDotEnv(".env")

    p := DefaultPaths()
    _ = p.Ensure()
    return Config{
        AppName:    envOr("GOTCHA_APP_NAME", "gotcha"),
        ShowSources: false,
        Paths: p,
        LLM: LLMConfig{
            Provider:    envOr("LLM_PROVIDER", "openai"),
            Model:       envOr("LLM_MODEL", "gpt-5-mini-2025-08-07"),
            APIKey:      os.Getenv("OPENAI_API_KEY"),
            BaseURL:     envOr("OPENAI_BASE_URL", "https://api.openai.com"),
            MaxTokens:   intEnvOr("LLM_MAX_TOKENS", 1500),
            Temperature: floatEnvOr("LLM_TEMPERATURE", 0.2),
        },
        ProxyURL: firstNonEmpty(
            os.Getenv("PROXY_URL"),
            os.Getenv("HTTPS_PROXY"),
            os.Getenv("https_proxy"),
            os.Getenv("HTTP_PROXY"),
            os.Getenv("http_proxy"),
            os.Getenv("ALL_PROXY"),
            os.Getenv("all_proxy"),
        ),
    }
}

func envOr(key, def string) string {
    if v := os.Getenv(key); v != "" { return v }
    return def
}

func intEnvOr(key string, def int) int {
    if v := os.Getenv(key); v != "" {
        if x, err := strconv.Atoi(v); err == nil { return x }
    }
    return def
}

func floatEnvOr(key string, def float64) float64 {
    if v := os.Getenv(key); v != "" {
        if x, err := strconv.ParseFloat(v, 64); err == nil { return x }
    }
    return def
}

func firstNonEmpty(vals ...string) string {
    for _, v := range vals { if v != "" { return v } }
    return ""
}
