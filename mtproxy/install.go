package main

// --install and --uninstall: a systemd service of the user who runs the command,
// or a system service when it is run by root

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/proxy"
)

const (
	serviceName = "laser-mtproxy"
	systemUser  = "laser-mtproxy" // account of the system service
)

type installMode struct {
	root     bool
	dir      string   // binary, configuration, logs and statistics
	unitPath string   // systemd unit
	ctl      []string // systemctl command
}

func (m installMode) binary() string { return filepath.Join(m.dir, serviceName) }
func (m installMode) config() string { return filepath.Join(m.dir, serviceName+".json") }

func detectMode() (installMode, error) {
	if runtime.GOOS != "linux" {
		return installMode{}, errors.New("--install and --uninstall work only on Linux with systemd")
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return installMode{}, errors.New("systemctl not found: systemd is required")
	}
	if os.Geteuid() == 0 {
		return installMode{root: true, dir: "/opt/" + serviceName,
			unitPath: "/etc/systemd/system/" + serviceName + ".service", ctl: []string{"systemctl"}}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return installMode{}, err
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	return installMode{dir: filepath.Join(dataHome, serviceName),
		unitPath: filepath.Join(configHome, "systemd", "user", serviceName+".service"),
		ctl:      []string{"systemctl", "--user"}}, nil
}

func (m installMode) systemctl(args ...string) (string, error) {
	cmd := exec.Command(m.ctl[0], append(m.ctl[1:], args...)...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// copyFile writes src to dst through a temporary file, so a running binary can be replaced
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

func quote(path string) string { return strconv.Quote(path) }

func (m installMode) unit() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, `# created by %s --install
[Unit]
Description=laser-mtproxy: MTProto proxy for Telegram and Bot API gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=%s
ExecStart=%s -config %s
Restart=always
RestartSec=3
LimitNOFILE=65536
`, serviceName, m.dir, quote(m.binary()), quote(m.config()))
	if m.root {
		fmt.Fprintf(&b, `User=%s
Group=%s
# ports below 1024, for example 443 for Fake TLS
AmbientCapabilities=CAP_NET_BIND_SERVICE
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=%s

[Install]
WantedBy=multi-user.target
`, systemUser, systemUser, m.dir)
	} else {
		b.WriteString(`
[Install]
WantedBy=default.target
`)
	}
	return b.String()
}

func ensureSystemUser(dir string) (uid, gid int, err error) {
	u, err := user.Lookup(systemUser)
	if err != nil {
		out, cerr := exec.Command("useradd", "--system", "--home-dir", dir, "--no-create-home",
			"--shell", "/usr/sbin/nologin", systemUser).CombinedOutput()
		if cerr != nil {
			return 0, 0, fmt.Errorf("can't create user %s: %v %s", systemUser, cerr, out)
		}
		fmt.Println("Created system user", systemUser)
		if u, err = user.Lookup(systemUser); err != nil {
			return 0, 0, err
		}
	}
	uid, _ = strconv.Atoi(u.Uid)
	gid, _ = strconv.Atoi(u.Gid)
	return uid, gid, nil
}

// runInstall installs or updates the service. seedConfig is copied as the configuration
// when the service has none yet
func runInstall(seedConfig string) error {
	m, err := detectMode()
	if err != nil {
		return err
	}
	kind := "user service of " + currentUserName()
	if m.root {
		kind = "system service"
	}
	fmt.Printf("Installing %s as a %s\n", serviceName, kind)

	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return err
	}
	_, statErr := os.Stat(m.unitPath)
	upgrade := statErr == nil
	if upgrade {
		m.systemctl("stop", serviceName)
	}

	self, err := os.Executable()
	if err != nil {
		return err
	}
	if self, err = filepath.EvalSymlinks(self); err != nil {
		return err
	}
	if self != m.binary() {
		if err := copyFile(self, m.binary(), 0o755); err != nil {
			return fmt.Errorf("copy binary: %w", err)
		}
	}

	if _, err := os.Stat(m.config()); errors.Is(err, fs.ErrNotExist) && seedConfig != "" {
		if err := copyFile(seedConfig, m.config(), 0o600); err != nil {
			return fmt.Errorf("copy configuration %s: %w", seedConfig, err)
		}
		fmt.Println("Configuration taken from", seedConfig)
	}
	cfg, _, err := config.Load(m.config())
	if err != nil {
		return err
	}

	if m.root {
		uid, gid, err := ensureSystemUser(m.dir)
		if err != nil {
			return err
		}
		// the service owns its data, the binary stays owned by root
		err = filepath.WalkDir(m.dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || path == m.binary() {
				return err
			}
			return os.Lchown(path, uid, gid)
		})
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(m.unitPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(m.unitPath, []byte(m.unit()), 0o644); err != nil {
		return err
	}
	if out, err := m.systemctl("daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %v %s", err, out)
	}
	if out, err := m.systemctl("enable", serviceName); err != nil {
		return fmt.Errorf("systemctl enable: %v %s", err, out)
	}
	if out, err := m.systemctl("restart", serviceName); err != nil {
		return fmt.Errorf("systemctl restart: %v %s", err, out)
	}
	if !m.root {
		ensureLinger()
	}

	time.Sleep(2 * time.Second)
	state, _ := m.systemctl("is-active", serviceName)
	fmt.Println()
	fmt.Println("Service:      ", serviceName, "("+state+")")
	fmt.Println("Directory:    ", m.dir)
	fmt.Println("Configuration:", m.config())
	fmt.Println("Unit:         ", m.unitPath)
	host := cfg.Web.Host
	if host == "0.0.0.0" || host == "::" {
		host = proxy.OutboundIP()
	}
	fmt.Println("Web interface: http://" + net.JoinHostPort(host, strconv.Itoa(cfg.Web.Port)))
	journal := "journalctl -u " + serviceName + " -f"
	if !m.root {
		journal = "journalctl --user -u " + serviceName + " -f"
	}
	fmt.Println("Log:          ", journal)
	if state != "active" {
		return fmt.Errorf("the service is %s, see the log: %s", state, journal)
	}
	return nil
}

