# Agent Picker

A terminal picker to start or resume an agent in the current directory, inspired by [harness-picker](https://github.com/devnull03/harness-picker). Built with Charm's Bubble Tea v2 and Lip Gloss v2; no fzf required.

## Install

Requires Go 1.27 or newer to build and at least one of `claude`, `codex`, `pi`, or `crush` on your PATH.

```sh
go install ./cmd/apicker
```

For a local checkout:

```sh
go build -o apicker ./cmd/apicker
```

Run `apicker` from a project directory. Type to filter by agent or session title, use ↑/↓ (or Ctrl-P/Ctrl-N) to navigate, Enter to launch, and Esc to cancel. Backspace removes a character; Ctrl-U clears the search. Only installed agents appear. New chats are listed first, followed by recent chats from this directory.

Sessions come from `~/.claude/history.jsonl`, `~/.codex/history.jsonl` plus `~/.codex/sessions`, and `~/.pi/agent/sessions`. Crush is queried with `crush session list --json` from the current directory. Session listing failures are reported but do not prevent starting a new chat.
