package network

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// RED 2.1: signed dBm is preserved (negative stays negative).
func TestParseSignalDBMPreservesSign(t *testing.T) {
	cases := []struct {
		line string
		want int
	}{
		{"signal: -70 dBm", -70},
		{"signal: -45.5 dBm", -45},
		{"signal: -90 dBm", -90},
	}
	for _, tt := range cases {
		got, err := ParseSignalDBM(tt.line)
		if err != nil {
			t.Fatalf("ParseSignalDBM(%q) error = %v", tt.line, err)
		}
		if got != tt.want {
			t.Fatalf("ParseSignalDBM(%q) = %d, want %d", tt.line, got, tt.want)
		}
		if got >= 0 {
			t.Fatalf("dBm must stay negative, got %d for %q", got, tt.line)
		}
	}
}

func TestDBMToPercentMapping(t *testing.T) {
	if got := DBMToPercent(-30); got != 100 {
		t.Fatalf("DBMToPercent(-30) = %d, want 100", got)
	}
	if got := DBMToPercent(-90); got != 0 {
		t.Fatalf("DBMToPercent(-90) = %d, want 0", got)
	}
	mid := DBMToPercent(-60)
	if mid <= 0 || mid >= 100 {
		t.Fatalf("DBMToPercent(-60) = %d, want strictly between 0 and 100", mid)
	}
}

// RED 2.1: login classifier — unknown/malformed responses fail closed, never false connected.
func TestClassifyLoginUnknownFailsClosed(t *testing.T) {
	cases := []struct {
		name string
		code int
		html string
	}{
		{"empty body", 200, ""},
		{"unknown html", 200, "<html><body>hello world</body></html>"},
		{"redirect without login markers", 200, "<html>random portal page</html>"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			status, err := ClassifyLoginResponse(tt.code, tt.html)
			if err == nil {
				t.Fatalf("ClassifyLoginResponse must fail closed for unknown response (status=%v)", status)
			}
			if status == PortalConnected {
				t.Fatalf("unknown response must never classify as connected")
			}
		})
	}
}

func TestClassifyLoginKnownFailures(t *testing.T) {
	if _, err := ClassifyLoginResponse(200, "usuario no existe"); err == nil {
		t.Fatalf("unknown user must be an error")
	}
	if _, err := ClassifyLoginResponse(200, "LoginServlet form"); err == nil {
		t.Fatalf("login form without success markers must be an error")
	}
}

// RED 2.1: speed rejection — bad status, partial read, empty URLs.
func TestSpeedRejectsBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("oops"))
	}))
	defer srv.Close()
	st := NewSpeedTestWithURLs([]string{srv.URL})
	res := st.Run()
	if res.Error == nil {
		t.Fatalf("speed test must reject HTTP 500")
	}
}

func TestSpeedRejectsPartialRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100000")
		_, _ = w.Write([]byte("short"))
	}))
	defer srv.Close()
	st := NewSpeedTestWithURLs([]string{srv.URL})
	res := st.Run()
	if res.Error == nil {
		t.Fatalf("speed test must reject truncated body (declared 100000, got 5)")
	}
}

func TestSpeedRejectsEmptyURLs(t *testing.T) {
	st := NewSpeedTestWithURLs(nil)
	res := st.Run()
	if res.Error == nil {
		t.Fatalf("speed test with empty URL list must fail")
	}
	_ = io.Discard
}
