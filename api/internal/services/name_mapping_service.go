package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/textutil"
	"github.com/hanko-field/api/internal/repositories"
)

var (
	errNameMappingInvalidInput       = errors.New("name_mapping: invalid input")
	errNameMappingUnsupportedLocale  = errors.New("name_mapping: unsupported locale")
	errNameMappingRepositoryRequired = errors.New("name_mapping: repository is required")
	errNameMappingClockRequired      = errors.New("name_mapping: clock is required")
)

// ErrNameMappingInvalidInput indicates the caller provided invalid data.
var ErrNameMappingInvalidInput = errNameMappingInvalidInput

// ErrNameMappingUnsupportedLocale indicates the requested locale is not supported by the underlying provider.
var ErrNameMappingUnsupportedLocale = errNameMappingUnsupportedLocale

// ErrNameMappingUnavailable indicates the service cannot complete the request due to missing dependencies.
var ErrNameMappingUnavailable = errors.New("name_mapping: service unavailable")

// ErrNameMappingNotFound indicates the requested mapping does not exist.
var ErrNameMappingNotFound = errors.New("name_mapping: not found")

// ErrNameMappingUnauthorized indicates the mapping does not belong to the caller.
var ErrNameMappingUnauthorized = errors.New("name_mapping: unauthorized")

// ErrNameMappingConflict indicates the requested operation conflicts with existing state.
var ErrNameMappingConflict = errors.New("name_mapping: conflict")

const (
	defaultNameMappingCacheTTL = 24 * time.Hour
	defaultNameMappingSource   = "hanko-fallback"
	maxNameMappingLatinLength  = 120
	maxNameMappingContextKeys  = 16
	maxNameMappingCandidates   = 20
	nameMappingIDPrefix        = "nmap_"
)

var supportedGenders = map[string]struct{}{
	"male":    {},
	"female":  {},
	"neutral": {},
	"":        {},
}

// TransliterationProvider describes the dependency capable of producing kanji mapping candidates.
type TransliterationProvider interface {
	Transliterate(ctx context.Context, req TransliterationRequest) (TransliterationResult, error)
}

// TransliterationRequest encapsulates the parameters sent to the transliteration service.
type TransliterationRequest struct {
	Latin   string
	Locale  string
	Gender  string
	Context map[string]string
}

// TransliterationCandidate represents a single transliteration result from the provider.
type TransliterationCandidate struct {
	ID       string
	Kanji    string
	Kana     []string
	Score    float64
	Notes    string
	Metadata map[string]any
}

// TransliterationResult captures the provider outputs and provenance metadata.
type TransliterationResult struct {
	Provider   string
	Candidates []TransliterationCandidate
	Metadata   map[string]any
}

// ErrTransliterationUnsupportedLocale signals a provider cannot serve the specified locale.
var ErrTransliterationUnsupportedLocale = errors.New("transliteration: unsupported locale")

// ErrTransliterationUnavailable indicates the provider could not complete the request due to dependency issues.
var ErrTransliterationUnavailable = errors.New("transliteration: unavailable")

// NameMappingServiceDeps wires the repository and transliteration dependencies for name mapping operations.
type NameMappingServiceDeps struct {
	Repository     repositories.NameMappingRepository
	Users          repositories.UserRepository
	Transliterator TransliterationProvider
	Clock          func() time.Time
	IDGenerator    func() string
	Logger         func(context.Context, string, map[string]any)
	CacheTTL       time.Duration
}

type nameMappingService struct {
	repo     repositories.NameMappingRepository
	profiles repositories.UserRepository
	provider TransliterationProvider
	now      func() time.Time
	newID    func() string
	logger   func(context.Context, string, map[string]any)
	cacheTTL time.Duration
	fallback TransliterationProvider
}

// NewNameMappingService constructs a NameMappingService with the provided dependencies.
func NewNameMappingService(deps NameMappingServiceDeps) (NameMappingService, error) {
	if deps.Repository == nil {
		return nil, errNameMappingRepositoryRequired
	}

	clock := deps.Clock
	if clock == nil {
		return nil, errNameMappingClockRequired
	}

	idGen := deps.IDGenerator
	if idGen == nil {
		idGen = func() string { return ulid.Make().String() }
	}

	cacheTTL := deps.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultNameMappingCacheTTL
	}

	logger := deps.Logger
	if logger == nil {
		logger = func(context.Context, string, map[string]any) {}
	}

	return &nameMappingService{
		repo:     deps.Repository,
		profiles: deps.Users,
		provider: deps.Transliterator,
		now:      func() time.Time { return clock().UTC() },
		newID:    func() string { return nameMappingIDPrefix + strings.ToLower(idGen()) },
		logger:   logger,
		cacheTTL: cacheTTL,
		fallback: &heuristicTransliterator{},
	}, nil
}

