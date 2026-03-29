// Package zuzoto provides a Go client for the Zuzoto memory API.
//
// Zuzoto is cognitive memory infrastructure for AI agents — hierarchical,
// temporal, self-consolidating.
//
// Quick start:
//
//	client := zuzoto.NewClient("http://localhost:8080",
//	    zuzoto.WithAPIKey("your-api-key"),
//	)
//	result, err := client.Add(ctx, &zuzoto.AddInput{
//	    Content: "User prefers dark mode",
//	    UserID:  "user-123",
//	})
package zuzoto

import "time"

// ---- core types ------------------------------------------------------------

// Memory is the fundamental unit of storage.
type Memory struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // "episodic", "semantic", "procedural"
	Content     string            `json:"content"`
	Embedding   []float32         `json:"embedding,omitempty"`
	Scope       Scope             `json:"scope"`
	Temporal    TemporalMetadata  `json:"temporal"`
	Provenance  Provenance        `json:"provenance"`
	Strength    float64           `json:"strength"`
	AccessCount int64             `json:"access_count"`
	LastAccess  time.Time         `json:"last_access"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
}

// ScoredMemory pairs a memory with its relevance score.
type ScoredMemory struct {
	Memory Memory  `json:"memory"`
	Score  float64 `json:"score"`
}

// Scope defines the multi-tenant hierarchy for a memory.
type Scope struct {
	OrgID     string `json:"org_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
}

// TemporalMetadata implements bi-temporal versioning.
type TemporalMetadata struct {
	TValid     time.Time  `json:"t_valid"`
	TInvalid   *time.Time `json:"t_invalid,omitempty"`
	TCreated   time.Time  `json:"t_created"`
	TModified  time.Time  `json:"t_modified"`
	Confidence float64    `json:"confidence"`
	Source     string     `json:"source"`
}

// Provenance tracks the origin and lineage of a memory.
type Provenance struct {
	SourceMemoryIDs []string `json:"source_memory_ids,omitempty"`
	CreatedBy       string   `json:"created_by,omitempty"`
	ConsolidationID string   `json:"consolidation_id,omitempty"`
	Version         int      `json:"version"`
}

// Entity is a node in the knowledge graph.
type Entity struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Aliases      []string       `json:"aliases,omitempty"`
	EntityType   string         `json:"entity_type"`
	Properties   map[string]any `json:"properties,omitempty"`
	Embedding    []float32      `json:"embedding,omitempty"`
	Scope        Scope          `json:"scope"`
	FirstSeen    time.Time      `json:"first_seen"`
	LastSeen     time.Time      `json:"last_seen"`
	MentionCount int64          `json:"mention_count"`
}

// Fact is a semantic knowledge graph edge.
type Fact struct {
	ID           string           `json:"id"`
	Type         string           `json:"type"`
	Content      string           `json:"content"`
	Subject      string           `json:"subject"`
	Predicate    string           `json:"predicate"`
	Object       string           `json:"object"`
	Negation     bool             `json:"negation"`
	SupersededBy *string          `json:"superseded_by,omitempty"`
	Scope        Scope            `json:"scope"`
	Temporal     TemporalMetadata `json:"temporal"`
	Provenance   Provenance       `json:"provenance"`
	Strength     float64          `json:"strength"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
}

// Episode is an episodic memory — a specific event or interaction.
type Episode struct {
	ID           string           `json:"id"`
	Type         string           `json:"type"`
	Content      string           `json:"content"`
	Participants []string         `json:"participants,omitempty"`
	Location     string           `json:"location,omitempty"`
	EventType    string           `json:"event_type,omitempty"`
	Sequence     int64            `json:"sequence"`
	Scope        Scope            `json:"scope"`
	Temporal     TemporalMetadata `json:"temporal"`
}

// Session represents a conversation session.
type Session struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	AgentID   string         `json:"agent_id,omitempty"`
	Status    string         `json:"status"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	ClosedAt  *time.Time     `json:"closed_at,omitempty"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty"`
}

// StateChange represents a change in entity state.
type StateChange struct {
	OldFact *Fact     `json:"old_fact"`
	NewFact *Fact     `json:"new_fact"`
	At      time.Time `json:"at"`
}

// ---- request types ---------------------------------------------------------

// AddInput is the input to Client.Add.
type AddInput struct {
	Content   string         `json:"content,omitempty"`
	Messages  []Message      `json:"messages,omitempty"`
	UserID    string         `json:"user_id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Message represents a conversation message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SearchQuery configures a hybrid search.
type SearchQuery struct {
	Text       string   `json:"text"`
	UserID     string   `json:"-"` // convenience: set on scope
	Scope      *Scope   `json:"scope,omitempty"`
	Strategies []string `json:"strategies,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	MinScore   float64  `json:"min_score,omitempty"`
	From       *time.Time `json:"-"` // convenience: set on temporal
	To         *time.Time `json:"-"` // convenience: set on temporal
}

