package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/dividebyzero/univelop-api-mcp/internal/client"
)

// registerWorkflows registers the workflows_* tools.
func (r *Registry) registerWorkflows(s *server.MCPServer) {
	flowArg := mcp.WithString("flowSpecNameOrId", mcp.Required(), mcp.Description("Name or id of the flow spec."))

	list := newWorkspaceTool(r, "workflows_list",
		"List workflow executions of a flow spec series, with optional filters and pagination.",
		flowArg,
		mcp.WithString("orderBy", mcp.Description("Field to order by.")),
		mcp.WithBoolean("orderByDescending", mcp.Description("Sort descending.")),
		mcp.WithString("filters", mcp.Description("JSON-encoded filter array.")),
		mcp.WithString("startAfterDocument", mcp.Description("Pagination cursor document id.")),
		mcp.WithInteger("limit", mcp.Min(1), mcp.Max(1000), mcp.Description("Maximum number of executions to return.")),
	)
	register(s, list, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		filters, badErr := queryJSONArg(args, "filters")
		if badErr != nil {
			return errResult(badErr.Error()), nil
		}
		p := client.WorkflowListParams{
			OrderBy:            strArg(args, "orderBy"),
			OrderByDescending:  boolArg(args, "orderByDescending"),
			Filters:            filters,
			StartAfterDocument: strArg(args, "startAfterDocument"),
			Limit:              intArg(args, "limit"),
		}
		resp, err := c.WorkflowsList(ctx, wsID, strArg(args, "flowSpecNameOrId"), p)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})

	last := newWorkspaceTool(r, "workflows_last_execution",
		"Get the most recent execution of a flow spec.",
		flowArg,
	)
	register(s, last, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		resp, err := c.WorkflowsLastExecution(ctx, wsID, strArg(args, "flowSpecNameOrId"))
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})

	run := newWorkspaceTool(r, "workflows_run",
		"Trigger a workflow run asynchronously and return immediately.",
		flowArg,
		mcp.WithString("triggerSpecId", mcp.Description("Optional trigger spec id to start the flow.")),
		mcp.WithString("payload", mcp.Description("Optional JSON payload passed to the workflow.")),
	)
	register(s, run, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		payload, badErr := payloadArg(args)
		if badErr != nil {
			return badErr, nil
		}
		resp, err := c.WorkflowsRun(ctx, wsID, strArg(args, "flowSpecNameOrId"), strArg(args, "triggerSpecId"), payload)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})

	runAndWait := newWorkspaceTool(r, "workflows_run_and_wait",
		"Trigger a workflow run and wait for its result (the API blocks server-side; tool timeout defaults to 60s).",
		flowArg,
		mcp.WithString("triggerSpecId", mcp.Description("Optional trigger spec id to start the flow.")),
		mcp.WithString("payload", mcp.Description("Optional JSON payload passed to the workflow.")),
	)
	register(s, runAndWait, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		payload, badErr := payloadArg(args)
		if badErr != nil {
			return badErr, nil
		}
		if d := intArg(args, "timeoutSeconds"); d > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(d)*time.Second)
			defer cancel()
		}
		resp, err := c.WorkflowsRunAndWait(ctx, wsID, strArg(args, "flowSpecNameOrId"), strArg(args, "triggerSpecId"), payload)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})
}

// payloadArg validates the optional "payload" JSON argument.
func payloadArg(args map[string]interface{}) (json.RawMessage, *mcp.CallToolResult) {
	raw := strArg(args, "payload")
	if raw == "" {
		return nil, nil
	}
	if !json.Valid([]byte(raw)) {
		return nil, errResult("argument \"payload\" must be a valid JSON value")
	}
	return json.RawMessage(raw), nil
}