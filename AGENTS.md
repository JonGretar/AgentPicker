# AgentPicker

`apicker` is a Go CLI that starts or resumes Claude, Codex, pi, and Crush sessions from the current directory. Its picker uses Charm Bubble Tea v2 and Lip Gloss v2; do not add an fzf dependency.

## Layout

- `cmd/apicker/main.go`: CLI entry point and dispatch to the selected harness.
- `cmd/apicker/harness.go`: installed-agent discovery and choice assembly.
- `cmd/apicker/harnesses/harness.go`: shared `Harness` interface, registry, session type, and history helpers.
- `cmd/apicker/harnesses/{claude,codex,pi,crush}.go`: one integration per file, each self-registering in `init()`.
- `cmd/apicker/picker.go`: searchable terminal picker.
- `cmd/apicker/harnesses/harness_test.go`: session fixtures and integration tests.
- `cmd/apicker/harness_test.go`: picker tests.

## Development

- `Justfile` provides `fmt`, `build`, `test`, `vet`, `check`, and `install` (`~/.local/bin/apicker`) recipes.
- Build: `go build -o apicker ./cmd/apicker`
- Test: `go test ./...`
- Check: `go vet ./...`
- Format changed Go files with `gofmt`.
- To add an agent, create `cmd/apicker/harnesses/<agent>.go` in package `harnesses`, implement `Harness`, and call `Register(<agent>{})` in `init()`. Go compiles all files in a package; it does not discover implementations without registration. Add tests for the new reader.
- Keep session discovery restricted to the current working directory; a malformed or missing history line should not prevent new sessions from being offered. In `--loop` mode, refresh discovery after each agent exits and stop on picker cancellation.
- Harnesses own their `IsAvailable`, `NewSession`, and `ResumeSession` behavior. Use `runInteractive` for terminal-based commands so stdin, stdout, and stderr remain attached after the picker closes.
- Use `jj` for version-control operations in this repository.