// ContextQuery configures context window assembly.
type ContextQuery struct {
	Query     string `json:"-"` // set as query param
	UserID    string `json:"-"`
	SessionID string `json:"-"`
	AgentID   string `json:"-"`
	MaxTokens int    `json:"-"`
}

// ForgetInput specifies what to forget.
type ForgetInput struct {
	MemoryID string `json:"memory_id,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Mode     string `json:"mode,omitempty"` // "soft", "hard", "gdpr"
}

// CreateEntityInput is input for creating an entity.
type CreateEntityInput struct {
	Name       string         `json:"name"`
	EntityType string         `json:"entity_type"`
	Aliases    []string       `json:"aliases,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	UserID     string         `json:"user_id,omitempty"`
}

// UpdateEntityInput is input for updating an entity.
type UpdateEntityInput struct {
	Name       *string        `json:"name,omitempty"`
	EntityType *string        `json:"entity_type,omitempty"`
	Aliases    []string       `json:"aliases,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

// CreateFactInput is input for creating a fact.
type CreateFactInput struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	Negation  bool   `json:"negation,omitempty"`
	UserID    string `json:"user_id,omitempty"`
}

// InvalidateFactInput is input for invalidating a fact.
type InvalidateFactInput struct {
	SupersededBy *string `json:"superseded_by,omitempty"`
}

// CreateSessionInput is input for creating a session.
type CreateSessionInput struct {
	UserID    string         `json:"user_id"`
	AgentID   string         `json:"agent_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	ExpiresIn string         `json:"expires_in,omitempty"`
}

// UpdateMemoryInput is input for updating a memory.
type UpdateMemoryInput struct {
	Content  *string        `json:"content,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Tags     []string       `json:"tags,omitempty"`
}

// ---- response types --------------------------------------------------------

// AddResult is returned from Client.Add.
type AddResult struct {
	Memories         []Memory `json:"memories"`
	EntitiesCreated  int      `json:"entities_created"`
	FactsCreated     int      `json:"facts_created"`
	FactsInvalidated int      `json:"facts_invalidated"`
	ProcessingMs     int64    `json:"processing_ms"`
}

// SearchResult contains scored memories from a search.
type SearchResult struct {
	Memories []ScoredMemory `json:"memories"`
	Total    int            `json:"total"`
}

// ContextWindow is an assembled context for an LLM.
type ContextWindow struct {
	Memories []ScoredMemory `json:"memories"`
	Facts    []Fact         `json:"facts"`
	Summary  string         `json:"summary"`
	Tokens   int            `json:"tokens"`
}

// BatchAddResult is returned from Client.BatchAdd.
type BatchAddResult struct {
	Results []AddResult `json:"results"`
	Total   int         `json:"total"`
	Errors  int         `json:"errors"`
}

// EntityState represents the state of an entity at a point in time.
type EntityState struct {
	Entity Entity    `json:"entity"`
	Facts  []Fact    `json:"facts"`
	AsOf   time.Time `json:"as_of"`
}

// EntityTimeline is a list of state changes.
type EntityTimeline struct {
	Changes []StateChange `json:"changes"`
}

// Page is a generic paginated result.
type Page[T any] struct {
	Items   []T    `json:"items"`
	Cursor  string `json:"cursor,omitempty"`
	HasMore bool   `json:"has_more"`
}

// ---- list options ----------------------------------------------------------

// ListEntitiesOpts configures entity listing.
type ListEntitiesOpts struct {
	UserID     string
	Type       string
	NamePrefix string
	Cursor     string
	Limit      int
}

// ListFactsOpts configures fact listing.
type ListFactsOpts struct {
	SubjectID      string
	ObjectID       string
	Predicate      string
	ValidAt        *time.Time
	IncludeInvalid bool
	Cursor         string
	Limit          int
}

// ListSessionsOpts configures session listing.
type ListSessionsOpts struct {
	UserID string
	Status string
	Cursor string
	Limit  int
}

// ListEpisodesOpts configures episode listing.
type ListEpisodesOpts struct {
	Cursor string
	Limit  int
}

// ---- delete mode -----------------------------------------------------------

const (
	DeleteModeSoft = "soft"
	DeleteModeHard = "hard"
	DeleteModeGDPR = "gdpr"
)
