package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/dividebyzero/univelop-api-mcp/internal/client"
	"github.com/dividebyzero/univelop-api-mcp/internal/types"
)

// recordTools is the argument surface shared by the record CRUD tools.
func (r *Registry) recordArgs() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithString("recordSpecName", mcp.Required(), mcp.Description("Name of the record spec (list/tile) to operate on.")),
	}
}

// registerRecords registers the records_* tools.
func (r *Registry) registerRecords(s *server.MCPServer) {
	r.registerRecordsList(s)
	r.registerRecordsWrite(s)
	r.registerRecordsItem(s)
	r.registerRecordsSearch(s)
}

func (r *Registry) registerRecordsList(s *server.MCPServer) {
	tool := newWorkspaceTool(r, "records_list",
		"List records of a record spec, with optional filters, sorting and pagination.",
		append(r.recordArgs(),
			mcp.WithString("filters", mcp.Description("JSON-encoded filter array (e.g. [{\"field\":\"status\",\"operator\":\"equals\",\"value\":\"done\"}]).")),
			mcp.WithInteger("limit", mcp.Min(1), mcp.Max(1000), mcp.Description("Maximum number of records to return.")),
			mcp.WithString("startAfterDocumentId", mcp.Description("Pagination cursor: return records after this document id.")),
			mcp.WithBoolean("resolveLinkedBricks", mcp.Description("Resolve linked brick content in the response.")),
			mcp.WithString("orderBy", mcp.Description("Field name to order by (prefix with prefix filter restriction semantics).")),
			mcp.WithBoolean("orderByDescending", mcp.Description("Sort descending instead of ascending.")),
			mcp.WithBoolean("bypassFilteringRestrictions", mcp.Description("Bypass filter restrictions for API clients.")),
		)...,
	)

	register(s, tool, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		filters, badErr := queryJSONArg(args, "filters")
		if badErr != nil {
			return errResult(badErr.Error()), nil
		}
		p := client.RecordListParams{
			Filters:                     filters,
			Limit:                       intArg(args, "limit"),
			StartAfterDocumentID:        strArg(args, "startAfterDocumentId"),
			ResolveLinkedBricks:         boolArg(args, "resolveLinkedBricks"),
			OrderBy:                     strArg(args, "orderBy"),
			OrderByDescending:           boolArg(args, "orderByDescending"),
			BypassFilteringRestrictions: boolArg(args, "bypassFilteringRestrictions"),
		}
		resp, err := c.RecordsList(ctx, wsID, strArg(args, "recordSpecName"), p)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})
}

func (r *Registry) registerRecordsWrite(s *server.MCPServer) {
	// records_create_or_update (POST), records_update_by_keys (PATCH),
	// records_upsert_by_keys (PUT) share an identical argument surface.
	bodyDesc := mcp.WithString("body", mcp.Required(),
		mcp.Description("Record content as a JSON object (RecordPart). Fields map to spec fields; `id` is optional."))

	create := newWorkspaceTool(r, "records_create_or_update",
		"Create a record, or update the record matching its unique keys (POST).",
		append(r.recordArgs(), bodyDesc)...,
	)
	register(s, create, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		body, badErr := bodyArg(args, "body", true)
		if badErr != nil {
			return errResult(badErr.Error()), nil
		}
		resp, err := c.RecordsCreateOrUpdate(ctx, wsID, strArg(args, "recordSpecName"), body)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})

	updateByKeys := newWorkspaceTool(r, "records_update_by_keys",
		"Update records matching the unique keys present in the record part (PATCH to the collection).",
		append(r.recordArgs(), bodyDesc)...,
	)
	register(s, updateByKeys, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		body, badErr := bodyArg(args, "body", true)
		if badErr != nil {
			return errResult(badErr.Error()), nil
		}
		resp, err := c.RecordsUpdateByKeys(ctx, wsID, strArg(args, "recordSpecName"), body)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})

	upsert := newWorkspaceTool(r, "records_upsert_by_keys",
		"Create or fully replace a record by its unique keys (PUT to the collection).",
		append(r.recordArgs(), bodyDesc)...,
	)
	register(s, upsert, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		body, badErr := bodyArg(args, "body", true)
		if badErr != nil {
			return errResult(badErr.Error()), nil
		}
		resp, err := c.RecordsUpsertByKeys(ctx, wsID, strArg(args, "recordSpecName"), body)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})
}

