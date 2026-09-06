package network

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Portal handles ETECSA captive portal operations
type Portal struct {
	config *NetworkConfig
	client *http.Client
}

// NewPortal creates a new portal handler
func NewPortal(config *NetworkConfig) *Portal {
	if config == nil {
		config = DefaultConfig()
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: config.Timeout,
		Jar:     jar,
		// TLS certificates are verified by default. No InsecureSkipVerify:
		// captive-portal exceptions must be explicit, scoped, and authorized.
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	return &Portal{
		config: config,
		client: client,
	}
}

// CheckPortal checks the status of the captive portal
func (p *Portal) CheckPortal() (*Connection, error) {
	conn := &Connection{
		Gateway:   p.config.Gateway,
		Interface: p.config.Interface,
		PortalURL: p.config.PortalURL,
		LastCheck: time.Now(),
	}

	// First, check if we have real internet connectivity
	if p.hasInternetConnectivity() {
		conn.Status = PortalConnected
		return conn, nil
	}

	// No internet - check portal status
	resp, err := p.client.Get(p.config.PortalURL + "/")
	if err != nil {
		conn.Status = PortalError
		conn.LastError = err
		return conn, nil // Return without error, status indicates failure
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	html := string(body)

	if strings.Contains(html, "ya está conectado") || strings.Contains(html, "ya conectado") {
		conn.Status = PortalConnected
	} else if strings.Contains(html, "LoginServlet") || strings.Contains(html, "Bienvenido") {
		conn.Status = PortalNeedsAuth
	} else {
		conn.Status = PortalNone
	}

	return conn, nil
}

// hasInternetConnectivity checks if we have real internet access
func (p *Portal) hasInternetConnectivity() bool {
	// Try to reach Google's connectivity check (returns 204 when connected)
	client := &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Check 1: Google connectivity check
	resp, err := client.Get("http://connectivitycheck.gstatic.com/generate_204")
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == 204 {
			return true
		}
	}

	// Check 2: Try reaching a known HTTP endpoint
	resp2, err := client.Get("http://httpbin.org/ip")
	if err == nil {
		resp2.Body.Close()
		if resp2.StatusCode == 200 {
			return true
		}
	}

	return false
}

// Login authenticates with the captive portal
func (p *Portal) Login(user, pass string) (*Connection, error) {
	if p == nil || p.config == nil || p.client == nil {
		return &Connection{Status: PortalError, LastCheck: time.Now()},
			fmt.Errorf("portal client not initialized")
	}
	conn := &Connection{
		Gateway:   p.config.Gateway,
		Interface: p.config.Interface,
		PortalURL: p.config.PortalURL,
		LastCheck: time.Now(),
	}
	// Never retain credentials on the connection: they must not leak into
	// logs, JSON responses, or examples. Username is intentionally omitted.

	if p == nil || p.config == nil || p.client == nil {
		conn.Status = PortalError
		conn.LastError = fmt.Errorf("portal client not initialized")
		return conn, conn.LastError
	}

	// Get login page for hidden fields
	resp, err := p.client.Get(p.config.PortalURL + "/")
	if err != nil {
		conn.Status = PortalError
		conn.LastError = fmt.Errorf("portal unreachable: %w", err)
		return conn, conn.LastError
	}
	defer resp.Body.Close()

	pageBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	html := string(pageBody)

	// Build form data
	form := url.Values{
		"username":    {user},
		"password":    {pass},
		"wlanuserip":  {extractInput(html, "wlanuserip")},
		"wlanacname":  {extractInput(html, "wlanacname")},
		"wlanmac":     {extractInput(html, "wlanmac")},
		"firsturl":    {extractInput(html, "firsturl")},
		"ssid":        {extractInput(html, "ssid")},
		"usertype":    {extractInput(html, "usertype")},
		"gotopage":    {extractInput(html, "gotopage")},
		"successpage": {extractInput(html, "successpage")},
		"loggerId":    {extractInput(html, "loggerId")},
		"lang":        {"es_ES"},
		"CSRFHW":      {extractInput(html, "CSRFHW")},
		"Enviar":      {"Aceptar"},
	}

	// Submit login (nil-request guard: malformed PortalURL fails closed, no credentials sent).
	req, err := http.NewRequest("POST", p.config.PortalURL+"//LoginServlet", strings.NewReader(form.Encode()))
	if err != nil || req == nil {
		conn.Status = PortalError
		conn.LastError = fmt.Errorf("invalid portal request")
		return conn, conn.LastError
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux) AppleWebKit/537.36")
	req.Header.Set("Origin", p.config.PortalURL)
	req.Header.Set("Referer", p.config.PortalURL+"/")

	resp2, err := p.client.Do(req)
	if err != nil {
		conn.Status = PortalError
		conn.LastError = fmt.Errorf("login request failed: %w", err)
		return conn, conn.LastError
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(io.LimitReader(resp2.Body, 1<<20))
	respHTML := string(body2)

	return p.parseLoginResponse(resp2.StatusCode, respHTML, conn)
}

// Logout terminates the session
func (p *Portal) Logout() error {
	// Get session
	resp, err := p.client.Get(p.config.PortalURL + "/")
	if err != nil {
		return fmt.Errorf("cannot access portal: %w", err)
	}
	resp.Body.Close()

	// POST logout
	req, err := http.NewRequest("POST", p.config.PortalURL+"//LogoutServlet", nil)
	if err != nil || req == nil {
		return fmt.Errorf("invalid portal request")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux) AppleWebKit/537.36")
	req.Header.Set("Origin", p.config.PortalURL)
	req.Header.Set("Referer", p.config.PortalURL+"/")

	resp2, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("logout request failed: %w", err)
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(io.LimitReader(resp2.Body, 1<<20))
	html := string(body2)

	if logoutSucceeded(resp2.StatusCode, html) {
		return nil
	}

	return fmt.Errorf("logout failed: HTTP %d", resp2.StatusCode)
}

// logoutSucceeded classifies a logout response as success. ETECSA answers
// HTTP 200 (markerless) on logout, while other paths redirect (301/302) or
// return the login-page markers. Any 2xx is treated as success.
func logoutSucceeded(code int, html string) bool {
	return code == 301 || code == 302 ||
		(code >= 200 && code < 300) ||
		strings.Contains(html, "LoginServlet") ||
		strings.Contains(html, "Bienvenido")
}

// ClassifyLoginResponse is the pure login classifier: known success markers map
// to connected, known failure markers map to errors, and anything unknown
// fails closed (never false connected).
func ClassifyLoginResponse(code int, html string) (PortalStatus, error) {
	lower := strings.ToLower(html)

	if strings.Contains(html, "ya está conectado") || strings.Contains(html, "ya conectado") {
		return PortalConnected, nil
	}
	if strings.Contains(lower, "logout") || strings.Contains(lower, "desconectar") ||
		strings.Contains(lower, "tiempo restante") || strings.Contains(lower, "bytes") {
		return PortalConnected, nil
	}
	if code == 302 || code == 301 {
		return PortalConnected, nil
	}
	if strings.Contains(html, "alert(") {
		if m := regexp.MustCompile(`alert\("([^"]+)"\)`).FindStringSubmatch(html); len(m) > 1 {
			return PortalError, fmt.Errorf("%s", m[1])
		}
	}
	if strings.Contains(lower, "usuario no existe") {
		return PortalError, fmt.Errorf("usuario no existe")
	}
	if strings.Contains(lower, "contraseña incorrecta") || strings.Contains(lower, "invalid password") {
		return PortalError, fmt.Errorf("contraseña incorrecta")
	}
	if strings.Contains(html, "LoginServlet") {
		return PortalError, fmt.Errorf("credenciales inválidas")
	}
	return PortalError, fmt.Errorf("unrecognized portal response")
}

// parseLoginResponse parses the login response and returns connection status
func (p *Portal) parseLoginResponse(code int, html string, conn *Connection) (*Connection, error) {
	status, err := ClassifyLoginResponse(code, html)
	conn.Status = status
	conn.LastError = err
	return conn, err
}

// extractInput extracts hidden form field values
func extractInput(html, name string) string {
	patterns := []string{
		fmt.Sprintf(`name=['"]%s['"][^>]*value=['"]([^'"]*)['"]`, name),
		fmt.Sprintf(`value=['"]([^'"]*)['"][^>]*name=['"]%s['"]`, name),
	}
	for _, p := range patterns {
		if m := regexp.MustCompile(p).FindStringSubmatch(html); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}
