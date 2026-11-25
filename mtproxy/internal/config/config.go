// Package config is the JSON configuration of laser-mtproxy
package config

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/access"
)

// Config is the whole configuration. JSON names of the MTProto part are the same as in tg-ws-proxy
type Config struct {
	// MTProto proxy for Telegram applications
	Host string `json:"host"` // listen address, 0.0.0.0 = all interfaces
	Port int    `json:"port"` // listen port
	// PublicHost is the address in links of users allowed from outside, empty = the address the web interface was opened with
	PublicHost string `json:"public_host"`
	// Users are clients of the proxy, each with its own secret and limit of connections
	Users []User `json:"users"`
	// Secret and ExternalSecret are the settings of version 1.0, at start they become users "lan" and "external"
	Secret         string `json:"secret,omitempty"`
	ExternalSecret string `json:"external_secret,omitempty"`

	DCIP     []string `json:"dc_ip"`     // "DC:IP" rules for the direct WebSocket route
	BufKB    int      `json:"buf_kb"`    // socket buffers, KB
	PoolSize int      `json:"pool_size"` // ready WebSocket connections per DC, 0 disables the direct route

	CFProxy                  bool     `json:"cfproxy"` // fallback through Cloudflare-proxied domains
	H2                       bool     `json:"h2"`      // media through HTTP/2 to Cloudflare domains
	CFProxyUserDomainEnabled bool     `json:"cfproxy_user_domain_enabled"`
	CFProxyUserDomain        []string `json:"cfproxy_user_domain"`
	CFProxyWorkerEnabled     bool     `json:"cfproxy_worker_enabled"`
	CFProxyWorkerDomain      []string `json:"cfproxy_worker_domain"`
	CFProxyDomainsURL        string   `json:"cfproxy_domains_url"` // list of CF domains refreshed every hour, empty = built-in list only
	NoSecure                 bool     `json:"no_secure"`           // port 80 without TLS for CF proxy and worker
	ForceTestDC              bool     `json:"force_test_dc"`
	FakeTLSDomain            string   `json:"fake_tls_domain"` // enables ee-secret and masking
	ProxyProtocol            bool     `json:"proxy_protocol"`  // PROXY protocol v1 header from nginx/haproxy

	Verbose  bool    `json:"verbose"`
	LogFile  string  `json:"log_file"` // relative to the config directory
	LogMaxMB float64 `json:"log_max_mb"`

	Access  AccessConfig  `json:"access"`
	Gateway GatewayConfig `json:"gateway"`
	Web     WebConfig     `json:"web"`
}

// User is a client of the MTProto proxy. The secret identifies the user: whoever has the link is this user
type User struct {
	Name   string `json:"name"`
	Secret string `json:"secret"` // 32 hex characters, generated when empty
	// MaxConnections: 0 = the user is disabled, N = up to N simultaneous connections.
	// One Telegram application keeps several connections (main one and more for media)
	MaxConnections int `json:"max_connections"`
	// AllowExternal accepts the user from every address, otherwise only from trusted networks
	AllowExternal bool   `json:"allow_external"`
	Comment       string `json:"comment,omitempty"`
}

// MaxUserNameLen is the maximal length of a user name
const MaxUserNameLen = 64

// PageToken returns the token of the personal page of a user with this secret.
// It doesn't reveal the secret and changes together with it
func PageToken(secret string) string {
	sum := sha256.Sum256([]byte("laser-mtproxy page:" + strings.ToLower(secret)))
	return hex.EncodeToString(sum[:12])
}

// FindUser returns the user with the name (case-insensitive)
func (c *Config) FindUser(name string) (User, bool) {
	for _, u := range c.Users {
		if strings.EqualFold(u.Name, name) {
			return u, true
		}
	}
	return User{}, false
}

