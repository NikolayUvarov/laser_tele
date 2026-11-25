// Package web is the administration interface: status, connection links, settings,
// tests of Cloudflare routes and logs. Trusted addresses work without a password
package web

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/access"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/app"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/proxy"
)

//go:embed static
var staticFiles embed.FS

const (
	sessionCookie   = "lmp_session"
	sessionTTL      = 12 * time.Hour
	loginFailLimit  = 5
	loginBlockTime  = time.Minute
	loginFailDelay  = time.Second
	csrfHeader      = "X-Laser-Mtproxy"
	maxRequestBytes = 1 << 20
)

type loginFails struct {
	count        int
	blockedUntil time.Time
}

// Server is the web interface
type Server struct {
	app *app.App

	mu       sync.Mutex
	sessions map[string]time.Time
	fails    map[string]*loginFails

	srv *http.Server
}

// New creates the web interface for a
func New(a *app.App) *Server {
	return &Server{app: a, sessions: map[string]time.Time{}, fails: map[string]*loginFails{}}
}

// Handler returns the routes of the interface
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(staticFiles, "static")
	fileServer := http.FileServer(http.FS(static))
	mux.Handle("GET /", s.securityHeaders(fileServer))
	mux.HandleFunc("GET /api/session", s.handleSession)
	mux.HandleFunc("POST /api/login", s.mutating(s.handleLogin))
	mux.HandleFunc("POST /api/logout", s.mutating(s.handleLogout))
	mux.HandleFunc("GET /api/status", s.authorized(s.handleStatus))
	mux.HandleFunc("GET /api/config", s.authorized(s.handleGetConfig))
	mux.HandleFunc("PUT /api/config", s.authorized(s.mutating(s.handlePutConfig)))
	mux.HandleFunc("POST /api/restart", s.authorized(s.mutating(s.handleRestart)))
	mux.HandleFunc("POST /api/test", s.authorized(s.mutating(s.handleTest)))
	mux.HandleFunc("GET /api/logs", s.authorized(s.handleLogs))
	mux.HandleFunc("GET /api/logs/download", s.authorized(s.handleLogDownload))
	mux.HandleFunc("GET /api/random-secret", s.authorized(s.handleRandomSecret))
	mux.HandleFunc("GET /api/users", s.authorized(s.handleGetUsers))
	mux.HandleFunc("PUT /api/users", s.authorized(s.mutating(s.handlePutUsers)))
	mux.HandleFunc("GET /api/report", s.authorized(s.handleReport))
	mux.HandleFunc("GET /api/report.csv", s.authorized(s.handleReportCSV))
	mux.HandleFunc("POST /api/report/reset", s.authorized(s.mutating(s.handleReportReset)))
	// personal pages are opened by the secret token in the address, without login
	mux.Handle("GET /p/{token}", s.securityHeaders(http.HandlerFunc(s.handlePersonalPage)))
	mux.HandleFunc("GET /p/{token}/qr.svg", s.handleQRSVG)
	mux.HandleFunc("GET /p/{token}/qr.png", s.handleQRPNG)
	return mux
}

// Start listens on the address of the configuration
func (s *Server) Start() error {
	cfg := s.app.Config()
	addr := net.JoinHostPort(cfg.Web.Host, strconv.Itoa(cfg.Web.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.srv = &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 2 * time.Minute}
	scheme := "http"
	certFile, keyFile := s.app.ResolvePath(cfg.Web.TLSCert), s.app.ResolvePath(cfg.Web.TLSKey)
	if certFile != "" {
		scheme = "https"
	}
	go func() {
		var err error
		if certFile != "" {
			err = s.srv.ServeTLS(ln, certFile, keyFile)
		} else {
			err = s.srv.Serve(ln)
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logx.Errorf("Web interface stopped: %v", err)
		}
	}()
	host := cfg.Web.Host
	if host == "0.0.0.0" || host == "::" {
		host = proxy.OutboundIP()
	}
	logx.Infof("Web interface: %s://%s", scheme, net.JoinHostPort(host, strconv.Itoa(cfg.Web.Port)))
	return nil
}

// Stop stops the web interface
func (s *Server) Stop() {
	if s.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		s.srv.Shutdown(ctx)
	}
}

