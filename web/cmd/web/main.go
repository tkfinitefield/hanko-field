package main

import (
    "flag"
    "fmt"
    "html/template"
    "io/fs"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"
    "sync"
    "bytes"

    handlersPkg "finitefield.org/hanko-web/internal/handlers"
    "finitefield.org/hanko-web/internal/format"
    "finitefield.org/hanko-web/internal/i18n"
    "finitefield.org/hanko-web/internal/nav"
    "finitefield.org/hanko-web/internal/seo"
    mw "finitefield.org/hanko-web/internal/middleware"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

var (
    templatesDir = "templates"
    publicDir    = "public"
    localesDir   = "locales"
    // devMode is set in main() based on env: HANKO_WEB_DEV (preferred) or DEV (fallback)
    devMode   bool
    tmplCache *template.Template
    i18nBundle *i18n.Bundle
    // per-page cache in production to avoid reparse on each request
    pageTmplCache = map[string]*template.Template{}
    pageTmplMu sync.RWMutex
)

func main() {
	// Flags/environment
	var (
		addr     string
		tmplPath string
		pubPath  string
	)
	// Port resolution: prefer HANKO_WEB_PORT, then Cloud Run's PORT, else 8080
	port := os.Getenv("HANKO_WEB_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}
	flag.StringVar(&addr, "addr", ":"+port, "HTTP listen address")
    flag.StringVar(&tmplPath, "templates", templatesDir, "templates directory")
    flag.StringVar(&pubPath, "public", publicDir, "public assets directory")
    flag.StringVar(&localesDir, "locales", localesDir, "locales directory")
    flag.Parse()

	templatesDir = tmplPath
	publicDir = pubPath

    // Dev mode: prefer HANKO_WEB_DEV, fallback to DEV
    devMode = os.Getenv("HANKO_WEB_DEV") != "" || os.Getenv("DEV") != ""

    // Load i18n bundle
    sup := []string{"ja", "en"}
    if v := os.Getenv("HANKO_WEB_LOCALES"); v != "" {
        sup = strings.Split(v, ",")
        for i := range sup { sup[i] = strings.TrimSpace(sup[i]) }
    }
    var err error
    i18nBundle, err = i18n.Load(localesDir, "ja", sup)
    if err != nil {
        log.Fatalf("i18n load failed: %v", err)
    }

	if !devMode {
		// Parse templates once in production
		tc, err := parseTemplates()
		if err != nil {
			log.Fatalf("parse templates: %v", err)
		}
		tmplCache = tc
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// If deployed behind a trusted reverse proxy/load balancer, RealIP will use
	// X-Forwarded-For to determine the client IP. Ensure only trusted proxies
	// can set these headers in production environments.
	r.Use(middleware.RealIP)
    r.Use(mw.HTMX)
    r.Use(mw.Session)
    if i18nBundle != nil {
        r.Use(mw.Locale(i18nBundle))
    }
    r.Use(mw.Auth)
    r.Use(mw.CSRF)
    r.Use(mw.VaryLocale)
    r.Use(mw.Logger)
    r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(30 * time.Second))

	// Health check
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Static assets under /assets/ (with Cache-Control and ETag)
	assetsRoot := filepath.Join(publicDir, "assets")
	assets := http.StripPrefix("/assets", mw.AssetsWithCache(assetsRoot))
	r.Handle("/assets/*", assets)

    // Home page
    r.Get("/", HomeHandler)
    // Top-level pages
    r.Get("/shop", ShopHandler)
    r.Get("/templates", TemplatesHandler)
    r.Get("/guides", GuidesHandler)
    r.Get("/account", AccountHandler)
    // Fragment endpoints (htmx)
    r.Get("/frags/compare/sku-table", CompareSKUTableFrag)
    r.Get("/frags/guides/latest", LatestGuidesFrag)
    // Modal demo fragment (htmx)
    r.Get("/modals/demo", DemoModalHandler)

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("web listening on %s (devMode=%v)", addr, devMode)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func tmplFuncMapFor(getT func() *template.Template) template.FuncMap {
    return template.FuncMap{
        "now":      time.Now,
        "nowf":     func(layout string) string { return time.Now().Format(layout) },
        "tlang":    func(lang, key string) string { if i18nBundle == nil { return key }; return i18nBundle.T(lang, key) },
        "fmtDate":  func(ts time.Time, lang string) string { return format.FmtDate(ts, lang) },
        "fmtMoney": func(amount int64, currency, lang string) string { return format.FmtCurrency(amount, currency, lang) },
        "seq":      func(n int) []int { if n < 0 { n = 0 }; s := make([]int, n); for i := range s { s[i] = i }; return s },
        // dict builds a string-keyed map for component props
        "dict":     func(v ...any) map[string]any { m := map[string]any{}; for i := 0; i+1 < len(v); i += 2 { k := fmt.Sprint(v[i]); m[k] = v[i+1] }; return m },
        // list returns a slice of the arguments
        "list":     func(v ...any) []any { return v },
        // safe marks a string as trusted HTML. Use sparingly.
        "safe":     func(s string) template.HTML { return template.HTML(s) },
        // slot executes another template by name, passing data, and returns trusted HTML
        "slot":     func(name string, data any) template.HTML {
            t := getT()
            if t == nil || name == "" {
                return ""
            }
            var buf bytes.Buffer
            if err := t.ExecuteTemplate(&buf, name, data); err != nil {
                // render an HTML comment with the error to aid debugging without breaking page
                return template.HTML("<!-- slot '" + template.HTMLEscapeString(name) + "' error: " + template.HTMLEscapeString(err.Error()) + " -->")
            }
            return template.HTML(buf.String())
        },
    }
}

func parseTemplates() (*template.Template, error) {
    // create root template and bind funcMap that can access it
    root := template.New("_root")
    funcMap := tmplFuncMapFor(func() *template.Template { return root })
    root = root.Funcs(funcMap)
	// Recursively discover and parse all .tmpl files. Note: ParseGlob doesn't support **.
	var files []string
	if err := filepath.WalkDir(templatesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".tmpl") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
    if len(files) == 0 {
        return nil, fmt.Errorf("no templates found under %s", templatesDir)
    }
    return root.ParseFiles(files...)
}

// parsePageTemplates builds a template set with the shared layout/partials and one page.
func parsePageTemplates(page string) (*template.Template, error) {
    root := template.New("_root")
    funcMap := tmplFuncMapFor(func() *template.Template { return root })
    root = root.Funcs(funcMap)
    var files []string
    // layouts
    _ = filepath.WalkDir(filepath.Join(templatesDir, "layouts"), func(path string, d fs.DirEntry, err error) error {
        if err != nil { return err }
        if d.IsDir() { return nil }
        if strings.HasSuffix(d.Name(), ".tmpl") { files = append(files, path) }
        return nil
    })
    // partials
    _ = filepath.WalkDir(filepath.Join(templatesDir, "partials"), func(path string, d fs.DirEntry, err error) error {
        if err != nil { return err }
        if d.IsDir() { return nil }
        if strings.HasSuffix(d.Name(), ".tmpl") { files = append(files, path) }
        return nil
    })
    // page
    files = append(files, filepath.Join(templatesDir, "pages", page+".tmpl"))
    return root.ParseFiles(files...)
}

// renderTemplate executes a named template (partial/fragment) without the base layout.
func renderTemplate(w http.ResponseWriter, r *http.Request, name string, data any) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    var t *template.Template
    if devMode {
        tc, err := parseTemplates()
        if err != nil {
            http.Error(w, fmt.Sprintf("template parse error: %v", err), http.StatusInternalServerError)
            return
        }
        t = tc
    } else {
        t = tmplCache
    }
    if t == nil {
        http.Error(w, "template not initialized", http.StatusInternalServerError)
        return
    }
    if err := t.ExecuteTemplate(w, name, data); err != nil {
        http.Error(w, fmt.Sprintf("template exec error: %v", err), http.StatusInternalServerError)
        return
    }
}

// render executes the base layout. In dev mode, templates are reparsed on each request.
func render(w http.ResponseWriter, r *http.Request, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var t *template.Template
	if devMode {
		tc, err := parseTemplates()
		if err != nil {
			http.Error(w, fmt.Sprintf("template parse error: %v", err), http.StatusInternalServerError)
			return
		}
		t = tc
	} else {
		t = tmplCache
	}
	if t == nil {
		http.Error(w, "template not initialized", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, fmt.Sprintf("template exec error: %v", err), http.StatusInternalServerError)
		return
	}
}

// renderPage executes the base layout with page-specific content definitions.
func renderPage(w http.ResponseWriter, r *http.Request, page string, data any) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    var t *template.Template
    if devMode {
        var err error
        t, err = parsePageTemplates(page)
        if err != nil {
            http.Error(w, fmt.Sprintf("template parse error: %v", err), http.StatusInternalServerError)
            return
        }
    } else {
        pageTmplMu.RLock()
        t = pageTmplCache[page]
        pageTmplMu.RUnlock()
        if t == nil {
            var err error
            t, err = parsePageTemplates(page)
            if err != nil {
                http.Error(w, fmt.Sprintf("template parse error: %v", err), http.StatusInternalServerError)
                return
            }
            pageTmplMu.Lock()
            pageTmplCache[page] = t
            pageTmplMu.Unlock()
        }
    }
    if t == nil {
        http.Error(w, "template not initialized", http.StatusInternalServerError)
        return
    }
    if err := t.ExecuteTemplate(w, "base", data); err != nil {
        http.Error(w, fmt.Sprintf("template exec error: %v", err), http.StatusInternalServerError)
        return
    }
}

// HomeHandler renders the landing page.
func HomeHandler(w http.ResponseWriter, r *http.Request) {
    lang := mw.Lang(r)
    vm := handlersPkg.BuildHomeData(lang)
    // augment common layout data
    vm.Path = r.URL.Path
    vm.Nav = nav.Build(vm.Path)
    vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
    vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
    if i18nBundle != nil {
        vm.SEO.Title = i18nBundle.T(lang, "home.seo.title")
        vm.SEO.Description = i18nBundle.T(lang, "home.seo.description")
    }
    // Canonical + OG URL/Site
    vm.SEO.Canonical = absoluteURL(r)
    vm.SEO.OG.URL = vm.SEO.Canonical
    vm.SEO.OG.SiteName = i18nOrDefault(lang, "brand.name", "Hanko Field")
    vm.SEO.Alternates = buildAlternates(r)
    // Default JSON-LD (Organization + WebSite)
    siteURL := siteBaseURL(r)
    org := seo.Organization(i18nOrDefault(lang, "brand.name", "Hanko Field"), siteURL, "")
    ws := seo.WebSite(i18nOrDefault(lang, "brand.name", "Hanko Field"), siteURL, siteURL+"/search?q=")
    vm.SEO.JSONLD = []string{seo.JSON(org), seo.JSON(ws)}
    // Add representative Product + Articles JSON-LD
    // Product
    prod := seo.Product(i18nOrDefault(lang, "home.compare.col.name", "Name")+": Classic Round", i18nOrDefault(lang, "home.seo.description", "Custom stamps and seals"), siteURL+"/shop", "", "T-100")
    vm.SEO.JSONLD = append(vm.SEO.JSONLD, seo.JSON(prod))
    // Articles (latest guides)
    art1 := seo.Article(i18nOrDefault(lang, "home.guides.title", "Latest Guides")+": Materials", siteURL+"/guides/materials", "", "Hanko Field", "2025-01-10")
    art2 := seo.Article(i18nOrDefault(lang, "home.guides.title", "Latest Guides")+": Design Basics", siteURL+"/guides/design-basics", "", "Hanko Field", "2025-01-05")
    vm.SEO.JSONLD = append(vm.SEO.JSONLD, seo.JSON(art1), seo.JSON(art2))
    renderPage(w, r, "home", vm)
}

// Generic page handlers
func ShopHandler(w http.ResponseWriter, r *http.Request) {
    lang := mw.Lang(r)
    vm := handlersPkg.PageData{Title: "Shop", Lang: lang}
    vm.Path = r.URL.Path
    vm.Nav = nav.Build(vm.Path)
    vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
    vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
    vm.SEO.Canonical = absoluteURL(r)
    vm.SEO.OG.URL = vm.SEO.Canonical
    vm.SEO.OG.SiteName = i18nOrDefault(lang, "brand.name", "Hanko Field")
    vm.SEO.Alternates = buildAlternates(r)
    renderPage(w, r, "shop", vm)
}

func TemplatesHandler(w http.ResponseWriter, r *http.Request) {
    lang := mw.Lang(r)
    vm := handlersPkg.PageData{Title: "Templates", Lang: lang}
    vm.Path = r.URL.Path
    vm.Nav = nav.Build(vm.Path)
    vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
    vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
    vm.SEO.Canonical = absoluteURL(r)
    vm.SEO.OG.URL = vm.SEO.Canonical
    vm.SEO.OG.SiteName = i18nOrDefault(lang, "brand.name", "Hanko Field")
    vm.SEO.Alternates = buildAlternates(r)
    renderPage(w, r, "templates", vm)
}

func GuidesHandler(w http.ResponseWriter, r *http.Request) {
    lang := mw.Lang(r)
    vm := handlersPkg.PageData{Title: "Guides", Lang: lang}
    vm.Path = r.URL.Path
    vm.Nav = nav.Build(vm.Path)
    vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
    vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
    vm.SEO.Canonical = absoluteURL(r)
    vm.SEO.OG.URL = vm.SEO.Canonical
    vm.SEO.OG.SiteName = i18nOrDefault(lang, "brand.name", "Hanko Field")
    vm.SEO.Alternates = buildAlternates(r)
    renderPage(w, r, "guides", vm)
}

func AccountHandler(w http.ResponseWriter, r *http.Request) {
    lang := mw.Lang(r)
    vm := handlersPkg.PageData{Title: "Account", Lang: lang}
    vm.Path = r.URL.Path
    vm.Nav = nav.Build(vm.Path)
    vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
    vm.Analytics = handlersPkg.LoadAnalyticsFromEnv()
    vm.SEO.Canonical = absoluteURL(r)
    vm.SEO.OG.URL = vm.SEO.Canonical
    vm.SEO.OG.SiteName = i18nOrDefault(lang, "brand.name", "Hanko Field")
    vm.SEO.Alternates = buildAlternates(r)
    renderPage(w, r, "account", vm)
}

// DemoModalHandler returns a demo modal fragment for HTMX insertion.
func DemoModalHandler(w http.ResponseWriter, r *http.Request) {
    lang := mw.Lang(r)
    _ = lang // reserved for future i18n of title/buttons
    props := map[string]any{
        "ID":    "demo-modal",
        "Title": "Demo Modal",
        "Body":  "This is a shared modal opened via HTMX. Press ESC or click the overlay to close.",
        // No FooterTmpl provided → default Close button with data-modal-close
    }
    renderTemplate(w, r, "c_modal", props)
}

// absoluteURL builds an absolute URL for the current request path, using X-Forwarded-Proto if present.
func absoluteURL(r *http.Request) string {
    scheme := r.Header.Get("X-Forwarded-Proto")
    if scheme == "" {
        if r.TLS != nil { scheme = "https" } else { scheme = "http" }
    }
    host := r.Host
    if host == "" { host = "localhost" }
    return scheme + "://" + host + r.URL.Path
}

// siteBaseURL returns the base site URL (scheme+host) inferred from the request.
func siteBaseURL(r *http.Request) string {
    scheme := r.Header.Get("X-Forwarded-Proto")
    if scheme == "" { if r.TLS != nil { scheme = "https" } else { scheme = "http" } }
    host := r.Host
    if host == "" { host = "localhost" }
    return scheme + "://" + host
}

// buildAlternates produces hreflang alternates for supported languages using the current path.
func buildAlternates(r *http.Request) []struct{ Href, Hreflang string } {
    var out []struct{ Href, Hreflang string }
    if i18nBundle == nil { return out }
    base := siteBaseURL(r)
    path := r.URL.Path
    supported := i18nBundle.Supported()
    for _, l := range supported {
        href := base + path + "?hl=" + l
        out = append(out, struct{ Href, Hreflang string }{Href: href, Hreflang: l})
    }
    // x-default points to fallback
    out = append(out, struct{ Href, Hreflang string }{Href: base + path, Hreflang: "x-default"})
    return out
}

func i18nOrDefault(lang, key, def string) string {
    if i18nBundle == nil { return def }
    v := i18nBundle.T(lang, key)
    if v == "" || v == key { return def }
    return v
}

// --- Fragments and supporting types ---

// SKU represents a simple product option for comparison.
type SKU struct {
    ID    string
    Name  string
    Shape string
    Size  string
    Price string // display price (e.g., "$12")
}

// skuData returns the canonical list of SKUs for comparison.
func skuData(lang string) []SKU {
    // Static seed data; in the future fetch from API/DB.
    // Translate names lightly depending on lang.
    round := map[string]string{"en": "Classic Round", "ja": "丸型クラシック"}
    square := map[string]string{"en": "Square Logo", "ja": "角形ロゴ"}
    rect := map[string]string{"en": "Business Seal", "ja": "ビジネス印"}
    tl := func(m map[string]string, l string) string { if v, ok := m[l]; ok { return v }; return m["en"] }
    return []SKU{
        {ID: "T-100", Name: tl(round, lang),  Shape: "round",  Size: "small",  Price: "$12"},
        {ID: "T-105", Name: tl(round, lang),  Shape: "round",  Size: "medium", Price: "$14"},
        {ID: "T-110", Name: tl(round, lang),  Shape: "round",  Size: "large",  Price: "$18"},
        {ID: "T-220", Name: tl(square, lang), Shape: "square", Size: "small",  Price: "$16"},
        {ID: "T-225", Name: tl(square, lang), Shape: "square", Size: "medium", Price: "$18"},
        {ID: "T-310", Name: tl(rect, lang),   Shape: "rect",   Size: "large",  Price: "$24"},
    }
}

// CompareSKUTableFrag renders the SKU comparison table with optional filters.
func CompareSKUTableFrag(w http.ResponseWriter, r *http.Request) {
    lang := mw.Lang(r)
    shape := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("shape")))
    size := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("size")))

    // Build rows with filters applied
    cols := []map[string]any{
        {"Key": "id", "Label": "ID", "Align": "left"},
        {"Key": "name", "Label": i18nOrDefault(lang, "home.compare.col.name", "Name"), "Align": "left"},
        {"Key": "shape", "Label": i18nOrDefault(lang, "home.compare.col.shape", "Shape"), "Align": "left"},
        {"Key": "size", "Label": i18nOrDefault(lang, "home.compare.col.size", "Size"), "Align": "left"},
        {"Key": "price", "Label": i18nOrDefault(lang, "home.compare.col.price", "Price"), "Align": "right"},
    }
    var rows []map[string]string
    for _, s := range skuData(lang) {
        if shape != "" && s.Shape != shape { continue }
        if size != "" && s.Size != size { continue }
        rows = append(rows, map[string]string{"id": s.ID, "name": s.Name, "shape": s.Shape, "size": s.Size, "price": s.Price})
    }

    // ETag simple hash of inputs
    etag := etagFor("sku:", lang, shape, size)
    if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
        w.WriteHeader(http.StatusNotModified)
        return
    }
    w.Header().Set("Cache-Control", "public, max-age=60")
    w.Header().Set("ETag", etag)

    props := map[string]any{
        "Columns": cols,
        "Rows":    rows,
        "Shape":   shape,
        "Size":    size,
        "Lang":    lang,
    }
    renderTemplate(w, r, "frag_compare_sku_table", props)
}

