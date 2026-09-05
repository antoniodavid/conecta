package vpn

import (
	"testing"
)

func TestCountryFlag(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"USA", "🇺🇸"},
		{"United States", "🇺🇸"},
		{"Spain", "🇪🇸"},
		{"España", "🇪🇸"},
		{"Germany", "🇩🇪"},
		{"Alemania", "🇩🇪"},
		{"France", "🇫🇷"},
		{"Francia", "🇫🇷"},
		{"UK", "🇬🇧"},
		{"Canada", "🇨🇦"},
		{"Mexico", "🇲🇽"},
		{"México", "🇲🇽"},
		{"Netherlands", "🇳🇱"},
		{"Italy", "🇮🇹"},
		{"Italia", "🇮🇹"},
		{"Brazil", "🇧🇷"},
		{"Brasil", "🇧🇷"},
		{"Argentina", "🇦🇷"},
		{"Chile", "🇨🇱"},
		{"Colombia", "🇨🇴"},
		{"Portugal", "🇵🇹"},
		{"Sweden", "🇸🇪"},
		{"Suecia", "🇸🇪"},
		{"Norway", "🇳🇴"},
		{"Noruega", "🇳🇴"},
		{"Switzerland", "🇨🇭"},
		{"Suiza", "🇨🇭"},
		{"Japan", "🇯🇵"},
		{"Japón", "🇯🇵"},
		{"Australia", "🇦🇺"},
		{"us", "🇺🇸"},
		{"ES", "🇪🇸"},
		{"  Spain  ", "🇪🇸"},
		{"spain", "🇪🇸"},
		{"", ""},
		{"Atlantis", ""},
		{"Binhex-Spain", "🇪🇸"},
		{"USA Premium", "🇺🇸"},
		{"My UK Server", "🇬🇧"},
		{"deutschland-vpn", ""},
	}
	for _, tt := range tests {
		t.Run("flag "+tt.name, func(t *testing.T) {
			if got := CountryFlag(tt.name); got != tt.want {
				t.Fatalf("CountryFlag(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestCountryCode(t *testing.T) {
	if got := CountryCode("USA"); got != "US" {
		t.Fatalf("CountryCode(USA) = %q, want US", got)
	}
	if got := CountryCode("es"); got != "ES" {
		t.Fatalf("bare code must be used verbatim uppercased, got %q", got)
	}
	if got := CountryCode(""); got != "" {
		t.Fatalf("empty must yield empty, got %q", got)
	}
	if got := CountryCode("Atlantis"); got != "" {
		t.Fatalf("unknown must yield empty, got %q", got)
	}
}