// AccessConfig defines where a password is required
type AccessConfig struct {
	// TrustedNetworks work without password: CIDR, IP, "loopback", "lan" (subnets of this machine), "private", "any"
	TrustedNetworks []string `json:"trusted_networks"`
	Username        string   `json:"username"`
	// Password in plain text: it is replaced by PasswordHash at start.
	// Without a password the web interface and the gateway refuse untrusted addresses
	Password     string `json:"password,omitempty"`
	PasswordHash string `json:"password_hash,omitempty"`
}

// GatewayConfig is the HTTP CONNECT / SOCKS5 gateway for Bot API clients (laser_tele bots)
type GatewayConfig struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	// AllowedHosts are destinations as "host:port", "*.domain:port", port "*" means any port
	AllowedHosts []string `json:"allowed_hosts"`
	// FallbackIPs are other addresses of a host tried when its DNS address is unreachable.
	// Frontends of Telegram serve api.telegram.org too, TLS stays end-to-end between the bot and Telegram
	FallbackIPs map[string][]string `json:"fallback_ips"`
	// ViaCFWorker sends connections through CF worker domains when the direct connection fails
	ViaCFWorker bool `json:"via_cfproxy_worker"`
}

// WebConfig is the web interface
type WebConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	TLSCert  string `json:"tls_cert"` // HTTPS when both files are set
	TLSKey   string `json:"tls_key"`
	Language string `json:"language"` // ru or en
	// AdminTrustedOnly: the administration works only from trusted networks, even with the password.
	// Personal pages of users (/p/...) work from everywhere
	AdminTrustedOnly bool `json:"admin_trusted_only"`
	// AllowedHosts are extra host names of the web interface for trusted access. IP addresses, localhost
	// and the name of this machine always work; other names are refused to protect from DNS rebinding
	AllowedHosts []string `json:"allowed_hosts"`
}

// DefaultCFProxyDomainsURL is the list of Cloudflare domains maintained by tg-ws-proxy
const DefaultCFProxyDomainsURL = "https://raw.githubusercontent.com/Flowseal/tg-ws-proxy/main/.github/cfproxy-domains.txt"

// Default returns the configuration of the distribution, secrets are empty
func Default() *Config {
	return &Config{
		Host:                "0.0.0.0",
		Port:                1443,
		DCIP:                []string{"2:149.154.167.220", "4:149.154.167.220"},
		BufKB:               256,
		PoolSize:            4,
		CFProxy:             true,
		H2:                  true,
		CFProxyUserDomain:   []string{},
		CFProxyWorkerDomain: []string{},
		CFProxyDomainsURL:   DefaultCFProxyDomainsURL,
		LogFile:             "laser-mtproxy.log",
		LogMaxMB:            5,
		Users:               []User{{Name: "lan", MaxConnections: 100, Comment: "общий секрет для локальной сети"}},
		Access: AccessConfig{
			TrustedNetworks: []string{access.KeywordLoopback, access.KeywordLAN},
			Username:        "admin",
		},
		Gateway: GatewayConfig{
			Enabled:      true,
			Host:         "0.0.0.0",
			Port:         1080,
			AllowedHosts: []string{"api.telegram.org:443"},
			FallbackIPs:  map[string][]string{"api.telegram.org": {"149.154.167.220"}},
			ViaCFWorker:  true,
		},
		Web: WebConfig{Host: "0.0.0.0", Port: 8080, Language: "ru", AllowedHosts: []string{}, AdminTrustedOnly: true},
	}
}

// Clone returns a deep copy
func (c *Config) Clone() *Config {
	data, _ := json.Marshal(c)
	clone := &Config{}
	json.Unmarshal(data, clone)
	return clone
}

// Load reads the configuration from path, missing keys get default values.
// A missing file is created. Empty secrets are generated and a plain password is hashed,
// then the file is saved again. changed reports that the file was written
func Load(path string) (cfg *Config, changed bool, err error) {
	return load(path, true)
}

// Check reads and validates the configuration without writing anything
func Check(path string) (*Config, error) {
	cfg, _, err := load(path, false)
	return cfg, err
}

