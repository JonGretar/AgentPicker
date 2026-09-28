package harnesses

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestClaudeSessions(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	path := filepath.Join(home, ".claude", "history.jsonl")
	fixture(t, path, `{"project":"`+cwd+`","sessionId":"a","display":"First prompt","timestamp":1000}`+"\n"+
		`{"project":"`+cwd+`/","sessionId":"a","display":"Next prompt","timestamp":3000}`+"\n"+
		`{"project":"/other","sessionId":"b","display":"Wrong directory","timestamp":3000}`+"\n"+
		`{"project":"`+cwd+`","sessionId":"c","display":"/help","timestamp":3000}`+"\n")
	rows, err := (claude{}).ListSessions(home, cwd)
	if err != nil || len(rows) != 1 || rows[0].Title != "First prompt" || rows[0].Modified.UnixMilli() != 3000 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}

func TestCodexSessions(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	id := "12345678-1234-1234-1234-123456789abc"
	fixture(t, filepath.Join(home, ".codex", "sessions", "2026", "rollout-2026-01-01T00-00-00-"+id+".jsonl"), `{"type":"session_meta","payload":{"cwd":"`+cwd+`"}}`+"\n")
	fixture(t, filepath.Join(home, ".codex", "history.jsonl"), `{"session_id":"`+id+`","text":"Build it","ts":100}`+"\n"+`{"session_id":"other","text":"Ignore","ts":200}`+"\n")
	rows, err := (codex{}).ListSessions(home, cwd)
	if err != nil || len(rows) != 1 || rows[0].ID != id || rows[0].Modified.Unix() != 100 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}

func TestPiSessions(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	path := filepath.Join(home, ".pi", "agent", "sessions", "x.jsonl")
	fixture(t, path, `{"cwd":"`+cwd+`"}`+"\n"+`{"message":{"role":"assistant","content":[{"type":"text","text":"no"}]}}`+"\n"+`{"message":{"role":"user","content":[{"type":"text","text":"Hello pi"}]}}`+"\n")
	rows, err := (pi{}).ListSessions(home, cwd)
	if err != nil || len(rows) != 1 || rows[0].ID != path || rows[0].Title != "Hello pi" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}

func TestOMPSessions(t *testing.T) {
	home, cwd, other := t.TempDir(), t.TempDir(), t.TempDir()
	h := omp{}
	if rows, err := h.ListSessions(home, cwd); err != nil || len(rows) != 0 {
		t.Fatalf("missing history: rows=%+v err=%v", rows, err)
	}
	root := filepath.Join(home, ".omp", "agent", "sessions")
	current := filepath.Join(root, "current", "new.jsonl")
	legacy := filepath.Join(root, "legacy", "old.jsonl")
	fixture(t, current, `{"type":"title","title":"Renamed chat"}`+"\n"+
		`{"type":"session","cwd":"`+cwd+`","title":"Original title"}`+"\n"+
		`{"type":"message","message":{"role":"user","content":[{"type":"text","text":"First prompt"}]}}`+"\n")
	fixture(t, legacy, `{"type":"session","cwd":"`+cwd+`/"}`+"\n"+
		`{"type":"message","message":{"role":"assistant","content":"Ignore"}}`+"\n"+
		`not json`+"\n"+
		`{"type":"message","message":{"role":"user","content":[{"type":"image"},{"type":"text","text":"Legacy prompt"}]}}`+"\n")
	fixture(t, filepath.Join(root, "other", "other.jsonl"), `{"type":"session","cwd":"`+other+`","title":"Other project"}`+"\n")
	fixture(t, filepath.Join(root, "bad", "bad.jsonl"), `{"type":"title","title":"Orphaned title"}`+"\n"+`not json`+"\n")
	fixture(t, filepath.Join(root, "empty", "empty.jsonl"), `{"type":"session","cwd":"`+cwd+`"}`+"\n")
	rows, err := h.ListSessions(home, cwd)
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	byID := map[string]Session{}
	for _, row := range rows {
		byID[row.ID] = row
		if row.Modified.IsZero() {
			t.Fatalf("missing modification time: %+v", row)
		}
	}
	if byID[current].Title != "Renamed chat" || byID[legacy].Title != "Legacy prompt" {
		t.Fatalf("rows=%+v", rows)
	}
}

