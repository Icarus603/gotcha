package agent

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "gotcha/internal/app"
    "gotcha/internal/llm"
    "gotcha/internal/platform"
)

// Researcher coordinates a minimal research pipeline using an LLM planner and writer.
// This MVP does not yet perform live web search; it focuses on plan+compose and
// persists a Markdown draft to the session report path.
type Researcher struct {
    bus EventBus
    llm llm.Client
    svc *app.Service
}

func NewResearcher(bus EventBus, llmClient llm.Client, svc *app.Service) *Researcher {
    return &Researcher{bus: bus, llm: llmClient, svc: svc}
}

// Start kicks off a background planning and composition run.
func (r *Researcher) Start(sessionID, prompt string) {
    go r.run(sessionID, prompt)
}

type plan struct {
    Title    string    `json:"title"`
    Sections []section `json:"sections"`
}
type section struct {
    Heading      string `json:"heading"`
    Instructions string `json:"instructions"`
}

func (r *Researcher) run(sessionID, prompt string) {
    ctx := context.Background()
    // Outline phase
    r.bus.Publish(ctx, Event{SessionID: sessionID, Phase: PhaseOutline, Type: "started", At: time.Now()})
    pl, err := r.plan(ctx, prompt)
    if err != nil {
        r.bus.Publish(ctx, Event{SessionID: sessionID, Phase: PhaseOutline, Type: "error", Err: err.Error(), At: time.Now()})
        return
    }
    r.bus.Publish(ctx, Event{SessionID: sessionID, Phase: PhaseOutline, Type: "done", At: time.Now(), Meta: map[string]any{"title": pl.Title, "sections": len(pl.Sections)}})

    // Compose phase
    total := len(pl.Sections)
    if total == 0 { total = 1 }
    r.bus.Publish(ctx, Event{SessionID: sessionID, Phase: PhaseCompose, Type: "started", At: time.Now(), Progress: Progress{Done: 0, Total: total}})
    var parts []string
    for i, s := range pl.Sections {
        txt, werr := r.writeSection(ctx, prompt, pl.Title, s)
        if werr != nil { txt = fmt.Sprintf("## %s\n\n(Unable to compose section: %v)\n", safeHead(s.Heading), werr) }
        parts = append(parts, txt)
        r.bus.Publish(ctx, Event{SessionID: sessionID, Phase: PhaseCompose, Type: "progress", At: time.Now(), Progress: Progress{Done: i+1, Total: total}})
    }
    doc := r.assembleMarkdown(pl.Title, parts)
    // Persist
    path := r.svc.ReportPath(sessionID)
    if err := platform.WriteFileAtomic(path, []byte(doc)); err != nil {
        r.bus.Publish(ctx, Event{SessionID: sessionID, Phase: PhaseCompose, Type: "error", Err: err.Error(), At: time.Now()})
        return
    }
    r.bus.Publish(ctx, Event{SessionID: sessionID, Phase: PhaseCompose, Type: "done", At: time.Now(), Meta: map[string]any{"path": path}})
}

func (r *Researcher) plan(ctx context.Context, userPrompt string) (plan, error) {
    // If no LLM configured, return a deterministic fallback plan.
    if r.llm == nil {
        return plan{
            Title:   fallbackTitle(userPrompt),
            Sections: []section{
                {Heading: "Overview", Instructions: "Explain key concepts, context, and relevance."},
                {Heading: "Key Points", Instructions: "List and explain main findings or considerations."},
                {Heading: "Conclusion", Instructions: "Summarize takeaways and next steps."},
            },
        }, nil
    }
    // Strict JSON planner prompt
    sys := "You are a meticulous research planner. Return strict JSON with fields: title (5-9 words), sections (array of {heading, instructions}). No extra text."
    u := fmt.Sprintf("Research prompt: %s\n\nReturn JSON only.", strings.TrimSpace(userPrompt))
    res, err := r.llm.Complete(ctx, llm.Request{System: sys, Prompt: u, MaxTokens: 600, Temperature: 0.2}, nil)
    if err != nil { return plan{}, err }
    // Extract JSON from possible code fences
    raw := strings.TrimSpace(res.Text)
    raw = trimFences(raw)
    var p plan
    if err := json.Unmarshal([]byte(raw), &p); err != nil {
        // Fallback simple plan
        p = plan{Title: fallbackTitle(userPrompt), Sections: []section{{Heading: "Overview", Instructions: "Explain key concepts, context, and relevance."}, {Heading: "Key Points", Instructions: "List and explain main findings or considerations."}, {Heading: "Conclusion", Instructions: "Summarize takeaways and next steps."}}}
    }
    if strings.TrimSpace(p.Title) == "" { p.Title = fallbackTitle(userPrompt) }
    return p, nil
}

