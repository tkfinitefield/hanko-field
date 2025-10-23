package helpers

import (
	"net/url"
	"testing"
)

func TestSetRawQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rawQuery string
		key      string
		value    string
		want     map[string]string
	}{
		{
			name:     "updates existing key",
			rawQuery: "status=open&page=2",
			key:      "page",
			value:    "3",
			want: map[string]string{
				"status": "open",
				"page":   "3",
			},
		},
		{
			name:     "adds new key when missing",
			rawQuery: "status=open",
			key:      "page",
			value:    "1",
			want: map[string]string{
				"status": "open",
				"page":   "1",
			},
		},
		{
			name:     "handles empty input",
			rawQuery: "",
			key:      "page",
			value:    "1",
			want: map[string]string{
				"page": "1",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SetRawQuery(tc.rawQuery, tc.key, tc.value)
			values, err := url.ParseQuery(got)
			if err != nil {
				t.Fatalf("ParseQuery returned error: %v", err)
			}
			for k, expected := range tc.want {
				if got := values.Get(k); got != expected {
					t.Errorf("expected %s=%s, got %s", k, expected, got)
				}
			}
		})
	}
}

func TestDelRawQuery(t *testing.T) {
	t.Parallel()

	raw := "status=open&page=2"
	got := DelRawQuery(raw, "page")
	values, err := url.ParseQuery(got)
	if err != nil {
		t.Fatalf("ParseQuery returned error: %v", err)
	}
	if values.Get("page") != "" {
		t.Errorf("expected page param removed, got %q", values.Get("page"))
	}
	if values.Get("status") != "open" {
		t.Errorf("expected status preserved, got %q", values.Get("status"))
	}
}

func TestBuildURL(t *testing.T) {
	t.Parallel()

	u := BuildURL("/admin/orders", "page=2&sort=created_at")
	if u != "/admin/orders?page=2&sort=created_at" {
		t.Errorf("unexpected URL: %s", u)
	}

	// handles empty raw query without trailing question mark
	u = BuildURL("/admin/orders?page=1", "")
	if u != "/admin/orders" {
		t.Errorf("expected query stripped when empty, got %s", u)
	}
}
