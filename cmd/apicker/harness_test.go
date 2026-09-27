package main

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/JonGretar/AgentPicker/cmd/apicker/harnesses"
)

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
	agents := harnesses.All()
	items := []choice{{harness: agents[0]}, {harness: agents[1]}, {harness: agents[3], session: &harnesses.Session{ID: "id", Title: "Fix bug", Modified: time.Now()}}}
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
