# Redpine CLI

MCP client for [Redpine Connect](https://app.redpine.ai). Search documents, list collections, and call upstream MCP tools from the terminal.

Built for agents — JSON output by default when piped, human-readable in the terminal.

## Install

```bash
brew install redpine-ai/tap/connect-cli
```

## Usage

```bash
# authenticate
redpine auth login                # browser-based OAuth
redpine auth set-key sk_live_...  # or use an API key

# search
redpine search "how does authentication work"
redpine search "rate limiting" --collection api-docs --limit 5

# filter — repeatable; key=value, key!=value to exclude, key>=N for ranges
redpine search corpus "crispr" --filter issn=1664-302X
redpine search corpus "crispr" --filter issn=1664-302X,1932-6203   # any-of
redpine search corpus "crispr" --filter 'issn!=1932-6203'          # exclude
redpine search corpus "crispr" --filter doi=10.1345/aph.1g425      # case-insensitive
redpine search corpus "crispr" --filter 'journal_metric.2yr_mean_citedness>=5'

# full DSL for OR / nesting
redpine search corpus "crispr" \
  --filter-json '{"or":[{"field":"issn","eq":"1664-302X"},{"field":"issn","eq":"1932-6203"}]}'

# collections
redpine collections list

# upstream MCP tools
redpine tools list
redpine tools call analytics--run_query query="SELECT * FROM events" limit=10

# pass JSON input (useful for agents)
echo '{"query": "test"}' | redpine tools call analytics--run_query
redpine tools call analytics--run_query --input '{"query": "test"}'
```

## Output

Terminal (TTY) gets human-readable output. Pipes and scripts get JSON.

```bash
# automatic — JSON when piped
redpine search "query" | jq '.data'

# force JSON in terminal
redpine search "query" --json

# force human-readable in pipes
redpine search "query" --pretty
```

JSON responses follow a consistent envelope:

```json
{"status": "ok", "data": { ... }}
{"status": "error", "error": {"code": "...", "message": "...", "suggestions": [...]}}
```

Exit codes: `0` success, `1` error, `2` auth, `3` bad input, `4` server error.

## Environment variables

| Variable | Description |
|----------|-------------|
| `CONNECT_API_KEY` | API key (skips `redpine auth login`) |
| `CONNECT_SERVER_URL` | Server URL override |
| `NO_COLOR` | Disable colored output |

## License

Apache-2.0
