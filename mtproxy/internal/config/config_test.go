package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCreatesDefaultConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "laser-mtproxy.json")
	cfg, changed, err := Load(path)
	if err != nil || !changed {
		t.Fatal(err, changed)
	}
	if len(cfg.Users) != 1 || cfg.Users[0].Name != "lan" || ValidateSecret(cfg.Users[0].Secret) != nil ||
		cfg.Users[0].MaxConnections == 0 || cfg.Users[0].AllowExternal {
		t.Fatalf("default user: %+v", cfg.Users)
	}
	data, _ := os.ReadFile(path)
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	trusted := raw["access"].(map[string]interface{})["trusted_networks"].([]interface{})
	if len(trusted) != 2 || trusted[0] != "loopback" || trusted[1] != "lan" {
		t.Fatalf("trusted networks must be written explicitly: %v", trusted)
	}
	again, changed, err := Load(path)
	if err != nil || changed || again.Users[0].Secret != cfg.Users[0].Secret {
		t.Fatal("second load must not change the file", err, changed)
	}
}

func TestPlainPasswordIsHashed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.json")
	os.WriteFile(path, []byte(`{"access": {"username": "admin", "password": "hello-world"}}`), 0o600)
	cfg, changed, err := Load(path)
	if err != nil || !changed {
		t.Fatal(err)
	}
	if cfg.Access.Password != "" || !CheckPassword(cfg.Access.PasswordHash, "hello-world") || CheckPassword(cfg.Access.PasswordHash, "other") {
		t.Fatal("password hash")
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "hello-world") {
		t.Fatal("plain password left in the file")
	}
	// defaults are kept for missing keys
	if cfg.Port != 1443 || cfg.Gateway.Port != 1080 || len(cfg.Access.TrustedNetworks) != 2 {
		t.Fatalf("defaults: %+v", cfg)
	}
}

func TestValidate(t *testing.T) {
	good := Default()
	good.Prepare()
	good.Users = append(good.Users, User{Name: "ivan", Secret: RandomSecret(), MaxConnections: 5, AllowExternal: true})
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := []func(c *Config){
		func(c *Config) { c.Users[0].Secret = "123" },
		func(c *Config) { c.Users[1].Secret = c.Users[0].Secret },
		func(c *Config) { c.Users[1].Name = "LAN" },
		func(c *Config) { c.Users[1].Name = " " },
		func(c *Config) { c.Users[1].MaxConnections = -1 },
		func(c *Config) { c.DCIP = []string{"2-1.1.1.1"} },
		func(c *Config) { c.Port = 70000 },
		func(c *Config) { c.Host = "localhost" },
		func(c *Config) { c.Access.TrustedNetworks = []string{"office"} },
		func(c *Config) { c.Gateway.Port = c.Port },
		func(c *Config) { c.Gateway.AllowedHosts = []string{"api.telegram.org"} },
		func(c *Config) { c.CFProxyWorkerDomain = []string{"bad domain"} },
	}
	for i, mutate := range bad {
		c := good.Clone()
		mutate(c)
		if c.Validate() == nil {
			t.Errorf("case %d must fail", i)
		}
	}
}

func TestHelpers(t *testing.T) {
	rules, err := ParseDCIP([]string{"2:149.154.167.220", " 4 : 149.154.167.220 ", ""})
	if err != nil || len(rules) != 2 || rules[4] != "149.154.167.220" {
		t.Fatal(rules, err)
	}
	got := CoerceDomainList([]string{"a.com, b.com;A.com", "c.com"})
	if strings.Join(got, " ") != "a.com b.com c.com" {
		t.Fatal(got)
	}
	if !IsValidDomain("kws2.web.telegram.org") || IsValidDomain("localhost") || IsValidDomain("-a.com") {
		t.Fatal("domain validation")
	}
}

func TestLegacySecretsBecomeUsers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.json")
	lan, ext := RandomSecret(), RandomSecret()
	os.WriteFile(path, []byte(`{"secret": "`+lan+`", "external_secret": "`+ext+`"}`), 0o600)
	cfg, changed, err := Load(path)
	if err != nil || !changed {
		t.Fatal(err)
	}
	if len(cfg.Users) != 2 || cfg.Users[0].Secret != lan || cfg.Users[0].AllowExternal ||
		cfg.Users[1].Secret != ext || !cfg.Users[1].AllowExternal || cfg.Secret != "" || cfg.ExternalSecret != "" {
		t.Fatalf("migration: %+v", cfg.Users)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), `"external_secret"`) {
		t.Fatal("old keys left in the file")
	}
}

func TestEmptyUserListStaysEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.json")
	os.WriteFile(path, []byte(`{"users": [{"name": "ivan", "max_connections": 3}]}`), 0o600)
	cfg, _, err := Load(path)
	if err != nil || len(cfg.Users) != 1 || ValidateSecret(cfg.Users[0].Secret) != nil || cfg.Users[0].MaxConnections != 3 {
		t.Fatalf("empty secret must be generated: %+v %v", cfg.Users, err)
	}
	os.WriteFile(path, []byte(`{"users": []}`), 0o600)
	if cfg, _, err = Load(path); err != nil || len(cfg.Users) != 0 {
		t.Fatalf("an explicit empty list must stay empty: %+v %v", cfg.Users, err)
	}
	if PageToken(RandomSecret()) == PageToken(RandomSecret()) || len(PageToken(RandomSecret())) != 24 {
		t.Fatal("page token")
	}
}

func TestCheckDoesntWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.json")
	original := []byte(`{"users": [{"name": "ivan", "max_connections": 3}]}`)
	os.WriteFile(path, original, 0o600)
	if _, err := Check(path); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(path); string(data) != string(original) {
		t.Fatal("Check changed the file")
	}
	if _, err := Check(filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatal("a missing file means defaults:", err)
	}
}
