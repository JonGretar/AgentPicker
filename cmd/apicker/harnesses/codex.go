package harnesses

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type codex struct{}

func init()                     { Register(codex{}) }
func (codex) Name() string      { return "codex" }
func (codex) IsAvailable() bool { _, err := exec.LookPath("codex"); return err == nil }
func (codex) NewSession() error { return runInteractive(exec.Command("codex")) }
func (codex) ResumeSession(session Session) error {
	return runInteractive(exec.Command("codex", "resume", session.ID))
}

func (codex) ListSessions(home, cwd string) ([]Session, error) {
	inDir := make(map[string]bool)
	err := walkJSONL(filepath.Join(home, ".codex", "sessions"), func(path string) error {
		stem := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		if !strings.HasPrefix(stem, "rollout-") || len(stem) < 36 {
			return nil
		}
		id := stem[len(stem)-36:]
		first := true
		return readLines(path, func(line []byte) {
			if !first {
				return
			}
			first = false
			var r struct {
				Payload struct {
					CWD string `json:"cwd"`
				} `json:"payload"`
				CWD string `json:"cwd"`
			}
			if json.Unmarshal(line, &r) == nil && (sameDir(r.Payload.CWD, cwd) || sameDir(r.CWD, cwd)) {
				inDir[id] = true
			}
		})
	})
	if err != nil {
		return nil, err
	}
	var rows []Session
	byID := make(map[string]int)
	err = readLines(filepath.Join(home, ".codex", "history.jsonl"), func(line []byte) {
		var r struct {
			ID   string `json:"session_id"`
			Text string `json:"text"`
			TS   int64  `json:"ts"`
		}
		if json.Unmarshal(line, &r) != nil || !inDir[r.ID] || r.Text == "" {
			return
		}
		modified := time.Unix(r.TS, 0)
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
