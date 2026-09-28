package harnesses

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type omp struct{}

func init()                   { Register(omp{}) }
func (omp) Name() string      { return "omp" }
func (omp) IsAvailable() bool { _, err := exec.LookPath("omp"); return err == nil }
func (omp) NewSession() error { return runInteractive(exec.Command("omp")) }
func (omp) ResumeSession(session Session) error {
	return runInteractive(exec.Command("omp", "--resume", session.ID))
}

func (omp) ListSessions(home, cwd string) ([]Session, error) {
	var rows []Session
	err := walkJSONL(filepath.Join(home, ".omp", "agent", "sessions"), func(path string) error {
		var titleSlot, title, firstPrompt string
		headerSeen, matches := false, false
		err := readLines(path, func(line []byte) {
			if !headerSeen {
				var header struct {
					Type  string `json:"type"`
					CWD   string `json:"cwd"`
					Title string `json:"title"`
				}
				if json.Unmarshal(line, &header) != nil {
					return
				}
				if header.Type == "title" {
					titleSlot = strings.TrimSpace(header.Title)
					return
				}
				headerSeen = true
				matches = header.Type == "session" && header.CWD != "" && sameDir(header.CWD, cwd)
				if matches {
					title = strings.TrimSpace(header.Title)
				}
				return
			}
			if !matches || firstPrompt != "" {
				return
			}
			var entry struct {
				Type    string `json:"type"`
				Message struct {
					Role    string          `json:"role"`
					Content json.RawMessage `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal(line, &entry) != nil || entry.Type != "message" || entry.Message.Role != "user" {
				return
			}
			if json.Unmarshal(entry.Message.Content, &firstPrompt) == nil {
				firstPrompt = strings.TrimSpace(firstPrompt)
				return
			}
			var blocks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if json.Unmarshal(entry.Message.Content, &blocks) == nil {
				for _, block := range blocks {
					if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
						firstPrompt = strings.TrimSpace(block.Text)
						break
					}
				}
			}
		})
		if err != nil {
			return err
		}
		if !matches {
			return nil
		}
		if titleSlot != "" {
			title = titleSlot
		}
		if title == "" {
			title = firstPrompt
		}
		if title == "" {
			return nil
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		rows = append(rows, Session{ID: path, Title: title, Modified: info.ModTime()})
		return nil
	})
	return rows, err
}
