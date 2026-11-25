package proxy

import (
	"context"
	"errors"
	"fmt"
	mrand "math/rand/v2"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
)

const (
	wsPoolMaxAge        = 120 * time.Second
	poolCheckInterval   = 5 * time.Second
	refillBackoffStart  = time.Second
	refillBackoffMax    = time.Hour
	workerPoolMaxAge    = 100 * time.Second
	workerPerDCLimit    = 1
	workerFailureCooldn = 60 * time.Second
	frontingSNI         = "sprinthost.ru"
)

func backoff(failures int) time.Duration {
	if failures > 13 {
		failures = 13
	}
	d := refillBackoffStart * time.Duration(1<<(failures-1))
	if d > refillBackoffMax {
		d = refillBackoffMax
	}
	return d
}

type poolKey struct {
	dc    int
	media bool
	test  bool
}

func (k poolKey) String() string {
	s := fmt.Sprintf("DC%d", k.dc)
	if k.test {
		s += "t"
	}
	if k.media {
		s += "m"
	}
	return s
}

type pooledWS struct {
	ws      *wsConn
	created time.Time
	domain  string // worker domain for the CF worker pool
}

// wsPool keeps ready WebSocket connections to DCs of the "DC → IP" rules
type wsPool struct {
	s                *Server
	mu               sync.Mutex
	idle             map[poolKey][]pooledWS
	refilling        map[poolKey]bool
	rotating         map[poolKey]bool
	refillFailures   map[poolKey]int
	refillAfter      map[poolKey]time.Time
	tryFrontingFirst atomic.Bool
}

func newWSPool(s *Server) *wsPool {
	return &wsPool{s: s, idle: map[poolKey][]pooledWS{}, refilling: map[poolKey]bool{}, rotating: map[poolKey]bool{},
		refillFailures: map[poolKey]int{}, refillAfter: map[poolKey]time.Time{}}
}

func isStale(p pooledWS, maxAge time.Duration, now time.Time) bool {
	return now.Sub(p.created) >= maxAge || p.ws.watchFailed()
}

// get returns a ready connection or nil, the pool is refilled in background
func (p *wsPool) get(dc int, media, test bool) *wsConn {
	targetIP, ok := p.s.rc.dcRedirects[dc]
	if !ok || p.s.rc.poolSize <= 0 {
		return nil
	}
	key := poolKey{dc, media, test}
	for {
		p.mu.Lock()
		bucket := p.idle[key]
		if len(bucket) == 0 {
			p.mu.Unlock()
			break
		}
		item := bucket[0]
		p.idle[key] = bucket[1:]
		left := len(bucket) - 1
		p.mu.Unlock()

		now := time.Now()
		if now.Sub(item.created) >= wsPoolMaxAge || !item.ws.stopWatch() {
			go item.ws.Close()
			continue
		}
		stats.PoolHits.Add(1)
		logx.Debugf("WS pool hit %s (age=%.1fs, left=%d)", key, now.Sub(item.created).Seconds(), left)
		p.scheduleRefill(key, targetIP)
		return item.ws
	}
	stats.PoolMisses.Add(1)
	p.scheduleRefill(key, targetIP)
	return nil
}

func (p *wsPool) scheduleRefill(key poolKey, targetIP string) {
	if p.s.rc.poolSize <= 0 || p.s.ctx.Err() != nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.rotating[key] {
		p.rotating[key] = true
		go p.rotate(key, targetIP)
	}
	if p.refilling[key] || time.Now().Before(p.refillAfter[key]) {
		return
	}
	p.refilling[key] = true
	go p.refill(key, targetIP)
}

func (p *wsPool) refill(key poolKey, targetIP string) {
	defer func() {
		p.mu.Lock()
		delete(p.refilling, key)
		p.mu.Unlock()
	}()
	p.mu.Lock()
	needed := p.s.rc.poolSize - len(p.idle[key])
	p.mu.Unlock()
	if needed <= 0 {
		return
	}
	path := wsPath
	if key.test {
		path = wsPathTest
	}
	domains := wsDomains(key.dc, key.media)
	results := make(chan *wsConn, needed)
	for i := 0; i < needed; i++ {
		go func() { results <- p.connectOne(targetIP, domains, path) }()
	}
	connected := 0
	for i := 0; i < needed; i++ {
		ws := <-results
		if ws == nil {
			continue
		}
		if p.s.ctx.Err() != nil {
			ws.Close()
			continue
		}
		ws.startWatch()
		p.mu.Lock()
		p.idle[key] = append(p.idle[key], pooledWS{ws: ws, created: time.Now()})
		p.mu.Unlock()
		connected++
	}
	p.mu.Lock()
	ready := len(p.idle[key])
	if connected > 0 {
		delete(p.refillFailures, key)
		delete(p.refillAfter, key)
		p.mu.Unlock()
	} else {
		p.refillFailures[key]++
		delay := backoff(p.refillFailures[key])
		p.refillAfter[key] = time.Now().Add(delay)
		p.mu.Unlock()
		if p.s.ctx.Err() == nil {
			logx.Infof("WS pool refill failed for %s, retry in %.0fs", key, delay.Seconds())
		}
	}
	logx.Debugf("WS pool refilled %s: %d ready", key, ready)
}

