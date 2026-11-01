// Command tailbondd is the tailbond management agent: it supervises mqvpn and
// serves the web UI and HTTP API (see docs/architecture.md).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tailbond/tailbond/agent/internal/agent"
	"github.com/tailbond/tailbond/agent/internal/config"
)

var version = "dev"

func main() {
	cfgPath := flag.String("config", config.DefaultPath, "path to tailbond.json")
	role := flag.String("role", "", "override the configured role (server|client)")
	listen := flag.String("listen", "", "override the configured HTTP listen address")
	webDir := flag.String("web-dir", "", "serve the UI from this directory instead of the embedded copy (development)")
	quiet := flag.Bool("quiet", false, "do not copy logs to stderr")
	showVersion := flag.Bool("version", false, "print version and exit")
	initCfg := flag.Bool("init", false, "write a default config to -config if it does not exist, then exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("tailbondd", version)
		return
	}
	if *initCfg {
		if _, err := os.Stat(*cfgPath); err == nil {
			fmt.Fprintf(os.Stderr, "%s already exists\n", *cfgPath)
			return
		}
		c := config.Default()
		if *role != "" {
			c.Role = *role
		}
		if err := config.WriteFile(*cfgPath, c); err != nil {
			fatal(err)
		}
		fmt.Println("wrote", *cfgPath)
		return
	}

	store, err := config.Load(*cfgPath)
	if err != nil {
		fatal(err)
	}
	if err := store.SetOverrides(*role, *listen); err != nil {
		fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	a := agent.New(store, agent.Options{Version: version, WebDir: *webDir, Echo: !*quiet})
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for range hup {
			a.Rescan()
		}
	}()
	if err := a.Run(ctx); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "tailbondd:", err)
	os.Exit(1)
}
