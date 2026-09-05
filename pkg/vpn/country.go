package vpn

import (
	"strings"
	"unicode/utf8"
)

// countryByName maps lowercase country names (English + Spanish aliases)
// to ISO 3166-1 alpha-2 codes.
var countryByName = map[string]string{
	"usa":                      "US",
	"united states":            "US",
	"united states of america": "US",
	"spain":                    "ES",
	"españa":                   "ES",
	"espana":                   "ES",
	"germany":                  "DE",
	"alemania":                 "DE",
	"france":                   "FR",
	"francia":                  "FR",
	"uk":                       "GB",
	"united kingdom":           "GB",
	"canada":                   "CA",
	"mexico":                   "MX",
	"méxico":                   "MX",
	"netherlands":              "NL",
	"holanda":                  "NL",
	"italy":                    "IT",
	"italia":                   "IT",
	"brazil":                   "BR",
	"brasil":                   "BR",
	"argentina":                "AR",
	"chile":                    "CL",
	"colombia":                 "CO",
	"portugal":                 "PT",
	"sweden":                   "SE",
	"suecia":                   "SE",
	"norway":                   "NO",
	"noruega":                  "NO",
	"switzerland":              "CH",
	"suiza":                    "CH",
	"japan":                    "JP",
	"japón":                    "JP",
	"japon":                    "JP",
	"australia":                "AU",
}

// CountryCode maps a profile name to an ISO 3166-1 alpha-2 code.
// Exact (case-insensitive) name match wins, then the longest table key
// contained in the name (so "Binhex-Spain" maps to ES); a bare 2-letter
// code is used verbatim when it looks like A-Z letters.
// Unknown/empty yields "".
func CountryCode(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	lowered := strings.ToLower(trimmed)
	if iso, ok := countryByName[lowered]; ok {
		return iso
	}
	best, bestLen := "", 0
	for key, iso := range countryByName {
		if len(key) > bestLen && strings.Contains(lowered, key) {
			best, bestLen = iso, len(key)
		}
	}
	if best != "" {
		return best
	}
	if len(trimmed) == 2 && isASCIILetters(trimmed) {
		return strings.ToUpper(trimmed)
	}
	return ""
}

// CountryFlag maps a profile name to a regional-indicator flag emoji.
// Unknown/empty yields "".
func CountryFlag(name string) string {
	return flagForISO(CountryCode(name))
}

func isASCIILetters(s string) bool {
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return utf8.RuneCountInString(s) == 2
}

// flagForISO converts an ISO alpha-2 code to regional indicators.
func flagForISO(iso string) string {
	if len(iso) != 2 {
		return ""
	}
	upper := strings.ToUpper(iso)
	var out []rune
	for _, c := range upper {
		if c < 'A' || c > 'Z' {
			return ""
		}
		out = append(out, 0x1F1E6+rune(c-'A'))
	}
	return string(out)
}
