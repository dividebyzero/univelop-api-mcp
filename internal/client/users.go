package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dividebyzero/univelop-api-mcp/internal/types"
)

// UserCreateRequest mirrors the POST /api/v2/workspaces/{workspaceId}/users body.
type UserCreateRequest struct {
	FirstName      string   `json:"firstName"`
	LastName       string   `json:"lastName"`
	Email          string   `json:"email"`
	Password       string   `json:"password"`
	Gender         string   `json:"gender,omitempty"` // male | female | other
	ActiveRoleID   string   `json:"activeRoleId"`
	EnabledRoleIDs []string `json:"enabledRoleIds"`
	License        string   `json:"license"`
	UseOAuth       bool     `json:"useOAuth"`
}

// UsersCreate creates a new user in the workspace.
func (c *Client) UsersCreate(ctx context.Context, workspaceID string, req UserCreateRequest) (*types.Response, error) {
	if req.FirstName == "" || req.LastName == "" || req.Email == "" {
		return nil, fmt.Errorf("firstName, lastName and email are required")
	}
	if req.ActiveRoleID == "" {
		return nil, fmt.Errorf("activeRoleId is required")
	}
	if len(req.EnabledRoleIDs) == 0 {
		return nil, fmt.Errorf("enabledRoleIds must contain at least one role")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling user create request: %w", err)
	}
	path := "/api/v2/workspaces/" + EscapePath(workspaceID) + "/users"
	return c.Post(ctx, path, nil, body)
}