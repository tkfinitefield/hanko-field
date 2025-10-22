package services

import (
	"context"
	"strings"
	"time"
	"unicode"
)

type heuristicRegistrabilityEvaluator struct {
	clock func() time.Time
}

// NewHeuristicRegistrabilityEvaluator returns a simple, synchronous evaluator that applies
// lightweight heuristics to approximate registrability without making external calls.
// It enables the registrability endpoint in environments where the real provider is
// unavailable while still producing actionable feedback for clients.
func NewHeuristicRegistrabilityEvaluator(clock func() time.Time) RegistrabilityEvaluator {
	if clock == nil {
		clock = time.Now
	}
	return &heuristicRegistrabilityEvaluator{
		clock: func() time.Time {
			return clock().UTC()
		},
	}
}

func (e *heuristicRegistrabilityEvaluator) Check(_ context.Context, payload RegistrabilityCheckPayload) (RegistrabilityAssessment, error) {
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return RegistrabilityAssessment{}, ErrRegistrabilityInvalidInput
	}

	now := e.clock()
	reasons := make([]string, 0, 2)

	runes := []rune(name)
	length := len(runes)
	var disallowed int
	for _, r := range runes {
		if unicode.IsSymbol(r) || unicode.IsPunct(r) {
			disallowed++
		}
	}

	lines := make([]string, 0, len(payload.TextLines))
	for _, line := range payload.TextLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	passed := true
	status := "pass"

	if length < 2 {
		passed = false
		status = "review"
		reasons = append(reasons, "name too short for official seal")
	}
	if length > 12 {
		passed = false
		status = "review"
		reasons = append(reasons, "name may exceed typical seal bounds")
	}
	if disallowed > 0 {
		passed = false
		status = "fail"
		reasons = append(reasons, "name contains unsupported symbols")
	}
	if len(lines) > 0 && len(lines) != len(payload.TextLines) {
		passed = false
		status = "review"
		reasons = append(reasons, "blank engraving lines detected")
	}

	// Basic confidence score: start from 1.0 and subtract penalties.
	score := 1.0
	if length < 2 {
		score -= 0.4
	}
	if length > 12 {
		score -= 0.2
	}
	if disallowed > 0 {
		score -= 0.5
	}
	if score < 0 {
		score = 0
	}

	expiry := now.Add(12 * time.Hour)

	return RegistrabilityAssessment{
		Status:    status,
		Passed:    passed,
		Score:     &score,
		Reasons:   reasons,
		ExpiresAt: &expiry,
		Metadata: map[string]any{
			"method":               "heuristic",
			"nameLength":           length,
			"disallowedCharacters": disallowed,
			"lines":                lines,
		},
	}, nil
}