func load(path string, save bool) (cfg *Config, changed bool, err error) {
	cfg = Default()
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		changed = true
	case err != nil:
		return nil, false, err
	default:
		// maps are merged by json, so a list of the file replaces the default one;
		// users missing in the file are created by Prepare
		defaultIPs := cfg.Gateway.FallbackIPs
		cfg.Gateway.FallbackIPs = nil
		cfg.Users = nil
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, false, fmt.Errorf("%s: %w", path, err)
		}
		if cfg.Gateway.FallbackIPs == nil {
			cfg.Gateway.FallbackIPs = defaultIPs
		}
	}
	if cfg.Prepare() {
		changed = true
	}
	if err := cfg.Validate(); err != nil {
		return nil, false, fmt.Errorf("%s: %w", path, err)
	}
	if changed && save {
		if err := cfg.Save(path); err != nil {
			return nil, false, err
		}
	}
	return cfg, changed, nil
}

// Prepare generates the empty secret, hashes the plain password, normalizes lists.
// It returns true if something was changed
func (c *Config) Prepare() bool {
	changed := false
	legacy := strings.ToLower(strings.TrimSpace(c.Secret))
	legacyExternal := strings.ToLower(strings.TrimSpace(c.ExternalSecret))
	if c.Users == nil {
		// no "users" key: a new installation or the configuration of version 1.0
		changed = true
		if legacy == "" && legacyExternal == "" {
			c.Users = []User{{Name: "lan", MaxConnections: 100, Comment: "общий секрет для локальной сети"}}
		}
	}
	addLegacy := func(name, secret string, external bool, max int, comment string) {
		if secret == "" {
			return
		}
		for _, u := range c.Users {
			if strings.EqualFold(u.Secret, secret) {
				return
			}
		}
		for i := 2; ; i++ {
			if _, taken := c.FindUser(name); !taken {
				break
			}
			name = fmt.Sprintf("%s-%d", strings.TrimRight(name, "-0123456789"), i)
		}
		c.Users = append(c.Users, User{Name: name, Secret: secret, MaxConnections: max, AllowExternal: external, Comment: comment})
	}
	addLegacy("lan", legacy, false, 100, "общий секрет для локальной сети")
	addLegacy("external", legacyExternal, true, 20, "внешний секрет")
	if c.Secret != "" || c.ExternalSecret != "" {
		c.Secret, c.ExternalSecret = "", ""
		changed = true
	}
	for i := range c.Users {
		u := &c.Users[i]
		u.Name = strings.TrimSpace(u.Name)
		u.Comment = strings.TrimSpace(u.Comment)
		if strings.TrimSpace(u.Secret) == "" {
			u.Secret = RandomSecret()
			changed = true
		}
		u.Secret = strings.ToLower(strings.TrimSpace(u.Secret))
	}
	if c.Users == nil {
		c.Users = []User{}
	}
	if c.Access.Password != "" {
		c.Access.PasswordHash = HashPassword(c.Access.Password)
		c.Access.Password = ""
		changed = true
	}
	c.CFProxyUserDomain = CoerceDomainList(c.CFProxyUserDomain)
	c.CFProxyWorkerDomain = CoerceDomainList(c.CFProxyWorkerDomain)
	c.FakeTLSDomain = strings.TrimSpace(c.FakeTLSDomain)
	c.PublicHost = strings.TrimSpace(c.PublicHost)
	return changed
}