// ConvertName generates or retrieves kanji candidates for the requested latin name.
func (s *nameMappingService) ConvertName(ctx context.Context, cmd NameConversionCommand) (NameMapping, error) {
	if s == nil || s.repo == nil {
		return NameMapping{}, ErrNameMappingUnavailable
	}

	userID := strings.TrimSpace(cmd.UserID)
	if userID == "" {
		return NameMapping{}, ErrNameMappingInvalidInput
	}

	latin := strings.TrimSpace(cmd.Latin)
	if latin == "" {
		return NameMapping{}, ErrNameMappingInvalidInput
	}
	if len([]rune(latin)) > maxNameMappingLatinLength {
		return NameMapping{}, ErrNameMappingInvalidInput
	}

	locale := strings.TrimSpace(cmd.Locale)
	if locale == "" {
		locale = "en"
	}
	locale = strings.ToLower(locale)

	gender := strings.TrimSpace(strings.ToLower(cmd.Gender))
	if _, ok := supportedGenders[gender]; !ok {
		return NameMapping{}, ErrNameMappingInvalidInput
	}

	contextMap := textutil.NormalizeStringMap(cmd.Context)
	if len(contextMap) > maxNameMappingContextKeys {
		return NameMapping{}, ErrNameMappingInvalidInput
	}

	now := s.now()

	var existing *domain.NameMapping
	if found, err := s.repo.FindByLookup(ctx, userID, latin, locale); err == nil {
		if !cmd.ForceRefresh && !mappingExpired(found, now) {
			return found, nil
		}
		existing = &found
	} else if !isRepoNotFound(err) {
		return NameMapping{}, s.translateRepoError(err)
	}

	req := TransliterationRequest{
		Latin:   latin,
		Locale:  locale,
		Gender:  gender,
		Context: contextMap,
	}
	result, usedFallback, err := s.performTransliteration(ctx, req)
	if err != nil {
		return NameMapping{}, err
	}
	if len(result.Candidates) == 0 {
		return NameMapping{}, ErrNameMappingUnavailable
	}

	// Ensure candidates are ordered by descending score.
	sort.Slice(result.Candidates, func(i, j int) bool {
		return result.Candidates[i].Score > result.Candidates[j].Score
	})

	if len(result.Candidates) > maxNameMappingCandidates {
		result.Candidates = result.Candidates[:maxNameMappingCandidates]
	}

	mapping := domain.NameMapping{
		Status:     domain.NameMappingStatusReady,
		Input:      domain.NameMappingInput{Latin: latin, Locale: locale, Gender: gender, Context: cloneStringMap(contextMap)},
		Candidates: make([]domain.NameMappingCandidate, 0, len(result.Candidates)),
		Metadata:   cloneMetadataMap(result.Metadata),
		UpdatedAt:  now,
	}

	if existing != nil {
		mapping.ID = existing.ID
		mapping.UserID = existing.UserID
		mapping.UserRef = existing.UserRef
		mapping.CreatedAt = existing.CreatedAt
	} else {
		mapping.ID = s.newID()
		mapping.UserID = userID
		mapping.UserRef = buildUserRef(userID)
		mapping.CreatedAt = now
	}

	expiry := now.Add(s.cacheTTL)
	mapping.ExpiresAt = &expiry
	source := result.Provider
	if strings.TrimSpace(source) == "" {
		if usedFallback {
			source = defaultNameMappingSource
		} else {
			source = "unknown"
		}
	}
	mapping.Source = source

	for idx, candidate := range result.Candidates {
		id := strings.TrimSpace(candidate.ID)
		if id == "" {
			id = fmt.Sprintf("%s_cand_%d", mapping.ID, idx+1)
		}
		score := candidate.Score
		if math.IsNaN(score) || math.IsInf(score, 0) {
			score = 0
		}
		if score < 0 {
			score = 0
		}
		mapping.Candidates = append(mapping.Candidates, domain.NameMappingCandidate{
			ID:       id,
			Kanji:    strings.TrimSpace(candidate.Kanji),
			Kana:     cloneSlice(candidate.Kana),
			Score:    score,
			Notes:    strings.TrimSpace(candidate.Notes),
			Metadata: cloneMetadataMap(candidate.Metadata),
		})
	}

	if existing == nil {
		if err := s.repo.Insert(ctx, mapping); err != nil {
			return NameMapping{}, s.translateRepoError(err)
		}
	} else {
		if err := s.repo.Update(ctx, mapping); err != nil {
			return NameMapping{}, s.translateRepoError(err)
		}
	}

	return mapping, nil
}

