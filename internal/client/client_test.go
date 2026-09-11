package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dividebyzero/univelop-api-mcp/internal/types"
)

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	c, err := New(baseURL, "secret-api-key", Options{Timeout: 5 * time.Second, MaxRetries: 1})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRequestSetsAuthHeader(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.Get(context.Background(), "/api/v2/workspaces/abc/records/spec", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if gotKey != "secret-api-key" {
		t.Fatalf("expected x-api-key 'secret-api-key', got %q", gotKey)
	}
}

func TestPathEscaping(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.RecordsGet(context.Background(), "ws/1", "my spec", "rec&id", false)
	if err != nil {
		t.Fatal(err)
	}
	if want := "/api/v2/workspaces/ws%2F1/records/my%20spec/rec&id"; gotPath != want {
		t.Fatalf("expected %q, got %q", want, gotPath)
	}
}

func TestQueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.RecordsList(context.Background(), "ws1", "spec1", RecordListParams{
		Filters:              `[{"field":"a"}]`,
		Limit:                25,
		OrderBy:              "title",
		OrderByDescending:    true,
		ResolveLinkedBricks:  true,
		BypassFilteringRestrictions: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"filters=%5B%7B%22field%22%3A%22a%22%7D%5D", "limit=25", "orderBy=title", "orderByDescending=true", "resolveLinkedBricks=true", "bypassFilteringRestrictions=true"} {
		if !strings.Contains(gotQuery, want) {
			t.Fatalf("query %q missing %q", gotQuery, want)
		}
	}
}

func TestRetryOn503ThenSuccess(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.Get(context.Background(), "/api/v2/workspaces/ab/records/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 after retry, got %d", resp.StatusCode)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", calls.Load())
	}
}

func TestNonRetryableError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request","message":"invalid filters"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Get(context.Background(), "/api/v2/workspaces/ab/records/x", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*types.APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", apiErr.StatusCode)
	}
	if !strings.Contains(err.Error(), "invalid filters") {
		t.Fatalf("expected body message in error, got %q", err.Error())
	}
}

func TestJSONBody(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		json.NewDecoder(r.Body).Decode(&gotBody)
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"new1"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.RecordsCreateOrUpdate(context.Background(), "ws1", "spec1", []byte(`{"title":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if gotBody["title"] != "hello" {
		t.Fatalf("expected body title hello, got %v", gotBody)
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10) // 10 rps
	start := time.Now()
	for i := 0; i < 10; i++ {
		if err := rl.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("limiter too slow: %v", time.Since(start))
	}
	// The 11th call should block for ~100ms
	start = time.Now()
	if err := rl.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) < 50*time.Millisecond {
		t.Fatalf("expected throttle, got %v", time.Since(start))
	}
}