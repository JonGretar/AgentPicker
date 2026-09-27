package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

func run() error {
	flags := flag.NewFlagSet("apicker", flag.ContinueOnError)
	loop := flags.Bool("loop", false, "return to the picker after an agent exits")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("usage: apicker [--loop]")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return pickAndRun(*loop, func() (*choice, error) {
		items := choices(home, cwd, os.Stderr)
		if len(items) == 0 {
			return nil, errors.New("no available agent found")
		}
		model := newPicker(items)
		if _, err := tea.NewProgram(model).Run(); err != nil {
			return nil, err
		}
		return model.picked, nil
	}, launch)
}

func launch(c choice) error {
	if c.session == nil {
		return c.harness.NewSession()
	}
	return c.harness.ResumeSession(*c.session)
}

func pickAndRun(loop bool, pick func() (*choice, error), start func(choice) error) error {
	for {
		c, err := pick()
		if err != nil || c == nil {
			return err
		}
		err = start(*c)
		if !loop {
			return err
		}
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				return err
			}
			fmt.Fprintf(os.Stderr, "apicker: agent exited with status %d\n", exit.ExitCode())
		}
	}
}

func main() {
	if err := run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "apicker:", err)
		os.Exit(1)
	}
}
