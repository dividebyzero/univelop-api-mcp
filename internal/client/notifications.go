package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dividebyzero/univelop-api-mcp/internal/types"
)

// Notification mirrors POST /api/v2/workspaces/{workspaceId}/notifications.
type Notification struct {
	Title   string   `json:"title"`
	Body    string   `json:"body"`
	UserIDs []string `json:"userIds,omitempty"`
	RoleIDs []string `json:"roleIds,omitempty"`
}

// NotificationsSend sends an in-app notification.
func (c *Client) NotificationsSend(ctx context.Context, workspaceID string, n Notification) (*types.Response, error) {
	if n.Title == "" {
		return nil, fmt.Errorf("notification title is required")
	}
	body, err := json.Marshal(n)
	if err != nil {
		return nil, fmt.Errorf("marshaling notification: %w", err)
	}
	path := "/api/v2/workspaces/" + EscapePath(workspaceID) + "/notifications"
	return c.Post(ctx, path, nil, body)
}