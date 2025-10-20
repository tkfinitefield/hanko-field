package seo

type OpenGraph struct {
    Title       string
    Description string
    Image       string
    Type        string
}

type Twitter struct {
    Card  string
    Site  string
    Image string
}

type Meta struct {
    Title       string
    Description string
    Canonical   string
    OG          OpenGraph
    Twitter     Twitter
}

