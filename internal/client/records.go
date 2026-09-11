package client

import (
	"context"

	"github.com/dividebyzero/univelop-api-mcp/internal/types"
)

// RecordListParams mirrors the query parameters of GET /api/v2/workspaces/{workspaceId}/records/{recordSpecName}.
type RecordListParams struct {
	Filters                    string
	Limit                      int
	StartAfterDocumentID       string
	ResolveLinkedBricks        bool
	OrderBy                    string
	OrderByDescending          bool
	BypassFilteringRestrictions bool
}

// RecordSearchParams mirrors the query parameters of GET .../records/{recordSpecName}/search.
type RecordSearchParams struct {
	Query   string
	Filters string
	Limit   int
	Page    int
}

// RecordsList lists records of a given record spec.
func (c *Client) RecordsList(ctx context.Context, workspaceID, recordSpecName string, p RecordListParams) (*types.Response, error) {
	q := NewQuery()
	q.SetString("filters", p.Filters)
	q.SetInt("limit", p.Limit)
	q.SetString("startAfterDocumentId", p.StartAfterDocumentID)
	q.SetBool("resolveLinkedBricks", p.ResolveLinkedBricks)
	q.SetString("orderBy", p.OrderBy)
	q.SetBool("orderByDescending", p.OrderByDescending)
	q.SetBool("bypassFilteringRestrictions", p.BypassFilteringRestrictions)

	path := "/api/v2/workspaces/" + EscapePath(workspaceID) + "/records/" + EscapePath(recordSpecName)
	return c.Get(ctx, path, q.Values)
}

// RecordsCreateOrUpdate creates a record or updates it by its unique keys (POST).
func (c *Client) RecordsCreateOrUpdate(ctx context.Context, workspaceID, recordSpecName string, body []byte) (*types.Response, error) {
	path := "/api/v2/workspaces/" + EscapePath(workspaceID) + "/records/" + EscapePath(recordSpecName)
	return c.Post(ctx, path, nil, body)
}

// RecordsUpdateByKeys updates records matching the unique keys in the record part (PATCH).
func (c *Client) RecordsUpdateByKeys(ctx context.Context, workspaceID, recordSpecName string, body []byte) (*types.Response, error) {
	path := "/api/v2/workspaces/" + EscapePath(workspaceID) + "/records/" + EscapePath(recordSpecName)
	return c.Patch(ctx, path, nil, body)
}

// RecordsUpsertByKeys is a full upsert by unique keys (PUT).
func (c *Client) RecordsUpsertByKeys(ctx context.Context, workspaceID, recordSpecName string, body []byte) (*types.Response, error) {
	path := "/api/v2/workspaces/" + EscapePath(workspaceID) + "/records/" + EscapePath(recordSpecName)
	return c.Put(ctx, path, nil, body)
}

// RecordsGet fetches a single record by ID.
func (c *Client) RecordsGet(ctx context.Context, workspaceID, recordSpecName, recordID string, resolveLinkedBricks bool) (*types.Response, error) {
	q := NewQuery()
	q.SetBool("resolveLinkedBricks", resolveLinkedBricks)
	path := "/api/v2/workspaces/" + EscapePath(workspaceID) + "/records/" + EscapePath(recordSpecName) + "/" + EscapePath(recordID)
	return c.Get(ctx, path, q.Values)
}

// RecordsUpdate updates a specific record by ID (PATCH).
func (c *Client) RecordsUpdate(ctx context.Context, workspaceID, recordSpecName, recordID string, body []byte) (*types.Response, error) {
	path := "/api/v2/workspaces/" + EscapePath(workspaceID) + "/records/" + EscapePath(recordSpecName) + "/" + EscapePath(recordID)
	return c.Patch(ctx, path, nil, body)
}

// RecordsDelete deletes a specific record by ID.
func (c *Client) RecordsDelete(ctx context.Context, workspaceID, recordSpecName, recordID string) (*types.Response, error) {
	path := "/api/v2/workspaces/" + EscapePath(workspaceID) + "/records/" + EscapePath(recordSpecName) + "/" + EscapePath(recordID)
	return c.Delete(ctx, path)
}

// RecordsSearch performs a full-text search over records of a spec.
func (c *Client) RecordsSearch(ctx context.Context, workspaceID, recordSpecName string, p RecordSearchParams) (*types.Response, error) {
	q := NewQuery()
	q.SetString("query", p.Query)
	q.SetString("filters", p.Filters)
	q.SetInt("limit", p.Limit)
	q.SetInt("page", p.Page)

	path := "/api/v2/workspaces/" + EscapePath(workspaceID) + "/records/" + EscapePath(recordSpecName) + "/search"
	return c.Get(ctx, path, q.Values)
}