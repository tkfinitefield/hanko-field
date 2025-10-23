package search

import (
	"context"
	"errors"
	"time"
)

// ErrNotConfigured indicates the search service dependency has not been provided.
var ErrNotConfigured = errors.New("search service not configured")

// Service describes the contract for performing global search queries.
type Service interface {
	// Search executes the provided query and returns grouped results.
	Search(ctx context.Context, token string, query Query) (ResultSet, error)
}

// Entity represents a searchable resource category.
type Entity string

const (
	// EntityOrder captures order level results.
	EntityOrder Entity = "order"
	// EntityUser captures customer or account level results.
	EntityUser Entity = "user"
	// EntityReview captures review or feedback entries.
	EntityReview Entity = "review"
)

// Query represents incoming filter parameters for a search request.
type Query struct {
	Term    string
	Scope   []Entity
	Persona string
	Start   *time.Time
	End     *time.Time
	Limit   int
	Cursor  map[Entity]string
}

// ResultSet contains grouped search responses.
type ResultSet struct {
	Total    int
	Duration time.Duration
	Groups   []ResultGroup
}

// ResultGroup groups hits for a single entity type.
type ResultGroup struct {
	Entity     Entity
	Label      string
	Icon       string
	Total      int
	HasMore    bool
	NextCursor string
	Hits       []Hit
}

// Hit represents a single item in the search results.
type Hit struct {
	ID          string
	Entity      Entity
	Title       string
	Description string
	Badge       string
	BadgeTone   string
	URL         string
	Score       float64
	OccurredAt  *time.Time
	Persona     string
	Metadata    []Metadata
}

// Metadata captures supplementary key/value pairs for a hit.
type Metadata struct {
	Key   string
	Value string
	Icon  string
}