func (p *wsPool) rotate(key poolKey, targetIP string) {
	defer func() {
		p.mu.Lock()
		delete(p.rotating, key)
		p.mu.Unlock()
	}()
	for p.s.rc.poolSize > 0 {
		now := time.Now()
		p.mu.Lock()
		var ready, expired []pooledWS
		for _, item := range p.idle[key] {
			if isStale(item, wsPoolMaxAge, now) {
				expired = append(expired, item)
			} else {
				ready = append(ready, item)
			}
		}
		p.idle[key] = ready
		wakeAt := now.Add(poolCheckInterval)
		for _, item := range ready {
			if t := item.created.Add(wsPoolMaxAge); t.Before(wakeAt) {
				wakeAt = t
			}
		}
		if after, ok := p.refillAfter[key]; ok && !p.refilling[key] && after.After(now) && after.Before(wakeAt) {
			wakeAt = after
		}
		need := len(ready) < p.s.rc.poolSize
		p.mu.Unlock()

		for _, item := range expired {
			go item.ws.Close()
		}
		if len(expired) > 0 {
			logx.Debugf("WS pool rotated %s: %d stale, %d ready", key, len(expired), len(ready))
		}
		if need {
			p.scheduleRefill(key, targetIP)
		}
		select {
		case <-p.s.ctx.Done():
			return
		case <-time.After(time.Until(wakeAt)):
		}
	}
}

// connectOne connects to the DC IP with SNI of the Telegram domain, or with domain fronting
func (p *wsPool) connectOne(targetIP string, domains []string, path string) *wsConn {
	for _, domain := range domains {
		modes := []bool{false, true}
		if p.tryFrontingFirst.Load() {
			modes = []bool{true, false}
		}
		for _, fronted := range modes {
			o := wsOptions{host: targetIP, domain: domain, path: path, secure: true, timeout: 8 * time.Second, bufSize: p.s.rc.bufferSize}
			if fronted {
				o.sni = frontingSNI
				o.timeout = 7 * time.Second
			}
			ws, err := wsConnect(p.s.ctx, o)
			if err != nil {
				if p.s.ctx.Err() != nil {
					return nil
				}
				stats.WSErrors.Add(1)
				logx.Debugf("WS pool connect %s%s via %s (fronting=%v): %v", domain, path, targetIP, fronted, err)
				continue
			}
			p.tryFrontingFirst.Store(fronted)
			if fronted {
				stats.ConnectionsFronting.Add(1)
			}
			logx.Debugf("WS pool connected %s%s via %s (fronting=%v)", domain, path, targetIP, fronted)
			return ws
		}
	}
	return nil
}

func (p *wsPool) warmup() {
	for dc, ip := range p.s.rc.dcRedirects {
		for _, media := range []bool{false, true} {
			p.scheduleRefill(poolKey{dc, media, p.s.rc.forceTestDC}, ip)
		}
	}
	logx.Infof("WS pool warmup started for %d DC(s)", len(p.s.rc.dcRedirects))
}

func (p *wsPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, bucket := range p.idle {
		for _, item := range bucket {
			go item.ws.Close()
		}
	}
	p.idle = map[poolKey][]pooledWS{}
}

// workerPool keeps one ready connection per DC through Cloudflare Worker domains
type workerPool struct {
	s              *Server
	mu             sync.Mutex
	idle           map[int][]pooledWS
	refilling      map[int]bool
	rotating       map[int]bool
	refillFailures map[int]int
	refillAfter    map[int]time.Time
	exhaustedUntil map[string]time.Time
	trySecureFirst atomic.Bool
}

