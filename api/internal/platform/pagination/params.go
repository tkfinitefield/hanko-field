package pagination

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	// DefaultPageSize defines the fallback number of items returned when the client omits pageSize.
	DefaultPageSize = 50
	// DefaultMaxPageSize caps the supported pageSize to prevent unbounded queries.
	DefaultMaxPageSize = 100

	maxFilterValueLength = 512
)

// Operator enumerates supported filter operators accepted via the query string.
type Operator string

const (
	OperatorEqual         Operator = "=="
	OperatorGreaterThan   Operator = ">"
	OperatorLessThan      Operator = "<"
	OperatorGreaterEqual  Operator = ">="
	OperatorLessEqual     Operator = "<="
	OperatorArrayContains Operator = "array-contains"
)

var (
	supportedOperators = map[Operator]struct{}{
		OperatorEqual:         {},
		OperatorGreaterThan:   {},
		OperatorLessThan:      {},
		OperatorGreaterEqual:  {},
		OperatorLessEqual:     {},
		OperatorArrayContains: {},
	}

	operatorPriority = []Operator{
		OperatorArrayContains,
		OperatorGreaterEqual,
		OperatorLessEqual,
		OperatorEqual,
		OperatorGreaterThan,
		OperatorLessThan,
	}
)

// Order describes a single order-by clause.
type Order struct {
	Field string
	Desc  bool
}

// Filter captures an individual filter predicate parsed from the query string.
type Filter struct {
	Field string
	Op    Operator
	Value string
}

// Cursor represents the Firestore pagination cursor payload.
type Cursor struct {
	StartAfter []any `json:"startAfter,omitempty"`
	StartAt    []any `json:"startAt,omitempty"`
}

// Params bundles pagination, sorting, and filtering values extracted from a request.
type Params struct {
	PageSize  int
	PageToken string
	Cursor    Cursor
	Orders    []Order
	Filters   []Filter
}

// Options control how Parse behaves for a given handler layer.
type Options struct {
	DefaultPageSize     int
	MaxPageSize         int
	AllowedOrderFields  []string
	AllowedFilterFields map[string][]Operator
}

var (
	ErrInvalidPageSize  = errors.New("pagination: invalid pageSize")
	ErrInvalidOrderBy   = errors.New("pagination: invalid orderBy")
	ErrInvalidFilter    = errors.New("pagination: invalid filter")
	ErrInvalidPageToken = errors.New("pagination: invalid pageToken")
)

// FromRequest parses the supported query parameters from the supplied request.
func FromRequest(r *http.Request, opts Options) (Params, error) {
	if r == nil {
		return Params{}, errors.New("pagination: nil request")
	}
	return Parse(r.URL.Query(), opts)
}

// Parse consumes the provided query values and returns the normalised Params representation.
func Parse(values url.Values, opts Options) (Params, error) {
	if values == nil {
		values = url.Values{}
	}

	pageSize, err := parsePageSize(values.Get("pageSize"), opts)
	if err != nil {
		return Params{}, err
	}

	params := Params{PageSize: pageSize}

	rawToken := strings.TrimSpace(values.Get("pageToken"))
	if rawToken != "" {
		cursor, err := DecodeToken(rawToken)
		if err != nil {
			return Params{}, err
		}
		params.PageToken = rawToken
		params.Cursor = cursor
	}

	orders, err := parseOrder(values["orderBy"], opts.AllowedOrderFields)
	if err != nil {
		return Params{}, err
	}
	params.Orders = orders

	filters, err := parseFilters(values["filter"], opts.AllowedFilterFields)
	if err != nil {
		return Params{}, err
	}
	params.Filters = filters

	return params, nil
}

func parsePageSize(raw string, opts Options) (int, error) {
	maxPageSize := opts.MaxPageSize
	if maxPageSize <= 0 {
		maxPageSize = DefaultMaxPageSize
	}

	defaultPageSize := opts.DefaultPageSize
	if defaultPageSize <= 0 {
		defaultPageSize = DefaultPageSize
	}
	if defaultPageSize > maxPageSize {
		defaultPageSize = maxPageSize
	}

	if strings.TrimSpace(raw) == "" {
		return defaultPageSize, nil
	}

	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%w: must be an integer", ErrInvalidPageSize)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%w: must be greater than zero", ErrInvalidPageSize)
	}
	if value > maxPageSize {
		value = maxPageSize
	}
	return value, nil
}

func parseOrder(values []string, allowed []string) ([]Order, error) {
	if len(values) == 0 {
		return nil, nil
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("%w: ordering not supported", ErrInvalidOrderBy)
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, field := range allowed {
		if field == "" {
			continue
		}
		allowedSet[field] = struct{}{}
	}

	seen := make(map[string]struct{})
	var orders []Order

	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			field, desc, err := parseSingleOrder(part)
			if err != nil {
				return nil, err
			}
			if _, ok := allowedSet[field]; !ok {
				return nil, fmt.Errorf("%w: field %q is not allowed", ErrInvalidOrderBy, field)
			}
			key := field + ":asc"
			if desc {
				key = field + ":desc"
			}
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			orders = append(orders, Order{Field: field, Desc: desc})
		}
	}

	return orders, nil
}

