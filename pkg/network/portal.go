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

// portalProbeTimeout caps portal page GETs: on networks where the captive
// portal is unreachable (e.g. a plain home LAN), a hung probe must not stall
// status/logout for the full main-client timeout.
const portalProbeTimeout = 5 * time.Second

// Portal handles ETECSA captive portal operations
type Portal struct {
	config *NetworkConfig
	client *http.Client
}

// probeClient returns a short-timeout client for portal page probes,
// mirroring the main client's cookie jar and redirect policy.
func (p *Portal) probeClient() *http.Client {
	return &http.Client{
		Timeout:       portalProbeTimeout,
		Jar:           p.client.Jar,
		CheckRedirect: p.client.CheckRedirect,
	}
}

// fetchPortalPage GETs the portal root with the short probe client and
// returns the page body.
func (p *Portal) fetchPortalPage() (string, error) {
	resp, err := p.probeClient().Get(p.config.PortalURL + "/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return string(body), nil
}

// classifyPortalPage maps a portal page body to a status. ETECSA session
// pages carry "ya está conectado"/"ya conectado"; login pages carry
// "LoginServlet"/"Bienvenido". Session markers win when both appear; any
// other page is PortalNone (no portal state to report).
func classifyPortalPage(html string) PortalStatus {
	if hasSessionMarkers(html) {
		return PortalConnected
	}
	if strings.Contains(html, "LoginServlet") || strings.Contains(html, "Bienvenido") {
		return PortalNeedsAuth
	}
	return PortalNone
}

// hasSessionMarkers reports whether a portal body carries the ETECSA
// active-session markers.
func hasSessionMarkers(html string) bool {
	return strings.Contains(html, "ya está conectado") || strings.Contains(html, "ya conectado")
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

// CheckPortal checks the status of the captive portal. The portal page is
// probed first with a short timeout, and the verdict is decided by
// portalVerdict: the page alone is not the whole truth, because ETECSA
// binds the session to the account/IP rather than to a persistent client
// cookie — a fresh (cookie-less) client can receive the login form while
// the session is live. Real internet connectivity resolves those ambiguous
// cases; it is probed only when the page did not already prove a session.
func (p *Portal) CheckPortal() (*Connection, error) {
	conn := &Connection{
		Gateway:   p.config.Gateway,
		Interface: p.config.Interface,
		PortalURL: p.config.PortalURL,
		LastCheck: time.Now(),
	}

	page, err := p.fetchPortalPage()
	if err != nil {
		conn.Status = portalVerdict(PortalNone, err, p.hasInternetConnectivity())
		if conn.Status == PortalError {
			conn.LastError = err
		}
		return conn, nil // Return without error, status indicates failure
	}

	status := classifyPortalPage(page)
	// Session markers are conclusive; only login-form and unrecognized pages
	// need the connectivity tie-breaker (which costs up to ~6s, so skip it
	// on the fast path).
	online := status != PortalConnected && p.hasInternetConnectivity()
	conn.Status = portalVerdict(status, nil, online)
	return conn, nil
}

// portalVerdict is the pure CheckPortal decision table. pageStatus is the
// classified portal page and is meaningful only when pageErr == nil (a nil
// pageErr means the portal GET succeeded); online reports whether real
// internet works. The caller keeps the fetch error when PortalError wins.
//
//	GET success, session markers      -> connected (session page is proof)
//	GET success, login form,  online  -> connected (account/IP-bound
//	                                     session; fresh client lacks cookie)
//	GET success, login form,  offline -> needs auth
//	GET success, unrecognized, online -> connected (reachable portal on a
//	                                     live ETECSA session)
//	GET success, unrecognized, offline -> no portal
//	GET failure, online               -> no portal (not an ETECSA captive
//	                                     network; portal simply not there)
//	GET failure, offline              -> PortalError
func portalVerdict(pageStatus PortalStatus, pageErr error, online bool) PortalStatus {
	if pageErr != nil {
		if online {
			return PortalNone
		}
		return PortalError
	}
	switch pageStatus {
	case PortalConnected:
		return PortalConnected
	case PortalNeedsAuth:
		if online {
			return PortalConnected
		}
		return PortalNeedsAuth
	default: // PortalNone (classifyPortalPage never yields PortalError)
		if online {
			return PortalConnected
		}
		return PortalNone
	}
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
		"currentURL":  {extractInput(html, "currentURL")},
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

	conn, err = p.parseLoginResponse(resp2.StatusCode, respHTML, conn)
	if err != nil {
		return conn, err
	}
	// parseLoginResponse returns nil only for PortalConnected (fast-path
	// markers). Never claim success from the POST alone: the final verdict
	// needs the POST body and a session re-check.
	if verr := loginPostVerdict(respHTML, p.fetchPortalPage); verr != nil {
		conn.Status = PortalError
		conn.LastError = verr
		return conn, verr
	}
	return conn, nil
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
	resp2.Body.Close()

	// Never claim success from the POST alone: verify the session actually
	// ended by re-reading the portal page (short probe client, shared jar).
	page, err := p.fetchPortalPage()
	if err != nil {
		return fmt.Errorf("cannot verify logout: %w", err)
	}
	return logoutVerdict(page)
}

// logoutVerdict decides whether a post-logout portal page proves the
// session ended. Only a login page (needs auth) is proof: a session page
// means the logout did not take effect, and an unrecognized page fails
// closed (success is never claimed without evidence).
func logoutVerdict(page string) error {
	switch classifyPortalPage(page) {
	case PortalNeedsAuth:
		return nil
	case PortalConnected:
		return fmt.Errorf("logout failed: portal session still active")
	default:
		return fmt.Errorf("cannot verify logout: portal returned an unrecognized page")
	}
}

// loginVerdict decides whether a post-login portal page proves the session
// established. Only a session page (connected) is proof: a login page means
// the credentials were rejected or the session did not establish, and an
// unrecognized page fails closed (success is never claimed without evidence).
func loginVerdict(page string) error {
	switch classifyPortalPage(page) {
	case PortalConnected:
		return nil
	case PortalNeedsAuth:
		return fmt.Errorf("login failed: portal still requires authentication — check the username and password")
	default:
		return fmt.Errorf("login failed: portal returned an unrecognized page")
	}
}

// loginPostVerdict decides the real login outcome after the POST response
// was classified as connected. Evidence is the POST body plus a session
// re-check (fetch):
//   - POST body with active-session markers -> the account already holds an
//     active session elsewhere; this client got no session, so fail closed
//     with a specific error instead of claiming success.
//   - otherwise re-check the portal page: a session page is the only
//     success proof; a login page points at bad credentials; an unrecognized
//     page or a fetch failure fails closed.
func loginPostVerdict(postBody string, fetch func() (string, error)) error {
	if hasSessionMarkers(postBody) {
		return fmt.Errorf("login failed: the account already has an active session — log out on the other device or wait for it to expire")
	}
	return verifyLogin(fetch)
}

// verifyLogin re-reads the portal page after a login POST to confirm the
// session actually established. fetch is injected so tests can exercise
// every verdict without the network; a fetch failure fails closed.
func verifyLogin(fetch func() (string, error)) error {
	page, err := fetch()
	if err != nil {
		return fmt.Errorf("cannot verify login: %w", err)
	}
	return loginVerdict(page)
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
