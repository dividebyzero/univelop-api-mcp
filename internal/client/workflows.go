package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dividebyzero/univelop-api-mcp/internal/types"
)

// WorkflowListParams mirrors the query parameters of GET .../workflows/{flowSpecNameOrId}.
type WorkflowListParams struct {
	OrderBy           string
	OrderByDescending bool
	Filters           string
	StartAfterDocument string
	Limit             int
}

// WorkflowsList lists workflows/follow-up flows of a flow spec series.
func (c *Client) WorkflowsList(ctx context.Context, workspaceID, flowSpecNameOrID string, p WorkflowListParams) (*types.Response, error) {
	q := NewQuery()
	q.SetString("orderBy", p.OrderBy)
	q.SetBool("orderByDescending", p.OrderByDescending)
	q.SetString("filters", p.Filters)
	q.SetString("startAfterDocument", p.StartAfterDocument)
	q.SetInt("limit", p.Limit)

	path := workflowPath(workspaceID, flowSpecNameOrID)
	return c.Get(ctx, path, q.Values)
}

// WorkflowsLastExecution returns the most recent execution for a flow spec.
func (c *Client) WorkflowsLastExecution(ctx context.Context, workspaceID, flowSpecNameOrID string) (*types.Response, error) {
	path := workflowPath(workspaceID, flowSpecNameOrID) + "/lastExecution"
	return c.Get(ctx, path, nil)
}

// WorkflowsRun triggers a workflow run. payload may be nil. Returns immediately.
func (c *Client) WorkflowsRun(ctx context.Context, workspaceID, flowSpecNameOrID, triggerSpecID string, payload json.RawMessage) (*types.Response, error) {
	path := workflowPath(workspaceID, flowSpecNameOrID) + "/run"
	return c.runWorkflow(ctx, path, triggerSpecID, payload)
}

// WorkflowsRunAndWait triggers a workflow run and waits for the result.
func (c *Client) WorkflowsRunAndWait(ctx context.Context, workspaceID, flowSpecNameOrID, triggerSpecID string, payload json.RawMessage) (*types.Response, error) {
	path := workflowPath(workspaceID, flowSpecNameOrID) + "/run-and-wait"
	return c.runWorkflow(ctx, path, triggerSpecID, payload)
}

func (c *Client) runWorkflow(ctx context.Context, path, triggerSpecID string, payload json.RawMessage) (*types.Response, error) {
	if len(payload) > 0 && !json.Valid(payload) {
		return nil, fmt.Errorf("payload must be a valid JSON value")
	}
	q := NewQuery()
	q.SetString("triggerSpecId", triggerSpecID)
	var body []byte
	if len(payload) > 0 {
		body = payload
	}
	return c.Post(ctx, path, q.Values, body)
}

func workflowPath(workspaceID, flowSpecNameOrID string) string {
	return "/api/v2/workspaces/" + EscapePath(workspaceID) + "/workflows/" + EscapePath(flowSpecNameOrID)
}