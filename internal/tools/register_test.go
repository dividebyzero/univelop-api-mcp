package tools

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/dividebyzero/univelop-api-mcp/internal/config"
)

func testConfig() *config.Config {
	cfg, err := config.Parse([]byte(`
mcp:
  port: 8443
  name: "Univelop API MCP"
  version: "0.1.0"
univelop:
  default_base_url: "https://app.univelop.de/"
  timeout: "30s"
  max_retries: 3
workspaces:
  production:
    workspace_id: "wkspc_prod_1"
    api_key: "key-prod"
    label: "Production"
  staging:
    workspace_id: "wkspc_stage_2"
    api_key: "key-stage"
    label: "Staging"
`))
	if err != nil {
		panic(err)
	}
	return cfg
}

func TestRegistryResolveByAlias(t *testing.T) {
	r, err := NewRegistry(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	c, w, err := r.Resolve("production")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("client is nil")
	}
	if w.ID != "wkspc_prod_1" {
		t.Fatalf("expected wkspc_prod_1, got %q", w.ID)
	}
}

func TestRegistryResolveByRawID(t *testing.T) {
	r, _ := NewRegistry(testConfig())
	_, w, err := r.Resolve("wkspc_stage_2")
	if err != nil {
		t.Fatal(err)
	}
	if w.ID != "wkspc_stage_2" {
		t.Fatalf("expected wkspc_stage_2, got %q", w.ID)
	}
}

func TestRegistryResolveUnknown(t *testing.T) {
	r, _ := NewRegistry(testConfig())
	_, _, err := r.Resolve("nope")
	if err == nil {
		t.Fatal("expected error for unknown workspace")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected workspace id in error, got %q", err.Error())
	}
}

func TestRegistryRegistersAllTools(t *testing.T) {
	r, err := NewRegistry(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	s := server.NewMCPServer("test", "0.0.0")
	r.Register(s)
	tools := s.ListTools()
	if len(tools) != 23 {
		t.Fatalf("expected 23 tools, got %d", len(tools))
	}
	for _, want := range []string{
		"workspace_get",
		"records_list", "records_create_or_update", "records_update_by_keys",
		"records_upsert_by_keys", "records_get", "records_update", "records_delete",
		"records_search",
		"bricks_next_serial", "bricks_markdown_get", "bricks_markdown_set",
		"bricks_delta_get", "bricks_delta_set",
		"workflows_list", "workflows_last_execution", "workflows_run", "workflows_run_and_wait",
		"notifications_send", "users_create",
		"files_upload", "files_download_zip", "files_delete",
	} {
		if _, ok := tools[want]; !ok {
			t.Errorf("missing tool %q", want)
		}
	}
}

func TestClientForUsesWorkspaceIDForPath(t *testing.T) {
	r, err := NewRegistry(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	args := map[string]interface{}{"workspaceId": "production"}
	c, pathID, bad := r.clientFor(args)
	if bad != nil {
		t.Fatalf("unexpected bad result from clientFor: %+v", bad.Content)
	}
	if c == nil {
		t.Fatal("client is nil")
	}
	if pathID != "wkspc_prod_1" {
		t.Fatalf("expected path id wkspc_prod_1, got %q", pathID)
	}
}

func TestToolSchemaShape(t *testing.T) {
	r, _ := NewRegistry(testConfig())
	s := server.NewMCPServer("test", "0.0.0")
	r.Register(s)
	tools := s.ListTools()
	rec := tools["records_list"]
	if rec == nil {
		t.Fatal("records_list not registered")
	}
	schema := rec.Tool.InputSchema
	props := schema.Properties
	if _, ok := props["workspaceId"]; !ok {
		t.Fatal("records_list missing workspaceId property")
	}
	wsRequired := false
	for _, name := range schema.Required {
		if name == "workspaceId" {
			wsRequired = true
		}
	}
	if !wsRequired {
		t.Fatal("workspaceId must be required in schema")
	}
	_ = mcp.WithString("unused") // keep import
}