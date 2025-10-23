package helpers

import "strings"

// HighlightSegment represents a split section of text with optional emphasis.
type HighlightSegment struct {
	Text  string
	Match bool
	Empty bool
}

// HighlightSegments splits text into segments with highlighted matches.
func HighlightSegments(text, term string) []HighlightSegment {
	if term = strings.TrimSpace(term); term == "" {
		if text == "" {
			return nil
		}
		return []HighlightSegment{{Text: text}}
	}

	lowerText := strings.ToLower(text)
	lowerTerm := strings.ToLower(term)

	if lowerTerm == "" || text == "" {
		return []HighlightSegment{{Text: text}}
	}

	var segments []HighlightSegment
	cursor := 0
	termLen := len(lowerTerm)

	for cursor <= len(lowerText) {
		index := strings.Index(lowerText[cursor:], lowerTerm)
		if index < 0 {
			break
		}
		if index > 0 {
			segment := text[cursor : cursor+index]
			if segment != "" {
				segments = append(segments, HighlightSegment{Text: segment})
			}
		}
		matchEnd := cursor + index + termLen
		match := text[cursor+index : matchEnd]
		segments = append(segments, HighlightSegment{Text: match, Match: true})
		cursor = matchEnd
	}

	if cursor < len(text) {
		segments = append(segments, HighlightSegment{Text: text[cursor:]})
	}

	if len(segments) == 0 && text != "" {
		segments = append(segments, HighlightSegment{Text: text})
	}

	return segments
}
