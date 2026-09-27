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
		return errors.New("no available agent found")
	}
	model := newPicker(items)
	if _, err := tea.NewProgram(model).Run(); err != nil {
		return err
	}
	if model.picked == nil {
		return nil
	}
	c := model.picked
	var launchErr error
	if c.session == nil {
		launchErr = c.harness.NewSession()
	} else {
		launchErr = c.harness.ResumeSession(*c.session)
	}
	if err := launchErr; err != nil {
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
