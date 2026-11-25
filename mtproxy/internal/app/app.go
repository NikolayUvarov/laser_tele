// Package app owns the configuration and the running proxy and gateway
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/access"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/proxy"
)

// Version of laser-mtproxy
const Version = "1.1.0"

// App runs the proxy and the gateway with the current configuration
type App struct {
	cfgPath string

	mu           sync.Mutex
	cfg          *config.Config
	acl          *access.ACL
	users        *proxy.Users
	rc           *proxy.Runtime
	server       *proxy.Server
	gateway      *proxy.Gateway
	proxyError   string
	gatewayError string
	startedAt    time.Time

	ctx    context.Context
	cancel context.CancelFunc
}

// New creates the application with a loaded configuration
func New(cfgPath string, cfg *config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())
	a := &App{cfgPath: cfgPath, cfg: cfg, startedAt: time.Now(), ctx: ctx, cancel: cancel}
	a.users = proxy.NewUsers(a.StatsPath())
	a.users.Update(cfg.Users)
	return a
}

// StatsPath is the file with counters of users next to the configuration
func (a *App) StatsPath() string {
	return strings.TrimSuffix(a.cfgPath, filepath.Ext(a.cfgPath)) + "-stats.json"
}

// Users returns the registry of users with their counters
func (a *App) Users() *proxy.Users { return a.users }

// ConfigDir is the directory of the configuration file, relative paths are resolved from it
func (a *App) ConfigDir() string { return filepath.Dir(a.cfgPath) }

// ResolvePath makes a path of the configuration absolute
func (a *App) ResolvePath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(a.ConfigDir(), path)
}

func (a *App) setupLogging(cfg *config.Config) {
	logx.SetVerbose(cfg.Verbose)
	if err := logx.SetFile(a.ResolvePath(cfg.LogFile), cfg.LogMaxMB); err != nil {
		logx.Errorf("Can't open log file: %v", err)
	}
}

// Start starts the proxy and the gateway. Errors of listening are kept in the status,
// so the web interface still works and the configuration can be fixed there
func (a *App) Start() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.setupLogging(a.cfg)
	a.startLocked()
	go a.refreshLoop()
}

func (a *App) refreshLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			if a.acl != nil {
				a.acl.Refresh()
			}
			a.mu.Unlock()
			if err := a.users.Save(); err != nil {
				logx.Warnf("Can't save user statistics: %v", err)
			}
		}
	}
}

func (a *App) startLocked() {
	a.proxyError, a.gatewayError = "", ""
	acl, err := access.New(a.cfg.Access.TrustedNetworks)
	if err != nil {
		a.proxyError = err.Error()
		logx.Errorf("Bad trusted networks: %v", err)
		return
	}
	a.acl = acl
	rc, err := proxy.NewRuntime(a.cfg)
	if err != nil {
		a.proxyError = err.Error()
		logx.Errorf("Bad configuration: %v", err)
		return
	}
	a.rc = rc
	logx.Infof("Starting proxy on %s:%d ...", a.cfg.Host, a.cfg.Port)
	server := proxy.NewServer(rc, acl, a.users)
	if err := server.Start(); err != nil {
		a.proxyError = err.Error()
		logx.Errorf("Can't start proxy: %v", err)
	} else {
		a.server = server
	}
	if a.cfg.Gateway.Enabled {
		gw := proxy.NewGateway(a.cfg, acl)
		if err := gw.Start(); err != nil {
			a.gatewayError = err.Error()
			logx.Errorf("Can't start Bot API gateway: %v", err)
		} else {
			a.gateway = gw
		}
	}
}

func (a *App) stopLocked() {
	if a.gateway != nil {
		a.gateway.Stop()
		a.gateway = nil
	}
	if a.server != nil {
		a.server.Stop()
		a.server = nil
		logx.Infof("Proxy stopped")
	}
}

// Stop stops everything
func (a *App) Stop() {
	a.cancel()
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stopLocked()
	if err := a.users.Save(); err != nil {
		logx.Warnf("Can't save user statistics: %v", err)
	}
	logx.Infof("Shutting down. Final stats: %s", proxy.GlobalStats().Summary())
}

// Restart restarts the proxy and the gateway with the current configuration
func (a *App) Restart() {
	a.mu.Lock()
	defer a.mu.Unlock()
	logx.Infof("Restarting proxy...")
	a.stopLocked()
	time.Sleep(500 * time.Millisecond)
	a.startLocked()
}

// Config returns a copy of the current configuration
func (a *App) Config() *config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg.Clone()
}

// ACL returns the current list of trusted networks
func (a *App) ACL() *access.ACL {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.acl == nil {
		acl, _ := access.New([]string{access.KeywordLoopback})
		return acl
	}
	return a.acl
}

// IsTrusted reports whether the address is in trusted networks
func (a *App) IsTrusted(ip net.IP) bool { return a.ACL().Contains(ip) }

