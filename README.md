# Connect CLI

A fast, agent-optimized CLI for the [Connect](https://redpine.ai) platform. Speaks MCP (Model Context Protocol) to discover and call tools — both built-in search and any upstream MCP servers registered on your Connect instance.

## Install

```bash
# Homebrew
brew install redpine/tap/connect-cli

# Direct download
curl -fsSL https://get.redpine.ai/cli | sh
```

## Quick Start

```bash
# Authenticate
connect auth set-key sk_live_your_api_key

# Search documents
connect search "authentication best practices"

# List collections
connect collections list

# List upstream MCP tools
connect tools list

# Call an upstream tool
connect tools call analytics--run_query query="SELECT * FROM events"
```

## Agent Usage

The CLI outputs JSON by default when piped or used non-interactively:

```bash
# Structured JSON output (automatic in non-TTY)
connect search "query" | jq '.data'

# Explicit JSON with field selection
connect tools list --json name,description

# Exit codes for scripting
connect update --check && echo "up to date" || echo "update available"
```

## Auth

```bash
connect auth login          # Browser-based OAuth
connect auth set-key <key>  # Store API key
connect auth status         # Check auth state
connect auth logout         # Clear credentials
```

Or use environment variables:
```bash
export CONNECT_API_KEY=sk_live_...
export CONNECT_SERVER_URL=https://api.redpine.ai
```

## License

MIT
