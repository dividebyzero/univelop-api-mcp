package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dividebyzero/univelop-api-mcp/internal/types"
)

// BrickNextSerial gets the next serial number for a brick.
func (c *Client) BrickNextSerial(ctx context.Context, workspaceID, recordSpecName, recordID, brickID string) (*types.Response, error) {
	path := recordBrickPath(workspaceID, recordSpecName, recordID, brickID) + "/next-serial-number"
	return c.Post(ctx, path, nil, nil)
}

// BrickMarkdownGet returns the markdown content of a brick.
func (c *Client) BrickMarkdownGet(ctx context.Context, workspaceID, recordSpecName, recordID, brickID string) (*types.Response, error) {
	path := recordBrickPath(workspaceID, recordSpecName, recordID, brickID) + "/markdown"
	return c.Get(ctx, path, nil)
}

// BrickMarkdownSet writes markdown content to a brick.
func (c *Client) BrickMarkdownSet(ctx context.Context, workspaceID, recordSpecName, recordID, brickID, markdown string, blockTyping bool) (*types.Response, error) {
	body, err := json.Marshal(map[string]string{"markdown": markdown})
	if err != nil {
		return nil, fmt.Errorf("marshaling markdown body: %w", err)
	}
	q := NewQuery()
	q.SetBool("blockTyping", blockTyping)
	path := recordBrickPath(workspaceID, recordSpecName, recordID, brickID) + "/markdown"
	return c.Post(ctx, path, q.Values, body)
}

// BrickDeltaGet returns the rich-text delta of a brick.
func (c *Client) BrickDeltaGet(ctx context.Context, workspaceID, recordSpecName, recordID, brickID string) (*types.Response, error) {
	path := recordBrickPath(workspaceID, recordSpecName, recordID, brickID) + "/delta"
	return c.Get(ctx, path, nil)
}

// BrickDeltaSet writes a rich-text delta (Quill ops JSON) to a brick.
// delta must be a valid JSON value (e.g. {"ops": [...]} or an array).
func (c *Client) BrickDeltaSet(ctx context.Context, workspaceID, recordSpecName, recordID, brickID string, delta json.RawMessage, blockTyping bool) (*types.Response, error) {
	if len(delta) == 0 {
		return nil, fmt.Errorf("delta is required")
	}
	if !json.Valid(delta) {
		return nil, fmt.Errorf("delta must be a valid JSON value")
	}
	body := []byte(`{"delta":` + string(delta) + `}`)
	q := NewQuery()
	q.SetBool("blockTyping", blockTyping)
	path := recordBrickPath(workspaceID, recordSpecName, recordID, brickID) + "/delta"
	return c.Post(ctx, path, q.Values, body)
}

func recordBrickPath(workspaceID, recordSpecName, recordID, brickID string) string {
	return "/api/v2/workspaces/" + EscapePath(workspaceID) +
		"/records/" + EscapePath(recordSpecName) +
		"/" + EscapePath(recordID) +
		"/bricks/" + EscapePath(brickID)
}