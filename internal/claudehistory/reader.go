package claudehistory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Reader reads Claude Code conversation history from disk.
type Reader struct {
	baseDir string // ~/.claude/projects/
}

// NewReader creates a new Reader with the default base directory.
func NewReader() *Reader {
	homeDir, _ := os.UserHomeDir()
	return &Reader{baseDir: filepath.Join(homeDir, ".claude", "projects")}
}

// NewReaderWithBase creates a Reader with a custom base directory.
func NewReaderWithBase(baseDir string) *Reader {
	return &Reader{baseDir: baseDir}
}

// BaseDir returns the base directory for Claude Code projects.
func (r *Reader) BaseDir() string {
	return r.baseDir
}

// ListProjects returns all projects in the Claude Code projects directory.
func (r *Reader) ListProjects() ([]Project, error) {
	entries, err := os.ReadDir(r.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No projects yet
		}
		return nil, fmt.Errorf("reading projects dir: %w", err)
	}

	var projects []Project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectPath := entry.Name()
		projectName := unescapePath(projectPath)

		// Count sessions
		sessionFiles, _ := filepath.Glob(filepath.Join(r.baseDir, projectPath, "*.jsonl"))
		sessionCount := len(sessionFiles)

		if sessionCount > 0 {
			projects = append(projects, Project{
				Path:          projectPath,
				Name:          projectName,
				TotalSessions: sessionCount,
			})
		}
	}

	// Sort by name
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}

// ListSessions returns session metadata for a project.
func (r *Reader) ListSessions(projectPath string) ([]SessionMeta, error) {
	projectDir := filepath.Join(r.baseDir, projectPath)

	files, err := filepath.Glob(filepath.Join(projectDir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	var sessions []SessionMeta
	for _, filePath := range files {
		meta, err := r.getSessionMeta(filePath)
		if err != nil {
			// Skip files we can't parse
			continue
		}
		sessions = append(sessions, meta)
	}

	// Sort by start time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartTime.After(sessions[j].StartTime)
	})

	return sessions, nil
}

// getSessionMeta reads the first and last lines of a JSONL file to extract metadata.
func (r *Reader) getSessionMeta(filePath string) (SessionMeta, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return SessionMeta{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return SessionMeta{}, err
	}

	scanner := bufio.NewScanner(file)

	// Set a larger buffer for potentially long lines
	const maxScanTokenSize = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxScanTokenSize)

	var firstEntry, lastEntry JSONLEntry
	var lineCount int
	var totalIn, totalOut int
	var model string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry JSONLEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if lineCount == 0 {
			firstEntry = entry
		}
		lastEntry = entry
		lineCount++

		// Accumulate token usage
		if entry.Message != nil && entry.Message.Usage != nil {
			totalIn += entry.Message.Usage.InputTokens
			totalOut += entry.Message.Usage.OutputTokens
		}

		// Get model from first assistant message
		if model == "" && entry.Message != nil && entry.Message.Model != "" {
			model = entry.Message.Model
		}
	}

	if err := scanner.Err(); err != nil {
		return SessionMeta{}, fmt.Errorf("scanning file: %w", err)
	}

	if lineCount == 0 {
		return SessionMeta{}, fmt.Errorf("empty session file")
	}

	// Parse timestamps
	startTime, _ := time.Parse(time.RFC3339, firstEntry.Timestamp)
	endTime, _ := time.Parse(time.RFC3339, lastEntry.Timestamp)

	// Extract session ID from filename
	sessionID := strings.TrimSuffix(filepath.Base(filePath), ".jsonl")

	return SessionMeta{
		ID:        sessionID,
		StartTime: startTime,
		EndTime:   endTime,
		TurnCount: lineCount / 2, // Approximate turn count (user + assistant pairs)
		TotalIn:   totalIn,
		TotalOut:  totalOut,
		Model:     model,
		FilePath:  filePath,
		FileSize:  info.Size(),
	}, nil
}

// GetSession reads a complete session by ID.
// The sessionID can be a full session ID or a JSONL filename.
func (r *Reader) GetSession(sessionID string) (*Session, error) {
	// Search for the session file
	filePath, err := r.findSessionFile(sessionID)
	if err != nil {
		return nil, err
	}
	return r.GetSessionByPath(filePath)
}

