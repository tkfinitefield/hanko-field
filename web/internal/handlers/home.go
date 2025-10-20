package handlers

import "finitefield.org/hanko-web/internal/nav"

// HomeData is the view model for the home page.
type HomeData struct {
    Title   string
    Message string
    Lang    string
    SEO     SEOData
    Analytics Analytics
    // Common layout fields
    Path        string
    Nav         []nav.RenderedItem
    Breadcrumbs []nav.Crumb
}

// BuildHomeData constructs the default view model for the landing page.
func BuildHomeData(lang string) HomeData {
    return HomeData{
        Title:   "Hanko Field",
        Message: "Welcome to Hanko Field (Web)",
        Lang:    lang,
        SEO: SEOData{
            Title:       "Hanko Field – Welcome",
            Description: "Hanko Field - Custom stamps and seals",
        },
    }
}

// SEOData is a lightweight copy to avoid importing the seo package here.
type SEOData struct {
    Title       string
    Description string
    Canonical   string
    Robots      string
    OG          struct{
        Title       string
        Description string
        Image       string
        Type        string
        URL         string
        SiteName    string
    }
    Twitter     struct{
        Card  string
        Site  string
        Image string
    }
    Alternates []struct{ Href, Hreflang string }
    JSONLD     []string
}
