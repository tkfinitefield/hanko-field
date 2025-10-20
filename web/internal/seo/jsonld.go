package seo

import (
    "encoding/json"
)

// JSON marshals v to a compact JSON string. It returns an empty string on error.
func JSON(v any) string {
    b, err := json.Marshal(v)
    if err != nil {
        return ""
    }
    return string(b)
}

// Organization returns a minimal Organization schema.
func Organization(name, url, logoURL string) map[string]any {
    m := map[string]any{
        "@context": "https://schema.org",
        "@type":    "Organization",
        "name":     name,
    }
    if url != "" { m["url"] = url }
    if logoURL != "" { m["logo"] = logoURL }
    return m
}

// WebSite returns a minimal WebSite schema with optional SearchAction.
func WebSite(name, url, searchActionURL string) map[string]any {
    m := map[string]any{
        "@context": "https://schema.org",
        "@type":    "WebSite",
        "name":     name,
    }
    if url != "" { m["url"] = url }
    if searchActionURL != "" {
        m["potentialAction"] = map[string]any{
            "@type": "SearchAction",
            "target": searchActionURL + "{search_term_string}",
            "query-input": "required name=search_term_string",
        }
    }
    return m
}

// BreadcrumbItem maps name and absolute item URL.
type BreadcrumbItem struct {
    Name string
    Item string
}

// BreadcrumbList builds schema.org BreadcrumbList.
func BreadcrumbList(items []BreadcrumbItem) map[string]any {
    el := make([]map[string]any, 0, len(items))
    for i, it := range items {
        el = append(el, map[string]any{
            "@type":    "ListItem",
            "position": i + 1,
            "name":     it.Name,
            "item":     it.Item,
        })
    }
    return map[string]any{
        "@context":        "https://schema.org",
        "@type":           "BreadcrumbList",
        "itemListElement": el,
    }
}

// Product returns a minimal product schema payload.
func Product(name, description, url, imageURL string, sku string) map[string]any {
    m := map[string]any{
        "@context":   "https://schema.org",
        "@type":      "Product",
        "name":       name,
        "description": description,
    }
    if url != "" { m["url"] = url }
    if imageURL != "" { m["image"] = imageURL }
    if sku != "" { m["sku"] = sku }
    return m
}

// Article returns a minimal Article schema payload.
func Article(headline, url, imageURL, authorName, datePublished string) map[string]any {
    m := map[string]any{
        "@context":      "https://schema.org",
        "@type":         "Article",
        "headline":      headline,
    }
    if url != "" { m["url"] = url }
    if imageURL != "" { m["image"] = imageURL }
    if authorName != "" { m["author"] = map[string]any{"@type": "Person", "name": authorName} }
    if datePublished != "" { m["datePublished"] = datePublished }
    return m
}