// findSessionFile locates a session JSONL file by session ID.
func (r *Reader) findSessionFile(sessionID string) (string, error) {
	// Direct path if sessionID is already a path
	if filepath.IsAbs(sessionID) {
		if _, err := os.Stat(sessionID); err == nil {
			return sessionID, nil
		}
	}

	// Add .jsonl extension if not present
	if !strings.HasSuffix(sessionID, ".jsonl") {
		sessionID += ".jsonl"
	}

	// Search in all project directories
	projects, err := r.ListProjects()
	if err != nil {
		return "", err
	}

	for _, project := range projects {
		path := filepath.Join(r.baseDir, project.Path, sessionID)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("session not found: %s", sessionID)
}

// GetSessionByPath reads a session from a specific file path.
func (r *Reader) GetSessionByPath(filePath string) (*Session, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening session file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Set a larger buffer for potentially long lines
	const maxScanTokenSize = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxScanTokenSize)

	session := &Session{
		ID: strings.TrimSuffix(filepath.Base(filePath), ".jsonl"),
	}

	// Extract project path from file path
	dir := filepath.Dir(filePath)
	session.ProjectPath = filepath.Base(dir)
	session.ProjectName = unescapePath(session.ProjectPath)

	var messages []Message
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry JSONLEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed lines
			continue
		}

		msg := convertEntry(entry)
		messages = append(messages, msg)

		// Update session metadata
		if session.StartTime.IsZero() || msg.Timestamp.Before(session.StartTime) {
			session.StartTime = msg.Timestamp
		}
		if msg.Timestamp.After(session.EndTime) {
			session.EndTime = msg.Timestamp
		}

		if msg.Usage != nil {
			session.TotalIn += msg.Usage.InputTokens
			session.TotalOut += msg.Usage.OutputTokens
			session.CacheRead += msg.Usage.CacheReadTokens
			session.CacheWrite += msg.Usage.CacheCreationTokens
		}

		if session.Model == "" && msg.Model != "" {
			session.Model = msg.Model
		}
		if session.GitBranch == "" && msg.GitBranch != "" {
			session.GitBranch = msg.GitBranch
		}
		if session.Cwd == "" && msg.Cwd != "" {
			session.Cwd = msg.Cwd
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning session file: %w", err)
	}

	session.Messages = messages
	session.TurnCount = len(messages) / 2

	return session, nil
}

// convertEntry transforms a raw JSONL entry into our Message format.
func convertEntry(entry JSONLEntry) Message {
	msg := Message{
		SessionID:  entry.SessionID,
		Type:       entry.Type,
		ParentUUID: entry.ParentUUID,
		UUID:       entry.UUID,
		RequestID:  entry.RequestID,
		GitBranch:  entry.GitBranch,
		Cwd:        entry.Cwd,
	}

	// Parse timestamp
	if t, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
		msg.Timestamp = t
	}

	// Process message content
	if entry.Message != nil {
		msg.Model = entry.Message.Model
		msg.MessageID = entry.Message.ID
		msg.StopReason = entry.Message.StopReason

		// Convert usage
		if entry.Message.Usage != nil {
			msg.Usage = &TokenUsage{
				InputTokens:         entry.Message.Usage.InputTokens,
				OutputTokens:        entry.Message.Usage.OutputTokens,
				CacheReadTokens:     entry.Message.Usage.CacheReadInputTokens,
				CacheCreationTokens: entry.Message.Usage.CacheCreationTokens,
			}
		}

		// Convert content blocks
		for _, c := range entry.Message.Content {
			block := ContentBlock{Type: c.Type}

			switch c.Type {
			case "text":
				block.Text = c.Text
			case "thinking":
				block.Thinking = c.Thinking
			case "tool_use":
				block.ToolUse = &ToolUseBlock{
					ID:    c.ID,
					Name:  c.Name,
					Input: c.Input,
				}
			case "tool_result":
				block.ToolResult = &ToolResultBlock{
					ToolUseID: c.ToolUseID,
					Content:   c.Content,
					IsError:   c.IsError,
				}
			}

			msg.Content = append(msg.Content, block)
		}
	}

	return msg
}

// unescapePath converts the escaped project path back to readable form.
// e.g., "-Users-mark-dev-sunholo-ailang" -> "/Users/mark/dev/sunholo/ailang"
func unescapePath(escaped string) string {
	// Replace leading dash with /
	if strings.HasPrefix(escaped, "-") {
		escaped = "/" + escaped[1:]
	}
	// Replace remaining dashes with /
	return strings.ReplaceAll(escaped, "-", "/")
}

// GetMessagesByTimeRange returns messages within a time window.
// Useful for correlating spans to chat context.
func (r *Reader) GetMessagesByTimeRange(sessionID string, start, end time.Time) ([]Message, error) {
	session, err := r.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	var filtered []Message
	for _, msg := range session.Messages {
		if (msg.Timestamp.Equal(start) || msg.Timestamp.After(start)) &&
			(msg.Timestamp.Equal(end) || msg.Timestamp.Before(end)) {
			filtered = append(filtered, msg)
		}
	}

	return filtered, nil
}
