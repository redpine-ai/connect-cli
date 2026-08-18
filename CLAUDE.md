# connect-cli

`redpine` — MCP client CLI for [Redpine Connect](https://app.redpine.ai). Search documents, list
collections, and call upstream MCP tools from the terminal or from an agent. Built for agents
first: human-readable output in a TTY, JSON envelope (`{"status": "ok", "data": ...}` /
`{"status": "error", "error": {...}}`) when piped or with `--json`.

## What it does

- `redpine auth login` / `redpine auth set-key` — OAuth browser flow or API key auth
  (`internal/config`, `internal/command/auth`)
- `redpine search` — full-text/semantic search against the public Connect API, with a
  filter DSL: repeatable `--filter key=value` (comma-separated = any-of, `!=` to exclude,
  `>=`/`<=` for ranges) or `--filter-json` for full OR/nesting (`internal/command/search`)
- `redpine collections list` — list searchable collections
- `redpine tools list` / `redpine tools call` — discover and invoke upstream MCP tools,
  designed for pipe-chaining between agent tool calls (`internal/mcp`)
- `redpine whoami`, `redpine update` — session/version introspection

Commands live under `internal/command/<name>`; the MCP transport/client lives in
`internal/mcp`; auth/session config in `internal/config`.

## Install

```bash
brew install redpine-ai/tap/connect-cli
```

Formula lives in `redpine-ai/homebrew-tap`, published automatically by GoReleaser on
every tagged release (see `.goreleaser.yml`).

## Releases

Push a `v*` tag → `.github/workflows/release.yml` runs GoReleaser → builds
linux/darwin/windows (amd64/arm64, no windows/arm64) → GitHub release + Homebrew tap update.
Never hand-edit the tap repo.

## CI

`.github/workflows/ci.yml` runs `test`/`lint`/`build` jobs. `test` and `build` run on a
2-OS matrix (`ubuntu-latest`, `macos-latest`) against Go `1.26`; `lint` (staticcheck) runs
ubuntu-only. `.github/workflows/security.yml` runs govulncheck, gosec, and gitleaks on every
PR, on push to `main`, and weekly.

See `CONTRIBUTING.md` for build/test/lint commands and the PR checklist.

## Status (2026-08-18)

- Module: `github.com/redpine-ai/connect-cli`, binary name `redpine` — confirmed `go.mod:1-3`,
  `.goreleaser.yml` (`binary: redpine`).
- Go toolchain pinned at `go 1.26.6` in `go.mod` (bumped from `1.26.4` this session to clear a
  govulncheck failure — GO-2026-6218/6090/6089/5972/5026, all fixed in `go1.26.6`).
- `--filter`/`--filter-json` on `redpine search` shipped recently (commit `7444758`,
  "feat(search): add --filter and --filter-json to the search command").
- No `CONTRIBUTING.md` existed before RED-1336 (this session added it alongside this file).
- License file (`LICENSE`) is Apache 2.0, but `.goreleaser.yml`'s Homebrew formula declares
  `license: "MIT"` — pre-existing mismatch, not touched by this session; flag before relying
  on either for compliance purposes.
