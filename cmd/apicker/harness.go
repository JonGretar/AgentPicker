package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type session struct {
	ID       string
	Title    string
	Modified time.Time
}

type harness struct {
	name       string
	program    string
	list       func(home, cwd string) ([]session, error)
	resumeArgs func(string) []string
}

var harnesses = []harness{
	{"claude", "claude", claudeSessions, func(id string) []string { return []string{"--resume", id} }},
	{"codex", "codex", codexSessions, func(id string) []string { return []string{"resume", id} }},
	{"pi", "pi", piSessions, func(id string) []string { return []string{"--session", id} }},
	{"crush", "crush", crushSessions, func(id string) []string { return []string{"--session", id} }},
}

type choice struct {
	harness harness
	session *session // nil starts a new chat
}

func choices(home, cwd string, warn io.Writer) []choice {
	var installed []harness
	for _, h := range harnesses {
		if path, err := exec.LookPath(h.program); err == nil {
			h.program = path
			installed = append(installed, h)
		}
	}
	var items []choice
	for _, h := range installed {
		items = append(items, choice{harness: h})
	}
	var previous []choice
	for _, h := range installed {
		sessions, err := h.list(home, cwd)
		if err != nil {
			fmt.Fprintf(warn, "apicker: could not list %s sessions: %v\n", h.name, err)
		}
		for _, s := range sessions {
			s := s
			previous = append(previous, choice{harness: h, session: &s})
		}
	}
	sort.SliceStable(previous, func(i, j int) bool { return previous[i].session.Modified.After(previous[j].session.Modified) })
	return append(items, previous...)
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

func claudeSessions(home, cwd string) ([]session, error) {
	var rows []session
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
			rows = append(rows, session{r.ID, r.Text, modified})
		}
	})
	return rows, err
}

func codexSessions(home, cwd string) ([]session, error) {
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
	var rows []session
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
			rows = append(rows, session{r.ID, r.Text, modified})
		}
	})
	return rows, err
}

func piSessions(home, cwd string) ([]session, error) {
	var rows []session
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
		rows = append(rows, session{path, title, info.ModTime()})
		return nil
	})
	return rows, err
}

func crushSessions(_, cwd string) ([]session, error) {
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
	var rows []session
	for _, r := range data {
		when, err := time.Parse(time.RFC3339Nano, r.Modified)
		if r.UUID != "" && r.Title != "" && err == nil {
			rows = append(rows, session{r.UUID, r.Title, when})
		}
	}
	return rows, nil
}
