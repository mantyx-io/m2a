# Contributing to m2a

Thanks for your interest in improving m2a. This document describes how to work on the project and what we look for in contributions.

## Prerequisites

- **Go** — Use the version in [`go.mod`](go.mod) (currently **1.24.4**). [go.dev/dl](https://go.dev/dl/) has installers for common platforms.
- A working **terminal** — m2a is an interactive TUI; you’ll run it locally to verify behavior.

## Getting started

```bash
git clone https://github.com/mantyx-io/m2a.git
cd m2a
go mod download
go build -o m2a ./cmd/m2a/
```

Smoke-test the binary:

```bash
./m2a -version
```

Run against a real or example agent when your change affects networking or the UI (see [`examples/gemini-agent/`](examples/gemini-agent/)).

## What to run before opening a PR

These match [CI](.github/workflows/ci.yml):

```bash
go mod verify
go test ./...
go build -o m2a ./cmd/m2a/
```

If you change behavior users rely on, add or extend tests in the same package (for example under [`cmd/m2a/`](cmd/m2a/) or [`internal/tui/`](internal/tui/)).

## Pull requests

1. **One logical change per PR** — Easier to review and merge than large mixed diffs.
2. **Describe the “why”** — A short summary in the PR description helps reviewers and future readers.
3. **Keep scope tight** — Avoid unrelated refactors or formatting-only churn unless that’s the point of the PR.
4. **Let CI pass** — Push updates if `main`’s checks fail on your branch.

## Code style

- Follow **idiomatic Go** and match existing patterns in the file you’re editing (imports, naming, error handling).
- Prefer **small, focused functions** and clear boundaries between `cmd/m2a` (CLI) and `internal/...` (libraries).
- **Comments** — Where behavior is non-obvious, a short comment is better than leaving the next reader guessing.

## Project layout

| Path | Role |
| --- | --- |
| [`cmd/m2a/`](cmd/m2a/) | CLI entrypoint, flags, argument handling |
| [`internal/tui/`](internal/tui/) | Bubble Tea UI, markdown rendering, chat transcript |
| [`internal/httpclient/`](internal/httpclient/) | HTTP client helpers (e.g. headers) |
| [`examples/gemini-agent/`](examples/gemini-agent/) | Sample A2A server for local testing |

## Releases (maintainers)

Tagged releases (`v*`) are built and published via [`.github/workflows/release.yml`](.github/workflows/release.yml). Version strings are embedded with `-ldflags`; see the [Releases](README.md#releases) section in the README.

## License

By contributing, you agree that your contributions will be licensed under the same terms as the project: the [MIT License](LICENSE).
