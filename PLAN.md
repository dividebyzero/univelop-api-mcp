# Univelop API v2 → MCP Server — Architecture Plan

## Overview

Build a **Go** binary (`univelop-mcp`) that exposes the [Univelop API v2](https://docs.univelop.de) as a Model Context Protocol (MCP) server. AI agents connect to it over the MCP protocol (`tools/list`, `tools/call`) and call Univelop operations — CRUD records, trigger workflows, send notifications, manage files and users — across multiple workspaces.

**Key properties:** single binary, HTTPS-only, multi-workspace, configurable port, SSE transport.

---

## MCP Tool Surface

Each current (non-deprecated) Univelop V2 route becomes one MCP tool. Tools requiring a workspace scope receive a `workspace` (or `workspaceId`) parameter to select which configured workspace handles the call.

| # | MCP Tool Name | HTTP Method | API Path | Purpose |
|---|---|---|---|---|
| 1 | `workspace_get` | GET | `/api/v2/workspaces/{workspaceId}` | Get workspace metadata |
| 2 | `records_list` | GET | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}` | List records (filter, sort, paginate) |
| 3 | `records_create_or_update` | POST | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}` | Create/upsert a record |
| 4 | `records_update_by_keys` | PATCH | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}` | Update by secondary keys |
| 5 | `records_upsert_by_keys` | PUT | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}` | Create/update by secondary keys |
| 6 | `records_get` | GET | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}/{recordId}` | Get a single record |
| 7 | `records_update` | PATCH | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}/{recordId}` | Partial update by ID |
| 8 | `records_delete` | DELETE | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}/{recordId}` | Delete by ID |
| 9 | `records_search` | GET | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}/search` | Full-text search (Beta) |
| 10 | `bricks_next_serial` | POST | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}/{recordId}/bricks/{brickId}/next-serial-number` | Generate next serial number |
| 11 | `bricks_markdown_get` | GET | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}/{recordId}/bricks/{brickId}/markdown` | Get editor brick as markdown |
| 12 | `bricks_markdown_set` | POST | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}/{recordId}/bricks/{brickId}/markdown` | Set editor brick from markdown |
| 13 | `bricks_delta_get` | GET | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}/{recordId}/bricks/{brickId}/delta` | Get editor brick as Quill Delta |
| 14 | `bricks_delta_set` | POST | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}/{recordId}/bricks/{brickId}/delta` | Set editor brick from Quill Delta |
| 15 | `workflows_list` | GET | `/api/v2/workspaces/{workspaceId}/workflows/{flowSpecNameOrId}` | List workflow executions |
| 16 | `workflows_last_execution` | GET | `/api/v2/workspaces/{workspaceId}/workflows/{flowSpecNameOrId}/lastExecution` | Get most recent execution |
| 17 | `workflows_run` | POST | `/api/v2/workspaces/{workspaceId}/workflows/{flowSpecNameOrId}/run` | Trigger async workflow |
| 18 | `workflows_run_and_wait` | POST | `/api/v2/workspaces/{workspaceId}/workflows/{flowSpecNameOrId}/run-and-wait` | Trigger workflow and wait for result |
| 19 | `notifications_send` | POST | `/api/v2/workspaces/{workspaceId}/notifications` | Send notification to users |
| 20 | `users_create` | POST | `/api/v2/workspaces/{workspaceId}/users` | Create a new user |
| 21 | `files_upload` | POST | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}/{recordId}/bricks/{brickId}` | Upload file(s) to brick |
| 22 | `files_download_zip` | POST | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}/{recordId}/bricks/{brickId}/download` | Download files as zip |
| 23 | `files_delete` | DELETE | `/api/v2/workspaces/{workspaceId}/records/{recordSpecName}/{recordId}/bricks/{brickId}/{fileName}` | Delete a file from brick |

**Excluded:** All 10 deprecated legacy routes (`/api/v2/{workspaceId}/...` without `/workspaces/`). The MCP server will only expose the current `/api/v2/workspaces/...` variants.

---

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                    AI Agent                           │
│         (Claude Desktop, Cursor, any MCP host)        │
└──────────────┬───────────────────────────┬───────────┘
               │ tools/list                │ tools/call
               ▼                           ▼
┌──────────────────────────────────────────────────────┐
│                 MCP Server (univelop-mcp)              │
│                                                        │
│  ┌──────────┐  ┌────────────┐  ┌───────────────────┐  │
│  │  MCP      │  │  Tool      │  │  Workspace        │  │
│  │  Protocol │──│  Registry  │──│  Router           │  │
│  │  Handler  │  │  (23 tools)│  │  (selects client  │  │
│  │           │  │            │  │   by workspaceId) │  │
│  └──────────┘  └────────────┘  └────────┬──────────┘  │
│                                          │             │
│                                 ┌────────▼────────┐   │
│                                 │  HTTP Client    │   │
│                                 │  Pool           │   │
│                                 │  (1 per ws)     │   │
│                                 └────────┬────────┘   │
└──────────────────────────────────────────┼────────────┘
                                           │ x-api-key
                                           ▼
┌──────────────────────────────────────────────────────┐
│              Univelop API v2                           │
│  https://app.univelop.de/api/v2/workspaces/{id}/...    │
└──────────────────────────────────────────────────────┘
```

