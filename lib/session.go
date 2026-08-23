package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// ChatSession is one saved conversation. Cwd scopes it to the directory it
// was started in so `/load` only offers sessions relevant to where scout is
// running, and Hint records the first prompt the user typed so a session can
// be recognized in the picker without opening it.
type ChatSession struct {
	SessionID string                   `json:"session_id"`
	Cwd       string                   `json:"cwd"`
	CreatedAt time.Time                `json:"created_at"`
	Hint      string                   `json:"hint"`
	History   []anthropic.MessageParam `json:"history"`
	// todo: add mutex for concurrency safety
}

// SessionSummary is the listing-level view of a saved session: enough to
// render the picker without decoding the message history into SDK union
// types, which is both the slow part and the part most likely to fail on an
// older or hand-edited file.
type SessionSummary struct {
	SessionID string
	Path      string
	CreatedAt time.Time
	Hint      string
	Messages  int
}

// SessionsDir is where sessions are persisted. os.UserHomeDir is used rather
// than $HOME so the path also resolves on Windows.
func SessionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating home directory: %w", err)
	}
	return filepath.Join(home, ".scout", "sessions"), nil
}

// NewChatSession starts a session for cwd, named so that it is both
// human-readable and sorts chronologically by filename.
func NewChatSession(cwd string, now time.Time) *ChatSession {
	return &ChatSession{
		SessionID: newSessionID(cwd, now),
		Cwd:       cwd,
		CreatedAt: now,
		History:   []anthropic.MessageParam{},
	}
}

// newSessionID builds an id like "20260823-181205-scout": a sortable
// timestamp plus the directory name. This replaces the old bare UUID, which
// said nothing about when a session ran or what it belonged to. The full cwd
// is kept in the session body, so two directories sharing a base name are
// still told apart on load; the numeric suffix only breaks ties between
// sessions started in the same second.
func newSessionID(cwd string, now time.Time) string {
	base := fmt.Sprintf("%s-%s", now.Format("20060102-150405"), slugify(filepath.Base(cwd)))

	dir, err := SessionsDir()
	if err != nil {
		return base
	}

	id := base
	for i := 2; i < 100; i++ {
		if _, err := os.Stat(filepath.Join(dir, id+".json")); os.IsNotExist(err) {
			return id
		}
		id = fmt.Sprintf("%s-%d", base, i)
	}
	return id
}

// slugify reduces a path segment to lowercase alphanumerics joined by
// dashes, so it is safe to use verbatim in a filename on any platform.
func slugify(s string) string {
	var b strings.Builder
	dashed := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dashed = false
		default:
			if !dashed && b.Len() > 0 {
				b.WriteByte('-')
				dashed = true
			}
		}
	}
	if out := strings.Trim(b.String(), "-"); out != "" {
		return out
	}
	return "session"
}

// Save writes the session to disk, creating the sessions directory if this
// is the first run.
func (cs *ChatSession) Save() error {
	dir, err := SessionsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	path := filepath.Join(dir, cs.SessionID+".json")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create session file: %w", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(cs); err != nil {
		return fmt.Errorf("failed to encode session to file: %w", err)
	}
	return nil
}

// ListSessions returns the saved sessions started in cwd, newest first.
// Files that aren't readable session JSON, and sessions from before the cwd
// field existed, are skipped rather than failing the whole listing.
func ListSessions(cwd string) ([]SessionSummary, error) {
	dir, err := SessionsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var summaries []SessionSummary
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// History is decoded as raw JSON so the listing only pays for a
		// message count, never for building content-block unions.
		var meta struct {
			SessionID string            `json:"session_id"`
			Cwd       string            `json:"cwd"`
			CreatedAt time.Time         `json:"created_at"`
			Hint      string            `json:"hint"`
			History   []json.RawMessage `json:"history"`
		}
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		if meta.Cwd != cwd || meta.SessionID == "" {
			continue
		}

		summaries = append(summaries, SessionSummary{
			SessionID: meta.SessionID,
			Path:      path,
			CreatedAt: meta.CreatedAt,
			Hint:      meta.Hint,
			Messages:  len(meta.History),
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].CreatedAt.After(summaries[j].CreatedAt)
	})
	return summaries, nil
}

// LoadSession reads a single session by id.
func LoadSession(sessionID string) (*ChatSession, error) {
	dir, err := SessionsDir()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(dir, sessionID+".json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read session %s: %w", sessionID, err)
	}

	var session ChatSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to decode session %s: %w", sessionID, err)
	}
	return &session, nil
}
