package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerWorkspace registers the workspace_get tool.
func (r *Registry) registerWorkspace(s *server.MCPServer) {
	tool := newWorkspaceTool(r, "workspace_get",
		"Get metadata about a workspace (id, labels, timestamps).")

	register(s, tool, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, _, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		wsID := strArg(args, "workspaceId")

		resp, err := c.WorkspaceGet(ctx, wsID)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})
}