### MCP Transport — SSE (HTTP)

The server listens on a configurable TCP port (default `8443`) over **HTTPS**. It uses **SSE** (Server-Sent Events) transport — the standard way to serve MCP over HTTP. The `mcp-go` library provides the `server.NewMCPServer()` and SSE infrastructure out of the box.

TLS is configured via a certificate file and key file path in the config (or auto-generated self-signed cert for development via `mkcert` pattern).

### Multi-Workspace Routing

1. Config declares a map of `workspaceId → { apiKey, baseURL, label }`.
2. Every MCP tool receives `workspaceId` as a required string argument.
3. The Workspace Router looks up the config entry, picks the right pre-built HTTP client (which carries the `x-api-key` header and base URL), and issues the request.
4. If the workspaceId doesn't match any config entry, the tool returns a clear error.

---

## Project Structure

```
univelop-api-mcp/
├── cmd/
│   └── univelop-mcp/
│       └── main.go              # Entry point: load config, start server
├── internal/
│   ├── config/
│   │   └── config.go            # Config parsing (YAML), validation, defaults
│   ├── client/
│   │   ├── client.go            # HTTP client: request builder, auth header, retries
│   │   ├── records.go           # Records API methods
│   │   ├── workflows.go         # Workflows API methods
│   │   ├── notifications.go     # Notifications API methods
│   │   ├── files.go             # Files API methods (multipart upload)
│   │   ├── users.go             # Users API methods
│   │   └── workspace.go         # Workspace info API methods
│   ├── tools/
│   │   ├── register.go          # tool/list registration, tool/call dispatch
│   │   ├── records.go           # MCP tool definitions for record endpoints
│   │   ├── workflows.go         # MCP tool definitions for workflow endpoints
│   │   ├── notifications.go     # MCP tool definition for notification endpoint
│   │   ├── files.go             # MCP tool definitions for file endpoints
│   │   ├── users.go             # MCP tool definition for user endpoint
│   │   └── workspace.go         # MCP tool definition for workspace endpoint
│   ├── mcp/
│   │   └── server.go            # MCP server setup (SSE, TLS, signal handling)
│   └── types/
│       └── types.go             # Shared types, error helpers, workspace config structs
├── config.example.yaml          # Example configuration file
├── go.mod
├── go.sum
├── Makefile                     # build, test, lint, cross-compile
├── Dockerfile                   # Multi-stage build for container deployment
├── LICENSE
└── README.md
```

---

## Configuration

**Format:** YAML. Chosen for readability, nested structures, and native Go support (`gopkg.in/yaml.v3`).