// LatestGuidesFrag renders a small set of localized guide cards.
func LatestGuidesFrag(w http.ResponseWriter, r *http.Request) {
    lang := mw.Lang(r)
    type Guide struct { Title, URL, Excerpt, Date string }
    // Seed localized guide list
    var guides []Guide
    if lang == "ja" {
        guides = []Guide{
            {Title: "はんこ素材の選び方", URL: "/guides/materials", Excerpt: "用途別に最適な素材を解説します。", Date: "2025-01-10"},
            {Title: "印影デザインの基本", URL: "/guides/design-basics", Excerpt: "読みやすさと個性のバランスを学びます。", Date: "2025-01-05"},
            {Title: "サイズ比較ガイド", URL: "/guides/size-guide", Excerpt: "丸・角・楕円のサイズ感を比較。", Date: "2024-12-20"},
        }
    } else {
        guides = []Guide{
            {Title: "How to Choose Materials", URL: "/guides/materials", Excerpt: "Pick the right material for your use.", Date: "2025-01-10"},
            {Title: "Seal Design Basics", URL: "/guides/design-basics", Excerpt: "Balance legibility with personality.", Date: "2025-01-05"},
            {Title: "Size Comparison Guide", URL: "/guides/size-guide", Excerpt: "Compare round, square, and rectangular.", Date: "2024-12-20"},
        }
    }

    // Simple 2-minute cache
    etag := etagFor("guides:", lang)
    if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
        w.WriteHeader(http.StatusNotModified)
        return
    }
    w.Header().Set("Cache-Control", "public, max-age=120")
    w.Header().Set("ETag", etag)

    props := map[string]any{"Guides": guides}
    renderTemplate(w, r, "frag_guides_latest", props)
}

// etagFor builds a weak pseudo-ETag from inputs.
func etagFor(prefix string, parts ...string) string {
    // very small non-crypto hash
    h := 1469598103934665603 ^ uint64(len(prefix))
    for _, s := range parts {
        for i := 0; i < len(s); i++ {
            h ^= uint64(s[i])
            h *= 1099511628211
        }
    }
    return fmt.Sprintf("W/\"%s%x\"", prefix, h)
}