// SelectCandidate persists the chosen candidate, locking the mapping for future use.
func (s *nameMappingService) SelectCandidate(ctx context.Context, cmd NameMappingSelectCommand) (NameMapping, error) {
	if s == nil || s.repo == nil {
		return NameMapping{}, ErrNameMappingUnavailable
	}

	userID := strings.TrimSpace(cmd.UserID)
	mappingID := strings.TrimSpace(cmd.MappingID)
	candidateID := strings.TrimSpace(cmd.CandidateID)

	if userID == "" || mappingID == "" || candidateID == "" {
		return NameMapping{}, ErrNameMappingInvalidInput
	}

	mapping, err := s.repo.FindByID(ctx, mappingID)
	if err != nil {
		if isRepoNotFound(err) {
			return NameMapping{}, ErrNameMappingNotFound
		}
		return NameMapping{}, ErrNameMappingUnavailable
	}

	if strings.TrimSpace(mapping.UserID) != userID {
		return NameMapping{}, ErrNameMappingUnauthorized
	}

	switch mapping.Status {
	case domain.NameMappingStatusReady, domain.NameMappingStatusSelected:
		// allowed
	case domain.NameMappingStatusExpired:
		return NameMapping{}, ErrNameMappingInvalidInput
	default:
		return NameMapping{}, ErrNameMappingInvalidInput
	}

	var selectedCandidate *domain.NameMappingCandidate
	for _, cand := range mapping.Candidates {
		if strings.TrimSpace(cand.ID) == candidateID {
			copy := cand
			selectedCandidate = &copy
			break
		}
	}
	if selectedCandidate == nil {
		return NameMapping{}, ErrNameMappingInvalidInput
	}

	now := s.now()

	if mapping.Status == domain.NameMappingStatusSelected {
		if mapping.SelectedCandidate != nil && strings.TrimSpace(mapping.SelectedCandidate.ID) == candidateID {
			needsUpdate := false
			if mapping.SelectedAt == nil {
				selectionTime := now
				mapping.SelectedAt = &selectionTime
				needsUpdate = true
			}
			if mapping.ExpiresAt != nil {
				mapping.ExpiresAt = nil
				needsUpdate = true
			}
			if needsUpdate {
				mapping.UpdatedAt = now
				if err := s.repo.Update(ctx, mapping); err != nil {
					return NameMapping{}, s.translateSelectionRepoError(err)
				}
			}
			if err := s.storeSelectionOnProfile(ctx, userID, mapping.ID); err != nil {
				return NameMapping{}, err
			}
			return mapping, nil
		}
		if !cmd.AllowOverride {
			return NameMapping{}, ErrNameMappingConflict
		}
	}

	mapping.Status = domain.NameMappingStatusSelected
	mapping.SelectedCandidate = selectedCandidate
	selectionTime := now
	mapping.SelectedAt = &selectionTime
	mapping.ExpiresAt = nil
	mapping.UpdatedAt = now

	if err := s.repo.Update(ctx, mapping); err != nil {
		return NameMapping{}, s.translateSelectionRepoError(err)
	}

	if err := s.storeSelectionOnProfile(ctx, userID, mapping.ID); err != nil {
		return NameMapping{}, err
	}

	return mapping, nil
}

func (s *nameMappingService) translateSelectionRepoError(err error) error {
	if err == nil {
		return nil
	}
	if isRepoNotFound(err) {
		return ErrNameMappingNotFound
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		if repoErr.IsConflict() {
			return ErrNameMappingConflict
		}
		if repoErr.IsUnavailable() {
			return ErrNameMappingUnavailable
		}
	}
	return ErrNameMappingUnavailable
}

func (s *nameMappingService) storeSelectionOnProfile(ctx context.Context, userID, mappingID string) error {
	if s.profiles == nil {
		return nil
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrNameMappingUnavailable
	}
	profile, err := s.profiles.FindByID(ctx, userID)
	if err != nil {
		if isRepoNotFound(err) {
			return ErrNameMappingUnavailable
		}
		return ErrNameMappingUnavailable
	}

	ref := strings.TrimSpace(mappingID)
	current := ""
	if profile.NameMappingRef != nil {
		current = strings.TrimSpace(*profile.NameMappingRef)
	}
	if current == ref {
		return nil
	}

	if ref == "" {
		profile.NameMappingRef = nil
	} else {
		value := ref
		profile.NameMappingRef = &value
	}
	if profile.LastSyncTime.IsZero() {
		profile.LastSyncTime = profile.UpdatedAt
	}
	if _, err := s.profiles.UpdateProfile(ctx, profile); err != nil {
		return ErrNameMappingUnavailable
	}
	return nil
}

