package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/app"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
)

func newTestServer(t *testing.T, password string) http.Handler {
	return newTestServerWith(t, password, false)
}

func newTestServerWith(t *testing.T, password string, adminTrustedOnly bool) http.Handler {
	cfg := config.Default()
	cfg.Prepare()
	cfg.Web.AdminTrustedOnly = adminTrustedOnly
	cfg.Users = append(cfg.Users, config.User{Name: "ivan", Secret: config.RandomSecret(), MaxConnections: 5, AllowExternal: true},
		config.User{Name: "off", Secret: config.RandomSecret(), MaxConnections: 0})
	cfg.Access.TrustedNetworks = []string{"loopback"}
	if password != "" {
		cfg.Access.PasswordHash = config.HashPassword(password)
	}
	testApp = app.New(filepath.Join(t.TempDir(), "c.json"), cfg)
	return New(testApp).Handler()
}

var testApp *app.App

func do(h http.Handler, method, path, remote, host, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = remote
	req.Host = host
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAccess(t *testing.T) {
	h := newTestServer(t, "long-password")
	csrf := map[string]string{csrfHeader: "1", "Content-Type": "application/json"}

	if rec := do(h, "GET", "/api/config", "127.0.0.1:5000", "127.0.0.1:8080", "", nil); rec.Code != 200 {
		t.Fatalf("trusted address: %d", rec.Code)
	}
	if rec := do(h, "GET", "/api/config", "203.0.113.5:5000", "proxy.example:8080", "", nil); rec.Code != 401 {
		t.Fatalf("untrusted address without session: %d", rec.Code)
	}
	// DNS rebinding: trusted source address, but a foreign host name
	if rec := do(h, "GET", "/api/config", "127.0.0.1:5000", "evil.example:8080", "", nil); rec.Code != 401 {
		t.Fatalf("foreign host name: %d", rec.Code)
	}
	if rec := do(h, "POST", "/api/restart", "127.0.0.1:5000", "127.0.0.1:8080", "{}", nil); rec.Code != 403 {
		t.Fatalf("mutation without CSRF header: %d", rec.Code)
	}
	cross := map[string]string{csrfHeader: "1", "Origin": "http://evil.example"}
	if rec := do(h, "POST", "/api/logout", "127.0.0.1:5000", "127.0.0.1:8080", "{}", cross); rec.Code != 403 {
		t.Fatalf("cross-origin mutation: %d", rec.Code)
	}

	bad := do(h, "POST", "/api/login", "203.0.113.5:5000", "proxy.example:8080", `{"username":"admin","password":"nope"}`, csrf)
	if bad.Code != 401 {
		t.Fatalf("wrong password: %d", bad.Code)
	}
	ok := do(h, "POST", "/api/login", "203.0.113.5:5000", "proxy.example:8080", `{"username":"admin","password":"long-password"}`, csrf)
	if ok.Code != 200 {
		t.Fatalf("login: %d %s", ok.Code, ok.Body)
	}
	cookie := ok.Result().Cookies()[0]
	withCookie := map[string]string{"Cookie": cookie.Name + "=" + cookie.Value}
	rec := do(h, "GET", "/api/config", "203.0.113.5:5000", "proxy.example:8080", "", withCookie)
	if rec.Code != 200 {
		t.Fatalf("with session: %d", rec.Code)
	}
	var view struct {
		Config      config.Config `json:"config"`
		HasPassword bool          `json:"has_password"`
	}
	json.Unmarshal(rec.Body.Bytes(), &view)
	if view.Config.Access.PasswordHash != "" || !view.HasPassword {
		t.Fatal("password hash must not leave the server")
	}
}

func TestPersonalPageAndQR(t *testing.T) {
	h := newTestServer(t, "")
	ivan, _ := testApp.Config().FindUser("ivan")
	token := config.PageToken(ivan.Secret)
	// the page is opened from anywhere by its token
	rec := do(h, "GET", "/p/"+token, "203.0.113.5:5000", "proxy.example:8080", "", nil)
	body := rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, ivan.Secret) || !strings.Contains(body, "tg://proxy?server=") {
		t.Fatalf("page: %d %s", rec.Code, body)
	}
	if rec.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("the token must not leak through Referer")
	}
	svg := do(h, "GET", "/p/"+token+"/qr.svg", "203.0.113.5:5000", "proxy.example:8080", "", nil)
	if svg.Code != 200 || !strings.HasPrefix(svg.Body.String(), "<svg") || !strings.Contains(svg.Body.String(), "M4 4h7") {
		t.Fatalf("qr.svg: %d %.80s", svg.Code, svg.Body.String())
	}
	if pngRec := do(h, "GET", "/p/"+token+"/qr.png", "203.0.113.5:5000", "", "", nil); pngRec.Code != 200 ||
		!strings.HasPrefix(pngRec.Body.String(), "\x89PNG") {
		t.Fatalf("qr.png: %d", pngRec.Code)
	}
	if rec := do(h, "GET", "/p/0123456789abcdef01234567", "127.0.0.1:1", "127.0.0.1", "", nil); rec.Code != 404 {
		t.Fatalf("unknown token: %d", rec.Code)
	}
	off, _ := testApp.Config().FindUser("off")
	offPage := do(h, "GET", "/p/"+config.PageToken(off.Secret), "127.0.0.1:1", "127.0.0.1", "", nil)
	if offPage.Code != 200 || strings.Contains(offPage.Body.String(), off.Secret) {
		t.Fatal("a disabled user must not get the link")
	}
	if rec := do(h, "GET", "/p/"+config.PageToken(off.Secret)+"/qr.svg", "127.0.0.1:1", "127.0.0.1", "", nil); rec.Code != 404 {
		t.Fatalf("QR of a disabled user: %d", rec.Code)
	}
}

