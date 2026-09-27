package main

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

func TestPickAndRun(t *testing.T) {
	c := choice{harness: harnesses.All()[0]}
	for _, tc := range []struct {
		name                  string
		loop                  bool
		wantPicks, wantStarts int
	}{
		{"default launches once", false, 1, 1},
		{"loop returns to picker", true, 3, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			picks, starts := 0, 0
			err := pickAndRun(tc.loop, func() (*choice, error) {
				picks++
				if picks == 3 { // Esc exits the loop
					return nil, nil
				}
				return &c, nil
			}, func(choice) error { starts++; return nil })
			if err != nil || picks != tc.wantPicks || starts != tc.wantStarts {
				t.Fatalf("picks=%d starts=%d err=%v", picks, starts, err)
			}
		})
	}
	failure := errors.New("launch failed")
	picks := 0
	err := pickAndRun(true, func() (*choice, error) { picks++; return &c, nil }, func(choice) error { return failure })
	if !errors.Is(err, failure) || picks != 1 {
		t.Fatalf("error=%v picks=%d", err, picks)
	}
}

func TestLoopAfterNonzeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell fixture")
	}
	c := choice{harness: harnesses.All()[0]}
	picks := 0
	exitErr := exec.Command("sh", "-c", "exit 7").Run()
	var exit *exec.ExitError
	if !errors.As(exitErr, &exit) {
		t.Fatalf("expected ExitError, got %v", exitErr)
	}
	err := pickAndRun(true, func() (*choice, error) {
		picks++
		if picks == 2 {
			return nil, nil
		}
		return &c, nil
	}, func(choice) error { return exitErr })
	if err != nil || picks != 2 {
		t.Fatalf("error=%v picks=%d", err, picks)
	}
	err = pickAndRun(false, func() (*choice, error) { return &c, nil }, func(choice) error { return exitErr })
	if !errors.Is(err, exitErr) {
		t.Fatalf("non-loop exit error = %v", err)
	}
}

func TestSessionRowStyles(t *testing.T) {
	var h harnesses.Harness
	for _, agent := range harnesses.All() {
		if agent.Name() == "opencode" {
			h = agent
			break
		}
	}
	if h == nil {
		t.Fatal("opencode harness not registered")
	}
	now := time.Unix(1700000000, 0)
	c := choice{harness: h, session: &harnesses.Session{Title: "Project greeting", Modified: now}}
	for _, selected := range []bool{false, true} {
		row := renderChoice(c, now, 80, 8, selected)
		for _, segment := range []string{
			harnessStyle.Bold(selected).Render("opencode"),
			ageStyle.Bold(selected).Render("   0m  "),
			titleStyle.Bold(selected).Render("Project greeting"),
		} {
			if !strings.Contains(row, segment) {
				t.Fatalf("selected=%v: missing styled segment %q in %q", selected, segment, row)
			}
		}
		if lipgloss.Width(row) > 80 {
			t.Fatalf("row too wide: %d", lipgloss.Width(row))
		}
	}
	if harnessStyle.Render("x") == ageStyle.Render("x") || ageStyle.Render("x") == titleStyle.Render("x") {
		t.Fatal("harness, age and title should have distinct colors")
	}
	narrow := renderChoice(c, now, 20, 8, false)
	if lipgloss.Width(narrow) > 20 {
		t.Fatalf("truncated row too wide: %d", lipgloss.Width(narrow))
	}
	newRow := renderChoice(choice{harness: h}, now, 80, 8, false)
	if !strings.Contains(newRow, harnessStyle.Render("opencode")) {
		t.Fatalf("new row missing colored harness: %q", newRow)
	}
}

func TestSessionAgeColumnAlignment(t *testing.T) {
	now := time.Unix(1700000000, 0)
	var rows []choice
	for _, h := range harnesses.All() {
		if h.Name() == "opencode" || h.Name() == "crush" {
			rows = append(rows, choice{harness: h, session: &harnesses.Session{Title: "Example", Modified: now}})
		}
	}
	m := newPicker(rows)
	if m.nameWidth != 8 {
		t.Fatalf("name column = %d, want 8", m.nameWidth)
	}
	for _, c := range rows {
		row := renderChoice(c, now, 80, m.nameWidth, false)
		ageStart := strings.Index(row, ageStyle.Render("   0m  "))
		if ageStart < 0 {
			t.Fatalf("missing age in %q", row)
		}
		if col := lipgloss.Width(row[:ageStart]); col != m.nameWidth {
			t.Fatalf("%s age starts in column %d, want %d", c.harness.Name(), col, m.nameWidth)
		}
	}
}

func TestPickerFilterAndSelect(t *testing.T) {
	agents := harnesses.All()
	items := []choice{{harness: agents[0]}, {harness: agents[1]}, {harness: agents[4], session: &harnesses.Session{ID: "id", Title: "Fix bug", Modified: time.Now()}}}
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
