package harnesses

import (
	"os"
	"os/exec"
	"path/filepath"
)

const aiderHistory = ".aider.chat.history.md"

type aider struct{}

func init()                     { Register(aider{}) }
func (aider) Name() string      { return "aider" }
func (aider) IsAvailable() bool { _, err := exec.LookPath("aider"); return err == nil }
func (aider) NewSession() error {
	return runInteractive(exec.Command("aider", "--no-auto-commits", "--no-restore-chat-history", "--chat-history-file", aiderHistory))
}
func (aider) ResumeSession(session Session) error {
	return runInteractive(exec.Command("aider", "--no-auto-commits", "--restore-chat-history", "--chat-history-file", session.ID))
}

func (aider) ListSessions(_, cwd string) ([]Session, error) {
	path := filepath.Join(cwd, aiderHistory)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return nil, nil
	}
	return []Session{{ID: path, Title: "Continue chat", Modified: info.ModTime()}}, nil
}
