package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// AssetsWithCache wraps a file server and applies Cache-Control, Vary, and ETag handling.
func AssetsWithCache(dir string) http.Handler {
	// precompute ETags for files under dir
	etags := map[string]string{}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		et, _ := fileETag(path)
		// store by URL path suffix relative to dir
		if rel, err := filepath.Rel(dir, path); err == nil {
			// always use '/' separators for URLs
			rel = filepath.ToSlash(rel)
			etags["/"+rel] = et
		}
		return nil
	})
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// add cache headers
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Cache-Control", "public, max-age=604800, stale-while-revalidate=86400")
		if et := etags[strings.TrimPrefix(r.URL.Path, "/assets")]; et != "" {
			w.Header().Set("ETag", et)
			if inm := r.Header.Get("If-None-Match"); inm != "" && inm == et {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		fs.ServeHTTP(w, r)
	})
}

func fileETag(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return `W/"` + hex.EncodeToString(h.Sum(nil)) + `"`, nil
}
