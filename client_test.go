package zuzoto_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	zuzoto "github.com/zuzoto/zuzoto-go"
)

func TestAdd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/memories" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatal("missing auth header")
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["content"] != "hello world" {
			t.Fatalf("expected content 'hello world', got %v", body["content"])
		}
		json.NewEncoder(w).Encode(zuzoto.AddResult{
			Memories:        []zuzoto.Memory{{ID: "mem-1", Content: "hello world"}},
			EntitiesCreated: 1,
			FactsCreated:    2,
		})
	}))
	defer srv.Close()

	client := zuzoto.NewClient(srv.URL, zuzoto.WithAPIKey("test-key"))
	result, err := client.Add(context.Background(), &zuzoto.AddInput{Content: "hello world"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Memories) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(result.Memories))
	}
	if result.EntitiesCreated != 1 {
		t.Fatalf("expected 1 entity, got %d", result.EntitiesCreated)
	}
}

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["text"] != "vim" {
			t.Fatalf("expected text 'vim', got %v", body["text"])
		}
		// Check scope was set from convenience UserID.
		scope, ok := body["scope"].(map[string]any)
		if !ok || scope["user_id"] != "u1" {
			t.Fatalf("expected scope.user_id 'u1', got %v", scope)
		}
		json.NewEncoder(w).Encode(zuzoto.SearchResult{
			Memories: []zuzoto.ScoredMemory{{Memory: zuzoto.Memory{Content: "uses vim"}, Score: 0.95}},
			Total:    1,
		})
	}))
	defer srv.Close()

	client := zuzoto.NewClient(srv.URL, zuzoto.WithAPIKey("k"))
	result, err := client.Search(context.Background(), &zuzoto.SearchQuery{
		Text:   "vim",
		UserID: "u1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total=1, got %d", result.Total)
	}
	if result.Memories[0].Score != 0.95 {
		t.Fatalf("expected score=0.95, got %f", result.Memories[0].Score)
	}
}

