package helpers

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/a-h/templ"
)

// Currency formats amounts (in minor units) with the given ISO currency code.
func Currency(amount int64, currency string) string {
	symbol := currencySymbol(currency)

	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}

	major := amount / 100
	minor := amount % 100

	return fmt.Sprintf("%s%s%d.%02d", sign, symbol, major, minor)
}

// Date formats the timestamp in the provided layout (defaults to 2006-01-02 15:04 MST).
func Date(ts time.Time, layout string) string {
	if layout == "" {
		layout = "2006-01-02 15:04 MST"
	}
	return ts.In(time.Local).Format(layout)
}

// Relative returns a coarse "time ago" string.
func Relative(ts time.Time) string {
	now := time.Now()
	diff := now.Sub(ts)
	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	return ts.Format("2006-01-02")
}

// I18N is a placeholder translation helper.
func I18N(key string, args ...any) string {
	if len(args) == 0 {
		return key
	}
	return fmt.Sprintf(key, args...)
}

func currencySymbol(code string) string {
	switch code {
	case "JPY":
		return "¥"
	case "USD":
		return "$"
	case "EUR":
		return "€"
	default:
		return code + " "
	}
}

// NavClass returns sidebar link classes.
func NavClass(active bool) string {
	if active {
		return "flex items-center gap-2 rounded-md bg-slate-900 px-3 py-2 text-sm font-medium text-white shadow-sm"
	}
	return "flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900"
}

// BadgeClass maps semantic tones to utility classes.
func BadgeClass(tone string) string {
	switch tone {
	case "success":
		return "inline-flex items-center rounded-full bg-emerald-100 px-2 py-1 text-xs font-medium text-emerald-700"
	case "warning":
		return "inline-flex items-center rounded-full bg-amber-100 px-2 py-1 text-xs font-medium text-amber-700"
	case "danger":
		return "inline-flex items-center rounded-full bg-rose-100 px-2 py-1 text-xs font-medium text-rose-700"
	default:
		return "inline-flex items-center rounded-full bg-slate-100 px-2 py-1 text-xs font-medium text-slate-700"
	}
}

// TextComponent returns a templ component that renders plain text.
func TextComponent(value string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, value)
		return err
	})
}

// TableRows converts [][]string to [][]templ.Component for tables.
func TableRows(rows [][]string) [][]templ.Component {
	result := make([][]templ.Component, 0, len(rows))
	for _, row := range rows {
		cells := make([]templ.Component, 0, len(row))
		for _, col := range row {
			cells = append(cells, TextComponent(col))
		}
		result = append(result, cells)
	}
	return result
}
