# Redpine CLI

`redpine` is the command-line client for [Redpine Connect](https://docs.redpine.ai): search licensed research collections, preview results before paying for them, and call upstream MCP tools from a terminal or an agent.

Built for agents: JSON on a pipe, human-readable in a terminal, one envelope and one set of exit codes everywhere.

## Install

```bash
brew install redpine-ai/tap/connect-cli
```

Binaries for Linux, macOS and Windows are on the [releases page](https://github.com/redpine-ai/connect-cli/releases). `redpine update` upgrades a release-page install in place and verifies the download against the release checksums.

## Authenticate

```bash
redpine auth login                 # browser-based OAuth
redpine auth set-key sk_live_...   # or store an API key
export REDPINE_API_KEY=sk_test_... # or use the environment (sandbox keys work too)
redpine whoami
```

A sandbox key (`sk_test_`) returns synthetic fixtures at no cost; `whoami` says so.

## Search

```bash
# search a collection (billed per result)
redpine search corpus "how does authentication work"
redpine search corpus "rate limiting" --limit 5

# preview for free, then pay only for what you want
redpine preview corpus "crispr delivery"
redpine confirm qry_a1b2c3d4e5f6                 # unlock every previewed result
redpine confirm qry_a1b2c3d4e5f6 abc123 def456   # unlock only these

# filter — repeatable; key=value, key!=value to exclude, key>=N for ranges
redpine search corpus "crispr" --filter issn=1664-302X
redpine search corpus "crispr" --filter issn=1664-302X,1932-6203   # any-of
redpine search corpus "crispr" --filter 'issn!=1932-6203'          # exclude
redpine search corpus "crispr" --filter doi=10.1345/aph.1g425      # case-insensitive
redpine search corpus "crispr" --filter 'journal_metric.2yr_mean_citedness>=5'

# full DSL for OR / nesting
redpine search corpus "crispr" \
  --filter-json '{"or":[{"field":"issn","eq":"1664-302X"},{"field":"issn","eq":"1932-6203"}]}'

# what can I search, and what do I have left?
redpine collections
redpine balance
```

`preview` takes the same `--limit`, `--filter` and `--filter-json` as `search`. Filters are documented at [docs.redpine.ai/docs/filtering](https://docs.redpine.ai/docs/filtering).

## MCP tools

```bash
redpine tools list
redpine tools info analytics--run_query
redpine tools call analytics--run_query query="SELECT * FROM events" limit=10

# pass JSON input (useful for agents)
echo '{"query": "test"}' | redpine tools call analytics--run_query
redpine tools call analytics--run_query --input '{"query": "test"}'

# pipe chaining auto-wires ids between calls
redpine tools call media--create_workspace --input '{"filter":{"query":"X"}}' \
  | redpine tools call media--daily_briefing
```

## Output

A terminal gets human-readable output. A pipe or a script gets the JSON envelope.

```bash
redpine search corpus "query" | jq '.data'   # JSON, automatically
redpine search corpus "query" --json         # force JSON in a terminal
redpine collections --pretty | less          # force human-readable in a pipe
```

```json
{"status": "ok", "data": { ... }}
{"status": "error", "error": {"code": "...", "message": "...", "hint": "...", "suggestions": [...]}}
```

Exit codes: `0` success, `1` error, `2` auth, `3` bad input, `4` server error.

## Other commands

```bash
redpine docs               # open docs.redpine.ai
redpine docs filtering     # open a page; short names like auth, sdk, preview work too
redpine completion zsh     # shell completion (Homebrew installs it for you)
redpine update             # upgrade a release-page install; Homebrew users: brew upgrade connect-cli
```

## Environment variables

| Variable | Description |
|----------|-------------|
| `REDPINE_API_KEY` | API key; skips `redpine auth login`. `CONNECT_API_KEY` is still read as a fallback |
| `REDPINE_BASE_URL` | Server URL override. `CONNECT_SERVER_URL` is still read as a fallback |
| `NO_COLOR` | Disable coloured output |
| `CLICOLOR_FORCE` | Force coloured output |

The same two names configure the [SDKs](https://docs.redpine.ai/docs/sdks), so one shell setup serves both.

## License

Apache-2.0
