package proxy

import (
	"fmt"
	"sync/atomic"
)

// Stats are counters of the process, they survive restarts of the proxy
type Stats struct {
	ConnectionsTotal       atomic.Int64
	ConnectionsActive      atomic.Int64
	ConnectionsWS          atomic.Int64
	ConnectionsH2          atomic.Int64
	H2TCPConnections       atomic.Int64
	H2Requests             atomic.Int64
	H2Errors               atomic.Int64
	ConnectionsTCPFallback atomic.Int64
	ConnectionsCFProxy     atomic.Int64
	ConnectionsFronting    atomic.Int64
	ConnectionsBad         atomic.Int64
	ConnectionsUntrusted   atomic.Int64 // refused: a user allowed only from trusted networks came from outside
	ConnectionsLimited     atomic.Int64 // refused: the user is disabled or has the maximum of connections
	ConnectionsMasked      atomic.Int64
	WSErrors               atomic.Int64
	BytesUp                atomic.Int64
	BytesDown              atomic.Int64
	PoolHits               atomic.Int64
	PoolMisses             atomic.Int64
	CFPoolHits             atomic.Int64
	CFPoolMisses           atomic.Int64

	GatewayTotal    atomic.Int64
	GatewayActive   atomic.Int64
	GatewayDenied   atomic.Int64
	GatewayViaCF    atomic.Int64
	GatewayBytesUp  atomic.Int64
	GatewayBytesDn  atomic.Int64
	GatewayFailures atomic.Int64
}

// stats is the only instance
var stats = &Stats{}

// GlobalStats returns counters of the process
func GlobalStats() *Stats { return stats }

func ratio(hits, misses int64) string {
	if hits+misses == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%d/%d", hits, hits+misses)
}

// Summary is one line for the log
func (s *Stats) Summary() string {
	return fmt.Sprintf("total=%d active=%d ws=%d h2=%d h2_tcp=%d h2_req=%d h2_err=%d tcp_fb=%d cf=%d front=%d "+
		"bad=%d untrusted=%d limited=%d masked=%d err=%d pool=%s cf_pool=%s up=%s down=%s gw=%d/%d gw_denied=%d",
		s.ConnectionsTotal.Load(), s.ConnectionsActive.Load(), s.ConnectionsWS.Load(), s.ConnectionsH2.Load(),
		s.H2TCPConnections.Load(), s.H2Requests.Load(), s.H2Errors.Load(), s.ConnectionsTCPFallback.Load(),
		s.ConnectionsCFProxy.Load(), s.ConnectionsFronting.Load(), s.ConnectionsBad.Load(),
		s.ConnectionsUntrusted.Load(), s.ConnectionsLimited.Load(), s.ConnectionsMasked.Load(), s.WSErrors.Load(),
		ratio(s.PoolHits.Load(), s.PoolMisses.Load()), ratio(s.CFPoolHits.Load(), s.CFPoolMisses.Load()),
		HumanBytes(s.BytesUp.Load()), HumanBytes(s.BytesDown.Load()),
		s.GatewayActive.Load(), s.GatewayTotal.Load(), s.GatewayDenied.Load())
}

// Snapshot returns counters for the web interface
func (s *Stats) Snapshot() map[string]int64 {
	return map[string]int64{
		"connections_total":        s.ConnectionsTotal.Load(),
		"connections_active":       s.ConnectionsActive.Load(),
		"connections_ws":           s.ConnectionsWS.Load(),
		"connections_h2":           s.ConnectionsH2.Load(),
		"h2_tcp_connections":       s.H2TCPConnections.Load(),
		"h2_requests":              s.H2Requests.Load(),
		"h2_errors":                s.H2Errors.Load(),
		"connections_tcp_fallback": s.ConnectionsTCPFallback.Load(),
		"connections_cfproxy":      s.ConnectionsCFProxy.Load(),
		"connections_fronting":     s.ConnectionsFronting.Load(),
		"connections_bad":          s.ConnectionsBad.Load(),
		"connections_untrusted":    s.ConnectionsUntrusted.Load(),
		"connections_limited":      s.ConnectionsLimited.Load(),
		"connections_masked":       s.ConnectionsMasked.Load(),
		"ws_errors":                s.WSErrors.Load(),
		"bytes_up":                 s.BytesUp.Load(),
		"bytes_down":               s.BytesDown.Load(),
		"pool_hits":                s.PoolHits.Load(),
		"pool_misses":              s.PoolMisses.Load(),
		"cf_pool_hits":             s.CFPoolHits.Load(),
		"cf_pool_misses":           s.CFPoolMisses.Load(),
		"gateway_total":            s.GatewayTotal.Load(),
		"gateway_active":           s.GatewayActive.Load(),
		"gateway_denied":           s.GatewayDenied.Load(),
		"gateway_via_cf":           s.GatewayViaCF.Load(),
		"gateway_bytes_up":         s.GatewayBytesUp.Load(),
		"gateway_bytes_down":       s.GatewayBytesDn.Load(),
		"gateway_failures":         s.GatewayFailures.Load(),
	}
}