func clientIP(r *http.Request) net.IP { return access.HostIP(r.RemoteAddr) }

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// hostAllowed protects trusted access from DNS rebinding: a foreign site must not
// read the interface through a name that resolves to this machine
func (s *Server) hostAllowed(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	host = strings.ToLower(strings.Trim(host, "[]"))
	if net.ParseIP(host) != nil || host == "localhost" {
		return true
	}
	cfg := s.app.Config()
	if cfg.PublicHost != "" && strings.EqualFold(host, cfg.PublicHost) {
		return true
	}
	for _, allowed := range cfg.Web.AllowedHosts {
		if strings.EqualFold(host, strings.TrimSpace(allowed)) {
			return true
		}
	}
	if name, err := os.Hostname(); err == nil {
		name = strings.ToLower(name)
		short, _, _ := strings.Cut(name, ".")
		if host == name || host == short || host == short+".local" {
			return true
		}
	}
	return false
}

func (s *Server) validSession(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	until, ok := s.sessions[c.Value]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		delete(s.sessions, c.Value)
		return false
	}
	return true
}

// access returns whether the request may use the interface and whether the address is trusted
func (s *Server) access(r *http.Request) (authorized, trusted bool) {
	trusted = s.app.IsTrusted(clientIP(r)) && s.hostAllowed(r)
	if !trusted && s.adminBlocked() {
		return false, false
	}
	return trusted || s.validSession(r), trusted
}

// adminBlocked reports that untrusted addresses can't use the administration even with the password
func (s *Server) adminBlocked() bool { return s.app.Config().Web.AdminTrustedOnly }

func (s *Server) authorized(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ok, trusted := s.access(r); !ok {
			if !trusted && s.adminBlocked() {
				writeError(w, http.StatusForbidden, "the administration is available only from trusted networks")
				return
			}
			writeError(w, http.StatusUnauthorized, "login required")
			return
		}
		next(w, r)
	}
}

// mutating requires the custom header, which a cross-site form or image can't send
func (s *Server) mutating(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(csrfHeader) != "1" {
			writeError(w, http.StatusForbidden, "missing "+csrfHeader+" header")
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			if !strings.HasSuffix(origin, "://"+r.Host) {
				writeError(w, http.StatusForbidden, "cross-origin request")
				return
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		next(w, r)
	}
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	authorized, trusted := s.access(r)
	cfg := s.app.Config()
	ip := clientIP(r)
	ipStr := ""
	if ip != nil {
		ipStr = ip.String()
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"authorized":    authorized,
		"trusted":       trusted,
		"trusted_ip":    s.app.IsTrusted(ip),
		"host_allowed":  s.hostAllowed(r),
		"client_ip":     ipStr,
		"has_password":  cfg.Access.PasswordHash != "",
		"admin_blocked": !trusted && cfg.Web.AdminTrustedOnly,
		"username":      cfg.Access.Username,
		"language":      cfg.Web.Language,
		"version":       app.Version,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r).String()
	if _, trusted := s.access(r); !trusted && s.adminBlocked() {
		logx.Warnf("[%s] web: login refused, the administration is only for trusted networks", ip)
		writeError(w, http.StatusForbidden, "the administration is available only from trusted networks")
		return
	}
	s.mu.Lock()
	f := s.fails[ip]
	if f != nil && time.Now().Before(f.blockedUntil) {
		s.mu.Unlock()
		writeError(w, http.StatusTooManyRequests, "too many attempts, try again later")
		return
	}
	s.mu.Unlock()

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}
	cfg := s.app.Config()
	ok := cfg.Access.PasswordHash != "" &&
		subtle.ConstantTimeCompare([]byte(req.Username), []byte(cfg.Access.Username)) == 1 &&
		config.CheckPassword(cfg.Access.PasswordHash, req.Password)
	if !ok {
		s.mu.Lock()
		if f == nil {
			f = &loginFails{}
			s.fails[ip] = f
		}
		f.count++
		if f.count >= loginFailLimit {
			f.count = 0
			f.blockedUntil = time.Now().Add(loginBlockTime)
		}
		s.mu.Unlock()
		logx.Warnf("[%s] web: failed login for %q", ip, req.Username)
		time.Sleep(loginFailDelay)
		writeError(w, http.StatusUnauthorized, "wrong username or password")
		return
	}
	token := make([]byte, 32)
	rand.Read(token)
	value := hex.EncodeToString(token)
	s.mu.Lock()
	delete(s.fails, ip)
	now := time.Now()
	for k, until := range s.sessions {
		if now.After(until) {
			delete(s.sessions, k)
		}
	}
	s.sessions[value] = now.Add(sessionTTL)
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: value, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil, MaxAge: int(sessionTTL.Seconds())})
	logx.Infof("[%s] web: login of %q", ip, req.Username)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.mu.Lock()
		delete(s.sessions, c.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// linkHost is the address the browser used, so links work from the same network
func linkHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	host = strings.Trim(host, "[]")
	if ip := net.ParseIP(host); (ip != nil && ip.IsLoopback()) || host == "localhost" || host == "" {
		return ""
	}
	return host
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.app.Status(linkHost(r)))
}

