package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
)

// TestResult is the result of one WebSocket upgrade attempt
type TestResult struct {
	DC    int    `json:"dc"`
	OK    bool   `json:"ok"`
	Via   string `json:"via,omitempty"`
	Error string `json:"error,omitempty"`
}

// DomainTest is the result for one domain
type DomainTest struct {
	Domain  string       `json:"domain"`
	Results []TestResult `json:"results"`
	OKCount int          `json:"ok_count"`
	Total   int          `json:"total"`
}

func testError(err error) string {
	var hs *wsHandshakeError
	switch {
	case errors.As(err, &hs):
		if hs.StatusLine == "" {
			return "no response"
		}
		return hs.StatusLine
	case isTimeout(err) || errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	default:
		msg := err.Error()
		if len(msg) > 80 {
			msg = msg[:80]
		}
		return msg
	}
}

type testCase struct {
	dc     int
	host   string
	domain string
	path   string
	sni    string
}

func runCases(ctx context.Context, cases []testCase, secure bool) []TestResult {
	results := make([]TestResult, len(cases))
	var wg sync.WaitGroup
	for i, c := range cases {
		wg.Add(1)
		go func(i int, c testCase) {
			defer wg.Done()
			ws, err := wsConnect(ctx, wsOptions{host: c.host, domain: c.domain, path: c.path, sni: c.sni, secure: secure, timeout: 6 * time.Second})
			if err != nil {
				results[i] = TestResult{DC: c.dc, Error: testError(err)}
				return
			}
			ws.Close()
			results[i] = TestResult{DC: c.dc, OK: true}
		}(i, c)
	}
	wg.Wait()
	return results
}

func summarize(domain string, results []TestResult) DomainTest {
	ok := 0
	for _, r := range results {
		if r.OK {
			ok++
		}
	}
	return DomainTest{Domain: domain, Results: results, OKCount: ok, Total: len(results)}
}

// TestCFProxyDomain checks the WebSocket route through a Cloudflare-proxied domain for every DC
func TestCFProxyDomain(ctx context.Context, domain string, secure bool) DomainTest {
	var cases []testCase
	for _, dc := range TestDCs {
		host := fmt.Sprintf("kws%d.%s", dc, domain)
		cases = append(cases, testCase{dc: dc, host: host, domain: host, path: wsPath})
	}
	return summarize(domain, runCases(ctx, cases, secure))
}

// TestCFWorkerDomain checks a Cloudflare Worker domain for every DC
func TestCFWorkerDomain(ctx context.Context, domain string, secure bool) DomainTest {
	var cases []testCase
	for _, dc := range TestDCs {
		path := "/apiws?" + url.Values{"dst": {DCDefaultIPs[dc]}, "dc": {strconv.Itoa(dc)}, "media": {"0"}}.Encode()
		cases = append(cases, testCase{dc: dc, host: domain, domain: domain, path: path})
	}
	return summarize(domain, runCases(ctx, cases, secure))
}

// TestCFProxyAuto checks the built-in domains until one works for all DCs, like the "auto" test of tg-ws-proxy
func TestCFProxyAuto(ctx context.Context, domains []string, secure bool) (best string, merged []TestResult) {
	byDC := map[int]TestResult{}
	for i := len(domains) - 1; i >= 0; i-- {
		if ctx.Err() != nil {
			break
		}
		res := TestCFProxyDomain(ctx, domains[i], secure)
		if res.OKCount == res.Total {
			return domains[i], res.Results
		}
		for _, r := range res.Results {
			if r.OK {
				r.Via = domains[i]
				byDC[r.DC] = r
				best = domains[i]
			} else if _, ok := byDC[r.DC]; !ok {
				byDC[r.DC] = r
			}
		}
	}
	for _, r := range byDC {
		merged = append(merged, r)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].DC < merged[j].DC })
	return best, merged
}

// TestDirectRoute checks the direct WebSocket route of "DC → IP" rules, with and without domain fronting
func TestDirectRoute(ctx context.Context, dcIP []string) ([]TestResult, error) {
	rules, err := config.ParseDCIP(dcIP)
	if err != nil {
		return nil, err
	}
	var results []TestResult
	for _, dc := range config.SortedDCs(rules) {
		ip := rules[dc]
		domain := wsDomains(dc, false)[0]
		res := runCases(ctx, []testCase{{dc: dc, host: ip, domain: domain, path: wsPath}}, true)[0]
		res.Via = "sni"
		if !res.OK {
			fronted := runCases(ctx, []testCase{{dc: dc, host: ip, domain: domain, path: wsPath, sni: frontingSNI}}, true)[0]
			if fronted.OK {
				fronted.Via = "fronting"
				res = fronted
			}
		}
		results = append(results, res)
	}
	return results, nil
}
