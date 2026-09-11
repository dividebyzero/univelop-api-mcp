package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// brickArgs returns the record/brick path arguments shared by all brick tools.
func (r *Registry) brickArgs() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithString("recordSpecName", mcp.Required(), mcp.Description("Name of the record spec.")),
		mcp.WithString("recordId", mcp.Required(), mcp.Description("Id of the record.")),
		mcp.WithString("brickId", mcp.Required(), mcp.Description("Id of the brick.")),
	}
}

// registerBricks registers the bricks_* tools.
func (r *Registry) registerBricks(s *server.MCPServer) {
	r.registerBrickSerial(s)
	r.registerBrickMarkdown(s)
	r.registerBrickDelta(s)
}

func (r *Registry) registerBrickSerial(s *server.MCPServer) {
	tool := newWorkspaceTool(r, "bricks_next_serial",
		"Generate the next serial number for a serial-number brick.",
		r.brickArgs()...,
	)
	register(s, tool, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		resp, err := c.BrickNextSerial(ctx, wsID, strArg(args, "recordSpecName"), strArg(args, "recordId"), strArg(args, "brickId"))
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})
}

func (r *Registry) registerBrickMarkdown(s *server.MCPServer) {
	get := newWorkspaceTool(r, "bricks_markdown_get",
		"Get an editor brick's content as markdown.",
		r.brickArgs()...,
	)
	register(s, get, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		resp, err := c.BrickMarkdownGet(ctx, wsID, strArg(args, "recordSpecName"), strArg(args, "recordId"), strArg(args, "brickId"))
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})

	set := newWorkspaceTool(r, "bricks_markdown_set",
		"Set an editor brick's content from markdown.",
		append(r.brickArgs(),
			mcp.WithString("markdown", mcp.Required(), mcp.Description("Markdown content to write.")),
			mcp.WithBoolean("blockTyping", mcp.Description("Block typing on the brick while writing (default false).")),
		)...,
	)
	register(s, set, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		resp, err := c.BrickMarkdownSet(ctx, wsID,
			strArg(args, "recordSpecName"), strArg(args, "recordId"), strArg(args, "brickId"),
			strArg(args, "markdown"), boolArg(args, "blockTyping"))
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})
}

func (r *Registry) registerBrickDelta(s *server.MCPServer) {
	get := newWorkspaceTool(r, "bricks_delta_get",
		"Get an editor brick's content as a Quill Delta document.",
		r.brickArgs()...,
	)
	register(s, get, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		resp, err := c.BrickDeltaGet(ctx, wsID, strArg(args, "recordSpecName"), strArg(args, "recordId"), strArg(args, "brickId"))
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})

	set := newWorkspaceTool(r, "bricks_delta_set",
		"Set an editor brick's content from a Quill Delta (JSON ops document).",
		append(r.brickArgs(),
			mcp.WithString("delta", mcp.Required(), mcp.Description("Quill Delta document as JSON, e.g. {\"ops\":[{\"insert\":\"Hello\"}]}.")),
			mcp.WithBoolean("blockTyping", mcp.Description("Block typing on the brick while writing (default false).")),
		)...,
	)
	register(s, set, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		raw := strArg(args, "delta")
		if !json.Valid([]byte(raw)) {
			return errResult("argument \"delta\" must be a valid JSON value"), nil
		}
		resp, err := c.BrickDeltaSet(ctx, wsID,
			strArg(args, "recordSpecName"), strArg(args, "recordId"), strArg(args, "brickId"),
			json.RawMessage([]byte(raw)), boolArg(args, "blockTyping"))
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})
}