package llm

import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"
)

// OpenAIClient is a lightweight client for OpenAI-compatible chat completions.
// It avoids external deps to keep the binary small and sandbox-friendly.
type OpenAIClient struct {
    apiKey   string
    baseURL  string // e.g., https://api.openai.com
    model    string
    timeout  time.Duration
    proxyURL string
}

func NewOpenAI(apiKey, baseURL, model string, proxyURL string) *OpenAIClient {
    if baseURL == "" { baseURL = "https://api.openai.com" }
    return &OpenAIClient{apiKey: apiKey, baseURL: strings.TrimRight(baseURL, "/"), model: model, timeout: 60 * time.Second, proxyURL: proxyURL}
}

func (c *OpenAIClient) Name() string { return "openai" }


// Complete performs a chat completion; if onToken is non-nil, it streams deltas.
func (c *OpenAIClient) Complete(ctx context.Context, req Request, onToken StreamHandler) (Response, error) {
    // Use Responses API (recommended) to support GPT-5 and reasoning models.
    if c.apiKey == "" { return Response{}, errors.New("openai: missing API key") }
    model := c.model
    if req.Model != "" { model = req.Model }
    rr := responsesReq{
        Model:                 model,
        Instructions:          strings.TrimSpace(req.System),
        Input:                 buildResponsesInputWithHistory(req.Prompt, req.ConversationHistory),
        MaxOutputTokens:       req.MaxTokens,
        Stream:                onToken != nil,
        Stop:                  req.Stop,
    }
    if supportsTemperature(model) && req.Temperature > 0 {
        rr.Temperature = req.Temperature
    }
    if len(req.Tools) > 0 { rr.Tools = req.Tools }
    if req.ToolChoice != "" { rr.ToolChoice = req.ToolChoice }
    if len(req.Include) > 0 { rr.Include = req.Include }
    if req.ReasoningEffort != "" { rr.Reasoning.Effort = req.ReasoningEffort }
    if req.ReasoningSummary != "" { rr.Reasoning.Summary = req.ReasoningSummary }
    body, _ := json.Marshal(rr)
    httpClient := &http.Client{Timeout: c.timeout, Transport: transportWithProxy(c.proxyURL)}
    endpoint := c.baseURL + "/v1/responses"
    httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    httpReq.Header.Set("Content-Type", "application/json")
    resp, err := httpClient.Do(httpReq)
    if err != nil { return Response{}, err }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        b, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
        // Fallback: some orgs/models don't permit streaming. Retry without stream.
        if onToken != nil && (bytes.Contains(b, []byte("\"param\":\"stream\"")) || bytes.Contains(bytes.ToLower(b), []byte("verify organization")) || bytes.Contains(b, []byte("unsupported_value"))) {
            // Re-issue same request without streaming
            rr.Stream = false
            body2, _ := json.Marshal(rr)
            httpReq2, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body2))
            httpReq2.Header.Set("Authorization", "Bearer "+c.apiKey)
            httpReq2.Header.Set("Content-Type", "application/json")
            resp2, err2 := httpClient.Do(httpReq2)
            if err2 != nil { return Response{}, err2 }
            defer resp2.Body.Close()
            if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
                b2, _ := io.ReadAll(io.LimitReader(resp2.Body, 8192))
                return Response{}, fmt.Errorf("openai: http %d: %s", resp2.StatusCode, string(b2))
            }
            var r responsesResp
            data2, err2 := io.ReadAll(resp2.Body)
            if err2 != nil { return Response{}, err2 }
            if err2 := json.Unmarshal(data2, &r); err2 != nil { return Response{}, err2 }
            text := r.OutputText
            if text == "" { text = r.AggregateOutputText() }
            if text != "" { onToken(text) }
            return Response{Text: strings.TrimSpace(text), PromptTokens: r.Usage.InputTokens, CompletionTokens: r.Usage.OutputTokens}, nil
        }
        return Response{}, fmt.Errorf("openai: http %d: %s", resp.StatusCode, string(b))
    }
    if onToken == nil {
        var r responsesResp
        data, err := io.ReadAll(resp.Body)
        if err != nil { return Response{}, err }
        if err := json.Unmarshal(data, &r); err != nil { return Response{}, err }
        text := r.OutputText
        if text == "" { text = r.AggregateOutputText() }
        return Response{Text: strings.TrimSpace(text), PromptTokens: r.Usage.InputTokens, CompletionTokens: r.Usage.OutputTokens}, nil
    }
    // Streaming via SSE events (event: ... \n data: ...)
    scanner := bufio.NewScanner(resp.Body)
    scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
    var event string
    var full strings.Builder
    needFallback := false
    var errPayload string
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "event:") {
            event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
            continue
        }
        if strings.HasPrefix(line, "data:") {
            payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
            switch event {
            case "response.output_text.delta":
                delta := parseDelta(payload)
                if delta != "" {
                    full.WriteString(delta)
                    onToken(delta)
                }
            case "response.tool_call.delta", "response.web_search_call.delta":
                onToken("\x00WEBSEARCH:" + payload)
            case "response.reasoning.delta", "response.summary.delta":
                onToken("\x00REASON:" + parseDelta(payload))
            case "response.completed":
                // done
            case "error":
                // Some org/model combos do not allow stream; fallback to non-stream
                needFallback = true
                errPayload = payload
            }
            // Fallback detection: if payload mentions web_search_call in other events
            if strings.Contains(payload, "web_search_call") {
                onToken("\x00WEBSEARCH:" + payload)
            }
            if strings.Contains(payload, "summary_text") || strings.Contains(payload, "\"type\":\"reasoning\"") {
                onToken("\x00REASON:" + parseDelta(payload))
            }
        }
        if needFallback { break }
    }
    if needFallback {
        // Retry same request without streaming
        rr.Stream = false
        body2, _ := json.Marshal(rr)
        httpClient2 := &http.Client{Timeout: c.timeout, Transport: transportWithProxy(c.proxyURL)}
        endpoint := c.baseURL + "/v1/responses"
        httpReq2, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body2))
        httpReq2.Header.Set("Authorization", "Bearer "+c.apiKey)
        httpReq2.Header.Set("Content-Type", "application/json")
        resp2, err2 := httpClient2.Do(httpReq2)
        if err2 != nil { return Response{}, fmt.Errorf("openai stream error then retry failed: %w | payload=%s", err2, errPayload) }
        defer resp2.Body.Close()
        if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
            b2, _ := io.ReadAll(io.LimitReader(resp2.Body, 8192))
            return Response{}, fmt.Errorf("openai: http %d: %s", resp2.StatusCode, string(b2))
        }
        var r responsesResp
        data2, err2 := io.ReadAll(resp2.Body)
        if err2 != nil { return Response{}, err2 }
        if err2 := json.Unmarshal(data2, &r); err2 != nil { return Response{}, err2 }
        text := r.OutputText
        if text == "" { text = r.AggregateOutputText() }
        if text != "" { onToken(text) }
        return Response{Text: strings.TrimSpace(text), PromptTokens: r.Usage.InputTokens, CompletionTokens: r.Usage.OutputTokens}, nil
    }
    if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) { return Response{}, err }
    return Response{Text: strings.TrimSpace(full.String())}, nil
}