```yaml
# config.example.yaml

mcp:
  # Default to 8443 (unprivileged, HTTPS)
  port: 8443
  # TLS certificate and key (PEM)
  cert_file: "/etc/univelop-mcp/cert.pem"
  key_file: "/etc/univelop-mcp/key.pem"
  # MCP server identity
  name: "Univelop API MCP"
  version: "0.1.0"

univelop:
  # Base URLs (default app.univelop.de; can override per workspace)
  default_base_url: "https://app.univelop.de/"
  # HTTP client settings
  timeout: 30s
  max_retries: 3

workspaces:
  production:
    workspace_id: "eT8cIxNkx5dL8A5jqHf9gqtBdn13"
    api_key: "${UNIVELOP_API_KEY_PROD}"      # env-var substitution
    base_url: "https://app.univelop.de/"      # optional override
    label: "Production"
  staging:
    workspace_id: "wkspc_staging_abc123"
    api_key: "${UNIVELOP_API_KEY_STAGING}"
    base_url: "https://dev.univelop.de/"
    label: "Staging"
```

Configuration is loaded from a path passed via CLI flag: `univelop-mcp --config /path/to/config.yaml`. Environment variable substitution (e.g. `${VAR_NAME}`) is supported so API keys never land in version control.

---

## Data Flow (One MCP `tools/call` Lifecycle)

```
1. MCP Host sends tools/call { name, arguments: { workspaceId, ...args } }
2. MCP Protocol handler (mark3labs/mcp-go) deserializes the call
3. Tool Registry dispatches to the matching handler function
4. Handler reassembles arguments into an internal request struct
5. Workspace Router resolves workspaceId → { apiKey, baseURL }
6. HTTP Client builds the full URL: {baseURL}/api/v2/workspaces/{workspaceId}/...
7. HTTP Client attaches x-api-key header, serializes body (JSON/multipart)
8. HTTP request is sent with configured timeout & retry policy
9. Response is parsed into a generic map[string]any or typed struct
10. Handler converts response into MCP ToolResult (text content or JSON)
11. Result is returned through MCP protocol to the host
```

### Argument to HTTP mapping

Path parameters, query parameters, and body fields are mapped from flat tool arguments. For example, `records_list` needs:

| Tool arg | Maps to | Type |
|---|---|---|
| `workspaceId` | Path: `{workspaceId}` | string |
| `recordSpecName` | Path: `{recordSpecName}` | string |
| `filters` | Query: `filters` (JSON string → URL-encoded) | string |
| `limit` | Query: `limit` | integer |
| `startAfterDocumentId` | Query: `startAfterDocumentId` | string |
| `resolveLinkedBricks` | Query: `resolveLinkedBricks` | boolean |
| `orderBy` | Query: `orderBy` | string |
| `orderByDescending` | Query: `orderByDescending` | boolean |

For JSON body endpoints (create/update records), the body content is passed as a JSON string argument, deserialized, and sent as `application/json`.

---

## Security

1. **HTTPS only** — the server will refuse plain HTTP connections. TLS cert+key are mandatory in config.
2. **No credential storage in binary** — API keys live in environment variables referenced from YAML config.
3. **Auth per workspace** — each workspace's API key is isolated. A misconfigured key for one workspace doesn't leak others.
4. **Input validation** — workspaceId is validated against the config before any HTTP call. Path parameters are checked for common injection patterns.
5. **Timeouts** — configurable per-workspace HTTP timeout (default 30s). Individual tool calls can specify their own if needed.
6. **Rate limiting** — optional built-in token-bucket limiter to stay under Univelop API rate limits (the spec declares a `429 TooManyRequests` response).

---

## MCP Server Library: `github.com/mark3labs/mcp-go`

**Why:** 9k+ stars, active maintenance, v1.0.0 released, implements MCP spec 2025-11-25, has first-class SSE transport support, simple API for tool registration.

```go
import (
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

// Tool definition
tool := mcp.NewTool("records_list",
    mcp.WithDescription("List records from a record list"),
    mcp.WithString("workspaceId", mcp.Required(), mcp.Description("...")),
    mcp.WithString("recordSpecName", mcp.Required(), mcp.Description("...")),
    mcp.WithString("filters", mcp.Description("JSON-encoded filter array")),
    mcp.WithNumber("limit", mcp.Description("Max 200")),
)

// Handler
handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    ws := request.Params.Arguments["workspaceId"].(string)
    // ... route, call, return
    return mcp.NewToolResultText(jsonResponse), nil
}

// Registration
s := server.NewMCPServer("Univelop API MCP", "0.1.0",
    server.WithResourceCapabilities(true, false),
)
s.AddTool(tool, handler)
server.ServeSSE(s, ":8443")  // with TLS wrapper
```