func newWorkerPool(s *Server) *workerPool {
	w := &workerPool{s: s, idle: map[int][]pooledWS{}, refilling: map[int]bool{}, rotating: map[int]bool{},
		refillFailures: map[int]int{}, refillAfter: map[int]time.Time{}, exhaustedUntil: map[string]time.Time{}}
	w.trySecureFirst.Store(true)
	return w
}

func (w *workerPool) get(dc int, dst string) (*wsConn, string) {
	domains := w.s.rc.workerDomains
	if w.s.rc.poolSize <= 0 || len(domains) == 0 {
		return nil, ""
	}
	for {
		w.mu.Lock()
		bucket := w.idle[dc]
		if len(bucket) == 0 {
			w.mu.Unlock()
			break
		}
		item := bucket[0]
		w.idle[dc] = bucket[1:]
		left := len(bucket) - 1
		w.mu.Unlock()
		now := time.Now()
		if now.Sub(item.created) >= workerPoolMaxAge || !item.ws.stopWatch() {
			go item.ws.Close()
			continue
		}
		stats.CFPoolHits.Add(1)
		logx.Debugf("CF worker pool hit DC%d via %s (age=%.1fs, left=%d)", dc, item.domain, now.Sub(item.created).Seconds(), left)
		w.scheduleRefill(dc, dst)
		return item.ws, item.domain
	}
	stats.CFPoolMisses.Add(1)
	w.scheduleRefill(dc, dst)
	return nil, ""
}

// deferIfAllPaused postpones the refill until the first worker domain is available again. Under w.mu
func (w *workerPool) deferIfAllPaused(dc int, now time.Time) bool {
	first := time.Time{}
	for i, d := range w.s.rc.workerDomains {
		until, ok := w.exhaustedUntil[d]
		if !ok || !until.After(now) {
			return false
		}
		if i == 0 || until.Before(first) {
			first = until
		}
	}
	w.refillAfter[dc] = first
	return true
}

func (w *workerPool) scheduleRefill(dc int, dst string) {
	if w.s.rc.poolSize <= 0 || len(w.s.rc.workerDomains) == 0 || w.s.ctx.Err() != nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.rotating[dc] {
		w.rotating[dc] = true
		go w.rotate(dc, dst)
	}
	now := time.Now()
	if w.refilling[dc] || w.deferIfAllPaused(dc, now) || now.Before(w.refillAfter[dc]) {
		return
	}
	w.refilling[dc] = true
	go w.refill(dc, dst)
}

func (w *workerPool) target() int {
	if w.s.rc.poolSize < workerPerDCLimit {
		return w.s.rc.poolSize
	}
	return workerPerDCLimit
}

