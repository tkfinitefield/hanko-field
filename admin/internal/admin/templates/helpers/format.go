package helpers

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
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
	base := []string{"badge"}
	switch tone {
	case "success":
		base = append(base, "badge-success")
	case "warning":
		base = append(base, "badge-warning")
	case "danger":
		base = append(base, "badge-danger")
	case "info":
		base = append(base, "badge-info")
	default:
		// neutral badge uses base styling
	}
	return ClassList(base...)
}

// ButtonClass returns the composed class string for a button variant/size combination.
func ButtonClass(variant, size string, fullWidth, loading bool) string {
	if variant == "" {
		variant = "primary"
	}
	if size == "" {
		size = "md"
	}
	classes := []string{"btn", "btn-" + variant, "btn-" + size}
	if fullWidth {
		classes = append(classes, "w-full")
	}
	if loading {
		classes = append(classes, "btn-loading")
	}
	return ClassList(classes...)
}

// ModalPanelClass returns the class string for the modal panel.
func ModalPanelClass(size string) string {
	classes := []string{"modal-panel"}
	if size == "lg" || size == "large" {
		classes = append(classes, "modal-lg")
	}
	return ClassList(classes...)
}

// ToastClass maps tones to toast UI classes.
func ToastClass(tone string) string {
	classes := []string{"toast"}
	switch tone {
	case "success":
		classes = append(classes, "toast-success")
	case "danger", "error":
		classes = append(classes, "toast-danger")
	case "warning":
		classes = append(classes, "toast-warning")
	case "info":
		classes = append(classes, "toast-info")
	default:
		classes = append(classes, "toast-info")
	}
	return ClassList(classes...)
}

// ClassList joins non-empty class names safely.
func ClassList(classes ...string) string {
	result := make([]string, 0, len(classes))
	for _, c := range classes {
		if strings.TrimSpace(c) == "" {
			continue
		}
		result = append(result, c)
	}
	return strings.Join(result, " ")
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

// SetRawQuery returns a new raw query string with the provided key set to the supplied value.
func SetRawQuery(rawQuery, key, value string) string {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		values = url.Values{}
	}
	values.Set(key, value)
	return values.Encode()
}

// DelRawQuery removes the provided key from the raw query string.
func DelRawQuery(rawQuery, key string) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}
	values.Del(key)
	return values.Encode()
}

// BuildURL combines a path and raw query string into a URL, preserving original encoding where possible.
func BuildURL(path, rawQuery string) string {
	if path == "" {
		path = "."
	}
	u, err := url.Parse(path)
	if err != nil {
		if rawQuery == "" {
			return path
		}
		return path + "?" + rawQuery
	}

	if rawQuery == "" {
		u.RawQuery = ""
		return u.String()
	}

	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		u.RawQuery = rawQuery
		return u.String()
	}
	u.RawQuery = values.Encode()
	return u.String()
}
