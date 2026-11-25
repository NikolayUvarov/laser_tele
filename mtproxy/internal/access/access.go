// Package access decides which client addresses are trusted.
// Trusted clients work without a password, the others must authenticate.
package access

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
)

// Keywords that can be used in the list of trusted networks besides CIDR and single IP addresses
const (
	KeywordLoopback = "loopback" // 127.0.0.0/8 and ::1
	KeywordLAN      = "lan"      // private subnets of the network interfaces of this machine
	KeywordPrivate  = "private"  // 10/8, 172.16/12, 192.168/16, 100.64/10, fc00::/7, fe80::/10
	KeywordAny      = "any"      // every address, passwords are never asked
)

var privateNets = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10", "169.254.0.0/16", "fc00::/7", "fe80::/10"}
var loopbackNets = []string{"127.0.0.0/8", "::1/128"}
var anyNets = []string{"0.0.0.0/0", "::/0"}

// ACL is a list of trusted networks. It is safe for concurrent use
type ACL struct {
	entries []string

	mu   sync.RWMutex
	nets []*net.IPNet
}

// ValidateEntry checks one item of the list of trusted networks
func ValidateEntry(entry string) error {
	e := strings.ToLower(strings.TrimSpace(entry))
	switch e {
	case KeywordLoopback, KeywordLAN, KeywordPrivate, KeywordAny:
		return nil
	case "":
		return fmt.Errorf("empty trusted network")
	}
	if _, _, err := net.ParseCIDR(e); err == nil {
		return nil
	}
	if net.ParseIP(e) != nil {
		return nil
	}
	return fmt.Errorf("wrong trusted network %q: expected CIDR, IP or one of loopback, lan, private, any", entry)
}

// New creates the ACL from entries (keywords, CIDR or IP addresses)
func New(entries []string) (*ACL, error) {
	acl := &ACL{}
	for _, entry := range entries {
		if err := ValidateEntry(entry); err != nil {
			return nil, err
		}
		acl.entries = append(acl.entries, strings.ToLower(strings.TrimSpace(entry)))
	}
	acl.Refresh()
	return acl, nil
}

// Refresh recomputes the networks of the interfaces for the "lan" keyword
func (a *ACL) Refresh() {
	var nets []*net.IPNet
	add := func(cidrs []string) {
		for _, cidr := range cidrs {
			_, n, err := net.ParseCIDR(cidr)
			if err == nil {
				nets = append(nets, n)
			}
		}
	}
	for _, entry := range a.entries {
		switch entry {
		case KeywordLoopback:
			add(loopbackNets)
		case KeywordPrivate:
			add(privateNets)
		case KeywordAny:
			add(anyNets)
		case KeywordLAN:
			nets = append(nets, InterfaceNetworks()...)
		default:
			if strings.Contains(entry, "/") {
				add([]string{entry})
			} else if ip := net.ParseIP(entry); ip != nil {
				bits := 128
				if ip.To4() != nil {
					ip = ip.To4()
					bits = 32
				}
				nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			}
		}
	}
	a.mu.Lock()
	a.nets = nets
	a.mu.Unlock()
}

var cgnat = &net.IPNet{IP: net.IPv4(100, 64, 0, 0).To4(), Mask: net.CIDRMask(10, 32)}

// isLocalAddress reports whether ip belongs to a local network: private ranges, CGNAT and link-local.
// Public subnets of an interface (a server in a datacenter) are not a local network
func isLocalAddress(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLinkLocalUnicast() || cgnat.Contains(ip)
}

// InterfaceNetworks returns private subnets of the up interfaces of this machine, except loopback
func InterfaceNetworks() []*net.IPNet {
	var nets []*net.IPNet
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() || !isLocalAddress(ipNet.IP) {
				continue
			}
			ip := ipNet.IP
			mask := ipNet.Mask
			if ip4 := ip.To4(); ip4 != nil && len(mask) == net.IPv6len {
				ip, mask = ip4, mask[12:]
			}
			nets = append(nets, &net.IPNet{IP: ip.Mask(mask), Mask: mask})
		}
	}
	return nets
}

// Contains reports whether ip belongs to a trusted network
func (a *ACL) Contains(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, n := range a.nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ContainsAddr is Contains for "host:port" or "host" strings
func (a *ACL) ContainsAddr(addr string) bool {
	return a.Contains(HostIP(addr))
}

// HostIP extracts IP address from "host:port" or "host"
func HostIP(addr string) net.IP {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.Trim(host, "[]")
	if i := strings.IndexByte(host, '%'); i >= 0 {
		host = host[:i]
	}
	return net.ParseIP(host)
}

// Entries returns the configured items of the list
func (a *ACL) Entries() []string {
	return append([]string(nil), a.entries...)
}

// Networks returns resolved networks as strings, sorted
func (a *ACL) Networks() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	seen := map[string]bool{}
	result := make([]string, 0, len(a.nets))
	for _, n := range a.nets {
		s := n.String()
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	sort.Strings(result)
	return result
}
