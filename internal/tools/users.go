package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/dividebyzero/univelop-api-mcp/internal/client"
)

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// registerUsers registers the users_create tool.
func (r *Registry) registerUsers(s *server.MCPServer) {
	tool := newWorkspaceTool(r, "users_create",
		"Create a new user in a workspace.",
		mcp.WithString("firstName", mcp.Required(), mcp.Description("User's first name.")),
		mcp.WithString("lastName", mcp.Required(), mcp.Description("User's last name.")),
		mcp.WithString("email", mcp.Required(), mcp.Description("User's email address.")),
		mcp.WithString("password", mcp.Description("Initial password (required unless useOAuth is true).")),
		mcp.WithString("gender", mcp.Description("Gender: male, female or other.")),
		mcp.WithString("activeRoleId", mcp.Required(), mcp.Description("Id of the user's active role.")),
		mcp.WithString("enabledRoleIds", mcp.Required(), mcp.Description("JSON array of enabled role ids, e.g. [\"role123\"].")),
		mcp.WithString("license", mcp.Description("License type, e.g. \"basic\".")),
		mcp.WithBoolean("useOAuth", mcp.DefaultBool(false), mcp.Description("Whether the user authenticates via OAuth.")),
	)

	register(s, tool, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}

		enabledRoles, badErr := parseStringArrayArg(args, "enabledRoleIds")
		if badErr != nil {
			return badErr, nil
		}

		req := client.UserCreateRequest{
			FirstName:      strArg(args, "firstName"),
			LastName:       strArg(args, "lastName"),
			Email:          strArg(args, "email"),
			Password:       strArg(args, "password"),
			Gender:         strArg(args, "gender"),
			ActiveRoleID:   strArg(args, "activeRoleId"),
			EnabledRoleIDs: enabledRoles,
			License:        strArg(args, "license"),
			UseOAuth:       boolArg(args, "useOAuth"),
		}
		resp, err := c.UsersCreate(ctx, wsID, req)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})
}