// Save writes the configuration atomically, readable only by the owner
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Validate checks all values
func (c *Config) Validate() error {
	if err := validateListenHost(c.Host); err != nil {
		return fmt.Errorf("host: %w", err)
	}
	if err := validatePort(c.Port); err != nil {
		return fmt.Errorf("port: %w", err)
	}
	if err := ValidateUsers(c.Users); err != nil {
		return err
	}
	if _, err := ParseDCIP(c.DCIP); err != nil {
		return fmt.Errorf("dc_ip: %w", err)
	}
	if c.BufKB < 4 {
		return errors.New("buf_kb must be at least 4")
	}
	if c.PoolSize < 0 {
		return errors.New("pool_size can't be negative")
	}
	if c.LogMaxMB < 0 {
		return errors.New("log_max_mb can't be negative")
	}
	for _, d := range append(append([]string{}, c.CFProxyUserDomain...), c.CFProxyWorkerDomain...) {
		if !IsValidDomain(strings.ToLower(d)) {
			return fmt.Errorf("wrong domain %q", d)
		}
	}
	if c.FakeTLSDomain != "" && !IsValidDomain(strings.ToLower(c.FakeTLSDomain)) {
		return fmt.Errorf("fake_tls_domain: wrong domain %q", c.FakeTLSDomain)
	}
	for _, n := range c.Access.TrustedNetworks {
		if err := access.ValidateEntry(n); err != nil {
			return fmt.Errorf("access.trusted_networks: %w", err)
		}
	}
	if c.Access.PasswordHash != "" && c.Access.Username == "" {
		return errors.New("access.username is empty")
	}
	if err := validateListenHost(c.Gateway.Host); err != nil {
		return fmt.Errorf("gateway.host: %w", err)
	}
	if err := validatePort(c.Gateway.Port); err != nil {
		return fmt.Errorf("gateway.port: %w", err)
	}
	for _, h := range c.Gateway.AllowedHosts {
		if _, _, err := ParseHostPattern(h); err != nil {
			return fmt.Errorf("gateway.allowed_hosts: %w", err)
		}
	}
	for host, ips := range c.Gateway.FallbackIPs {
		if !isValidHostname(strings.ToLower(host)) {
			return fmt.Errorf("gateway.fallback_ips: wrong host %q", host)
		}
		for _, ip := range ips {
			if net.ParseIP(ip) == nil {
				return fmt.Errorf("gateway.fallback_ips: wrong IP %q of %s", ip, host)
			}
		}
	}
	if err := validateListenHost(c.Web.Host); err != nil {
		return fmt.Errorf("web.host: %w", err)
	}
	if err := validatePort(c.Web.Port); err != nil {
		return fmt.Errorf("web.port: %w", err)
	}
	if (c.Web.TLSCert == "") != (c.Web.TLSKey == "") {
		return errors.New("web.tls_cert and web.tls_key must be set together")
	}
	ports := map[string]string{}
	for name, addr := range map[string]string{
		"port":         net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		"gateway.port": net.JoinHostPort(c.Gateway.Host, strconv.Itoa(c.Gateway.Port)),
		"web.port":     net.JoinHostPort(c.Web.Host, strconv.Itoa(c.Web.Port)),
	} {
		if name == "gateway.port" && !c.Gateway.Enabled {
			continue
		}
		_, port, _ := net.SplitHostPort(addr)
		if other, ok := ports[port]; ok {
			return fmt.Errorf("%s and %s use the same port %s", other, name, port)
		}
		ports[port] = name
	}
	return nil
}

// ValidateUsers checks names, secrets and limits of users
func ValidateUsers(users []User) error {
	names := map[string]bool{}
	secrets := map[string]string{}
	for i, u := range users {
		name := strings.TrimSpace(u.Name)
		if name == "" {
			return fmt.Errorf("users[%d]: empty name", i)
		}
		if len([]rune(name)) > MaxUserNameLen {
			return fmt.Errorf("user %q: name is longer than %d characters", name, MaxUserNameLen)
		}
		for _, r := range name {
			if r < 0x20 || r == 0x7f {
				return fmt.Errorf("user %q: control characters in the name", name)
			}
		}
		key := strings.ToLower(name)
		if names[key] {
			return fmt.Errorf("user %q is listed twice", name)
		}
		names[key] = true
		if err := ValidateSecret(u.Secret); err != nil {
			return fmt.Errorf("user %q: secret %w", name, err)
		}
		if other, ok := secrets[strings.ToLower(u.Secret)]; ok {
			return fmt.Errorf("users %q and %q have the same secret", other, name)
		}
		secrets[strings.ToLower(u.Secret)] = name
		if u.MaxConnections < 0 || u.MaxConnections > 100000 {
			return fmt.Errorf("user %q: max_connections must be 0 (disabled) to 100000", name)
		}
	}
	return nil
}

