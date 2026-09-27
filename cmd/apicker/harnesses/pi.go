package harnesses

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
)

type pi struct{}

func init()                  { Register(pi{}) }
func (pi) Name() string      { return "pi" }
func (pi) IsAvailable() bool { _, err := exec.LookPath("pi"); return err == nil }
func (pi) NewSession() error { return runInteractive(exec.Command("pi")) }
func (pi) ResumeSession(session Session) error {
	return runInteractive(exec.Command("pi", "--session", session.ID))
}

func (pi) ListSessions(home, cwd string) ([]Session, error) {
	var rows []Session
	err := walkJSONL(filepath.Join(home, ".pi", "agent", "sessions"), func(path string) error {
		first := true
		matches := false
		title := ""
		err := readLines(path, func(line []byte) {
			if first {
				first = false
				var header struct {
					CWD string `json:"cwd"`
				}
				if json.Unmarshal(line, &header) == nil {
					matches = sameDir(header.CWD, cwd)
				}
			}
			if !matches || title != "" {
				return
			}
			var r struct {
				Message struct {
					Role    string          `json:"role"`
					Content json.RawMessage `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal(line, &r) != nil || r.Message.Role != "user" {
				return
			}
			// pi stores user content as either text or an array of text blocks.
			if json.Unmarshal(r.Message.Content, &title) == nil {
				return
			}
			var blocks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if json.Unmarshal(r.Message.Content, &blocks) == nil {
				for _, b := range blocks {
					if b.Type == "text" && b.Text != "" {
						title = b.Text
						break
					}
				}
			}
		})
		if err != nil {
			return err
		}
		if title == "" {
			return nil
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		rows = append(rows, Session{path, title, info.ModTime()})
		return nil
	})
	return rows, err
}
