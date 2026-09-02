package network

import (
	"crypto/tls"
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
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
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

// Login authenticates with the captive portal
func (p *Portal) Login(user, pass string) (*Connection, error) {
	conn := &Connection{
		Username:  user,
		Gateway:   p.config.Gateway,
		Interface: p.config.Interface,
		PortalURL: p.config.PortalURL,
		LastCheck: time.Now(),
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

	// Submit login
	req, _ := http.NewRequest("POST", p.config.PortalURL+"//LoginServlet", strings.NewReader(form.Encode()))
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
	req, _ := http.NewRequest("POST", p.config.PortalURL+"//LogoutServlet", nil)
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

	if resp2.StatusCode == 302 || resp2.StatusCode == 301 ||
		strings.Contains(html, "LoginServlet") || strings.Contains(html, "Bienvenido") {
		return nil
	}

	return fmt.Errorf("logout failed: HTTP %d", resp2.StatusCode)
}

// parseLoginResponse parses the login response and returns connection status
func (p *Portal) parseLoginResponse(code int, html string, conn *Connection) (*Connection, error) {
	lower := strings.ToLower(html)

	if strings.Contains(html, "ya está conectado") || strings.Contains(html, "ya conectado") {
		conn.Status = PortalConnected
		return conn, nil
	}

	if strings.Contains(lower, "logout") || strings.Contains(lower, "desconectar") ||
		strings.Contains(lower, "tiempo restante") || strings.Contains(lower, "bytes") {
		conn.Status = PortalConnected
		return conn, nil
	}

	if code == 302 || code == 301 {
		conn.Status = PortalConnected
		return conn, nil
	}

	if strings.Contains(html, "alert(") {
		if m := regexp.MustCompile(`alert\("([^"]+)"\)`).FindStringSubmatch(html); len(m) > 1 {
			conn.Status = PortalError
			conn.LastError = fmt.Errorf("%s", m[1])
			return conn, conn.LastError
		}
	}

	if strings.Contains(lower, "usuario no existe") {
		conn.Status = PortalError
		conn.LastError = fmt.Errorf("usuario no existe")
		return conn, conn.LastError
	}

	if strings.Contains(lower, "contraseña incorrecta") || strings.Contains(lower, "invalid password") {
		conn.Status = PortalError
		conn.LastError = fmt.Errorf("contraseña incorrecta")
		return conn, conn.LastError
	}

	if strings.Contains(html, "LoginServlet") {
		conn.Status = PortalError
		conn.LastError = fmt.Errorf("credenciales inválidas")
		return conn, conn.LastError
	}

	conn.Status = PortalConnected
	return conn, nil
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
