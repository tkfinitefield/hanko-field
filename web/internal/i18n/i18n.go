package i18n

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strconv"
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

// Fallback returns the configured fallback language.
func (b *Bundle) Fallback() string { return b.fallback }

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
    type langPref struct {
        base string
        q    float64
        pos  int
    }
    prefs := make([]langPref, 0, 8)
    parts := strings.Split(acceptLang, ",")
    for i, raw := range parts {
        p := strings.TrimSpace(raw)
        if p == "" {
            continue
        }
        q := 1.0
        if sc := strings.IndexByte(p, ';'); sc != -1 {
            // parse ;q=...
            params := strings.TrimSpace(p[sc+1:])
            p = strings.TrimSpace(p[:sc])
            if strings.HasPrefix(params, "q=") {
                if v, err := parseQValue(strings.TrimPrefix(params, "q=")); err == nil {
                    q = v
                }
            }
        }
        base := p
        if dash := strings.IndexByte(p, '-'); dash != -1 {
            base = p[:dash]
        }
        base = strings.ToLower(base)
        prefs = append(prefs, langPref{base: base, q: q, pos: i})
    }
    // sort by q desc then by original order
    sort.SliceStable(prefs, func(i, j int) bool {
        if prefs[i].q == prefs[j].q {
            return prefs[i].pos < prefs[j].pos
        }
        return prefs[i].q > prefs[j].q
    })
    for _, lp := range prefs {
        if b.isSupported(lp.base) {
            return lp.base
        }
    }
    return b.fallback
}

// parseQValue parses a qvalue per RFC 7231 (0.0 to 1.0).
func parseQValue(s string) (float64, error) {
    s = strings.TrimSpace(s)
    // Only simple parser needed: 1, 1.0, 0.8, etc.
    var v float64
    var err error
    // fast path for common values
    switch s {
    case "1", "1.0", "1.00":
        return 1.0, nil
    case "0", "0.0", "0.00":
        return 0.0, nil
    }
    v, err = strconv.ParseFloat(s, 64)
    if err != nil {
        return 0, err
    }
    if v < 0 {
        v = 0
    } else if v > 1 {
        v = 1
    }
    return v, nil
}