func TestUsersAPI(t *testing.T) {
	h := newTestServer(t, "long-password")
	if rec := do(h, "GET", "/api/users", "203.0.113.5:5000", "proxy.example:8080", "", nil); rec.Code != 401 {
		t.Fatalf("users without login: %d", rec.Code)
	}
	if rec := do(h, "GET", "/api/report", "203.0.113.5:5000", "proxy.example:8080", "", nil); rec.Code != 401 {
		t.Fatalf("report without login: %d", rec.Code)
	}
	csrf := map[string]string{csrfHeader: "1", "Content-Type": "application/json"}
	body := `{"users": [{"name": "maria", "max_connections": 3, "allow_external": true}]}`
	rec := do(h, "PUT", "/api/users", "127.0.0.1:5000", "127.0.0.1:8080", body, csrf)
	if rec.Code != 200 {
		t.Fatalf("put users: %d %s", rec.Code, rec.Body)
	}
	var resp struct {
		Users []userView `json:"users"`
	}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Users) != 1 || resp.Users[0].Name != "maria" || config.ValidateSecret(resp.Users[0].Secret) != nil ||
		!strings.HasPrefix(resp.Users[0].Page, "p/") || resp.Users[0].Link.TG == "" {
		t.Fatalf("users: %+v", resp.Users)
	}
	bad := `{"users": [{"name": "a", "max_connections": 1}, {"name": "A", "max_connections": 1}]}`
	if rec := do(h, "PUT", "/api/users", "127.0.0.1:5000", "127.0.0.1:8080", bad, csrf); rec.Code != 400 {
		t.Fatalf("duplicate names: %d", rec.Code)
	}
	csv := do(h, "GET", "/api/report.csv", "127.0.0.1:5000", "127.0.0.1:8080", "", nil)
	if csv.Code != 200 || !strings.Contains(csv.Body.String(), "maria;3;true;0") {
		t.Fatalf("csv: %s", csv.Body)
	}
}

func TestAdminOnlyFromTrustedNetworks(t *testing.T) {
	h := newTestServerWith(t, "long-password", true)
	csrf := map[string]string{csrfHeader: "1", "Content-Type": "application/json"}
	outside := "203.0.113.5:5000"
	login := do(h, "POST", "/api/login", outside, "proxy.example:8080", `{"username":"admin","password":"long-password"}`, csrf)
	if login.Code != 403 {
		t.Fatalf("login from outside with the right password: %d", login.Code)
	}
	if rec := do(h, "GET", "/api/config", outside, "proxy.example:8080", "", nil); rec.Code != 403 {
		t.Fatalf("admin API from outside: %d", rec.Code)
	}
	var session map[string]interface{}
	json.Unmarshal(do(h, "GET", "/api/session", outside, "proxy.example:8080", "", nil).Body.Bytes(), &session)
	if session["admin_blocked"] != true || session["authorized"] != false {
		t.Fatalf("session: %v", session)
	}
	if rec := do(h, "GET", "/api/config", "127.0.0.1:5000", "127.0.0.1:8080", "", nil); rec.Code != 200 {
		t.Fatalf("admin API inside: %d", rec.Code)
	}
	ivan, _ := testApp.Config().FindUser("ivan")
	if rec := do(h, "GET", "/p/"+config.PageToken(ivan.Secret), outside, "proxy.example:8080", "", nil); rec.Code != 200 {
		t.Fatalf("personal page from outside: %d", rec.Code)
	}
}
