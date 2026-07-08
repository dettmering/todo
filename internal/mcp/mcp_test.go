package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/till/todo/internal/events"
	"github.com/till/todo/internal/service"
	"github.com/till/todo/internal/storage/sqlite"
)

func newTestService(t *testing.T) *service.Service {
	t.Helper()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return service.New(store.Tasks(), store.Projects(), store.Audit(), store.Comments(), store.Absences(), events.NewBus())
}

// connect wires an in-memory MCP client to the todo server and returns a
// connected client session.
func connect(t *testing.T, svc *service.Service) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	srv := (&server{svc: svc}).build()

	serverT, clientT := mcp.NewInMemoryTransports()
	if _, err := srv.Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// callJSON invokes a tool and unmarshals its text content into v. It fails the
// test if the tool reports an error.
func callJSON(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any, v any) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("call %s returned tool error: %+v", name, res.Content)
	}
	if v == nil {
		return
	}
	if len(res.Content) == 0 {
		t.Fatalf("call %s: no content", name)
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("call %s: content[0] is %T, want *TextContent", name, res.Content[0])
	}
	if err := json.Unmarshal([]byte(tc.Text), v); err != nil {
		t.Fatalf("call %s: unmarshal %q: %v", name, tc.Text, err)
	}
}

func TestToolsRegistered(t *testing.T) {
	cs := connect(t, newTestService(t))
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	got := map[string]bool{}
	for _, tool := range res.Tools {
		got[tool.Name] = true
	}
	want := []string{
		"list_tasks", "get_task", "create_task", "update_task",
		"complete_task", "reopen_task", "delete_task",
		"list_projects", "create_project",
		"list_absences", "create_absence", "next_workday", "parse_task",
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("tool %q not registered", name)
		}
	}
	if len(res.Tools) != len(want) {
		t.Errorf("got %d tools, want %d", len(res.Tools), len(want))
	}
}

func TestCreateAndListTask(t *testing.T) {
	cs := connect(t, newTestService(t))

	var created taskOut
	callJSON(t, cs, "create_task", map[string]any{
		"title":    "Buy milk",
		"priority": 1,
	}, &created)
	if created.ID == "" {
		t.Fatal("create_task returned empty id")
	}
	if created.Priority != 1 {
		t.Errorf("priority = %d, want 1", created.Priority)
	}
	if created.ProjectID != "inbox" {
		t.Errorf("project_id = %q, want inbox", created.ProjectID)
	}

	var list tasksOutput
	callJSON(t, cs, "list_tasks", map[string]any{}, &list)
	found := false
	for _, task := range list.Tasks {
		if task.ID == created.ID {
			found = true
			if task.Title != "Buy milk" {
				t.Errorf("title = %q, want Buy milk", task.Title)
			}
		}
	}
	if !found {
		t.Errorf("created task %s not in list", created.ID)
	}

	// complete_task should flip status to completed.
	var completed taskOut
	callJSON(t, cs, "complete_task", map[string]any{"id": created.ID}, &completed)
	if completed.Status != "completed" {
		t.Errorf("status after complete = %q, want completed", completed.Status)
	}
}

func TestToolErrorSurfacesValidation(t *testing.T) {
	cs := connect(t, newTestService(t))
	// Missing title violates service validation → tool error, not transport error.
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "create_task",
		Arguments: map[string]any{"title": ""},
	})
	if err != nil {
		t.Fatalf("CallTool transport error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError for empty title")
	}
}

func TestBearerAuth(t *testing.T) {
	const token = "s3cret-token"
	var reached bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true })
	h := bearerAuth(token, next)

	// No header → 401.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/mcp", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: status = %d, want 401", rec.Code)
	}
	if reached {
		t.Error("handler reached without token")
	}

	// Wrong token → 401.
	reached = false
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: status = %d, want 401", rec.Code)
	}

	// Correct token via header → passes through.
	reached = false
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	h.ServeHTTP(rec, req)
	if !reached {
		t.Error("handler not reached with correct header token")
	}

	// Correct token via ?token= query (claude.ai path) → passes through.
	reached = false
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/mcp?token="+token, nil))
	if !reached {
		t.Error("handler not reached with correct query token")
	}

	// Wrong query token → 401.
	reached = false
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/mcp?token=nope", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong query token: status = %d, want 401", rec.Code)
	}
}
