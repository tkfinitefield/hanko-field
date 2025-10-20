package testutil

import (
	"bytes"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// ParseHTML parses the provided HTML payload into a goquery document for assertions.
func ParseHTML(t testing.TB, body []byte) *goquery.Document {
	t.Helper()

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	return doc
}