func currentUserName() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return strconv.Itoa(os.Getuid())
}

// ensureLinger keeps user services running without a login session
func ensureLinger() {
	name := currentUserName()
	out, err := exec.Command("loginctl", "show-user", name, "-p", "Linger").Output()
	if err == nil && strings.TrimSpace(string(out)) == "Linger=yes" {
		return
	}
	if err := exec.Command("loginctl", "enable-linger", name).Run(); err != nil {
		fmt.Println("Warning: the service stops when you log out. Run as root: loginctl enable-linger", name)
		return
	}
	fmt.Println("Enabled lingering, the service works without a login session")
}

// runUninstall stops and removes the service; with purge it also removes the data
func runUninstall(purge bool) error {
	m, err := detectMode()
	if err != nil {
		return err
	}
	if _, err := os.Stat(m.unitPath); errors.Is(err, fs.ErrNotExist) {
		fmt.Println("The service is not installed:", m.unitPath, "not found")
	} else {
		m.systemctl("disable", "--now", serviceName)
		if err := os.Remove(m.unitPath); err != nil {
			return err
		}
		m.systemctl("daemon-reload")
		fmt.Println("Service", serviceName, "removed")
	}
	if !purge {
		fmt.Println("Data kept in", m.dir, "(configuration with secrets, logs, statistics); --uninstall --purge removes it")
		return nil
	}
	if err := os.RemoveAll(m.dir); err != nil {
		return err
	}
	fmt.Println("Removed", m.dir)
	if m.root {
		if _, err := user.Lookup(systemUser); err == nil {
			if out, err := exec.Command("userdel", systemUser).CombinedOutput(); err != nil {
				fmt.Printf("Can't remove user %s: %v %s\n", systemUser, err, out)
			} else {
				fmt.Println("Removed system user", systemUser)
			}
		}
	}
	return nil
}
