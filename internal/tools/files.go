package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/dividebyzero/univelop-api-mcp/internal/client"
)

// registerFiles registers the files_* tools.
func (r *Registry) registerFiles(s *server.MCPServer) {
	brArgs := []mcp.ToolOption{
		mcp.WithString("recordSpecName", mcp.Required(), mcp.Description("Name of the record spec.")),
		mcp.WithString("recordId", mcp.Required(), mcp.Description("Id of the record.")),
		mcp.WithString("brickId", mcp.Required(), mcp.Description("Id of the file brick.")),
	}

	upload := newWorkspaceTool(r, "files_upload",
		"Upload one or more files to a file brick. Files are passed as a JSON array of {name, content (base64), mimeType, favorite}.",
		append(brArgs,
			mcp.WithString("files", mcp.Required(),
				mcp.Description("JSON array of file objects: [{\"name\":\"doc.pdf\",\"content\":\"base64...\",\"mimeType\":\"application/pdf\",\"favorite\":false}]")),
		)...,
	)
	register(s, upload, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		filesStr := strArg(args, "files")
		if filesStr == "" {
			return errResult("missing required argument \"files\""), nil
		}
		var items []client.FileItem
		if err := json.Unmarshal([]byte(filesStr), &items); err != nil {
			return errResult("argument \"files\" must be a valid JSON array of file objects: " + err.Error()), nil
		}
		resp, err := c.FilesUpload(ctx, wsID, strArg(args, "recordSpecName"), strArg(args, "recordId"), strArg(args, "brickId"), items)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})

	download := newWorkspaceTool(r, "files_download_zip",
		"Download selected files from a file brick as a zip archive.",
		append(brArgs,
			mcp.WithString("files", mcp.Required(),
				mcp.Description("JSON array of file names to include, e.g. [\"doc1.pdf\",\"img.png\"]")),
		)...,
	)
	register(s, download, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		filesStr := strArg(args, "files")
		if filesStr == "" {
			return errResult("missing required argument \"files\""), nil
		}
		var names []string
		if err := json.Unmarshal([]byte(filesStr), &names); err != nil {
			return errResult("argument \"files\" must be a valid JSON array of file name strings: " + err.Error()), nil
		}
		resp, err := c.FilesDownloadZip(ctx, wsID, strArg(args, "recordSpecName"), strArg(args, "recordId"), strArg(args, "brickId"), names)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return resultText(resp.Body), nil
	})

	del := newWorkspaceTool(r, "files_delete",
		"Delete a single file from a file brick.",
		append(brArgs,
			mcp.WithString("fileName", mcp.Required(), mcp.Description("Name of the file to delete.")),
		)...,
	)
	register(s, del, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		c, wsID, bad := r.clientFor(args)
		if bad != nil {
			return bad, nil
		}
		resp, err := c.FilesDelete(ctx, wsID, strArg(args, "recordSpecName"), strArg(args, "recordId"), strArg(args, "brickId"), strArg(args, "fileName"))
		if err != nil {
			return errResult(err.Error()), nil
		}
		if len(resp.Body) == 0 {
			return mcp.NewToolResultText("file deleted"), nil
		}
		return resultText(resp.Body), nil
	})
}