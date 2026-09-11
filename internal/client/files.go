package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dividebyzero/univelop-api-mcp/internal/types"
)

// FileItem is one file in an upload or download request.
type FileItem struct {
	Name     string `json:"name"`
	Content  string `json:"content,omitempty"` // base64-encoded for uploads
	MimeType string `json:"mimeType,omitempty"`
	Favorite bool   `json:"favorite,omitempty"`
}

// FileUploadBody is the JSON body for POST .../bricks/{brickId}.
type FileUploadBody struct {
	Files []FileItem `json:"files"`
}

// FilesUpload uploads files (base64 content) to a brick.
func (c *Client) FilesUpload(ctx context.Context, workspaceID, recordSpecName, recordID, brickID string, files []FileItem) (*types.Response, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("at least one file is required")
	}
	for _, f := range files {
		if f.Name == "" {
			return nil, fmt.Errorf("each file must have a name")
		}
	}
	body, err := json.Marshal(FileUploadBody{Files: files})
	if err != nil {
		return nil, fmt.Errorf("marshaling upload request: %w", err)
	}
	path := recordBrickPath(workspaceID, recordSpecName, recordID, brickID)
	return c.Post(ctx, path, nil, body)
}

// FilesDownloadZip downloads the named files from a brick as a ZIP archive.
func (c *Client) FilesDownloadZip(ctx context.Context, workspaceID, recordSpecName, recordID, brickID string, fileNames []string) (*types.Response, error) {
	if len(fileNames) == 0 {
		return nil, fmt.Errorf("at least one file name is required")
	}
	body, err := json.Marshal(map[string][]string{"files": fileNames})
	if err != nil {
		return nil, fmt.Errorf("marshaling download request: %w", err)
	}
	path := recordBrickPath(workspaceID, recordSpecName, recordID, brickID) + "/download"
	return c.Post(ctx, path, nil, body)
}

// FilesDelete deletes a single file from a brick.
func (c *Client) FilesDelete(ctx context.Context, workspaceID, recordSpecName, recordID, brickID, fileName string) (*types.Response, error) {
	path := recordBrickPath(workspaceID, recordSpecName, recordID, brickID) + "/" + EscapePath(fileName)
	return c.Delete(ctx, path)
}