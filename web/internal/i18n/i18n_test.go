package i18n

import "testing"

func TestResolveHonorsQValues(t *testing.T) {
    b, err := Load("../../locales", "ja", []string{"ja", "en"})
    if err != nil {
        t.Fatalf("load: %v", err)
    }
    got := b.Resolve("ja;q=0.8, en;q=0.9")
    if got != "en" {
        t.Fatalf("expected en, got %s", got)
    }
}

