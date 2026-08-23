package lib

import (
	"bufio"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// saveSession writes a session for cwd into the test's fake home.
func saveSession(t *testing.T, cwd, hint string, at time.Time, history []anthropic.MessageParam) *ChatSession {
	t.Helper()

	session := NewChatSession(cwd, at)
	session.Hint = hint
	session.History = history
	if err := session.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	return session
}

func TestSessionIDIsReadableAndSortable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	at := time.Date(2026, 8, 23, 18, 12, 5, 0, time.UTC)
	if got, want := newSessionID("/home/connor/projects/scout", at), "20260823-181205-scout"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	// Same second, same directory: the tie is broken rather than clobbering
	// the existing file.
	saveSession(t, "/home/connor/projects/scout", "first", at, nil)
	if got, want := newSessionID("/home/connor/projects/scout", at), "20260823-181205-scout-2"; got != want {
		t.Fatalf("collision: got %q, want %q", got, want)
	}
}

func TestListSessionsIsScopedToCwdAndNewestFirst(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	base := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	saveSession(t, "/proj/scout", "older scout prompt", base, nil)
	saveSession(t, "/proj/scout", "newer scout prompt", base.Add(time.Hour), nil)
	saveSession(t, "/proj/other", "different project", base.Add(2*time.Hour), nil)

	got, err := ListSessions("/proj/scout")
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 sessions scoped to /proj/scout, got %d", len(got))
	}
	if got[0].Hint != "newer scout prompt" {
		t.Fatalf("expected newest first, got %q", got[0].Hint)
	}
	if got[1].Hint != "older scout prompt" {
		t.Fatalf("expected older second, got %q", got[1].Hint)
	}
}

func TestListSessionsSkipsUnreadableAndLegacyFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	saveSession(t, "/proj/scout", "real session", time.Now(), nil)

	dir, err := SessionsDir()
	if err != nil {
		t.Fatal(err)
	}
	// A pre-cwd session (old UUID filename, no cwd field) and a junk file.
	writeFile(t, dir, "9f1c-legacy.json", `{"SessionID":"9f1c-legacy","History":[]}`)
	writeFile(t, dir, "broken.json", `{not json`)

	got, err := ListSessions("/proj/scout")
	if err != nil {
		t.Fatalf("list should not fail on bad files: %v", err)
	}
	if len(got) != 1 || got[0].Hint != "real session" {
		t.Fatalf("expected only the valid session, got %+v", got)
	}
}

func TestSelectSessionRestoresToolCallHistory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// A history with a tool_use / tool_result pair -- the shape that has to
	// survive a save/load round-trip for a resumed session to be replayable.
	history := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("list the files")),
		anthropic.NewAssistantMessage(anthropic.ContentBlockParamUnion{
			OfToolUse: &anthropic.ToolUseBlockParam{
				ID: "tu_1", Name: "shell_command",
				Input: map[string]any{"command": "ls"},
			},
		}),
		anthropic.NewUserMessage(anthropic.NewToolResultBlock("tu_1", "main.go", false)),
	}

	base := time.Date(2026, 8, 23, 9, 0, 0, 0, time.UTC)
	saveSession(t, "/proj/scout", "list the files", base, history)
	saveSession(t, "/proj/scout", "newest prompt", base.Add(time.Hour), nil)

	// "2" picks the older session, since the list is newest-first.
	scanner := bufio.NewScanner(strings.NewReader("2\n"))
	got, err := SelectSession(scanner, "/proj/scout")
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if got == nil {
		t.Fatal("expected a session")
	}

	if got.Hint != "list the files" {
		t.Fatalf("picked the wrong session: %q", got.Hint)
	}
	if len(got.History) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(got.History))
	}
	if tu := got.History[1].Content[0].OfToolUse; tu == nil {
		t.Fatal("tool_use block lost in round-trip")
	} else if tu.ID != "tu_1" {
		t.Fatalf("tool_use id lost: %q", tu.ID)
	}
	if got.History[2].Content[0].OfToolResult == nil {
		t.Fatal("tool_result block lost in round-trip")
	}
}

func TestSelectSessionCancelsAndHandlesEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Nothing saved for this cwd at all.
	got, err := SelectSession(bufio.NewScanner(strings.NewReader("\n")), "/proj/empty")
	if err != nil || got != nil {
		t.Fatalf("expected (nil, nil) for no sessions, got (%v, %v)", got, err)
	}

	saveSession(t, "/proj/scout", "a prompt", time.Now(), nil)

	// Bad input then a blank line: reprompts, then cancels.
	got, err = SelectSession(bufio.NewScanner(strings.NewReader("99\nabc\n\n")), "/proj/scout")
	if err != nil || got != nil {
		t.Fatalf("expected cancel to return (nil, nil), got (%v, %v)", got, err)
	}
}

func TestFormatHint(t *testing.T) {
	if got := formatHint(""); got != "(no prompt recorded)" {
		t.Fatalf("empty hint: %q", got)
	}
	if got := formatHint("  spaced   out\tprompt "); got != "spaced out prompt" {
		t.Fatalf("whitespace not collapsed: %q", got)
	}

	long := strings.Repeat("x", 200)
	got := formatHint(long)
	if len([]rune(got)) != maxHintWidth {
		t.Fatalf("expected %d runes, got %d", maxHintWidth, len([]rune(got)))
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis, got %q", got)
	}
}

func TestResumeThenSaveWritesBackToTheSameSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	original := saveSession(t, "/proj/scout", "original prompt",
		time.Date(2026, 8, 23, 9, 0, 0, 0, time.UTC),
		[]anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock("original prompt"))})

	loaded, err := LoadSession(original.SessionID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// A fresh client starts its own session, then resumes the loaded one.
	sc := &ScoutClient{Session: NewChatSession("/proj/scout", time.Now())}
	sc.ResumeSession(loaded)
	sc.AddMessage("a follow up")
	if err := sc.SaveSession(); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := ListSessions("/proj/scout")
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("resuming should not fork a new session; got %d sessions", len(got))
	}
	if got[0].SessionID != original.SessionID {
		t.Fatalf("wrote to %q, want %q", got[0].SessionID, original.SessionID)
	}
	if got[0].Messages != 2 {
		t.Fatalf("expected the follow-up to be appended, got %d messages", got[0].Messages)
	}
	if got[0].Hint != "original prompt" {
		t.Fatalf("hint should stay the session's first prompt, got %q", got[0].Hint)
	}
}
