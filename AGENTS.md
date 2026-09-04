# connect-cli

`redpine` — command-line client for [Redpine Connect](https://docs.redpine.ai). Search
licensed collections, preview and unlock results, list collections, check balance, and call
upstream MCP tools from a terminal or from an agent. Agents first: JSON envelope
(`{"status": "ok", "data": ...}` / `{"status": "error", "error": {...}}`) whenever stdout is
not a terminal or `--json` is passed; human-readable in a TTY or with `--pretty`.

## What it does

- `redpine auth login` / `auth set-key` / `auth logout` / `auth status` — OAuth browser flow
  or API key auth (`internal/config`, `internal/command/auth`). `auth status` is `whoami`.
- `redpine search <collection> <query>` — MCP `search` tool, billed per result. Filter DSL:
  repeatable `--filter key=value` (comma-separated = any-of, `!=` to exclude, `>=`/`<=` for
  ranges) or `--filter-json` for full OR/nesting (`internal/command/search`).
- `redpine preview <collection> <query>` — MCP `preview` tool wrapping the same search
  arguments (`search.BuildSearchArgs`): free teasers plus the price, returns a `queryId`.
- `redpine confirm <queryId> [resultId...]` (alias `unlock`) — MCP `confirm` tool. This is
  the call that bills.
- `redpine collections` (also `collections list`) — MCP `list_collections`.
- `redpine balance` — MCP `get_balance`, free.
- `redpine tools list` / `tools info` / `tools call` — MCP `find-tools`, `inspect-tool`, and
  a direct `tools/call`; built for pipe-chaining between agent tool calls (`internal/mcp`,
  `internal/command/tools`).
- `redpine whoami` — token source and type; labels `sk_test_` keys as sandbox.
- `redpine docs [topic]` — opens `docs.redpine.ai/docs/<topic>`; short names such as `auth`,
  `sdk`, `preview` are aliased (`internal/command/docs`).
- `redpine update` — self-update from GitHub releases, verified against `checksums.txt`.
  A newer release is a one-line stderr notice on other commands, never a block.

Commands live under `internal/command/<name>`; the MCP transport/client lives in
`internal/mcp`; auth/session config in `internal/config`; the factory that wires them in
`internal/factory`.

## Wire facts an agent must not break

- Everything is MCP streamable HTTP: `POST {server}/mcp`, `DELETE {server}/mcp`. No REST.
  Protocol version `mcp.ProtocolVersion` (`2025-03-26`), sent in `initialize` and as the
  `MCP-Protocol-Version` header. `Accept: application/json, text/event-stream`,
  `User-Agent: redpine-cli/<version>`, 120 s request timeout (`internal/mcp/transport.go`).
- Session id round-trips in `Mcp-Session-Id`, cached per server URL under `os.TempDir()`.
- Config resolution order, identical for the token and the server URL: flag, then
  `REDPINE_API_KEY` / `REDPINE_BASE_URL`, then the legacy `CONNECT_API_KEY` /
  `CONNECT_SERVER_URL`, then keyring / config file. The SDKs use the same names and order;
  keep them in step (`internal/config/config.go`, `auth.go`).
- `clientInfo.name` in `initialize` is `connect-cli`. The server keys client detection off
  it; do not rename without changing connect-api's `mcp/client_detection.py`.
- `find-tools --format json` output has a balance banner appended after the JSON; the client
  decodes with `json.NewDecoder(...).Decode`, which tolerates trailing text. Do not switch to
  `json.Unmarshal`.

## Install

```bash
brew install redpine-ai/tap/connect-cli
```

Formula lives in `redpine-ai/homebrew-tap`, published automatically by GoReleaser on every
tagged release (see `.goreleaser.yml`; note GoReleaser v2 spells the archive key `formats:`).

## Releases

Push a `v*` tag → `.github/workflows/release.yml` runs GoReleaser → builds
linux/darwin/windows (amd64/arm64, no windows/arm64) → GitHub release with `checksums.txt`
+ Homebrew tap update. Never hand-edit the tap repo. `redpine update` refuses a release
that has no `checksums.txt`.

## CI

`.github/workflows/ci.yml` runs `test`/`lint`/`build`. `test` and `build` run on a 2-OS
matrix (`ubuntu-latest`, `macos-latest`) against Go `1.27.1` (also the `go` directive in
`go.mod`); `lint` (staticcheck, pinned) runs ubuntu-only. `.github/workflows/security.yml`
runs govulncheck, gosec (both pinned) and gitleaks on every PR, on push to `main`, and
weekly. Run `gofmt -l .` before pushing; CI does not, reviewers will.

See `CONTRIBUTING.md` for build/test/lint commands and the PR checklist.

## Known gaps

- `search` and `preview` take one collection. The MCP `search` tool has no `collections`
  (multi-collection) field yet, so the CLI cannot offer it until the server does.
- `include_figures` is not exposed: the CLI renders only `text` content blocks, so image
  blocks would be silently dropped.
- Assisted search and the quota endpoint are REST-only on the server; the CLI has no MCP
  route to them.
