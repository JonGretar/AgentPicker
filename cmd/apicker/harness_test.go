package main

import (
	"errors"
	"os/exec"
	"runtime"
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
