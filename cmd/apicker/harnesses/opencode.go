package harnesses

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"time"
)

type opencode struct{}

func init()                        { Register(opencode{}) }
func (opencode) Name() string      { return "opencode" }
func (opencode) IsAvailable() bool { _, err := exec.LookPath("opencode"); return err == nil }
func (opencode) NewSession() error { return runInteractive(exec.Command("opencode")) }
func (opencode) ResumeSession(session Session) error {
	return runInteractive(exec.Command("opencode", "--session", session.ID))
}

func (opencode) ListSessions(_, cwd string) ([]Session, error) {
	cmd := exec.Command("opencode", "session", "list", "--format", "json")
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseOpenCodeSessions(output, cwd)
}

func parseOpenCodeSessions(output []byte, cwd string) ([]Session, error) {
	// OpenCode prints nothing when there are no sessions.
	if len(bytes.TrimSpace(output)) == 0 {
		return nil, nil
	}
	var data []struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Directory string `json:"directory"`
		Updated   int64  `json:"updated"`
	}
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, err
	}
	var rows []Session
	for _, s := range data {
		if s.ID == "" || s.Title == "" || s.Directory == "" || !sameDir(s.Directory, cwd) {
			continue
		}
		rows = append(rows, Session{ID: s.ID, Title: s.Title, Modified: time.UnixMilli(s.Updated)})
	}
	return rows, nil
}
