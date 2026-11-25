// laser-mtproxy is an MTProto proxy for Telegram applications with WebSocket transport
// (a Go port of tg-ws-proxy), a gateway for Bot API clients and a web interface
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/app"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/web"
)

func main() {
	configPath := flag.String("config", "laser-mtproxy.json", "path of the configuration file, it is created if missing")
	verbose := flag.Bool("v", false, "debug logging (overrides \"verbose\" of the configuration)")
	checkOnly := flag.Bool("check", false, "check the configuration and exit")
	showVersion := flag.Bool("version", false, "print the version and exit")
	install := flag.Bool("install", false, "install or update the systemd service: a user service, or a system service when run by root;\n"+
		"-config gives the initial configuration, an existing one is kept")
	uninstall := flag.Bool("uninstall", false, "stop and remove the systemd service, the data is kept")
	purge := flag.Bool("purge", false, "with -uninstall: also remove the configuration, logs and statistics")
	flag.Parse()

	if *showVersion {
		fmt.Println("laser-mtproxy", app.Version)
		return
	}
	if *install || *uninstall {
		var err error
		if *install {
			seed := ""
			flag.Visit(func(f *flag.Flag) {
				if f.Name == "config" {
					seed, _ = filepath.Abs(*configPath)
				}
			})
			err = runInstall(seed)
		} else {
			err = runUninstall(*purge)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return
	}
	path, err := filepath.Abs(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *checkOnly {
		if _, err := config.Check(path); err != nil {
			fmt.Fprintln(os.Stderr, "Configuration error:", err)
			os.Exit(1)
		}
		fmt.Println("Configuration is valid:", path)
		return
	}
	cfg, changed, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Configuration error:", err)
		os.Exit(1)
	}
	if *verbose {
		cfg.Verbose = true
	}
	if changed {
		logx.Infof("Configuration written: %s", path)
	}

	a := app.New(path, cfg)
	a.Start()
	w := web.New(a)
	if err := w.Start(); err != nil {
		logx.Errorf("Can't start web interface: %v", err)
		a.Stop()
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	w.Stop()
	a.Stop()
}
