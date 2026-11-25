package proxy

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
)

const cfTimeoutCooldown = 30 * time.Second

var cfConnectTimeouts = []time.Duration{3 * time.Second, 3 * time.Second, 5 * time.Second, 5 * time.Second, 10 * time.Second}

// probing marks a (domain, tls) pair that is being tried after its cooldown by another connection
var probing = time.Unix(math.MaxInt32, 0)

type cfKey struct {
	name string
	tls  bool
}

// cfState remembers timeouts of Cloudflare domains and whether TLS or plain connections work better
type cfState struct {
	mu           sync.Mutex
	timeoutUntil map[cfKey]time.Time
	tlsOK        int
	plainOK      int
}

func newCFState() *cfState { return &cfState{timeoutUntil: map[cfKey]time.Time{}} }

func (s *cfState) prefersTLS() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tlsOK >= s.plainOK
}

func (s *cfState) tlsInCooldown(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	until, ok := s.timeoutUntil[cfKey{name, true}]
	return ok && time.Now().Before(until)
}

type cfCandidate struct {
	name string // base domain
	host string // kwsN.<base> or the worker domain
}

// connectCFDomains tries candidates one by one, alternating TLS and plain connections after timeouts
func (s *Server) connectCFDomains(ctx context.Context, candidates []cfCandidate, label string, secure bool, path string, timeout time.Duration) (*wsConn, string, error) {
	st := s.cf
	domains := append([]cfCandidate(nil), candidates...)
	retrySingle := secure && len(domains) == 1
	if retrySingle {
		domains = append(domains, domains[0])
	}
	timeouts := 0
	tlsFirst := st.prefersTLS()
	lastErr := errors.New("no CF domains")

	for index, c := range domains {
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		useTLS := secure && (tlsFirst != (timeouts%2 == 1))
		if retrySingle && index == 1 && timeouts == 0 {
			// the first try was skipped because of the cooldown
			useTLS = !tlsFirst
		}
		key := cfKey{c.name, useTLS}

		st.mu.Lock()
		retryAt, hadRetry := st.timeoutUntil[key]
		if hadRetry {
			if time.Now().Before(retryAt) {
				st.mu.Unlock()
				continue
			}
			st.timeoutUntil[key] = probing
		}
		st.mu.Unlock()

		attemptTimeout := cfConnectTimeouts[timeouts]
		if timeout < attemptTimeout {
			attemptTimeout = timeout
		}
		ws, err := wsConnect(ctx, wsOptions{host: c.host, domain: c.host, path: path, secure: useTLS,
			timeout: attemptTimeout, bufSize: s.rc.bufferSize})

		st.mu.Lock()
		if err != nil && isTimeout(err) && ctx.Err() == nil {
			now := time.Now()
			_, present := st.timeoutUntil[key]
			if hadRetry || !present {
				for k, until := range st.timeoutUntil {
					if until != probing && !until.Add(cfTimeoutCooldown).After(now) {
						delete(st.timeoutUntil, k)
					}
				}
				st.timeoutUntil[key] = now.Add(cfTimeoutCooldown)
				st.mu.Unlock()
				logx.Warnf("[%s] CF WebSocket timed out (host=%s tls=%v); retry in %.0fs", label, c.host, useTLS, cfTimeoutCooldown.Seconds())
			} else {
				st.mu.Unlock()
			}
			lastErr = err
			timeouts++
			if timeouts >= len(cfConnectTimeouts) {
				break
			}
			continue
		}
		delete(st.timeoutUntil, key)
		if err == nil {
			if useTLS {
				st.tlsOK++
			} else {
				st.plainOK++
			}
		}
		st.mu.Unlock()

		if err == nil {
			return ws, c.name, nil
		}
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		logx.Warnf("[%s] CF WebSocket failed (host=%s tls=%v): %v", label, c.host, useTLS, err)
		lastErr = err
		if retrySingle {
			break
		}
	}
	return nil, "", lastErr
}
