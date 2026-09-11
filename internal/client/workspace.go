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