func parseSingleOrder(part string) (string, bool, error) {
	part = strings.TrimSpace(part)
	if part == "" {
		return "", false, fmt.Errorf("%w: empty orderBy value", ErrInvalidOrderBy)
	}

	if strings.Contains(part, ":") && !strings.Contains(part, " ") {
		part = strings.ReplaceAll(part, ":", " ")
	}

	segments := strings.Fields(part)
	if len(segments) == 0 {
		return "", false, fmt.Errorf("%w: empty orderBy value", ErrInvalidOrderBy)
	}
	if len(segments) > 2 {
		return "", false, fmt.Errorf("%w: invalid orderBy format %q", ErrInvalidOrderBy, part)
	}

	field := segments[0]
	if !isAllowedFieldName(field) {
		return "", false, fmt.Errorf("%w: invalid field %q", ErrInvalidOrderBy, field)
	}

	desc := false
	if len(segments) == 2 {
		switch strings.ToLower(segments[1]) {
		case "asc":
			desc = false
		case "desc":
			desc = true
		default:
			return "", false, fmt.Errorf("%w: invalid direction %q", ErrInvalidOrderBy, segments[1])
		}
	}

	return field, desc, nil
}

func parseFilters(values []string, allowed map[string][]Operator) ([]Filter, error) {
	if len(values) == 0 {
		return nil, nil
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("%w: filtering not supported", ErrInvalidFilter)
	}

	allowedConfig := make(map[string]map[Operator]struct{}, len(allowed))
	for field, ops := range allowed {
		if !isAllowedFieldName(field) {
			continue
		}
		var opSet map[Operator]struct{}
		if len(ops) == 0 {
			opSet = cloneOperatorSet(supportedOperators)
		} else {
			opSet = make(map[Operator]struct{}, len(ops))
			for _, op := range ops {
				if _, ok := supportedOperators[op]; ok {
					opSet[op] = struct{}{}
				}
			}
			if len(opSet) == 0 {
				opSet = cloneOperatorSet(supportedOperators)
			}
		}
		allowedConfig[field] = opSet
	}

	if len(allowedConfig) == 0 {
		return nil, fmt.Errorf("%w: filtering not supported", ErrInvalidFilter)
	}

	filters := make([]Filter, 0, len(values))
	for _, raw := range values {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		filter, err := parseSingleFilter(raw)
		if err != nil {
			return nil, err
		}
		allowedOps, ok := allowedConfig[filter.Field]
		if !ok {
			return nil, fmt.Errorf("%w: field %q is not allowed", ErrInvalidFilter, filter.Field)
		}
		if _, ok := allowedOps[filter.Op]; !ok {
			return nil, fmt.Errorf("%w: operator %q is not allowed for field %q", ErrInvalidFilter, filter.Op, filter.Field)
		}
		filters = append(filters, filter)
	}

	return filters, nil
}

func parseSingleFilter(raw string) (Filter, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Filter{}, fmt.Errorf("%w: empty filter value", ErrInvalidFilter)
	}

	field, op, value, err := splitFilter(raw)
	if err != nil {
		return Filter{}, err
	}
	if !isAllowedFieldName(field) {
		return Filter{}, fmt.Errorf("%w: invalid field %q", ErrInvalidFilter, field)
	}
	if _, ok := supportedOperators[op]; !ok {
		return Filter{}, fmt.Errorf("%w: unsupported operator %q", ErrInvalidFilter, op)
	}

	value = sanitizeFilterValue(value)
	if value == "" {
		return Filter{}, fmt.Errorf("%w: empty value for field %q", ErrInvalidFilter, field)
	}

	return Filter{Field: field, Op: op, Value: value}, nil
}

func splitFilter(raw string) (string, Operator, string, error) {
	for _, candidate := range operatorPriority {
		token := string(candidate)
		idx := strings.Index(raw, token)
		if idx <= 0 {
			continue
		}
		field := strings.TrimSpace(raw[:idx])
		value := strings.TrimSpace(raw[idx+len(token):])
		if field == "" || value == "" {
			continue
		}
		return field, candidate, value, nil
	}
	return "", "", "", fmt.Errorf("%w: missing operator in %q", ErrInvalidFilter, raw)
}

func sanitizeFilterValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 0 {
		return ""
	}
	value = strings.Trim(value, "\"'")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.TrimSpace(value)
	if len(value) > maxFilterValueLength {
		value = value[:maxFilterValueLength]
	}
	return value
}

func cloneOperatorSet(src map[Operator]struct{}) map[Operator]struct{} {
	dst := make(map[Operator]struct{}, len(src))
	for op := range src {
		dst[op] = struct{}{}
	}
	return dst
}

func isAllowedFieldName(field string) bool {
	if field == "" {
		return false
	}
	for _, r := range field {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '.':
		default:
			return false
		}
	}
	return true
}

// Must ensures PageSize is always initialised with a sensible default before use.
func Must(params Params) Params {
	if params.PageSize <= 0 {
		params.PageSize = DefaultPageSize
	}
	return params
}
