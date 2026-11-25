package proxy

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	mrand "math/rand/v2"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
)

// Built-in Cloudflare-proxied domains of tg-ws-proxy, stored encoded like there
var cfProxyEncoded = []string{
	"virkgj.com", "vmmzovy.com", "mkuosckvso.com", "zaewayzmplad.com", "twdmbzcm.com",
	"awzwsldi.com", "clngqrflngqin.com", "tjacxbqtj.com", "bxaxtxmrw.com", "dmohrsgmohcrwb.com",
	"vwbmtmoi.com", "khgrre.com", "ulihssf.com", "tmhqsdqmfpmk.com", "xwuwoqbm.com",
	"orgcnunpj.com", "zhkuldz.com", "zypoljnslxa.com", "efabnxaowuzs.com", "zaftuzsftqdq.com",
}

const cfProxyMinValidDomains = 3

func isAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

// decodeCFDomain reverses the encoding of tg-ws-proxy (shift of letters, ".com" -> ".co.uk")
func decodeCFDomain(s string) string {
	if !strings.HasSuffix(s, ".com") {
		return s
	}
	p := s[:len(s)-4]
	n := 0
	for i := 0; i < len(p); i++ {
		if isAlpha(p[i]) {
			n++
		}
	}
	out := make([]byte, len(p))
	for i := 0; i < len(p); i++ {
		c := p[i]
		if !isAlpha(c) {
			out[i] = c
			continue
		}
		base := 65
		if c > '`' {
			base = 97
		}
		out[i] = byte(((int(c)-base-n)%26+26)%26 + base)
	}
	return string(out) + ".co.uk"
}

// DefaultCFDomains returns the decoded built-in list
func DefaultCFDomains() []string {
	result := make([]string, 0, len(cfProxyEncoded))
	for _, d := range cfProxyEncoded {
		result = append(result, decodeCFDomain(d))
	}
	return result
}

// balancer chooses a Cloudflare domain for every DC and remembers the one that worked
type balancer struct {
	mu        sync.Mutex
	domains   []string
	dcDomains map[int]string
}

func newBalancer() *balancer { return &balancer{dcDomains: map[int]string{}} }

func sameDomains(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x := append([]string(nil), a...)
	y := append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

func (b *balancer) updateDomains(domains []string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if sameDomains(b.domains, domains) || len(domains) == 0 {
		return
	}
	b.domains = append([]string(nil), domains...)
	b.dcDomains = map[int]string{}
	for _, dc := range []int{1, 2, 3, 4, 5, 203} {
		b.dcDomains[dc] = b.domains[mrand.IntN(len(b.domains))]
	}
}

func (b *balancer) updateDomainForDC(dc int, domain string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.dcDomains[dc] == domain {
		return false
	}
	b.dcDomains[dc] = domain
	return true
}

// domainsForDC returns the current domain of the DC first, then the others shuffled
func (b *balancer) domainsForDC(dc int) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]string, 0, len(b.domains))
	current, ok := b.dcDomains[dc]
	if ok {
		result = append(result, current)
	}
	others := make([]string, 0, len(b.domains))
	for _, d := range b.domains {
		if d != current {
			others = append(others, d)
		}
	}
	mrand.Shuffle(len(others), func(i, j int) { others[i], others[j] = others[j], others[i] })
	return append(result, others...)
}

func (b *balancer) list() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.domains...)
}

// pinned addresses of GitHub hosts, tried first in case DNS is spoofed
var githubIPs = map[string]string{
	"release-assets.githubusercontent.com": "185.199.109.133",
	"raw.githubusercontent.com":            "185.199.109.133",
}

func githubClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, _ := net.SplitHostPort(addr)
			if ip, ok := githubIPs[host]; ok {
				if conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip, port)); err == nil {
					return conn, nil
				}
			}
			return dialer.DialContext(ctx, network, addr)
		},
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &http.Client{Transport: transport, Timeout: 15 * time.Second}
}

func randomLetters(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}

func fetchCFDomainList(ctx context.Context, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"?"+randomLetters(7), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "tg-ws-proxy")
	resp, err := githubClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var result []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		result = append(result, decodeCFDomain(line))
	}
	return result, scanner.Err()
}

func normalizeDomainPool(domains []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, d := range domains {
		item := strings.ToLower(strings.TrimSpace(d))
		if !config.IsValidDomain(item) || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

// refreshCFDomains keeps the domain list updated from url every hour until ctx is done
func (b *balancer) refreshLoop(ctx context.Context, url string) {
	refresh := func() {
		fetched, err := fetchCFDomainList(ctx, url)
		if err != nil {
			if ctx.Err() == nil {
				logx.Warnf("Failed to fetch CF proxy domain list: %v", err)
			}
			return
		}
		pool := normalizeDomainPool(fetched)
		if len(pool) >= cfProxyMinValidDomains {
			b.updateDomains(pool)
			logx.Infof("CF proxy domain pool updated (%d domains)", len(pool))
			return
		}
		logx.Warnf("Ignoring fetched CF proxy domains (total=%d, valid=%d, required>=%d); keeping current domain pool",
			len(fetched), len(pool), cfProxyMinValidDomains)
	}
	refresh()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refresh()
		}
	}
}