func (r *Registry) registerRecordsItem(s *server.MCPServer) {
	idArg := mcp.WithString("recordId", mcp.Required(), mcp.Description("Id of the record."))

	get := newWorkspaceTool(r, "records_get",
		"Get a single record by id, optionally resolving linked brick content.",
		append(r.recordArgs(),
			idArg,
			mcp.WithBoolean("resolveLinkedBricks", mcp.Description("Resolve linked brick content in the response.")),
		)...,
	)
	register(s, get, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		resp, err := c.RecordsGet(ctx, wsID, strArg(args, "recordSpecName"), strArg(args, "recordId"), boolArg(args, "resolveLinkedBricks"))
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})

	update := newWorkspaceTool(r, "records_update",
		"Partially update a single record by id (PATCH).",
		append(r.recordArgs(),
			idArg,
			mcp.WithString("body", mcp.Required(), mcp.Description("Fields to update as a JSON object (RecordPart).")),
		)...,
	)
	register(s, update, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		body, badErr := bodyArg(args, "body", true)
		if badErr != nil {
			return errResult(badErr.Error()), nil
		}
		resp, err := c.RecordsUpdate(ctx, wsID, strArg(args, "recordSpecName"), strArg(args, "recordId"), body)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})

	del := newWorkspaceTool(r, "records_delete",
		"Delete a single record by id.",
		mcp.WithString("recordSpecName", mcp.Required(), mcp.Description("Name of the record spec.")),
		idArg,
	)
	register(s, del, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		resp, err := c.RecordsDelete(ctx, wsID, strArg(args, "recordSpecName"), strArg(args, "recordId"))
		if err != nil {
			return errResult(err.Error()), nil
		}
		if len(resp.Body) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("record %q deleted", strArg(args, "recordId"))), nil
		}
		return resultText(resp.Body), nil
	})
}

func (r *Registry) registerRecordsSearch(s *server.MCPServer) {
	tool := newWorkspaceTool(r, "records_search",
		"Full-text search (beta) over records of a record spec, with pagination metadata.",
		append(r.recordArgs(),
			mcp.WithString("query", mcp.Description("Free-text search query.")),
			mcp.WithString("filters", mcp.Description("JSON-encoded filter array.")),
			mcp.WithInteger("limit", mcp.Min(1), mcp.Max(1000), mcp.Description("Page size.")),
			mcp.WithInteger("page", mcp.Min(0), mcp.Description("Page number (0-based).")),
		)...,
	)

	register(s, tool, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		filters, badErr := queryJSONArg(args, "filters")
		if badErr != nil {
			return errResult(badErr.Error()), nil
		}
		p := client.RecordSearchParams{
			Query:   strArg(args, "query"),
			Filters: filters,
			Limit:   intArg(args, "limit"),
			Page:    intArg(args, "page"),
		}
		resp, err := c.RecordsSearch(ctx, wsID, strArg(args, "recordSpecName"), p)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return searchResult(resp), nil
	})
}

// searchResult wraps the search response body with pagination metadata
// extracted from the Total-Hits / Total-Pages / Current-Page / Limit headers.
func searchResult(resp *types.Response) *mcp.CallToolResult {
	meta := struct {
		TotalHits    int             `json:"totalHits"`
		TotalPages   int             `json:"totalPages"`
		CurrentPage  int             `json:"currentPage"`
		Limit        int             `json:"limit"`
		Records      json.RawMessage `json:"records"`
	}{}
	meta.TotalHits = headerInt(resp.Header, "Total-Hits")
	meta.TotalPages = headerInt(resp.Header, "Total-Pages")
	meta.CurrentPage = headerInt(resp.Header, "Current-Page")
	meta.Limit = headerInt(resp.Header, "Limit")
	meta.Records = json.RawMessage(resp.Body)
	out, err := json.Marshal(meta)
	if err != nil {
		return mcp.NewToolResultText(string(resp.Body))
	}
	return resultText(out)
}