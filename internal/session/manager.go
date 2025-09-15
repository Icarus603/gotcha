package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Metadata holds global session information
type Metadata struct {
	LastSessionID string            `json:"last_session_id"`
	Sessions      []SessionInfo     `json:"sessions"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

// SessionInfo holds basic session information
type SessionInfo struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Title     string    `json:"title,omitempty"`
}

// Context holds the complete conversation state for persistence
type Context struct {
	SessionID     string     `json:"session_id"`
	Conversations []ChatMsg  `json:"conversations"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	NoteCount     int        `json:"note_count"`
	LastSaveIndex int        `json:"last_save_index"` // Track which conversations have been saved to transcript
}

// ChatMsg represents a conversation message (matches TUI structure)
type ChatMsg struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// Manager handles session creation, loading, and persistence
type Manager struct {
	basePath string
}

// NewManager creates a new session manager
func NewManager() *Manager {
	cwd, _ := os.Getwd()
	basePath := filepath.Join(cwd, ".gotcha")
	return &Manager{basePath: basePath}
}

// CreateNewSession creates a new session with sequential ID
func (m *Manager) CreateNewSession() (string, error) {
	if err := m.ensureDirectories(); err != nil {
		return "", err
	}

	metadata, err := m.loadMetadata()
	if err != nil {
		return "", err
	}

	// Generate next session ID
	sessionID := m.generateNextSessionID(metadata.Sessions)

	// Create session directory
	sessionDir := filepath.Join(m.basePath, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create session directory: %w", err)
	}

	// Create initial context
	context := Context{
		SessionID:     sessionID,
		Conversations: []ChatMsg{},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		NoteCount:     0,
	}

	if err := m.saveContext(sessionID, context); err != nil {
		return "", err
	}

	// Update metadata
	sessionInfo := SessionInfo{
		ID:        sessionID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	metadata.Sessions = append(metadata.Sessions, sessionInfo)
	metadata.LastSessionID = sessionID
	metadata.UpdatedAt = time.Now()

	if err := m.saveMetadata(metadata); err != nil {
		return "", err
	}

	return sessionID, nil
}

// LoadSession loads an existing session context
func (m *Manager) LoadSession(sessionID string) (Context, error) {
	contextPath := filepath.Join(m.basePath, "sessions", sessionID, "context.json")

	data, err := os.ReadFile(contextPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new context for existing session directory
			return Context{
				SessionID:     sessionID,
				Conversations: []ChatMsg{},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
				NoteCount:     0,
			}, nil
		}
		return Context{}, fmt.Errorf("failed to read context: %w", err)
	}

	var context Context
	if err := json.Unmarshal(data, &context); err != nil {
		return Context{}, fmt.Errorf("failed to parse context: %w", err)
	}

	return context, nil
}

// SaveSession saves the current session context
func (m *Manager) SaveSession(sessionID string, context Context) error {
	context.UpdatedAt = time.Now()
	return m.saveContext(sessionID, context)
}

// GetLastSession returns the most recent session ID
func (m *Manager) GetLastSession() (string, error) {
	metadata, err := m.loadMetadata()
	if err != nil {
		return "", err
	}

	if metadata.LastSessionID == "" && len(metadata.Sessions) > 0 {
		// Fallback to most recent session
		sort.Slice(metadata.Sessions, func(i, j int) bool {
			return metadata.Sessions[i].UpdatedAt.After(metadata.Sessions[j].UpdatedAt)
		})
		return metadata.Sessions[0].ID, nil
	}

	return metadata.LastSessionID, nil
}

// ListSessions returns all available sessions
func (m *Manager) ListSessions() ([]SessionInfo, error) {
	metadata, err := m.loadMetadata()
	if err != nil {
		return nil, err
	}

	// Sort by most recent first
	sessions := make([]SessionInfo, len(metadata.Sessions))
	copy(sessions, metadata.Sessions)

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// Private helper methods

func (m *Manager) ensureDirectories() error {
	sessionsDir := filepath.Join(m.basePath, "sessions")
	return os.MkdirAll(sessionsDir, 0o755)
}

func (m *Manager) loadMetadata() (Metadata, error) {
	metadataPath := filepath.Join(m.basePath, "metadata.json")

	// Return empty metadata if file doesn't exist
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return Metadata{
			Sessions:  []SessionInfo{},
			UpdatedAt: time.Now(),
		}, nil
	}

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return Metadata{}, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return Metadata{}, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return metadata, nil
}

func (m *Manager) saveMetadata(metadata Metadata) error {
	metadataPath := filepath.Join(m.basePath, "metadata.json")

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return os.WriteFile(metadataPath, data, 0o644)
}

func (m *Manager) saveContext(sessionID string, context Context) error {
	contextPath := filepath.Join(m.basePath, "sessions", sessionID, "context.json")

	// Ensure session directory exists
	sessionDir := filepath.Dir(contextPath)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	data, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	return os.WriteFile(contextPath, data, 0o644)
}

func (m *Manager) generateNextSessionID(sessions []SessionInfo) string {
	maxNum := 0

	for _, session := range sessions {
		if strings.HasPrefix(session.ID, "session-") {
			numStr := strings.TrimPrefix(session.ID, "session-")
			if num, err := strconv.Atoi(numStr); err == nil && num > maxNum {
				maxNum = num
			}
		}
	}

	return fmt.Sprintf("session-%d", maxNum+1)
}

// SaveTranscript generates or appends to the session's transcript.md file
func (m *Manager) SaveTranscript(sessionID string, context Context) error {
	transcriptPath := filepath.Join(m.basePath, "sessions", sessionID, "transcript.md")

	// Check if file exists
	fileExists := true
	if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
		fileExists = false
	}

	// Open file for writing (create or append)
	var file *os.File
	var err error
	if fileExists {
		file, err = os.OpenFile(transcriptPath, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(transcriptPath)
	}
	if err != nil {
		return fmt.Errorf("failed to open transcript file: %w", err)
	}
	defer file.Close()

	// Write header if this is a new file
	if !fileExists {
		header := fmt.Sprintf("# Session %s\n\n**Created:** %s\n\n",
			sessionID, context.CreatedAt.Format("2006-01-02 15:04:05"))
		if _, err := file.WriteString(header); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
	}

	// Determine which conversations are new since last save
	newConversations := context.Conversations[context.LastSaveIndex:]
	if len(newConversations) == 0 {
		return nil // Nothing new to save
	}

	// Write save section header
	saveTime := time.Now().Format("2006-01-02 15:04:05")
	sectionHeader := fmt.Sprintf("\n---\n\n**Saved:** %s\n\n", saveTime)
	if _, err := file.WriteString(sectionHeader); err != nil {
		return fmt.Errorf("failed to write section header: %w", err)
	}

	// Write new conversations
	for _, msg := range newConversations {
		var roleLabel string
		switch msg.Role {
		case "user":
			roleLabel = "**You:**"
		case "assistant":
			roleLabel = "**Assistant:**"
		default:
			roleLabel = fmt.Sprintf("**%s:**", strings.Title(msg.Role))
		}

		conversation := fmt.Sprintf("%s\n%s\n\n", roleLabel, msg.Text)
		if _, err := file.WriteString(conversation); err != nil {
			return fmt.Errorf("failed to write conversation: %w", err)
		}
	}

	return nil
}

// UpdateLastSaveIndex updates the context to mark conversations as saved
func (m *Manager) UpdateLastSaveIndex(sessionID string, context Context) error {
	context.LastSaveIndex = len(context.Conversations)
	return m.SaveSession(sessionID, context)
}