func (r *Researcher) writeSection(ctx context.Context, userPrompt, title string, s section) (string, error) {
    if r.llm == nil {
        // Deterministic offline content so the app remains usable without API keys.
        body := fmt.Sprintf("## %s\n\n%s\n\n- Prompt: %s\n- Note: LLM not configured; this is a placeholder.\n",
            safeHead(s.Heading), strings.TrimSpace(s.Instructions), strings.TrimSpace(userPrompt))
        return body, nil
    }
    sys := "You write concise, well-structured Markdown sections. No preamble, no chatty tone. Use headings provided. Cite with footnotes [^1] style only if sources are given; otherwise omit citations."
    prompt := fmt.Sprintf("Title: %s\nUser Prompt: %s\n\nWrite the section below as Markdown.\nHeading: %s\nInstructions: %s\n",
        strings.TrimSpace(title), strings.TrimSpace(userPrompt), safeHead(s.Heading), strings.TrimSpace(s.Instructions))
    res, err := r.llm.Complete(ctx, llm.Request{System: sys, Prompt: prompt, MaxTokens: 800, Temperature: 0.4}, nil)
    if err != nil { return "", err }
    out := strings.TrimSpace(res.Text)
    if !strings.HasPrefix(out, "#") && !strings.HasPrefix(strings.ToLower(out), fmt.Sprintf("## %s", strings.ToLower(s.Heading))) {
        out = fmt.Sprintf("## %s\n\n%s", safeHead(s.Heading), out)
    }
    return out, nil
}

func (r *Researcher) assembleMarkdown(title string, sections []string) string {
    var b strings.Builder
    // front matter
    b.WriteString("---\n")
    b.WriteString("title: \""+escapeYAML(title)+"\"\n")
    b.WriteString("generated_at: \""+time.Now().Format(time.RFC3339)+"\"\n")
    b.WriteString("tool: gotcha\n")
    b.WriteString("---\n\n")
    b.WriteString("# "+title+"\n\n")
    for _, s := range sections { b.WriteString(s); if !strings.HasSuffix(s, "\n") { b.WriteString("\n") }; b.WriteString("\n") }
    // Placeholder sources block (to be populated once web research lands)
    b.WriteString("\n---\n\n")
    b.WriteString("## Sources\n\n")
    b.WriteString("(Sources to be added by web research pipeline.)\n")
    return b.String()
}

func trimFences(s string) string {
    s = strings.TrimSpace(s)
    if strings.HasPrefix(s, "```") {
        // drop first line (``` or ```json)
        if idx := strings.IndexByte(s, '\n'); idx >= 0 {
            s = s[idx+1:]
        } else {
            s = ""
        }
        s = strings.TrimSpace(s)
        if i := strings.LastIndex(s, "```"); i >= 0 { s = strings.TrimSpace(s[:i]) }
    }
    return s
}

func safeHead(h string) string { return strings.TrimSpace(strings.Trim(h, "# ")) }

func escapeYAML(s string) string { return strings.ReplaceAll(s, "\"", "\\\"") }

func fallbackTitle(s string) string {
    s = strings.TrimSpace(s)
    if s == "" { return "Untitled Research" }
    // naive: capture first ~8 words
    parts := strings.Fields(s)
    if len(parts) > 8 { parts = parts[:8] }
    return strings.Join(parts, " ")
}
