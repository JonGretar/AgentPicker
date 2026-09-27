# AgentPicker

`apicker` is a Go CLI that starts or resumes Claude, Codex, pi, and Crush sessions from the current directory. Its picker uses Charm Bubble Tea v2 and Lip Gloss v2; do not add an fzf dependency.

## Layout

- `cmd/apicker/main.go`: CLI entry point and agent launch.
- `cmd/apicker/harness.go`: installed-agent discovery and session readers.
- `cmd/apicker/picker.go`: searchable terminal picker.
- `cmd/apicker/harness_test.go`: session fixtures and picker tests.

## Development

- Build: `go build -o apicker ./cmd/apicker`
- Test: `go test ./...`
- Check: `go vet ./...`
- Format changed Go files with `gofmt`.
- Keep session discovery restricted to the current working directory; a malformed or missing history line should not prevent new sessions from being offered.
- Keep agent commands interactive: stdin, stdout, and stderr must remain attached to the terminal after the picker closes.
- Use `jj` for version-control operations in this repository.
