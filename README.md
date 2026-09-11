# Univelop API v2 → MCP Server

A **Go** binary (`univelop-mcp`) that exposes the [Univelop API v2](https://docs.univelop.de) as a **Model Context Protocol** (MCP) server. AI agents can connect over SSE/HTTPS (or stdio for local use) to call Univelop operations — CRUD records, trigger workflows, send notifications, manage files and users — across multiple workspaces.

## Features

- **23 MCP tools** covering the full Univelop API v2 surface
- **Multi-workspace** — configure any number of workspaces, each with its own API key
- **SSE over HTTPS** (default) or **stdio** mode for local MCP hosts (Claude Desktop, etc.)
- **Environment-var substitution** in YAML config for API keys
- **Rate limiting**, **retry with backoff**, **configurable timeouts**
- **Auto-generated dev certificates** (`--dev-tls`) for local testing
- **TLS required** in SSE mode — production-grade security

## Quick Start

### Prerequisites
- Go 1.23+
- A Univelop workspace and API key

### Build

```bash
make build
```

Or manually:

```bash
go build -ldflags="-s -w -X main.version=$(git describe --tags --always)" \
  -o bin/univelop-mcp ./cmd/univelop-mcp/
```

### Configure

Copy `config.example.yaml` to `config.yaml` and fill in:

```yaml
workspaces:
  production:
    workspace_id: "your-workspace-id"
    api_key: "${UNIVELOP_API_KEY_PROD}"  # or paste the key directly
```

### Run

**Development mode** (auto-generated self-signed TLS certificate):

```bash
./bin/univelop-mcp --dev-tls --config config.yaml
```

**Local stdio mode** (no TLS needed; for Claude Desktop etc.):

```bash
./bin/univelop-mcp --stdio --config config.yaml
```

**Production** (certificate files in config):

```bash
./bin/univelop-mcp --config /etc/univelop-mcp/config.yaml
```

### Validate Configuration

```bash
./bin/univelop-mcp validate --config config.yaml
```

## MCP Tool Surface

| # | Tool | Description |
|---|------|-------------|
| 1 | `workspace_get` | Get workspace metadata |
| 2 | `records_list` | List records with filter/sort/pagination |
| 3 | `records_create_or_update` | Create or update a record |
| 4 | `records_update_by_keys` | Update by secondary keys |
| 5 | `records_upsert_by_keys` | Upsert by secondary keys |
| 6 | `records_get` | Get a single record |
| 7 | `records_update` | Partial update by ID |
| 8 | `records_delete` | Delete by ID |
| 9 | `records_search` | Full-text search (Beta) |
| 10 | `bricks_next_serial` | Generate next serial number |
| 11 | `bricks_markdown_get` | Get editor brick as markdown |
| 12 | `bricks_markdown_set` | Set editor brick from markdown |
| 13 | `bricks_delta_get` | Get editor brick as Quill Delta |
| 14 | `bricks_delta_set` | Set editor brick from Quill Delta |
| 15 | `workflows_list` | List workflow executions |
| 16 | `workflows_last_execution` | Get most recent execution |
| 17 | `workflows_run` | Trigger async workflow |
| 18 | `workflows_run_and_wait` | Trigger workflow and wait for result |
| 19 | `notifications_send` | Send in-app notification |
| 20 | `users_create` | Create a new user |
| 21 | `files_upload` | Upload file(s) to brick (base64) |
| 22 | `files_download_zip` | Download files as zip |
| 23 | `files_delete` | Delete a file from a brick |

Every tool takes a `workspaceId` argument (config alias or raw workspace ID).

## Deployment

### Docker

```bash
docker build -t univelop-mcp .
docker run -p 8443:8443 \
  -v $(pwd)/config.yaml:/etc/univelop-mcp/config.yaml:ro \
  -e UNIVELOP_API_KEY_PROD=your-key \
  univelop-mcp
```

### Docker Compose

```yaml
services:
  mcp:
    build: .
    ports:
      - "8443:8443"
    volumes:
      - ./config.yaml:/etc/univelop-mcp/config.yaml:ro
    environment:
      - UNIVELOP_API_KEY_PROD=${UNIVELOP_API_KEY_PROD}
```

## Config Reference

| Key | Default | Description |
|-----|---------|-------------|
| `mcp.port` | `8443` | TCP port |
| `mcp.cert_file` | — | TLS certificate path (PEM; required in SSE mode) |
| `mcp.key_file` | — | TLS key path (PEM; required in SSE mode) |
| `mcp.name` | `univelop-mcp` | MCP server name |
| `mcp.version` | — | MCP server version |
| `univelop.default_base_url` | `https://app.univelop.de/` | Base API URL |
| `univelop.timeout` | `30s` | HTTP request timeout |
| `univelop.max_retries` | `3` | Max retries for idempotent requests |
| `univelop.rate_limit_per_second` | `0` (unlimited) | Token-bucket rate limit |
| `workspaces.<key>.workspace_id` | (required) | Univelop workspace ID |
| `workspaces.<key>.api_key` | (required) | API key (or `${ENV_VAR}`) |
| `workspaces.<key>.base_url` | default | Per-workspace base URL override |
| `workspaces.<key>.label` | — | Optional label |

## Development

```bash
make test       # run tests
make vet        # run go vet
make lint       # run golangci-lint (if installed)
make cross      # cross-compile for all platforms
make clean      # remove build artifacts
```

## License

MIT