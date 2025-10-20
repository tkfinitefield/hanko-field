package handlers

import "os"

// Analytics holds client instrumentation configuration surfaced to templates.
type Analytics struct {
    GA4MeasurementID string // e.g. G-XXXXXXXXXX
    GTMContainerID   string // e.g. GTM-XXXXXXX
    SegmentWriteKey  string // Segment browser key
    Debug            bool
}

// LoadAnalyticsFromEnv builds Analytics from environment variables.
func LoadAnalyticsFromEnv() Analytics {
    return Analytics{
        GA4MeasurementID: os.Getenv("HANKO_WEB_GA_MEASUREMENT_ID"),
        GTMContainerID:   os.Getenv("HANKO_WEB_GTM_CONTAINER_ID"),
        SegmentWriteKey:  os.Getenv("HANKO_WEB_SEGMENT_WRITE_KEY"),
        Debug:            os.Getenv("HANKO_WEB_ANALYTICS_DEBUG") != "",
    }
}

