package pagination

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestParseDefaults(t *testing.T) {
	params, err := Parse(url.Values{}, Options{})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if params.PageSize != DefaultPageSize {
		t.Fatalf("expected default page size %d got %d", DefaultPageSize, params.PageSize)
	}
	if params.PageToken != "" {
		t.Fatalf("expected empty page token got %q", params.PageToken)
	}
	if !reflect.DeepEqual(params.Cursor, Cursor{}) {
		t.Fatalf("expected zero cursor, got %#v", params.Cursor)
	}
	if params.Orders != nil {
		t.Fatalf("expected nil orders, got %#v", params.Orders)
	}
	if params.Filters != nil {
		t.Fatalf("expected nil filters, got %#v", params.Filters)
	}
}

func TestParsePageSize(t *testing.T) {
	opts := Options{DefaultPageSize: 25, MaxPageSize: 40}
	values := url.Values{}
	values.Set("pageSize", "30")

	params, err := Parse(values, opts)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if params.PageSize != 30 {
		t.Fatalf("expected page size 30 got %d", params.PageSize)
	}

	values.Set("pageSize", "400")
	params, err = Parse(values, opts)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if params.PageSize != opts.MaxPageSize {
		t.Fatalf("expected page size clamped to %d got %d", opts.MaxPageSize, params.PageSize)
	}
}

func TestParseInvalidPageSize(t *testing.T) {
	values := url.Values{}
	values.Set("pageSize", "abc")

	if _, err := Parse(values, Options{}); !errors.Is(err, ErrInvalidPageSize) {
		t.Fatalf("expected ErrInvalidPageSize got %v", err)
	}

	values.Set("pageSize", "0")
	if _, err := Parse(values, Options{}); !errors.Is(err, ErrInvalidPageSize) {
		t.Fatalf("expected ErrInvalidPageSize for zero got %v", err)
	}
}

func TestParsePageToken(t *testing.T) {
	cursor := Cursor{StartAfter: []any{"abc", 123}}
	token, err := EncodeToken(cursor)
	if err != nil {
		t.Fatalf("EncodeToken returned error: %v", err)
	}

	values := url.Values{}
	values.Set("pageToken", token)

	params, err := Parse(values, Options{})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if params.PageToken != token {
		t.Fatalf("expected page token %q got %q", token, params.PageToken)
	}
	if got := len(params.Cursor.StartAfter); got != len(cursor.StartAfter) {
		t.Fatalf("expected cursor length %d got %d", len(cursor.StartAfter), got)
	}
	if s, ok := params.Cursor.StartAfter[0].(string); !ok || s != "abc" {
		t.Fatalf("expected first cursor value %q got %#v", "abc", params.Cursor.StartAfter[0])
	}
	if fmt.Sprint(params.Cursor.StartAfter[1]) != "123" {
		t.Fatalf("expected numeric cursor value %q got %#v", "123", params.Cursor.StartAfter[1])
	}
}

func TestParseInvalidPageToken(t *testing.T) {
	values := url.Values{}
	values.Set("pageToken", "!!!invalid!!!")

	if _, err := Parse(values, Options{}); !errors.Is(err, ErrInvalidPageToken) {
		t.Fatalf("expected ErrInvalidPageToken got %v", err)
	}
}

func TestParseOrderBy(t *testing.T) {
	values := url.Values{}
	values.Add("orderBy", "createdAt desc")
	values.Add("orderBy", "updatedAt asc,score desc")

	opts := Options{AllowedOrderFields: []string{"createdAt", "updatedAt", "score"}}

	params, err := Parse(values, opts)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	expected := []Order{{Field: "createdAt", Desc: true}, {Field: "updatedAt", Desc: false}, {Field: "score", Desc: true}}
	if !reflect.DeepEqual(params.Orders, expected) {
		t.Fatalf("expected orders %#v got %#v", expected, params.Orders)
	}
}

func TestParseOrderByInvalid(t *testing.T) {
	values := url.Values{}
	values.Add("orderBy", "createdAt desc")

	if _, err := Parse(values, Options{}); !errors.Is(err, ErrInvalidOrderBy) {
		t.Fatalf("expected ErrInvalidOrderBy got %v", err)
	}

	values = url.Values{}
	values.Add("orderBy", "createdAt invalid")
	opts := Options{AllowedOrderFields: []string{"createdAt"}}
	if _, err := Parse(values, opts); !errors.Is(err, ErrInvalidOrderBy) {
		t.Fatalf("expected ErrInvalidOrderBy for direction got %v", err)
	}

	values = url.Values{}
	values.Add("orderBy", "unknown desc")
	if _, err := Parse(values, opts); !errors.Is(err, ErrInvalidOrderBy) {
		t.Fatalf("expected ErrInvalidOrderBy for field got %v", err)
	}
}

