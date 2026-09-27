package harnesses

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type claude struct{}

func init()                      { Register(claude{}) }
func (claude) Name() string      { return "claude" }
func (claude) IsAvailable() bool { _, err := exec.LookPath("claude"); return err == nil }
func (claude) NewSession() error { return runInteractive(exec.Command("claude")) }
func (claude) ResumeSession(session Session) error {
	return runInteractive(exec.Command("claude", "--resume", session.ID))
}

func (claude) ListSessions(home, cwd string) ([]Session, error) {
	var rows []Session
	byID := make(map[string]int)
	err := readLines(filepath.Join(home, ".claude", "history.jsonl"), func(line []byte) {
		var r struct {
			Project   string `json:"project"`
			ID        string `json:"sessionId"`
			Text      string `json:"display"`
			Timestamp int64  `json:"timestamp"`
		}
		if json.Unmarshal(line, &r) != nil || !sameDir(r.Project, cwd) || r.ID == "" || r.Text == "" || strings.HasPrefix(r.Text, "/") {
			return
		}
		modified := time.UnixMilli(r.Timestamp)
		if i, ok := byID[r.ID]; ok {
			if modified.After(rows[i].Modified) {
				rows[i].Modified = modified
			}
		} else {
			byID[r.ID] = len(rows)
			rows = append(rows, Session{r.ID, r.Text, modified})
		}
	})
	return rows, err
}
