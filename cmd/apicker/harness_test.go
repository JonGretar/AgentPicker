package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
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
	rows, err := claudeSessions(home, cwd)
	if err != nil || len(rows) != 1 || rows[0].Title != "First prompt" || rows[0].Modified.UnixMilli() != 3000 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}

func TestCodexSessions(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	id := "12345678-1234-1234-1234-123456789abc"
	fixture(t, filepath.Join(home, ".codex", "sessions", "2026", "rollout-2026-01-01T00-00-00-"+id+".jsonl"), `{"type":"session_meta","payload":{"cwd":"`+cwd+`"}}`+"\n")
	fixture(t, filepath.Join(home, ".codex", "history.jsonl"), `{"session_id":"`+id+`","text":"Build it","ts":100}`+"\n"+`{"session_id":"other","text":"Ignore","ts":200}`+"\n")
	rows, err := codexSessions(home, cwd)
	if err != nil || len(rows) != 1 || rows[0].ID != id || rows[0].Modified.Unix() != 100 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}

func TestPiSessions(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	path := filepath.Join(home, ".pi", "agent", "sessions", "x.jsonl")
	fixture(t, path, `{"cwd":"`+cwd+`"}`+"\n"+`{"message":{"role":"assistant","content":[{"type":"text","text":"no"}]}}`+"\n"+`{"message":{"role":"user","content":[{"type":"text","text":"Hello pi"}]}}`+"\n")
	rows, err := piSessions(home, cwd)
	if err != nil || len(rows) != 1 || rows[0].ID != path || rows[0].Title != "Hello pi" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	default:
		return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
	}
}

func TestPickerFilterAndSelect(t *testing.T) {
	items := []choice{{harness: harness{name: "claude"}}, {harness: harness{name: "codex"}}, {harness: harness{name: "pi"}, session: &session{ID: "id", Title: "Fix bug", Modified: time.Now()}}}
	m := newPicker(items)
	for _, c := range "fix" {
		m.Update(key(string(c)))
	}
	if len(m.filtered) != 1 || m.filtered[0] != 2 {
		t.Fatalf("filtered=%v", m.filtered)
	}
	m.Update(key("enter"))
	if m.picked == nil || m.picked.session.ID != "id" {
		t.Fatalf("picked=%+v", m.picked)
	}
	m = newPicker(items)
	m.Update(key("esc"))
	if m.picked != nil {
		t.Fatal("cancel picked an item")
	}
	if got := compact("  hi\n there  ", 8); !strings.Contains(got, "hi there") {
		t.Fatalf("compact=%q", got)
	}
}
