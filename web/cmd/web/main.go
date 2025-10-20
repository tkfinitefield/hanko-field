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
    if i18nBundle != nil {
        vm.SEO.Title = i18nBundle.T(lang, "home.seo.title")
        vm.SEO.Description = i18nBundle.T(lang, "home.seo.description")
    }
    renderPage(w, r, "home", vm)
}

// Generic page handlers
func ShopHandler(w http.ResponseWriter, r *http.Request) {
    lang := mw.Lang(r)
    vm := handlersPkg.PageData{Title: "Shop", Lang: lang}
    vm.Path = r.URL.Path
    vm.Nav = nav.Build(vm.Path)
    vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
    renderPage(w, r, "shop", vm)
}

func TemplatesHandler(w http.ResponseWriter, r *http.Request) {
    lang := mw.Lang(r)
    vm := handlersPkg.PageData{Title: "Templates", Lang: lang}
    vm.Path = r.URL.Path
    vm.Nav = nav.Build(vm.Path)
    vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
    renderPage(w, r, "templates", vm)
}

func GuidesHandler(w http.ResponseWriter, r *http.Request) {
    lang := mw.Lang(r)
    vm := handlersPkg.PageData{Title: "Guides", Lang: lang}
    vm.Path = r.URL.Path
    vm.Nav = nav.Build(vm.Path)
    vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
    renderPage(w, r, "guides", vm)
}

func AccountHandler(w http.ResponseWriter, r *http.Request) {
    lang := mw.Lang(r)
    vm := handlersPkg.PageData{Title: "Account", Lang: lang}
    vm.Path = r.URL.Path
    vm.Nav = nav.Build(vm.Path)
    vm.Breadcrumbs = nav.Breadcrumbs(vm.Path)
    renderPage(w, r, "account", vm)
}
