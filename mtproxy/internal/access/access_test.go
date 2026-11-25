package access

import (
	"net"
	"testing"
)

func TestACL(t *testing.T) {
	acl, err := New([]string{"loopback", "10.1.0.0/16", "192.168.5.7"})
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]bool{
		"127.0.0.1": true, "::1": true, "::ffff:127.0.0.1": true, "10.1.200.3": true, "10.2.0.1": false,
		"192.168.5.7": true, "192.168.5.8": false, "8.8.8.8": false,
	}
	for ip, want := range cases {
		if got := acl.Contains(net.ParseIP(ip)); got != want {
			t.Errorf("%s: %v, want %v", ip, got, want)
		}
	}
	if !acl.ContainsAddr("[::1]:1234") || acl.ContainsAddr("8.8.8.8:53") {
		t.Fatal("ContainsAddr")
	}
	if _, err := New([]string{"office"}); err == nil {
		t.Fatal("unknown keyword accepted")
	}
	empty, _ := New(nil)
	if empty.Contains(net.ParseIP("127.0.0.1")) {
		t.Fatal("empty list trusts nobody")
	}
}

func TestLANIncludesOwnAddresses(t *testing.T) {
	acl, _ := New([]string{"lan"})
	for _, n := range InterfaceNetworks() {
		if !acl.Contains(n.IP) {
			t.Errorf("own network %s not trusted", n)
		}
	}
	if acl.Contains(net.ParseIP("127.0.0.1")) {
		t.Error("lan must not include loopback")
	}
}

func TestPublicSubnetsAreNotLAN(t *testing.T) {
	for ip, want := range map[string]bool{
		"192.168.14.77": true, "10.1.2.3": true, "172.17.0.1": true, "100.64.1.1": true, "fe80::1": true, "fd00::1": true,
		"109.74.142.77": false, "8.8.8.8": false, "2a00:1450::1": false,
	} {
		if got := isLocalAddress(net.ParseIP(ip)); got != want {
			t.Errorf("%s: %v, want %v", ip, got, want)
		}
	}
}