func validateListenHost(host string) error {
	if host == "" {
		return errors.New("empty address")
	}
	if net.ParseIP(host) == nil {
		return fmt.Errorf("wrong IP address %q", host)
	}
	return nil
}

func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d is out of 1-65535", port)
	}
	return nil
}

// ValidateSecret checks MTProto secret: 32 hex characters
func ValidateSecret(secret string) error {
	if len(secret) != 32 {
		return errors.New("must be exactly 32 hex characters (16 bytes)")
	}
	if _, err := hex.DecodeString(secret); err != nil {
		return errors.New("must contain only hex characters 0-9, a-f")
	}
	return nil
}

// RandomSecret returns a new MTProto secret
func RandomSecret() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ParseDCIP parses "DC:IP" rules
func ParseDCIP(list []string) (map[int]string, error) {
	result := map[int]string{}
	for _, entry := range list {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		dcS, ipS, ok := strings.Cut(entry, ":")
		if !ok {
			return nil, fmt.Errorf("wrong format %q, expected DC:IP", entry)
		}
		dc, err := strconv.Atoi(strings.TrimSpace(dcS))
		ip := net.ParseIP(strings.TrimSpace(ipS))
		if err != nil || dc <= 0 || ip == nil || ip.To4() == nil {
			return nil, fmt.Errorf("wrong rule %q", entry)
		}
		result[dc] = ip.To4().String()
	}
	return result, nil
}

// CoerceDomainList splits items by spaces, commas and semicolons and removes duplicates
func CoerceDomainList(items []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, item := range items {
		for _, part := range strings.FieldsFunc(item, func(r rune) bool {
			return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
		}) {
			key := strings.ToLower(part)
			if part == "" || seen[key] {
				continue
			}
			seen[key] = true
			result = append(result, part)
		}
	}
	return result
}

// IsValidDomain checks a domain name like tg-ws-proxy does
func IsValidDomain(domain string) bool {
	if domain == "" || len(domain) > 253 || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, ch := range label {
			if !(ch == '-' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
				return false
			}
		}
	}
	tld := labels[len(labels)-1]
	hasAlpha := false
	for _, ch := range tld {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			hasAlpha = true
		}
	}
	return len(tld) >= 2 && hasAlpha
}

// isValidHostname accepts single-label names (localhost, a LAN host) besides domains
func isValidHostname(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, ch := range label {
			if !(ch == '-' || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')) {
				return false
			}
		}
	}
	return true
}

// ParseHostPattern parses "host:port" of gateway.allowed_hosts, port may be "*"
func ParseHostPattern(pattern string) (host, port string, err error) {
	host, port, err = net.SplitHostPort(strings.TrimSpace(pattern))
	if err != nil {
		return "", "", fmt.Errorf("wrong destination %q, expected host:port", pattern)
	}
	if port != "*" {
		if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
			return "", "", fmt.Errorf("wrong port in %q", pattern)
		}
	}
	h := strings.TrimPrefix(strings.ToLower(host), "*.")
	if net.ParseIP(h) == nil && !isValidHostname(h) {
		return "", "", fmt.Errorf("wrong host in %q", pattern)
	}
	return strings.ToLower(host), port, nil
}

const passwordIterations = 120000

// HashPassword returns "pbkdf2-sha256$iterations$salt$hash"
func HashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	key, _ := pbkdf2.Key(sha256.New, password, salt, passwordIterations, 32)
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", passwordIterations,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
}

// CheckPassword compares password with a hash made by HashPassword
func CheckPassword(hash, password string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[2])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

// SortedDCs returns DC numbers of the rules in ascending order
func SortedDCs(rules map[int]string) []int {
	dcs := make([]int, 0, len(rules))
	for dc := range rules {
		dcs = append(dcs, dc)
	}
	sort.Ints(dcs)
	return dcs
}