func TestParseFilters(t *testing.T) {
	values := url.Values{}
	values.Add("filter", "status == active")
	values.Add("filter", "score >= 10")
	values.Add("filter", "tags array-contains premium")

	opts := Options{AllowedFilterFields: map[string][]Operator{
		"status": {OperatorEqual},
		"score":  {OperatorGreaterEqual},
		"tags":   {OperatorArrayContains},
	}}

	params, err := Parse(values, opts)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	expected := []Filter{
		{Field: "status", Op: OperatorEqual, Value: "active"},
		{Field: "score", Op: OperatorGreaterEqual, Value: "10"},
		{Field: "tags", Op: OperatorArrayContains, Value: "premium"},
	}
	if !reflect.DeepEqual(params.Filters, expected) {
		t.Fatalf("expected filters %#v got %#v", expected, params.Filters)
	}
}

func TestParseFiltersInvalid(t *testing.T) {
	values := url.Values{}
	values.Add("filter", "status == active")

	if _, err := Parse(values, Options{}); !errors.Is(err, ErrInvalidFilter) {
		t.Fatalf("expected ErrInvalidFilter got %v", err)
	}

	opts := Options{AllowedFilterFields: map[string][]Operator{"status": {OperatorEqual}}}

	values = url.Values{}
	values.Add("filter", "status != active")
	if _, err := Parse(values, opts); !errors.Is(err, ErrInvalidFilter) {
		t.Fatalf("expected ErrInvalidFilter for operator got %v", err)
	}

	values = url.Values{}
	values.Add("filter", "unknown == value")
	if _, err := Parse(values, opts); !errors.Is(err, ErrInvalidFilter) {
		t.Fatalf("expected ErrInvalidFilter for field got %v", err)
	}
}

func TestEncodeDecodeToken(t *testing.T) {
	cursor := Cursor{StartAfter: []any{"id-1"}, StartAt: []any{123}}
	token, err := EncodeToken(cursor)
	if err != nil {
		t.Fatalf("EncodeToken returned error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	decoded, err := DecodeToken(token)
	if err != nil {
		t.Fatalf("DecodeToken returned error: %v", err)
	}
	if got := len(decoded.StartAfter); got != len(cursor.StartAfter) {
		t.Fatalf("expected startAfter length %d got %d", len(cursor.StartAfter), got)
	}
	if s, ok := decoded.StartAfter[0].(string); !ok || s != "id-1" {
		t.Fatalf("expected first cursor value %q got %#v", "id-1", decoded.StartAfter[0])
	}
	if got := len(decoded.StartAt); got != len(cursor.StartAt) {
		t.Fatalf("expected startAt length %d got %d", len(cursor.StartAt), got)
	}
	if fmt.Sprint(decoded.StartAt[0]) != "123" {
		t.Fatalf("expected numeric startAt value %q got %#v", "123", decoded.StartAt[0])
	}

	emptyToken, err := EncodeToken(Cursor{})
	if err != nil {
		t.Fatalf("EncodeToken for empty cursor returned error: %v", err)
	}
	if emptyToken != "" {
		t.Fatalf("expected empty token got %q", emptyToken)
	}
}

func TestDecodeTokenInvalid(t *testing.T) {
	if _, err := DecodeToken("not-base64"); !errors.Is(err, ErrInvalidPageToken) {
		t.Fatalf("expected ErrInvalidPageToken got %v", err)
	}
}

func TestContextHelpers(t *testing.T) {
	params := Params{PageSize: 12}
	ctx := WithParams(nil, params)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected context to return params")
	}
	if !reflect.DeepEqual(got, params) {
		t.Fatalf("expected params %#v got %#v", params, got)
	}

	defaultParams := FromContextOrDefault(context.Background())
	if defaultParams.PageSize != DefaultPageSize {
		t.Fatalf("expected default page size %d got %d", DefaultPageSize, defaultParams.PageSize)
	}
}

func TestFromRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/?pageSize=20", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	params, err := FromRequest(req, Options{})
	if err != nil {
		t.Fatalf("FromRequest returned error: %v", err)
	}
	if params.PageSize != 20 {
		t.Fatalf("expected page size 20 got %d", params.PageSize)
	}
}

func TestMust(t *testing.T) {
	ensured := Must(Params{})
	if ensured.PageSize != DefaultPageSize {
		t.Fatalf("expected default page size %d got %d", DefaultPageSize, ensured.PageSize)
	}

	ensured = Must(Params{PageSize: 15})
	if ensured.PageSize != 15 {
		t.Fatalf("expected page size 15 got %d", ensured.PageSize)
	}
}
