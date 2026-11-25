package proxy

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
)

// UserTotals are counters of a user saved between restarts
type UserTotals struct {
	Connections      int64     `json:"connections"`
	RefusedLimit     int64     `json:"refused_limit"`
	RefusedDisabled  int64     `json:"refused_disabled"`
	RefusedUntrusted int64     `json:"refused_untrusted"`
	BytesUp          int64     `json:"bytes_up"`
	BytesDown        int64     `json:"bytes_down"`
	Peak             int       `json:"peak"`
	PeakAt           time.Time `json:"peak_at,omitempty"`
	LastSeen         time.Time `json:"last_seen,omitempty"`
	LastIP           string    `json:"last_ip,omitempty"`
}

type userConn struct {
	conn    net.Conn
	ip      string
	trusted bool
	user    *userState
}

type userState struct {
	name     string
	secret   []byte
	max      int
	external bool

	totals UserTotals // under Users.mu
	conns  map[*userConn]struct{}
	up     atomic.Int64 // bytes not yet added to totals
	down   atomic.Int64
}

// Users are the clients of the proxy, their limits and counters. They survive restarts of the proxy
// and are updated without them
type Users struct {
	mu        sync.Mutex
	list      []*userState
	byName    map[string]*userState
	saved     map[string]UserTotals // totals of users not in the configuration now
	since     time.Time
	statsPath string
}

type statsFile struct {
	Since time.Time             `json:"since"`
	Users map[string]UserTotals `json:"users"`
}

// NewUsers creates the registry, counters are loaded from statsPath (empty: not saved)
func NewUsers(statsPath string) *Users {
	u := &Users{byName: map[string]*userState{}, saved: map[string]UserTotals{}, since: time.Now(), statsPath: statsPath}
	if statsPath == "" {
		return u
	}
	data, err := os.ReadFile(statsPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logx.Warnf("Can't read user statistics: %v", err)
		}
		return u
	}
	var f statsFile
	if err := json.Unmarshal(data, &f); err != nil {
		logx.Warnf("Can't read user statistics %s: %v", statsPath, err)
		return u
	}
	if !f.Since.IsZero() {
		u.since = f.Since
	}
	for name, t := range f.Users {
		u.saved[strings.ToLower(name)] = t
	}
	return u
}

// Update applies the list of users of the configuration. Connections of users that were removed,
// disabled, got a new secret or lost access from outside are closed
func (u *Users) Update(users []config.User) {
	u.mu.Lock()
	var toClose []*userConn
	newByName := map[string]*userState{}
	var list []*userState
	for _, cu := range users {
		key := strings.ToLower(cu.Name)
		secret, err := hex.DecodeString(cu.Secret)
		if err != nil || len(secret) != 16 {
			continue
		}
		st, ok := u.byName[key]
		if !ok {
			st = &userState{conns: map[*userConn]struct{}{}}
			if t, ok := u.saved[key]; ok {
				st.totals = t
				delete(u.saved, key)
			}
		} else {
			secretChanged := string(st.secret) != string(secret)
			for c := range st.conns {
				if cu.MaxConnections == 0 || secretChanged || (!cu.AllowExternal && !c.trusted) {
					toClose = append(toClose, c)
				}
			}
		}
		st.name, st.secret, st.max, st.external = cu.Name, secret, cu.MaxConnections, cu.AllowExternal
		newByName[key] = st
		list = append(list, st)
	}
	for key, st := range u.byName {
		if _, kept := newByName[key]; kept {
			continue
		}
		for c := range st.conns {
			toClose = append(toClose, c)
		}
		u.flushLocked(st)
		u.saved[key] = st.totals
	}
	u.byName, u.list = newByName, list
	u.mu.Unlock()

	for _, c := range toClose {
		logx.Infof("[%s] user %s: connection closed after the change of the user", c.ip, c.user.name)
		c.conn.Close()
	}
}

type userCandidate struct {
	user   *userState
	secret []byte
}

// candidates returns secrets to try for a client: all users for a trusted address,
// users allowed from outside for others
func (u *Users) candidates(trusted bool) []userCandidate {
	u.mu.Lock()
	defer u.mu.Unlock()
	result := make([]userCandidate, 0, len(u.list))
	for _, st := range u.list {
		if trusted || st.external {
			result = append(result, userCandidate{st, st.secret})
		}
	}
	return result
}

// localOnly returns users allowed only from trusted networks, to explain refusals
func (u *Users) localOnly() []userCandidate {
	u.mu.Lock()
	defer u.mu.Unlock()
	var result []userCandidate
	for _, st := range u.list {
		if !st.external {
			result = append(result, userCandidate{st, st.secret})
		}
	}
	return result
}

