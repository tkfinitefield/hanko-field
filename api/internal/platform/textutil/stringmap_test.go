package textutil

import (
	"reflect"
	"testing"
)

func TestNormalizeStringMap(t *testing.T) {
	t.Helper()

	t.Run("trims keys and values", func(t *testing.T) {
		input := map[string]string{
			" Title ":     " About ",
			"description": " Learn ",
			"empty":       " ",
			" ":           "ignored",
			"":            "ignore",
		}

		expected := map[string]string{
			"Title":       "About",
			"description": "Learn",
			"empty":       "",
		}

		actual := NormalizeStringMap(input)
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("expected %#v got %#v", expected, actual)
		}
	})

	t.Run("returns nil for nil or empty input", func(t *testing.T) {
		if NormalizeStringMap(nil) != nil {
			t.Fatalf("expected nil for nil input")
		}
		if NormalizeStringMap(map[string]string{}) != nil {
			t.Fatalf("expected nil for empty map")
		}
	})
}