func transportWithProxy(proxy string) http.RoundTripper {
    // Default transport copies http.DefaultTransport settings but allows proxy override.
    tr := &http.Transport{
        Proxy: http.ProxyFromEnvironment,
    }
    if proxy != "" {
        if u, err := url.Parse(proxy); err == nil {
            tr.Proxy = func(_ *http.Request) (*url.URL, error) { return u, nil }
        }
    }
    return tr
}

// Responses API structures
type responsesReq struct {
    Model               string      `json:"model"`
    Instructions        string      `json:"instructions,omitempty"`
    Input               any         `json:"input"`
    MaxOutputTokens     int         `json:"max_output_tokens,omitempty"`
    Temperature         float64     `json:"temperature,omitempty"`
    Stream              bool        `json:"stream,omitempty"`
    Stop                []string    `json:"stop,omitempty"`
    Tools               []map[string]any `json:"tools,omitempty"`
    ToolChoice          string           `json:"tool_choice,omitempty"`
    Include             []string         `json:"include,omitempty"`
    Reasoning           struct {
        Effort  string `json:"effort,omitempty"`
        Summary string `json:"summary,omitempty"`
    } `json:"reasoning,omitempty"`
}

type responsesResp struct {
    Output     []struct {
        Type    string `json:"type"`
        Role    string `json:"role"`
        Content []struct {
            Type string `json:"type"`
            Text string `json:"text"`
        } `json:"content"`
    } `json:"output"`
    OutputText string `json:"output_text"`
    Usage struct {
        InputTokens  int `json:"input_tokens"`
        OutputTokens int `json:"output_tokens"`
    } `json:"usage"`
}

func (r responsesResp) AggregateOutputText() string {
    var b strings.Builder
    for _, o := range r.Output {
        if len(o.Content) == 0 { continue }
        for _, c := range o.Content { if c.Type == "output_text" || c.Type == "text" { b.WriteString(c.Text) } }
    }
    return b.String()
}


func buildResponsesInputWithHistory(prompt string, history []ConversationMessage) any {
    if len(history) == 0 {
        return strings.TrimSpace(prompt)
    }
    
    var builder strings.Builder
    builder.WriteString("Previous conversation:\n")

    for _, msg := range history {
        switch msg.Role {
        case "user":
            builder.WriteString("User: ")
        case "assistant":
            builder.WriteString("Assistant: ")
        default:
            continue // Skip tool/reason messages for API
        }
        builder.WriteString(msg.Text)
        builder.WriteString("\n\n")
    }
    
    builder.WriteString("Current user message: ")
    builder.WriteString(strings.TrimSpace(prompt))
    
    return builder.String()
}

func parseDelta(payload string) string {
    // payload can be a JSON object {"delta":"..."} or a raw string
    if len(payload) > 0 && payload[0] == '{' {
        var tmp struct{ Delta string `json:"delta"` }
        if err := json.Unmarshal([]byte(payload), &tmp); err == nil { return tmp.Delta }
    }
    return payload
}

func supportsTemperature(model string) bool {
    // Some reasoning models do not accept temperature; omit for gpt-5/o3 families.
    m := strings.ToLower(model)
    if strings.HasPrefix(m, "gpt-5") || strings.HasPrefix(m, "o3") {
        return false
    }
    return true
}


