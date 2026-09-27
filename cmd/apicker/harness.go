package main

import (
	"fmt"
	"io"

	"sort"

	"github.com/JonGretar/AgentPicker/cmd/apicker/harnesses"
)

type choice struct {
	harness harnesses.Harness

	session *harnesses.Session // nil starts a new chat
}

func choices(home, cwd string, warn io.Writer) []choice {
	var items []choice
	var previous []choice
	for _, h := range harnesses.All() {
		if !h.IsAvailable() {
			continue
		}
		c := choice{harness: h}
		items = append(items, c)
		sessions, err := h.ListSessions(home, cwd)
		if err != nil {
			fmt.Fprintf(warn, "apicker: could not list %s sessions: %v\n", h.Name(), err)
		}
		for _, s := range sessions {
			s := s
			previous = append(previous, choice{harness: h, session: &s})
		}
	}
	sort.SliceStable(previous, func(i, j int) bool { return previous[i].session.Modified.After(previous[j].session.Modified) })
	return append(items, previous...)
}