// first returns the first enabled user allowed from trusted networks, for the start banner
func (u *Users) first() *userState {
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, st := range u.list {
		if st.max > 0 {
			return st
		}
	}
	return nil
}

func (u *Users) summary() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	var parts []string
	for _, st := range u.list {
		s := st.name
		switch {
		case st.max == 0:
			s += " (off)"
		case st.external:
			s += " (" + itoa(st.max) + ", external)"
		default:
			s += " (" + itoa(st.max) + ")"
		}
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// errRefused explains why a connection of a known user is refused
type errRefused string

func (e errRefused) Error() string { return string(e) }

// acquire takes a connection slot of the user
func (u *Users) acquire(st *userState, conn net.Conn, ip string, trusted bool) (*userConn, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.byName[strings.ToLower(st.name)] != st {
		return nil, errRefused("user was removed")
	}
	if st.max == 0 {
		st.totals.RefusedDisabled++
		return nil, errRefused("user is disabled")
	}
	if len(st.conns) >= st.max {
		st.totals.RefusedLimit++
		return nil, errRefused("limit of " + itoa(st.max) + " connections reached")
	}
	c := &userConn{conn: conn, ip: ip, trusted: trusted, user: st}
	st.conns[c] = struct{}{}
	now := time.Now()
	st.totals.Connections++
	st.totals.LastSeen = now
	st.totals.LastIP = ip
	if n := len(st.conns); n > st.totals.Peak {
		st.totals.Peak, st.totals.PeakAt = n, now
	}
	return c, nil
}

func (u *Users) refusedUntrusted(st *userState) {
	u.mu.Lock()
	st.totals.RefusedUntrusted++
	u.mu.Unlock()
}

func (u *Users) release(c *userConn) {
	u.mu.Lock()
	delete(c.user.conns, c)
	c.user.totals.LastSeen = time.Now()
	u.mu.Unlock()
}

func (u *Users) flushLocked(st *userState) {
	st.totals.BytesUp += st.up.Swap(0)
	st.totals.BytesDown += st.down.Swap(0)
}

// UserReport is one row of the report
type UserReport struct {
	Name           string   `json:"name"`
	MaxConnections int      `json:"max_connections"`
	AllowExternal  bool     `json:"allow_external"`
	Active         int      `json:"active"`
	ActiveIPs      []string `json:"active_ips"`
	UserTotals
}

// Report is the statistics of users
type Report struct {
	Since     time.Time    `json:"since"`
	Generated time.Time    `json:"generated"`
	Users     []UserReport `json:"users"`
}

// Report returns counters of the users of the configuration in its order
func (u *Users) Report() Report {
	u.mu.Lock()
	defer u.mu.Unlock()
	r := Report{Since: u.since, Generated: time.Now(), Users: []UserReport{}}
	for _, st := range u.list {
		u.flushLocked(st)
		ips := map[string]int{}
		for c := range st.conns {
			ips[c.ip]++
		}
		active := make([]string, 0, len(ips))
		for ip, n := range ips {
			if n > 1 {
				ip += " ×" + itoa(n)
			}
			active = append(active, ip)
		}
		sort.Strings(active)
		r.Users = append(r.Users, UserReport{Name: st.name, MaxConnections: st.max, AllowExternal: st.external,
			Active: len(st.conns), ActiveIPs: active, UserTotals: st.totals})
	}
	return r
}

// ActiveCount returns the number of connections of all users
func (u *Users) ActiveCount() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	n := 0
	for _, st := range u.list {
		n += len(st.conns)
	}
	return n
}

// Reset clears the counters, the current connections stay
func (u *Users) Reset() {
	u.mu.Lock()
	for _, st := range u.list {
		st.up.Store(0)
		st.down.Store(0)
		st.totals = UserTotals{Peak: len(st.conns)}
		if len(st.conns) > 0 {
			st.totals.PeakAt = time.Now()
		}
	}
	u.saved = map[string]UserTotals{}
	u.since = time.Now()
	u.mu.Unlock()
	u.Save()
}

// Save writes the counters to the statistics file
func (u *Users) Save() error {
	if u.statsPath == "" {
		return nil
	}
	u.mu.Lock()
	f := statsFile{Since: u.since, Users: map[string]UserTotals{}}
	for key, t := range u.saved {
		f.Users[key] = t
	}
	for _, st := range u.list {
		u.flushLocked(st)
		f.Users[st.name] = st.totals
	}
	u.mu.Unlock()
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(u.statsPath), 0o755); err != nil {
		return err
	}
	tmp := u.statsPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, u.statsPath)
}
