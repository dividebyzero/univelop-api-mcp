package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/dividebyzero/univelop-api-mcp/internal/client"
)

// registerNotifications registers the notifications_send tool.
func (r *Registry) registerNotifications(s *server.MCPServer) {
	tool := newWorkspaceTool(r, "notifications_send",
		"Send an in-app notification to users or roles in a workspace.",
		mcp.WithString("title", mcp.Required(), mcp.Description("Notification title.")),
		mcp.WithString("body", mcp.Description("Notification body text.")),
		mcp.WithString("userIds", mcp.Description("JSON array of user ids to notify, e.g. [\"user123\"].")),
		mcp.WithString("roleIds", mcp.Description("JSON array of role ids to notify, e.g. [\"role123\"].")),
	)

	register(s, tool, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}

		userIDs, badErr := parseStringArrayArg(args, "userIds")
		if badErr != nil {
			return badErr, nil
		}
		roleIDs, badErr := parseStringArrayArg(args, "roleIds")
		if badErr != nil {
			return badErr, nil
		}

		n := client.Notification{
			Title:   strArg(args, "title"),
			Body:    strArg(args, "body"),
			UserIDs: userIDs,
			RoleIDs: roleIDs,
		}
		resp, err := c.NotificationsSend(ctx, wsID, n)
		if err != nil {
			return errResult(err.Error()), nil
		}
		if len(resp.Body) == 0 {
			return mcp.NewToolResultText("notification sent"), nil
		}
		return resultText(resp.Body), nil
	})
}

// parseStringArrayArg parses a JSON array-of-strings argument.
func parseStringArrayArg(args map[string]interface{}, name string) ([]string, *mcp.CallToolResult) {
	raw := strArg(args, name)
	if raw == "" {
		return nil, nil
	}
	var out []string
	if err := jsonUnmarshal([]byte(raw), &out); err != nil {
		return nil, errResultf("argument %q must be a JSON array of strings: %v", name, err)
	}
	return out, nil
}