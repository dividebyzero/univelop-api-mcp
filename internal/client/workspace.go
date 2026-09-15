package client

import (
	"context"

	"github.com/dividebyzero/univelop-api-mcp/internal/types"
)

// WorkspaceGet fetches workspace metadata.
func (c *Client) WorkspaceGet(ctx context.Context, workspaceID string) (*types.Response, error) {
	path := "/api/v2/workspaces/" + EscapePath(workspaceID)
	return c.Get(ctx, path, nil)
}

// WorkspaceSchema fetches the schema of a workspace with all record specs and bricks.
func (c *Client) WorkspaceSchema(ctx context.Context, workspaceID string) (*types.Response, error) {
	path := "/api/v2/workspaces/" + EscapePath(workspaceID) + "/schema"
	return c.Get(ctx, path, nil)
}