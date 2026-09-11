// Package tools defines the full MCP tool surface for the Univelop API v2.
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/dividebyzero/univelop-api-mcp/internal/client"
	"github.com/dividebyzero/univelop-api-mcp/internal/config"
)

// Registry holds one pre-built HTTP client per configured workspace and
// registers the MCP tool set on an MCPServer.
type Registry struct {
	cfg       *config.Config
	clients   map[string]*client.Client // config alias -> client
	byID      map[string]*client.Client // workspace_id -> client
	toolCount int
}

// NewRegistry builds the per-workspace HTTP client pool from a validated config.
func NewRegistry(cfg *config.Config) (*Registry, error) {
	r := &Registry{
		cfg:     cfg,
		clients: make(map[string]*client.Client),
		byID:    make(map[string]*client.Client),
	}
	for alias, w := range cfg.Workspaces {
		baseURL := w.BaseURL
		if baseURL == "" {
			baseURL = config.DefaultBaseURL
		}
		c, err := client.New(baseURL, w.APIKey, client.Options{
			Timeout:    cfg.Timeout(),
			MaxRetries: cfg.Univelop.MaxRetries,
			RateLimit:  cfg.Univelop.RateLimitPerSecond,
			Label:      alias,
		})
		if err != nil {
			return nil, fmt.Errorf("workspace %q: %w", alias, err)
		}
		r.clients[alias] = c
		if w.ID != "" {
			r.byID[w.ID] = c
		}
	}
	if len(r.clients) == 0 {
		return nil, fmt.Errorf("at least one workspace must be configured")
	}
	return r, nil
}

// WorkspaceCount returns the number of configured workspaces.
func (r *Registry) WorkspaceCount() int {
	return len(r.clients)
}

// ToolCount returns the number of registered MCP tools.
func (r *Registry) ToolCount() int {
	return r.toolCount
}

// Resolve maps a tool's workspaceId argument (config alias or raw workspace id)
// to its pre-built HTTP client.
func (r *Registry) Resolve(workspaceID string) (*client.Client, *config.Workspace, error) {
	if c, ok := r.clients[workspaceID]; ok {
		w := r.cfg.Workspaces[workspaceID]
		return c, &w, nil
	}
	if c, ok := r.byID[workspaceID]; ok {
		for _, w := range r.cfg.Workspaces {
			if w.ID == workspaceID {
				ww := w
				return c, &ww, nil
			}
		}
	}
	return nil, nil, fmt.Errorf(
		"workspaceId %q is not configured; available workspaces: %s",
		workspaceID, r.workspaceNames(),
	)
}

func (r *Registry) workspaceNames() string {
	names := make([]string, 0, len(r.cfg.Workspaces))
	for alias := range r.cfg.Workspaces {
		names = append(names, alias)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// Register adds all 23 MCP tools to the server.
func (r *Registry) Register(s *server.MCPServer) {
	r.registerWorkspace(s)
	r.registerRecords(s)
	r.registerBricks(s)
	r.registerWorkflows(s)
	r.registerNotifications(s)
	r.registerUsers(s)
	r.registerFiles(s)
	r.toolCount = len(s.ListTools())
}

// ---- argument helpers ----

func (r *Registry) clientFor(args map[string]interface{}) (*client.Client, string, *mcp.CallToolResult) {
	wsID, ok := args["workspaceId"].(string)
	if !ok || strings.TrimSpace(wsID) == "" {
		return nil, "", errResultf("missing required argument: workspaceId")
	}
	c, w, err := r.Resolve(wsID)
	if err != nil {
		return nil, "", errResult(err.Error())
	}
	// Use the workspace ID field for the API path; fallback to the alias.
	pathID := w.ID
	if pathID == "" {
		pathID = wsID
	}
	return c, pathID, nil
}

func strArg(args map[string]interface{}, name string) string {
	v, _ := args[name].(string)
	return v
}

func intArg(args map[string]interface{}, name string) int {
	switch v := args[name].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0
		}
		return int(n)
	}
	return 0
}

func boolArg(args map[string]interface{}, name string) bool {
	v, _ := args[name].(bool)
	return v
}

// bodyArg validates and returns a JSON body argument as raw bytes.
func bodyArg(args map[string]interface{}, name string, required bool) ([]byte, error) {
	s := strings.TrimSpace(strArg(args, name))
	if s == "" {
		if required {
			return nil, fmt.Errorf("missing required JSON body argument: %s", name)
		}
		return nil, nil
	}
	if !json.Valid([]byte(s)) {
		return nil, fmt.Errorf("argument %q must be a valid JSON value", name)
	}
	return []byte(s), nil
}

// queryJSONArg validates a JSON-string query parameter (e.g. filters).
func queryJSONArg(args map[string]interface{}, name string) (string, error) {
	s := strArg(args, name)
	if s == "" {
		return "", nil
	}
	if !json.Valid([]byte(s)) {
		return "", fmt.Errorf("argument %q must be a valid JSON value", name)
	}
	return s, nil
}

// ---- result helpers ----

// resultText pretty-prints a JSON API response as MCP text content.
func resultText(data []byte) *mcp.CallToolResult {
	var buf bytes.Buffer
	if json.Indent(&buf, data, "", "  ") == nil {
		return mcp.NewToolResultText(buf.String())
	}
	return mcp.NewToolResultText(string(data))
}

func errResult(text string) *mcp.CallToolResult {
	return mcp.NewToolResultError(text)
}

func errResultf(format string, a ...interface{}) *mcp.CallToolResult {
	return mcp.NewToolResultError(fmt.Sprintf(format, a...))
}

// toolHandler adapts a typed handler to mcp-go's ToolHandlerFunc.
type toolHandler func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error)

func (h toolHandler) handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, _ := req.Params.Arguments.(map[string]interface{})
	if args == nil {
		args = map[string]interface{}{}
	}
	return h(ctx, args)
}

// newWorkspaceTool builds an MCP tool that always requires a workspaceId.
func newWorkspaceTool(r *Registry, name, description string, extra ...mcp.ToolOption) mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(description),
		mcp.WithString("workspaceId",
			mcp.Required(),
			mcp.Description("Workspace to operate on: a config alias (e.g. \"production\") or the raw Univelop workspace id."),
		),
	}
	opts = append(opts, extra...)
	return mcp.NewTool(name, opts...)
}

// register adds a tool and its handler to the server.
func register(s *server.MCPServer, tool mcp.Tool, h toolHandler) {
	s.AddTool(tool, h.handle)
}