// Apply validates and saves the new configuration and restarts the proxy.
// webRestart reports that the address of the web interface changed and the process must be restarted
func (a *App) Apply(newCfg *config.Config) (webRestart bool, err error) {
	newCfg.Prepare()
	if err := newCfg.Validate(); err != nil {
		return false, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := newCfg.Save(a.cfgPath); err != nil {
		return false, fmt.Errorf("can't save configuration: %w", err)
	}
	old := a.cfg
	webRestart = old.Web.Host != newCfg.Web.Host || old.Web.Port != newCfg.Web.Port ||
		old.Web.TLSCert != newCfg.Web.TLSCert || old.Web.TLSKey != newCfg.Web.TLSKey
	a.cfg = newCfg
	a.users.Update(newCfg.Users)
	if onlyUsersDiffer(old, newCfg) {
		logx.Infof("Users updated: %d, the proxy keeps working", len(newCfg.Users))
		return false, nil
	}
	logx.Infof("Configuration saved, restarting proxy...")
	a.setupLogging(newCfg)
	a.stopLocked()
	time.Sleep(500 * time.Millisecond)
	a.startLocked()
	if a.proxyError != "" {
		return webRestart, errors.New("configuration saved, but the proxy didn't start: " + a.proxyError)
	}
	if a.gatewayError != "" {
		return webRestart, errors.New("configuration saved, but the gateway didn't start: " + a.gatewayError)
	}
	return webRestart, nil
}

// onlyUsersDiffer reports that the configurations differ only in users, so the proxy needs no restart
func onlyUsersDiffer(a, b *config.Config) bool {
	x, y := a.Clone(), b.Clone()
	x.Users, y.Users = nil, nil
	dx, _ := json.Marshal(x)
	dy, _ := json.Marshal(y)
	return bytes.Equal(dx, dy)
}

// UserLink returns the connection link of a user; host is the address the web interface was opened with
func (a *App) UserLink(u config.User, host string) proxy.Link {
	a.mu.Lock()
	rc := a.rc
	cfg := a.cfg
	a.mu.Unlock()
	if rc == nil {
		var err error
		if rc, err = proxy.NewRuntime(cfg); err != nil {
			return proxy.Link{}
		}
	}
	return rc.UserLink(u.Secret, u.AllowExternal, host)
}

// Status is the state shown by the web interface
type Status struct {
	Version         string           `json:"version"`
	Running         bool             `json:"running"`
	ProxyError      string           `json:"proxy_error,omitempty"`
	GatewayEnabled  bool             `json:"gateway_enabled"`
	GatewayRunning  bool             `json:"gateway_running"`
	GatewayError    string           `json:"gateway_error,omitempty"`
	Listen          string           `json:"listen"`
	GatewayListen   string           `json:"gateway_listen"`
	GatewayURL      string           `json:"gateway_url"`
	GatewayAllowed  []string         `json:"gateway_allowed"`
	ProcessStarted  time.Time        `json:"process_started"`
	ProxyStarted    time.Time        `json:"proxy_started"`
	Stats           map[string]int64 `json:"stats"`
	Users           int              `json:"users"`
	UsersEnabled    int              `json:"users_enabled"`
	UsersActive     int              `json:"users_active"`
	TrustedNetworks []string         `json:"trusted_networks"`
	TrustedEntries  []string         `json:"trusted_entries"`
	HasPassword     bool             `json:"has_password"`
	FakeTLSDomain   string           `json:"fake_tls_domain"`
	DCIP            []string         `json:"dc_ip"`
	CFDomains       int              `json:"cf_domains"`
}

// Status returns the state; linkHost is the address for links (empty = automatic)
func (a *App) Status(linkHost string) Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	cfg := a.cfg
	st := Status{
		Version: Version, Running: a.server != nil, ProxyError: a.proxyError,
		GatewayEnabled: cfg.Gateway.Enabled, GatewayRunning: a.gateway != nil, GatewayError: a.gatewayError,
		Listen:         net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		GatewayListen:  net.JoinHostPort(cfg.Gateway.Host, strconv.Itoa(cfg.Gateway.Port)),
		GatewayAllowed: cfg.Gateway.AllowedHosts,
		ProcessStarted: a.startedAt, Stats: proxy.GlobalStats().Snapshot(),
		HasPassword: cfg.Access.PasswordHash != "", Users: len(cfg.Users), UsersActive: a.users.ActiveCount(),
		FakeTLSDomain: cfg.FakeTLSDomain, DCIP: cfg.DCIP,
	}
	if a.acl != nil {
		st.TrustedNetworks = a.acl.Networks()
		st.TrustedEntries = a.acl.Entries()
	}
	for _, u := range cfg.Users {
		if u.MaxConnections > 0 {
			st.UsersEnabled++
		}
	}
	if a.rc != nil {
		host := a.rc.LinkHost(linkHost, false)
		st.GatewayURL = "http://" + net.JoinHostPort(host, strconv.Itoa(cfg.Gateway.Port))
	}
	if a.server != nil {
		st.ProxyStarted = a.server.StartedAt()
		st.CFDomains = len(a.server.CFDomains())
	}
	return st
}

// LogFile returns the absolute path of the log file
func (a *App) LogFile() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ResolvePath(a.cfg.LogFile)
}
