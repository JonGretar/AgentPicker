package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

func run() error {
	if len(os.Args) > 1 {
		return fmt.Errorf("usage: apicker (run in a project directory)")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	items := choices(home, cwd, os.Stderr)
	if len(items) == 0 {
		return errors.New("no supported agent found on PATH (claude, codex, pi, crush)")
	}
	model := newPicker(items)
	if _, err := tea.NewProgram(model).Run(); err != nil {
		return err
	}
	if model.picked == nil {
		return nil
	}
	c := model.picked
	var args []string
	if c.session != nil {
		args = c.harness.resumeArgs(c.session.ID)
	}
	cmd := exec.Command(c.harness.program, args...)
	cmd.Dir = cwd
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		return err
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "apicker:", err)
		os.Exit(1)
	}
}
