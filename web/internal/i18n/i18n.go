package i18n

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

type Bundle struct {
    dict      map[string]map[string]string
    fallback  string
    supported map[string]struct{}
}

func Load(dir string, fallback string, supported []string) (*Bundle, error) {
    b := &Bundle{
        dict:      map[string]map[string]string{},
        fallback:  fallback,
        supported: map[string]struct{}{},
    }
    if len(supported) == 0 {
        supported = []string{"ja", "en"}
    }
    for _, l := range supported {
        b.supported[l] = struct{}{}
        path := filepath.Join(dir, l+".json")
        raw, err := os.ReadFile(path)
        if err != nil {
            // allow missing file for non-default locales
            if l == fallback {
                return nil, fmt.Errorf("load locale %s: %w", l, err)
            }
            continue
        }
        var m map[string]string
        if err := json.Unmarshal(raw, &m); err != nil {
            return nil, fmt.Errorf("unmarshal %s: %w", l, err)
        }
        b.dict[l] = m
    }
    if _, ok := b.dict[fallback]; !ok {
        return nil, fmt.Errorf("fallback locale %s not loaded", fallback)
    }
    return b, nil
}

func (b *Bundle) Supported() []string {
    out := make([]string, 0, len(b.supported))
    for k := range b.supported {
        out = append(out, k)
    }
    sort.Strings(out)
    return out
}

func (b *Bundle) isSupported(lang string) bool {
    _, ok := b.supported[lang]
    return ok
}

// T returns translation for key in lang, falling back to default and finally key.
func (b *Bundle) T(lang, key string) string {
    if lang != "" {
        if m, ok := b.dict[lang]; ok {
            if v, ok := m[key]; ok {
                return v
            }
        }
    }
    if m, ok := b.dict[b.fallback]; ok {
        if v, ok := m[key]; ok {
            return v
        }
    }
    return key
}

// Resolve chooses best language from Accept-Language header.
func (b *Bundle) Resolve(acceptLang string) string {
    // Very small parser: split by comma, take primary tag, match supported.
    parts := strings.Split(acceptLang, ",")
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p == "" {
            continue
        }
        // strip ;q=...
        if i := strings.IndexByte(p, ';'); i != -1 {
            p = p[:i]
        }
        // primary subtag only (e.g., ja-JP -> ja)
        base := p
        if i := strings.IndexByte(p, '-'); i != -1 {
            base = p[:i]
        }
        base = strings.ToLower(base)
        if b.isSupported(base) {
            return base
        }
    }
    return b.fallback
}

