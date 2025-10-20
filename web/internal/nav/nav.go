package nav

import (
    "path"
    "strings"
)

// Item represents a top-level navigation item.
type Item struct {
    Path     string // e.g. "/shop"
    LabelKey string // i18n key, e.g. "nav.shop"
}

// RenderedItem is a view model for templates.
type RenderedItem struct {
    Href     string
    LabelKey string
    Active   bool
}

// Crumb represents a breadcrumb entry. If LabelKey is empty, use Label.
type Crumb struct {
    Href     string
    LabelKey string
    Label    string
    Active   bool
}

// Main is the primary navigation definition.
var Main = []Item{
    {Path: "/shop", LabelKey: "nav.shop"},
    {Path: "/templates", LabelKey: "nav.templates"},
    {Path: "/guides", LabelKey: "nav.guides"},
    {Path: "/account", LabelKey: "nav.account"},
}

// Build renders navigation items with active state given the current path.
func Build(currentPath string) []RenderedItem {
    if currentPath == "" {
        currentPath = "/"
    }
    items := make([]RenderedItem, 0, len(Main))
    for _, it := range Main {
        active := isActive(it.Path, currentPath)
        items = append(items, RenderedItem{
            Href:     it.Path,
            LabelKey: it.LabelKey,
            Active:   active,
        })
    }
    return items
}

func isActive(itemPath, currentPath string) bool {
    if itemPath == "/" {
        return currentPath == "/"
    }
    // match exact or prefix boundary: "/shop" or "/shop/..."
    if currentPath == itemPath {
        return true
    }
    if strings.HasPrefix(currentPath, itemPath+"/") {
        return true
    }
    return false
}

// Breadcrumbs builds breadcrumb entries from the current path.
// Rules:
// - Always start with Home
// - For known top-level sections, use nav label keys
// - For deeper segments, use a prettified segment label
func Breadcrumbs(currentPath string) []Crumb {
    var crumbs []Crumb
    // Home
    if currentPath == "" {
        currentPath = "/"
    }
    crumbs = append(crumbs, Crumb{Href: "/", LabelKey: "nav.home", Active: currentPath == "/"})
    if currentPath == "/" {
        return crumbs
    }

    // Normalize and split
    clean := path.Clean(currentPath)
    if clean == "." { // should not happen but guard
        clean = "/"
    }
    parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")

    // Top-level mapping from Main
    if len(parts) > 0 && parts[0] != "" {
        top := "/" + parts[0]
        // default label
        labelKey := ""
        for _, it := range Main {
            if it.Path == top {
                labelKey = it.LabelKey
                break
            }
        }
        crumbs = append(crumbs, Crumb{Href: top, LabelKey: labelKey, Label: titleFromSegment(parts[0]), Active: len(parts) == 1})
    }

    // Deeper segments
    if len(parts) > 1 {
        href := "/" + parts[0]
        for i := 1; i < len(parts); i++ {
            href = href + "/" + parts[i]
            crumbs = append(crumbs, Crumb{
                Href:   href,
                Label:  titleFromSegment(parts[i]),
                Active: i == len(parts)-1,
            })
        }
    }
    return crumbs
}

func titleFromSegment(seg string) string {
    if seg == "" {
        return seg
    }
    // replace hyphens/underscores with spaces and capitalize first letter
    s := strings.ReplaceAll(seg, "-", " ")
    s = strings.ReplaceAll(s, "_", " ")
    // very small titlecase: first rune upper
    r := []rune(s)
    r[0] = toUpper(r[0])
    return string(r)
}

func toUpper(r rune) rune {
    // ASCII only is sufficient for slugs here
    if r >= 'a' && r <= 'z' {
        return r - ('a' - 'A')
    }
    return r
}

