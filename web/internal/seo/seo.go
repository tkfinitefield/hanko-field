package seo

// OpenGraph contains Open Graph meta fields.
type OpenGraph struct {
    Title       string
    Description string
    Image       string
    Type        string
    URL         string
    SiteName    string
}

// Twitter contains Twitter card meta fields.
type Twitter struct {
    Card  string
    Site  string
    Image string
}

// AlternateLink represents an hreflang alternate URL.
type AlternateLink struct {
    Href     string
    Hreflang string // e.g. "ja", "en", "x-default"
}

// Meta is the top-level SEO metadata container passed to templates.
type Meta struct {
    Title       string
    Description string
    Canonical   string
    Robots      string // e.g. "index,follow" or "noindex,nofollow"
    SiteName    string
    OG          OpenGraph
    Twitter     Twitter
    Alternates  []AlternateLink
    // JSONLD contains raw JSON strings for script type="application/ld+json"
    JSONLD      []string
}
