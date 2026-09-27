package harnesses

import (
	"encoding/json"
	"os/exec"
	"time"
)

type crush struct{}

func init()                     { Register(crush{}) }
func (crush) Name() string      { return "crush" }
func (crush) IsAvailable() bool { _, err := exec.LookPath("crush"); return err == nil }
func (crush) NewSession() error { return runInteractive(exec.Command("crush")) }
func (crush) ResumeSession(session Session) error {
	return runInteractive(exec.Command("crush", "--session", session.ID))
}

func (crush) ListSessions(_, cwd string) ([]Session, error) {
	cmd := exec.Command("crush", "session", "list", "--json")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var data []struct {
		UUID     string `json:"uuid"`
		Title    string `json:"title"`
		Modified string `json:"modified"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, err
	}
	var rows []Session
	for _, r := range data {
		when, err := time.Parse(time.RFC3339Nano, r.Modified)
		if r.UUID != "" && r.Title != "" && err == nil {
			rows = append(rows, Session{r.UUID, r.Title, when})
		}
	}
	return rows, nil
}
