package zuzoto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client is the Zuzoto API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Option configures the client.
type Option func(*Client)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient creates a new Zuzoto client.
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ---- memory operations -----------------------------------------------------

// Add ingests content or conversation messages into memory.
func (c *Client) Add(ctx context.Context, input *AddInput) (*AddResult, error) {
	var result AddResult
	if err := c.post(ctx, "/v1/memories", input, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// BatchAdd ingests multiple memories in one call (max 100).
func (c *Client) BatchAdd(ctx context.Context, items []*AddInput) (*BatchAddResult, error) {
	body := struct {
		Items []*AddInput `json:"items"`
	}{Items: items}
	var result BatchAddResult
	if err := c.post(ctx, "/v1/memories/batch", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a memory by ID.
func (c *Client) Get(ctx context.Context, id string) (*Memory, error) {
	var mem Memory
	if err := c.get(ctx, "/v1/memories/"+url.PathEscape(id), nil, &mem); err != nil {
		return nil, err
	}
	return &mem, nil
}

// Update patches a memory.
func (c *Client) Update(ctx context.Context, id string, input *UpdateMemoryInput) (*Memory, error) {
	var mem Memory
	if err := c.patch(ctx, "/v1/memories/"+url.PathEscape(id), input, &mem); err != nil {
		return nil, err
	}
	return &mem, nil
}

// Delete removes a memory by ID.
func (c *Client) Delete(ctx context.Context, id string, mode string) error {
	params := url.Values{}
	if mode != "" {
		params.Set("mode", mode)
	}
	return c.del(ctx, "/v1/memories/"+url.PathEscape(id), params)
}

// Search runs hybrid search across memories.
func (c *Client) Search(ctx context.Context, query *SearchQuery) (*SearchResult, error) {
	// Build the API request body with scope and temporal from convenience fields.
	type apiSearchReq struct {
		Text       string   `json:"text"`
		Scope      *Scope   `json:"scope,omitempty"`
		Temporal   any      `json:"temporal,omitempty"`
		Strategies []string `json:"strategies,omitempty"`
		Limit      int      `json:"limit,omitempty"`
		MinScore   float64  `json:"min_score,omitempty"`
	}

	req := apiSearchReq{
		Text:       query.Text,
		Scope:      query.Scope,
		Strategies: query.Strategies,
		Limit:      query.Limit,
		MinScore:   query.MinScore,
	}

	// Apply convenience fields.
	if query.UserID != "" {
		if req.Scope == nil {
			req.Scope = &Scope{}
		}
		req.Scope.UserID = query.UserID
	}
	if query.From != nil || query.To != nil {
		req.Temporal = struct {
			From *time.Time `json:"from,omitempty"`
			To   *time.Time `json:"to,omitempty"`
		}{From: query.From, To: query.To}
	}

	var result SearchResult
	if err := c.post(ctx, "/v1/memories/search", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetContext assembles a token-budgeted context window for an LLM.
func (c *Client) GetContext(ctx context.Context, query *ContextQuery) (*ContextWindow, error) {
	params := url.Values{}
	params.Set("query", query.Query)
	if query.UserID != "" {
		params.Set("user_id", query.UserID)
	}
	if query.SessionID != "" {
		params.Set("session_id", query.SessionID)
	}
	if query.AgentID != "" {
		params.Set("agent_id", query.AgentID)
	}
	if query.MaxTokens > 0 {
		params.Set("max_tokens", strconv.Itoa(query.MaxTokens))
	}

	var window ContextWindow
	if err := c.get(ctx, "/v1/memories/context", params, &window); err != nil {
		return nil, err
	}
	return &window, nil
}

// Forget deletes memories by ID or user.
func (c *Client) Forget(ctx context.Context, input *ForgetInput) error {
	return c.postNoResponse(ctx, "/v1/memories/forget", input)
}

// PointInTime queries memories as they existed at a specific time.
func (c *Client) PointInTime(ctx context.Context, query string, asOf time.Time, limit int) (*SearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("as_of", asOf.Format(time.RFC3339))
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	var result SearchResult
	if err := c.get(ctx, "/v1/memories/point-in-time", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ---- entity operations -----------------------------------------------------

// ListEntities lists entities with optional filters.
func (c *Client) ListEntities(ctx context.Context, opts *ListEntitiesOpts) (*Page[Entity], error) {
	params := url.Values{}
	if opts != nil {
		if opts.UserID != "" {
			params.Set("user_id", opts.UserID)
		}
		if opts.Type != "" {
			params.Set("type", opts.Type)
		}
		if opts.NamePrefix != "" {
			params.Set("name_prefix", opts.NamePrefix)
		}
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
	}

	var page Page[Entity]
	if err := c.get(ctx, "/v1/entities", params, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// CreateEntity creates a new entity.
func (c *Client) CreateEntity(ctx context.Context, input *CreateEntityInput) (*Entity, error) {
	var entity Entity
	if err := c.post(ctx, "/v1/entities", input, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

// GetEntity retrieves an entity by ID.
func (c *Client) GetEntity(ctx context.Context, id string) (*Entity, error) {
	var entity Entity
	if err := c.get(ctx, "/v1/entities/"+url.PathEscape(id), nil, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

// UpdateEntity patches an entity.
func (c *Client) UpdateEntity(ctx context.Context, id string, input *UpdateEntityInput) (*Entity, error) {
	var entity Entity
	if err := c.patch(ctx, "/v1/entities/"+url.PathEscape(id), input, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

// DeleteEntity deletes an entity.
func (c *Client) DeleteEntity(ctx context.Context, id string, mode string) error {
	params := url.Values{}
	if mode != "" {
		params.Set("mode", mode)
	}
	return c.del(ctx, "/v1/entities/"+url.PathEscape(id), params)
}

// GetEntityState returns the entity's state (with facts) at a point in time.
func (c *Client) GetEntityState(ctx context.Context, id string, asOf *time.Time) (*EntityState, error) {
	params := url.Values{}
	if asOf != nil {
		params.Set("as_of", asOf.Format(time.RFC3339))
	}

	var state EntityState
	if err := c.get(ctx, "/v1/entities/"+url.PathEscape(id)+"/state", params, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// GetEntityTimeline returns the entity's change history.
func (c *Client) GetEntityTimeline(ctx context.Context, id string, from, to *time.Time) (*EntityTimeline, error) {
	params := url.Values{}
	if from != nil {
		params.Set("from", from.Format(time.RFC3339))
	}
	if to != nil {
		params.Set("to", to.Format(time.RFC3339))
	}

	var timeline EntityTimeline
	if err := c.get(ctx, "/v1/entities/"+url.PathEscape(id)+"/timeline", params, &timeline); err != nil {
		return nil, err
	}
	return &timeline, nil
}

// ---- fact operations -------------------------------------------------------

// ListFacts lists facts with optional filters.
func (c *Client) ListFacts(ctx context.Context, opts *ListFactsOpts) (*Page[Fact], error) {
	params := url.Values{}
	if opts != nil {
		if opts.UserID != "" {
			params.Set("user_id", opts.UserID)
		}
		if opts.SubjectID != "" {
			params.Set("subject_id", opts.SubjectID)
		}
		if opts.ObjectID != "" {
			params.Set("object_id", opts.ObjectID)
		}
		if opts.Predicate != "" {
			params.Set("predicate", opts.Predicate)
		}
		if opts.ValidAt != nil {
			params.Set("valid_at", opts.ValidAt.Format(time.RFC3339))
		}
		if opts.IncludeInvalid {
			params.Set("include_invalid", "true")
		}
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
	}

	var page Page[Fact]
	if err := c.get(ctx, "/v1/facts", params, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// CreateFact creates a new fact.
func (c *Client) CreateFact(ctx context.Context, input *CreateFactInput) (*Fact, error) {
	var fact Fact
	if err := c.post(ctx, "/v1/facts", input, &fact); err != nil {
		return nil, err
	}
	return &fact, nil
}

// GetFact retrieves a fact by ID.
func (c *Client) GetFact(ctx context.Context, id string) (*Fact, error) {
	var fact Fact
	if err := c.get(ctx, "/v1/facts/"+url.PathEscape(id), nil, &fact); err != nil {
		return nil, err
	}
	return &fact, nil
}

// InvalidateFact marks a fact as no longer true.
func (c *Client) InvalidateFact(ctx context.Context, id string, input *InvalidateFactInput) error {
	if input == nil {
		input = &InvalidateFactInput{}
	}
	return c.postNoResponse(ctx, "/v1/facts/"+url.PathEscape(id)+"/invalidate", input)
}

// DeleteFact deletes a fact.
func (c *Client) DeleteFact(ctx context.Context, id string, mode string) error {
	params := url.Values{}
	if mode != "" {
		params.Set("mode", mode)
	}
	return c.del(ctx, "/v1/facts/"+url.PathEscape(id), params)
}

// ---- session operations ----------------------------------------------------

// CreateSession creates a new conversation session.
func (c *Client) CreateSession(ctx context.Context, input *CreateSessionInput) (*Session, error) {
	var sess Session
	if err := c.post(ctx, "/v1/sessions", input, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// ListSessions lists sessions with optional filters.
func (c *Client) ListSessions(ctx context.Context, opts *ListSessionsOpts) (*Page[Session], error) {
	params := url.Values{}
	if opts != nil {
		if opts.UserID != "" {
			params.Set("user_id", opts.UserID)
		}
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
	}

	var page Page[Session]
	if err := c.get(ctx, "/v1/sessions", params, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// GetSession retrieves a session by ID.
func (c *Client) GetSession(ctx context.Context, id string) (*Session, error) {
	var sess Session
	if err := c.get(ctx, "/v1/sessions/"+url.PathEscape(id), nil, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// CloseSession closes a session.
func (c *Client) CloseSession(ctx context.Context, id string) error {
	return c.postNoResponse(ctx, "/v1/sessions/"+url.PathEscape(id)+"/close", nil)
}

// ListSessionEpisodes lists episodes within a session.
func (c *Client) ListSessionEpisodes(ctx context.Context, sessionID string, opts *ListEpisodesOpts) (*Page[Episode], error) {
	params := url.Values{}
	if opts != nil {
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
	}

	var page Page[Episode]
	if err := c.get(ctx, "/v1/sessions/"+url.PathEscape(sessionID)+"/episodes", params, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// ---- HTTP transport --------------------------------------------------------

// APIError is returned when the server responds with an error.
type APIError struct {
	StatusCode int
	Type       string // RFC 7807 problem type URI
	Title      string // RFC 7807 short summary
	Message    string // human-readable detail
	Instance   string // request ID for debugging
}

func (e *APIError) Error() string {
	if e.Instance != "" {
		return fmt.Sprintf("zuzoto: HTTP %d: %s (instance: %s)", e.StatusCode, e.Message, e.Instance)
	}
	return fmt.Sprintf("zuzoto: HTTP %d: %s", e.StatusCode, e.Message)
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("User-Agent", "zuzoto-go/0.1.0")
	return c.httpClient.Do(req)
}

func (c *Client) get(ctx context.Context, path string, params url.Values, dst any) error {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.decodeResponse(resp, dst)
}

func (c *Client) post(ctx context.Context, path string, body, dst any) error {
	resp, err := c.sendJSON(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.decodeResponse(resp, dst)
}

func (c *Client) postNoResponse(ctx context.Context, path string, body any) error {
	resp, err := c.sendJSON(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return c.readError(resp)
	}
	return nil
}

func (c *Client) patch(ctx context.Context, path string, body, dst any) error {
	resp, err := c.sendJSON(ctx, http.MethodPatch, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.decodeResponse(resp, dst)
}

func (c *Client) del(ctx context.Context, path string, params url.Values) error {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return c.readError(resp)
	}
	return nil
}

func (c *Client) sendJSON(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("zuzoto: encode request: %w", err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) decodeResponse(resp *http.Response, dst any) error {
	if resp.StatusCode >= 400 {
		return c.readError(resp)
	}
	if dst == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func (c *Client) readError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	// Try RFC 7807 ProblemDetail format first.
	var problem struct {
		Type     string `json:"type"`
		Title    string `json:"title"`
		Detail   string `json:"detail"`
		Instance string `json:"instance"`
	}
	if json.Unmarshal(body, &problem) == nil && problem.Type != "" {
		msg := problem.Detail
		if msg == "" {
			msg = problem.Title
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Type:       problem.Type,
			Title:      problem.Title,
			Message:    msg,
			Instance:   problem.Instance,
		}
	}

	// Fallback: try {"error": "..."} format.
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		return &APIError{StatusCode: resp.StatusCode, Message: errResp.Error}
	}

	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	return &APIError{StatusCode: resp.StatusCode, Message: msg}
}