func TestSearchWithTemporal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		temporal, ok := body["temporal"].(map[string]any)
		if !ok {
			t.Fatal("expected temporal in body")
		}
		if temporal["from"] == nil {
			t.Fatal("expected from in temporal")
		}
		json.NewEncoder(w).Encode(zuzoto.SearchResult{Total: 0})
	}))
	defer srv.Close()

	client := zuzoto.NewClient(srv.URL)
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := client.Search(context.Background(), &zuzoto.SearchQuery{
		Text: "test",
		From: &from,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") != "prefs" {
			t.Fatalf("expected query=prefs, got %s", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("user_id") != "u1" {
			t.Fatalf("expected user_id=u1")
		}
		if r.URL.Query().Get("max_tokens") != "2048" {
			t.Fatalf("expected max_tokens=2048, got %s", r.URL.Query().Get("max_tokens"))
		}
		json.NewEncoder(w).Encode(zuzoto.ContextWindow{
			Summary: "user likes vim",
			Tokens:  512,
		})
	}))
	defer srv.Close()

	client := zuzoto.NewClient(srv.URL)
	ctx, err := client.GetContext(context.Background(), &zuzoto.ContextQuery{
		Query:     "prefs",
		UserID:    "u1",
		MaxTokens: 2048,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Tokens != 512 {
		t.Fatalf("expected 512 tokens, got %d", ctx.Tokens)
	}
}

func TestForget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["memory_id"] != "mem-1" {
			t.Fatalf("expected memory_id=mem-1")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := zuzoto.NewClient(srv.URL)
	err := client.Forget(context.Background(), &zuzoto.ForgetInput{MemoryID: "mem-1"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBatchAdd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Items []map[string]any `json:"items"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if len(body.Items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(body.Items))
		}
		json.NewEncoder(w).Encode(zuzoto.BatchAddResult{Total: 2, Errors: 0})
	}))
	defer srv.Close()

	client := zuzoto.NewClient(srv.URL)
	result, err := client.BatchAdd(context.Background(), []*zuzoto.AddInput{
		{Content: "first"},
		{Content: "second"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total=2, got %d", result.Total)
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "text is required."})
	}))
	defer srv.Close()

	client := zuzoto.NewClient(srv.URL)
	_, err := client.Search(context.Background(), &zuzoto.SearchQuery{})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*zuzoto.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "text is required." {
		t.Fatalf("expected 'text is required.', got %q", apiErr.Message)
	}
}

func TestEntityCRUD(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/entities":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(zuzoto.Entity{ID: "ent-1", Name: "Alice"})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/entities/ent-1":
			json.NewEncoder(w).Encode(zuzoto.Entity{ID: "ent-1", Name: "Alice"})
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/entities/ent-1":
			json.NewEncoder(w).Encode(zuzoto.Entity{ID: "ent-1", Name: "Alice Updated"})
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/entities/ent-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	client := zuzoto.NewClient(srv.URL)
	ctx := context.Background()

	entity, err := client.CreateEntity(ctx, &zuzoto.CreateEntityInput{Name: "Alice", EntityType: "person"})
	if err != nil {
		t.Fatal(err)
	}
	if entity.Name != "Alice" {
		t.Fatalf("expected Alice, got %s", entity.Name)
	}

	got, err := client.GetEntity(ctx, "ent-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "ent-1" {
		t.Fatalf("expected ent-1, got %s", got.ID)
	}

	name := "Alice Updated"
	updated, err := client.UpdateEntity(ctx, "ent-1", &zuzoto.UpdateEntityInput{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Alice Updated" {
		t.Fatalf("expected 'Alice Updated', got %s", updated.Name)
	}

	err = client.DeleteEntity(ctx, "ent-1", zuzoto.DeleteModeSoft)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFactCRUD(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/facts":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(zuzoto.Fact{ID: "f-1", Subject: "Alice", Predicate: "works_at", Object: "Acme"})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/facts/f-1":
			json.NewEncoder(w).Encode(zuzoto.Fact{ID: "f-1"})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/facts/f-1/invalidate":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/facts/f-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	client := zuzoto.NewClient(srv.URL)
	ctx := context.Background()

	fact, err := client.CreateFact(ctx, &zuzoto.CreateFactInput{Subject: "Alice", Predicate: "works_at", Object: "Acme"})
	if err != nil {
		t.Fatal(err)
	}
	if fact.Subject != "Alice" {
		t.Fatalf("expected Alice, got %s", fact.Subject)
	}

	err = client.InvalidateFact(ctx, "f-1", nil)
	if err != nil {
		t.Fatal(err)
	}

	err = client.DeleteFact(ctx, "f-1", zuzoto.DeleteModeHard)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSessionLifecycle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/sessions":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(zuzoto.Session{ID: "s-1", UserID: "u1", Status: "active"})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/sessions/s-1":
			json.NewEncoder(w).Encode(zuzoto.Session{ID: "s-1", Status: "active"})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/sessions/s-1/close":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	client := zuzoto.NewClient(srv.URL)
	ctx := context.Background()

	sess, err := client.CreateSession(ctx, &zuzoto.CreateSessionInput{UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if sess.Status != "active" {
		t.Fatalf("expected active, got %s", sess.Status)
	}

	err = client.CloseSession(ctx, "s-1")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCustomHTTPClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "zuzoto-go/0.1.0" {
			t.Fatalf("expected zuzoto-go user agent, got %s", r.Header.Get("User-Agent"))
		}
		json.NewEncoder(w).Encode(zuzoto.Memory{ID: "m-1"})
	}))
	defer srv.Close()

	custom := &http.Client{Timeout: 5 * time.Second}
	client := zuzoto.NewClient(srv.URL, zuzoto.WithHTTPClient(custom))
	_, err := client.Get(context.Background(), "m-1")
	if err != nil {
		t.Fatal(err)
	}
}
