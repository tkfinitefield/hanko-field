package domain

import "time"

// NameMappingStatus represents the lifecycle state for a transliterated name mapping.
type NameMappingStatus string

const (
	// NameMappingStatusPending indicates the mapping is awaiting candidate generation.
	NameMappingStatusPending NameMappingStatus = "pending"
	// NameMappingStatusReady indicates the mapping has candidate options ready for selection.
	NameMappingStatusReady NameMappingStatus = "ready"
	// NameMappingStatusSelected indicates the user has chosen a final candidate.
	NameMappingStatusSelected NameMappingStatus = "selected"
	// NameMappingStatusExpired indicates the mapping can no longer be used without regeneration.
	NameMappingStatusExpired NameMappingStatus = "expired"
)

// NameMappingInput captures the parameters supplied to generate a name mapping.
type NameMappingInput struct {
	Latin   string
	Locale  string
	Gender  string
	Context map[string]string
}

// NameMappingCandidate represents a single transliteration candidate with scoring metadata.
type NameMappingCandidate struct {
	ID       string
	Kanji    string
	Kana     []string
	Score    float64
	Notes    string
	Metadata map[string]any
}

// NameMapping stores transliteration candidates, selection state, and audit metadata.
type NameMapping struct {
	ID                string
	UserID            string
	UserRef           string
	Input             NameMappingInput
	Status            NameMappingStatus
	Candidates        []NameMappingCandidate
	SelectedCandidate *NameMappingCandidate
	SelectedAt        *time.Time
	Source            string
	ExpiresAt         *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Metadata          map[string]any
}