type configView struct {
	Config           *config.Config `json:"config"`
	HasPassword      bool           `json:"has_password"`
	DefaultCFDomains int            `json:"default_cf_domains"`
	InterfaceNets    []string       `json:"interface_networks"`
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.app.Config()
	hasPassword := cfg.Access.PasswordHash != ""
	cfg.Access.PasswordHash = ""
	cfg.Access.Password = ""
	var nets []string
	seen := map[string]bool{}
	for _, n := range access.InterfaceNetworks() {
		if s := n.String(); !seen[s] {
			seen[s] = true
			nets = append(nets, s)
		}
	}
	writeJSON(w, http.StatusOK, configView{Config: cfg, HasPassword: hasPassword,
		DefaultCFDomains: len(proxy.DefaultCFDomains()), InterfaceNets: nets})
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Config        *config.Config `json:"config"`
		NewPassword   string         `json:"new_password"`
		ClearPassword bool           `json:"clear_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Config == nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}
	current := s.app.Config()
	cfg := req.Config
	cfg.Access.Password = ""
	switch {
	case req.ClearPassword:
		cfg.Access.PasswordHash = ""
	case req.NewPassword != "":
		if len(req.NewPassword) < 8 {
			writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		cfg.Access.PasswordHash = config.HashPassword(req.NewPassword)
	default:
		cfg.Access.PasswordHash = current.Access.PasswordHash
	}
	// a client from an untrusted address must not lock itself out by removing the password
	if _, trusted := s.access(r); !trusted && cfg.Access.PasswordHash == "" {
		writeError(w, http.StatusBadRequest, "the password can be removed only from a trusted network")
		return
	}
	ip := clientIP(r)
	webRestart, err := s.app.Apply(cfg)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	logx.Infof("[%s] web: configuration changed", ip)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true, "web_restart_required": webRestart})
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	s.app.Restart()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Kind    string   `json:"kind"` // cfproxy, cfproxy_auto, cfworker, direct
		Domains []string `json:"domains"`
		Secure  bool     `json:"secure"`
		DCIP    []string `json:"dc_ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	domains := config.CoerceDomainList(req.Domains)
	for _, d := range domains {
		if !config.IsValidDomain(strings.ToLower(d)) {
			writeError(w, http.StatusBadRequest, "wrong domain "+d)
			return
		}
	}
	switch req.Kind {
	case "cfproxy", "cfworker":
		if len(domains) == 0 {
			writeError(w, http.StatusBadRequest, "no domains")
			return
		}
		results := make([]proxy.DomainTest, len(domains))
		var wg sync.WaitGroup
		for i, d := range domains {
			wg.Add(1)
			go func(i int, d string) {
				defer wg.Done()
				if req.Kind == "cfproxy" {
					results[i] = proxy.TestCFProxyDomain(ctx, strings.ToLower(d), req.Secure)
				} else {
					results[i] = proxy.TestCFWorkerDomain(ctx, strings.ToLower(d), req.Secure)
				}
			}(i, d)
		}
		wg.Wait()
		writeJSON(w, http.StatusOK, map[string]interface{}{"domains": results})
	case "cfproxy_auto":
		best, merged := proxy.TestCFProxyAuto(ctx, proxy.DefaultCFDomains(), req.Secure)
		ok := 0
		for _, r := range merged {
			if r.OK {
				ok++
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"best": best != "", "results": merged,
			"ok_count": ok, "total": len(merged)})
	case "direct":
		results, err := proxy.TestDirectRoute(ctx, req.DCIP)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
	default:
		writeError(w, http.StatusBadRequest, "unknown test")
	}
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	after, _ := strconv.ParseUint(r.URL.Query().Get("after"), 10, 64)
	entries := logx.Since(after, 1000)
	writeJSON(w, http.StatusOK, map[string]interface{}{"entries": entries})
}

func (s *Server) handleLogDownload(w http.ResponseWriter, r *http.Request) {
	path := s.app.LogFile()
	if path == "" {
		writeError(w, http.StatusNotFound, "log file is disabled")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "can't open log file")
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="laser-mtproxy.log"`)
	info, _ := f.Stat()
	http.ServeContent(w, r, "laser-mtproxy.log", info.ModTime(), f)
}

func (s *Server) handleRandomSecret(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"secret": config.RandomSecret()})
}