func (s *nameMappingService) performTransliteration(ctx context.Context, req TransliterationRequest) (TransliterationResult, bool, error) {
	if s.provider != nil {
		result, err := s.provider.Transliterate(ctx, req)
		if err == nil {
			return normaliseResult(result), false, nil
		}
		if errors.Is(err, ErrTransliterationUnsupportedLocale) {
			return TransliterationResult{}, false, ErrNameMappingUnsupportedLocale
		}
		if errors.Is(err, ErrTransliterationUnavailable) {
			s.logger(ctx, "name_mapping.transliterate_unavailable", map[string]any{
				"locale": req.Locale,
			})
		} else {
			s.logger(ctx, "name_mapping.transliterate_error", map[string]any{
				"locale": req.Locale,
				"error":  err.Error(),
			})
		}
	}

	fallbackResult, err := s.fallback.Transliterate(ctx, req)
	if err != nil {
		return TransliterationResult{}, true, ErrNameMappingUnavailable
	}
	return normaliseResult(fallbackResult), true, nil
}

func (s *nameMappingService) translateRepoError(err error) error {
	if err == nil {
		return nil
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		if repoErr.IsNotFound() {
			return ErrNameMappingUnavailable
		}
		return ErrNameMappingUnavailable
	}
	return ErrNameMappingUnavailable
}

func normaliseResult(result TransliterationResult) TransliterationResult {
	if result.Metadata == nil {
		result.Metadata = map[string]any{}
	}
	if result.Candidates == nil {
		result.Candidates = []TransliterationCandidate{}
	}
	return result
}

func mappingExpired(mapping domain.NameMapping, now time.Time) bool {
	if mapping.Status == domain.NameMappingStatusExpired {
		return true
	}
	if mapping.ExpiresAt == nil {
		return false
	}
	return mapping.ExpiresAt.Before(now)
}

func cloneSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func cloneMetadataMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func buildUserRef(userID string) string {
	return fmt.Sprintf("/users/%s", strings.TrimSpace(userID))
}

type heuristicTransliterator struct{}

func (h *heuristicTransliterator) Transliterate(_ context.Context, req TransliterationRequest) (TransliterationResult, error) {
	latin := strings.TrimSpace(req.Latin)
	if latin == "" {
		return TransliterationResult{}, ErrTransliterationUnavailable
	}
	base := toHeuristicKanji(latin)
	kana := toKatakana(latin)

	candidates := []TransliterationCandidate{
		{
			ID:    "fallback_primary",
			Kanji: base,
			Kana: []string{
				kana,
			},
			Score: 0.6,
			Notes: "heuristic mapping",
			Metadata: map[string]any{
				"strategy": "heuristic",
			},
		},
		{
			ID:    "fallback_alt",
			Kanji: reverseKanji(base),
			Kana: []string{
				kana,
			},
			Score: 0.45,
			Notes: "heuristic alternate",
			Metadata: map[string]any{
				"strategy": "heuristic_alt",
			},
		},
	}

	return TransliterationResult{
		Provider:   defaultNameMappingSource,
		Candidates: candidates,
		Metadata: map[string]any{
			"fallback": true,
		},
	}, nil
}

func toHeuristicKanji(input string) string {
	alphabet := map[rune]string{
		'a': "安", 'b': "武", 'c': "千", 'd': "大", 'e': "恵",
		'f': "富", 'g': "雅", 'h': "浜", 'i': "伊", 'j': "仁",
		'k': "佳", 'l': "良", 'm': "真", 'n': "名", 'o': "緒",
		'p': "平", 'q': "玖", 'r': "礼", 's': "志", 't': "智",
		'u': "宇", 'v': "美", 'w': "和", 'x': "希", 'y': "優", 'z': "善",
	}

	var builder strings.Builder
	runes := []rune(strings.ToLower(input))
	for _, r := range runes {
		if val, ok := alphabet[r]; ok {
			builder.WriteString(val)
		} else if r == ' ' || r == '-' || r == '\'' {
			builder.WriteString("・")
		} else {
			builder.WriteString("〇")
		}
	}
	return builder.String()
}

func toKatakana(input string) string {
	upper := strings.ToUpper(input)
	replacements := map[string]string{
		"A": "ア", "I": "イ", "U": "ウ", "E": "エ", "O": "オ",
		"K": "カ", "S": "サ", "T": "タ", "N": "ナ", "H": "ハ",
		"M": "マ", "Y": "ヤ", "R": "ラ", "W": "ワ",
	}

	var builder strings.Builder
	for _, r := range upper {
		if rep, ok := replacements[string(r)]; ok {
			builder.WriteString(rep)
			continue
		}
		switch r {
		case ' ':
			builder.WriteString("・")
		case '-':
			builder.WriteString("ー")
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func reverseKanji(input string) string {
	runes := []rune(input)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
