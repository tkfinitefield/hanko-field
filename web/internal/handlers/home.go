package handlers

// HomeData is the view model for the home page.
type HomeData struct {
    Title   string
    Message string
    Lang    string
    SEO     SEOData
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
}
