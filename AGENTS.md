# AGENTS.md

## Cursor Cloud specific instructions

This is a Go CLI project (algorithmic crossword puzzle generator). No external services, databases, or Docker are required.

- **Go version**: 1.24.4 (specified in `go.mod`)
- **Tests**: `go test -v ./...`
- **Benchmarks**: `go test -bench . -benchmem ./...`
- **Lint**: `go vet ./...`
- **Build**: `go build ./...`
- **Run CLI**: `go run ./cmd/xwcli/ --file=testdata/words.txt --width=5 --first` (use `--first` to avoid interactive prompt)
- The `--first` flag exits after the first generated grid. Without it, the CLI prompts `Continue? [Y/n]` interactively.
- See `README.md` for usage and `go run ./cmd/xwcli/ -help` for all CLI flags.