func (w *workerPool) refill(dc int, dst string) {
	defer func() {
		w.mu.Lock()
		delete(w.refilling, dc)
		w.mu.Unlock()
	}()
	w.mu.Lock()
	needed := w.target() - len(w.idle[dc])
	w.mu.Unlock()
	connected := 0
	for i := 0; i < needed; i++ {
		ws, domain := w.connectOne(w.s.ctx, dc, dst, 8*time.Second)
		if ws == nil {
			break
		}
		if w.s.ctx.Err() != nil {
			ws.Close()
			return
		}
		ws.startWatch()
		w.mu.Lock()
		w.idle[dc] = append(w.idle[dc], pooledWS{ws: ws, created: time.Now(), domain: domain})
		w.mu.Unlock()
		connected++
	}
	if needed <= 0 {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if connected > 0 {
		delete(w.refillFailures, dc)
		delete(w.refillAfter, dc)
		return
	}
	now := time.Now()
	if w.deferIfAllPaused(dc, now) {
		return
	}
	w.refillFailures[dc]++
	delay := backoff(w.refillFailures[dc])
	w.refillAfter[dc] = now.Add(delay)
	if w.s.ctx.Err() == nil {
		logx.Infof("CF worker pool refill failed for DC%d, retry in %.0fs", dc, delay.Seconds())
	}
}

func (w *workerPool) rotate(dc int, dst string) {
	defer func() {
		w.mu.Lock()
		delete(w.rotating, dc)
		w.mu.Unlock()
	}()
	for w.s.rc.poolSize > 0 {
		now := time.Now()
		w.mu.Lock()
		var ready, expired []pooledWS
		for _, item := range w.idle[dc] {
			if isStale(item, workerPoolMaxAge, now) {
				expired = append(expired, item)
			} else {
				ready = append(ready, item)
			}
		}
		w.idle[dc] = ready
		wakeAt := now.Add(poolCheckInterval)
		for _, item := range ready {
			if t := item.created.Add(workerPoolMaxAge); t.Before(wakeAt) {
				wakeAt = t
			}
		}
		if after, ok := w.refillAfter[dc]; ok && !w.refilling[dc] && after.After(now) && after.Before(wakeAt) {
			wakeAt = after
		}
		need := len(ready) < w.target()
		w.mu.Unlock()
		for _, item := range expired {
			go item.ws.Close()
		}
		if need {
			w.scheduleRefill(dc, dst)
		}
		select {
		case <-w.s.ctx.Done():
			return
		case <-time.After(time.Until(wakeAt)):
		}
	}
}

func workerPath(dst string, dc int) string {
	return "/apiws?" + url.Values{"dst": {dst}, "dc": {strconv.Itoa(dc)}}.Encode()
}

// connectOne connects through one of the available worker domains to dst
func (w *workerPool) connectOne(ctx context.Context, dc int, dst string, timeout time.Duration) (*wsConn, string) {
	path := workerPath(dst, dc)
	for _, domain := range w.availableDomains() {
		modes := []bool{false}
		if !w.s.rc.noSecure {
			first := w.trySecureFirst.Load()
			modes = []bool{first, !first}
		}
		for attempt, secure := range modes {
			ws, err := wsConnect(ctx, wsOptions{host: domain, domain: domain, path: path, secure: secure, timeout: timeout, bufSize: w.s.rc.bufferSize})
			if err != nil {
				if ctx.Err() != nil {
					return nil, ""
				}
				logx.Debugf("CF worker connect DC%d via %s for %s failed (tls=%v): %v", dc, domain, dst, secure, err)
				if attempt < len(modes)-1 && isTimeout(err) {
					continue
				}
				w.reportFailure(domain, err)
				break
			}
			w.trySecureFirst.Store(secure)
			return ws, domain
		}
	}
	return nil, ""
}

func (w *workerPool) availableDomains() []string {
	now := time.Now()
	w.mu.Lock()
	defer w.mu.Unlock()
	seen := map[string]bool{}
	var domains []string
	for _, d := range w.s.rc.workerDomains {
		if seen[d] {
			continue
		}
		seen[d] = true
		if until, ok := w.exhaustedUntil[d]; ok {
			if until.After(now) {
				continue
			}
			delete(w.exhaustedUntil, d)
		}
		domains = append(domains, d)
	}
	mrand.Shuffle(len(domains), func(i, j int) { domains[i], domains[j] = domains[j], domains[i] })
	return domains
}

func (w *workerPool) reportFailure(domain string, err error) {
	now := time.Now()
	w.mu.Lock()
	defer w.mu.Unlock()
	if until, ok := w.exhaustedUntil[domain]; ok && until.After(now) {
		return
	}
	var until time.Time
	var reason string
	var hsErr *wsHandshakeError
	switch {
	case errors.As(err, &hsErr):
		switch {
		case hsErr.StatusCode == 429:
			// the free plan of Workers reports the daily request limit as 429
			secs := 86400 - time.Now().Unix()%86400
			until, reason = now.Add(time.Duration(secs)*time.Second), "daily request limit"
		case hsErr.StatusCode >= 500:
			until, reason = now.Add(workerFailureCooldn), fmt.Sprintf("HTTP %d", hsErr.StatusCode)
		default:
			// a challenge or a misconfigured route is not a reason to disable the domain
			return
		}
	default:
		until, reason = now.Add(workerFailureCooldn), err.Error()
	}
	w.exhaustedUntil[domain] = until
	logx.Warnf("CF worker %s disabled for %d seconds: %s", domain, int(until.Sub(now).Seconds()), reason)
}

func (w *workerPool) warmup() {
	if w.s.rc.poolSize <= 0 || len(w.s.rc.workerDomains) == 0 {
		return
	}
	count := 0
	for dc, ip := range DCDefaultIPs {
		if _, ok := w.s.rc.dcRedirects[dc]; ok {
			continue
		}
		w.scheduleRefill(dc, ip)
		count++
	}
	if count > 0 {
		logx.Infof("CF worker pool warmup started for %d DC(s)", count)
	}
}

func (w *workerPool) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, bucket := range w.idle {
		for _, item := range bucket {
			go item.ws.Close()
		}
	}
	w.idle = map[int][]pooledWS{}
}
