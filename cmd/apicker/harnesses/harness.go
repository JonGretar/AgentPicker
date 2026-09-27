// Package harnesses defines agent integrations. Each implementation registers
// itself in its own file, so the picker never needs a central harness list.
package harnesses

import (
	"bufio"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Session struct {
	ID       string
	Title    string
	Modified time.Time
}

// Harness is the contract for an installed agent and its local chat history.
type Harness interface {
	Name() string
	IsAvailable() bool
	ListSessions(home, cwd string) ([]Session, error)
	NewSession() error
	ResumeSession(session Session) error
}

var registered []Harness

// Register is called from an integration's init function. Files in this
// package are compiled together; no list of implementations needs updating.
func Register(h Harness) {
	for _, existing := range registered {
		if existing.Name() == h.Name() {
			panic("duplicate harness: " + h.Name())
		}
	}
	registered = append(registered, h)
}

func All() []Harness {
	result := append([]Harness(nil), registered...)
	sort.Slice(result, func(i, j int) bool { return result[i].Name() < result[j].Name() })
	return result
}

// runInteractive preserves the terminal streams after Bubble Tea has exited.
// Integrations own their commands; this only supplies the common I/O wiring.
func runInteractive(cmd *exec.Cmd) error {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func sameDir(a, b string) bool {
	clean := func(p string) string {
		p = strings.TrimPrefix(p, `\\?\`)
		p = filepath.Clean(p)
		if runtime.GOOS == "windows" {
			p = strings.ToLower(p)
		}
		return p
	}
	return clean(a) == clean(b)
}

// Missing history files are normal for agents that haven't run yet.
func readLines(path string, visit func([]byte)) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		visit(scanner.Bytes())
	}
	return scanner.Err()
}

func walkJSONL(root string, visit func(string) error) error {
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".jsonl") {
			return visit(path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