**Note on TLS:** `mcp-go`'s SSE server uses `net/http` underneath. We wrap it with `http.ListenAndServeTLS()` or configure TLS on the HTTP listener directly.

---

## Build & Release

### Build

```makefile
BINARY=univelop-mcp
VERSION=$(shell git describe --tags --always)

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION)" \
	  -o bin/$(BINARY) ./cmd/univelop-mcp/

.PHONY: cross
cross:
	GOOS=linux   GOARCH=amd64 make build
	GOOS=linux   GOARCH=arm64 make build
	GOOS=darwin  GOARCH=amd64 make build
	GOOS=darwin  GOARCH=arm64 make build
	GOOS=windows GOARCH=amd64 make build
```

### Docker

Multi-stage Dockerfile: `golang:1.23-alpine` build → `scratch` or `alpine:3.20` runtime (ca-certificates + tzdata).

### Release

Git tags (`v0.1.0`, `v0.2.0`, …) trigger a GitHub Actions pipeline that:
1. Runs `go test ./...`
2. `golangci-lint run`
3. Cross-compile for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
4. Build Docker image and push to GHCR
5. Upload binaries as release artifacts

---

## Implementation Order

| Phase | What | Est. Effort |
|---|---|---|
| **1** | Scaffold project: go.mod, dirs, config parsing, Makefile | 1 day |
| **2** | HTTP client layer: client.go with auth, retries, timeout | 1 day |
| **3** | MCP server setup: SSE, TLS, tool registration infra | 1 day |
| **4** | Records toolset: list, get, create, update, delete, search, upsert | 2 days |
| **5** | Workflow tools: list, lastExecution, run, run-and-wait | 1 day |
| **6** | Notification tool | 0.5 day |
| **7** | File upload tool (multipart), file download, file delete | 1 day |
| **8** | User creation tool | 0.5 day |
| **9** | Workspace info tool | 0.5 day |
| **10** | Config validation, error handling polish, input sanitization | 1 day |
| **11** | Dockerfile, CI pipeline, README, documentation | 1 day |

**Total: ~10 days**

---

## Open Questions

1. **TLS certificate management for the MCP server** — should the binary generate a self-signed cert on first run, or require explicit cert_path + key_path in config? **Decision: require explicit paths.** Self-signed certs are a separate utility.

2. **MCP transport** — SSE (HTTP) or stdio? AI agents like Claude Desktop support both natively. SSE is more flexible for remote deployments; stdio is simpler for local. **Decision: SSE as primary, with a `--stdio` flag** for local-only use (SSE needs TLS, stdio doesn't).

3. **File upload** — the API supports both `multipart/form-data` and `application/json` (base64). For MCP, JSON + base64 is much cleaner (no temporary file management). **Decision: use application/json with base64** file content in all file upload tools.

4. **Rate limiting** — how aggressive should the built-in limiter be? The spec doesn't document exact rate limits. **Decision: configurable, disabled by default**, with a note to enable once rates are known.

5. **MCP resource support** — beyond tools/list and tools/call, MCP 2025 supports resources (readable data). Should we expose workspaces/records as resources? **Decision: defer to v0.2.** Start with tools only.

6. **`records_search` pagination** — returns `Total-Hits`, `Total-Pages`, `Current-Page`, `Limit` headers. How to surface these in MCP? Return them as metadata in the JSON result alongside the records array.

7. **Deprecated routes** — the spec has 10 deprecated legacy routes at `/api/v2/{workspaceId}/...` (without `/workspaces/`). Skip them entirely.

8. **Should `workflow_run_and_wait` block the MCP connection?** The Univelop API waits server-side, so this is safe. But the tool should have a configurable timeout (default 60s) and return the execution result or timeout error.