func TestAiderSessions(t *testing.T) {
	home, cwd, other := t.TempDir(), t.TempDir(), t.TempDir()
	h := aider{}
	if sessions, err := h.ListSessions(home, cwd); err != nil || len(sessions) != 0 {
		t.Fatalf("missing history: sessions=%+v err=%v", sessions, err)
	}
	fixture(t, filepath.Join(other, aiderHistory), "unrelated chat")
	fixture(t, filepath.Join(cwd, aiderHistory), "")
	if sessions, err := h.ListSessions(home, cwd); err != nil || len(sessions) != 0 {
		t.Fatalf("empty history: sessions=%+v err=%v", sessions, err)
	}
	path := filepath.Join(cwd, aiderHistory)
	fixture(t, path, "## user\nHello aider\n")
	sessions, err := h.ListSessions(home, cwd)
	if err != nil || len(sessions) != 1 || sessions[0].ID != path || sessions[0].Title != "Continue chat" || sessions[0].Modified.IsZero() {
		t.Fatalf("history: sessions=%+v err=%v", sessions, err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell fixture")
	}
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	t.Setenv("PATH", dir)
	t.Setenv("APICKER_TEST_ARGS", argsFile)
	if h.IsAvailable() {
		t.Fatal("aider available without executable")
	}
	fixture(t, filepath.Join(dir, "aider"), "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$APICKER_TEST_ARGS\"\n")
	if err := os.Chmod(filepath.Join(dir, "aider"), 0755); err != nil {
		t.Fatal(err)
	}
	if !h.IsAvailable() {
		t.Fatal("aider unavailable with executable on PATH")
	}
	if err := h.NewSession(); err != nil {
		t.Fatal(err)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if want := "--no-auto-commits\n--no-restore-chat-history\n--chat-history-file\n" + aiderHistory + "\n"; string(args) != want {
		t.Fatalf("new args=%q, want %q", args, want)
	}
	if err := h.ResumeSession(sessions[0]); err != nil {
		t.Fatal(err)
	}
	args, err = os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if want := "--no-auto-commits\n--restore-chat-history\n--chat-history-file\n" + path + "\n"; string(args) != want {
		t.Fatalf("resume args=%q, want %q", args, want)
	}
}

func TestHarnessLaunches(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell fixture")
	}
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	t.Setenv("PATH", dir)
	t.Setenv("APICKER_TEST_ARGS", argsFile)
	agents := []struct {
		h      Harness
		resume string
	}{
		{claude{}, "--resume"}, {codex{}, "resume"}, {crush{}, "--session"}, {omp{}, "--resume"}, {opencode{}, "--session"}, {pi{}, "--session"},
	}
	for _, agent := range agents {
		t.Run(agent.h.Name(), func(t *testing.T) {
			if agent.h.IsAvailable() {
				t.Fatal("agent available without executable")
			}
			fixture(t, filepath.Join(dir, agent.h.Name()), "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$APICKER_TEST_ARGS\"\n")
			if err := os.Chmod(filepath.Join(dir, agent.h.Name()), 0755); err != nil {
				t.Fatal(err)
			}
			if !agent.h.IsAvailable() {
				t.Fatal("agent unavailable with executable on PATH")
			}
			if err := agent.h.NewSession(); err != nil {
				t.Fatal(err)
			}
			out, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(string(out)) != "" {
				t.Fatalf("new session args = %q", out)
			}
			if err := agent.h.ResumeSession(Session{ID: "chat"}); err != nil {
				t.Fatal(err)
			}
			out, err = os.ReadFile(argsFile)
			if err != nil {
				t.Fatal(err)
			}
			if string(out) != agent.resume+"\nchat\n" {
				t.Fatalf("resume args = %q", out)
			}
		})
	}
}

func TestCrushSessionsDoNotCreateDatabase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell fixture")
	}
	cwd, bin := t.TempDir(), t.TempDir()
	argsFile := filepath.Join(bin, "args")
	fixture(t, filepath.Join(bin, "crush"), "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\" > \"$APICKER_TEST_ARGS\"\nprintf '%s\\n' '[]'\n")
	if err := os.Chmod(filepath.Join(bin, "crush"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("APICKER_TEST_ARGS", argsFile)
	h := crush{}
	checkSkipped := func(dir string) {
		t.Helper()
		rows, err := h.ListSessions("", dir)
		if err != nil || len(rows) != 0 {
			t.Fatalf("missing database: rows=%+v err=%v", rows, err)
		}
		if _, err := os.Stat(argsFile); !os.IsNotExist(err) {
			t.Fatalf("Crush CLI invoked without database: %v", err)
		}
	}
	checkSkipped(cwd)
	if _, err := os.Stat(filepath.Join(cwd, ".crush")); !os.IsNotExist(err) {
		t.Fatalf("Crush created a data directory: %v", err)
	}
	if err := os.Mkdir(filepath.Join(cwd, ".crush"), 0755); err != nil {
		t.Fatal(err)
	}
	checkSkipped(cwd)
	fixture(t, filepath.Join(cwd, ".crush", "crush.db"), "")
	nested := filepath.Join(cwd, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	rows, err := h.ListSessions("", nested)
	if err != nil || len(rows) != 0 {
		t.Fatalf("existing database: rows=%+v err=%v", rows, err)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil || string(args) != nested+"\nsession\nlist\n--json\n" {
		t.Fatalf("Crush CLI cwd/args=%q err=%v", args, err)
	}
	if err := os.Remove(argsFile); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(nested, ".crush"), 0755); err != nil {
		t.Fatal(err)
	}
	checkSkipped(nested)
}

func TestOpenCodeSessions(t *testing.T) {
	home, cwd, other := t.TempDir(), t.TempDir(), t.TempDir()
	data, err := json.Marshal([]map[string]any{
		{"id": "ses_one", "title": "First", "directory": cwd, "updated": int64(1700000000123)},
		{"id": "ses_two", "title": "Second", "directory": cwd + string(os.PathSeparator), "updated": int64(1700000000456)},
		{"id": "ses_other", "title": "Different project", "directory": other, "updated": int64(1700000000789)},
		{"id": "", "title": "No ID", "directory": cwd, "updated": int64(1700000000123)},
		{"id": "ses_unknown", "title": "No directory", "updated": int64(1700000000123)},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseOpenCodeSessions(data, cwd)
	if err != nil || len(rows) != 2 || rows[0].ID != "ses_one" || rows[0].Modified.UnixMilli() != 1700000000123 || rows[1].ID != "ses_two" {
		t.Fatalf("sessions=%+v err=%v", rows, err)
	}
	if rows, err := parseOpenCodeSessions(nil, cwd); err != nil || len(rows) != 0 {
		t.Fatalf("empty sessions=%+v err=%v", rows, err)
	}
	if _, err := parseOpenCodeSessions([]byte("invalid JSON"), cwd); err == nil {
		t.Fatal("invalid JSON accepted")
	}
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell fixture")
	}
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	dataFile := filepath.Join(dir, "sessions.json")
	fixture(t, dataFile, string(data))
	fixture(t, filepath.Join(dir, "opencode"), "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\" > \"$APICKER_TEST_ARGS\"\n/bin/cat \"$APICKER_TEST_SESSIONS\"\n")
	if err := os.Chmod(filepath.Join(dir, "opencode"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("APICKER_TEST_ARGS", argsFile)
	t.Setenv("APICKER_TEST_SESSIONS", dataFile)
	rows, err = (opencode{}).ListSessions(home, cwd)
	if err != nil || len(rows) != 2 {
		t.Fatalf("CLI sessions=%+v err=%v", rows, err)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(args) != cwd+"\nsession\nlist\n--format\njson\n" {
		t.Fatalf("CLI cwd/args=%q", args)
	}
}

func TestRegisteredHarnesses(t *testing.T) {
	got := All()
	if len(got) != 7 {
		t.Fatalf("registered %d harnesses, want 7", len(got))
	}
	for i, name := range []string{"aider", "claude", "codex", "crush", "omp", "opencode", "pi"} {
		if got[i].Name() != name {
			t.Fatalf("harness %d = %s, want %s", i, got[i].Name(), name)
		}

	}
}
