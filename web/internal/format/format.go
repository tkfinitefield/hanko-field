package format

import (
    "fmt"
    "strings"
    "time"
)

// Currency formats amount in minor units for basic currencies.
// Example: FmtCurrency(12345, "JPY", "ja") => "¥12345" (no thousands separator for JPY)
func FmtCurrency(minor int64, currency, lang string) string {
    currency = strings.ToUpper(currency)
    switch currency {
    case "JPY":
        // Japanese Yen is commonly displayed without a thousands separator in Japan.
        // We intentionally avoid thousand separators here to match local expectations.
        return fmt.Sprintf("¥%d", minor)
    case "USD":
        // assume cents; format with 2 decimals
        neg := minor < 0
        if neg { minor = -minor }
        major := minor / 100
        cents := minor % 100
        head := thousandSep(major)
        tail := fmt.Sprintf("%02d", cents)
        if neg { return "-$" + head + "." + tail }
        return "$" + head + "." + tail
    default:
        // generic minor units
        return fmt.Sprintf("%s %s", currency, thousandSep(minor))
    }
}

func thousandSep(n int64) string {
    s := fmt.Sprintf("%d", n)
    neg := false
    if strings.HasPrefix(s, "-") { neg = true; s = s[1:] }
    out := ""
    for i, c := range s {
        if i != 0 && (len(s)-i)%3 == 0 { out += "," }
        out += string(c)
    }
    if neg { return "-" + out }
    return out
}

// Date formats time in a locale-friendly short form.
func FmtDate(t time.Time, lang string) string {
    switch strings.ToLower(lang) {
    case "ja":
        return t.Format("2006-01-02")
    default:
        return t.Format("Jan 2, 2006")
    }
}
