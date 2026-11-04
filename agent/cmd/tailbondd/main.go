// Command tailbondd is the tailbond management agent: it supervises mqvpn and
// serves the web UI and HTTP API (see docs/architecture.md).
//
//	tailbondd [-config FILE] ...                  run the agent
//	tailbondd backup --out FILE [--plain]         encrypted backup (or a summary)
//	tailbondd restore FILE [--yes]                restore one
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kaovilai/tailbond/agent/internal/agent"
	"github.com/kaovilai/tailbond/agent/internal/backup"
	"github.com/kaovilai/tailbond/agent/internal/config"
	"github.com/kaovilai/tailbond/agent/internal/sysproxy"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "backup":
			cmdBackup(os.Args[2:])
			return
		case "restore":
			cmdRestore(os.Args[2:])
			return
		case "restore-proxy":
			cmdRestoreProxy(os.Args[2:])
			return
		}
	}
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
		if errors.Is(err, agent.ErrRestart) {
			os.Exit(agent.ExitRestart)
		}
		fatal(err)
	}
}

// cmdRestoreProxy puts back the system proxy a phone hub left set
// (docs/phone-hub.md), for uninstall.sh: tailbondd does it itself at start
// and on the way out.
func cmdRestoreProxy(args []string) {
	fs := flag.NewFlagSet("restore-proxy", flag.ExitOnError)
	cfgPath := fs.String("config", config.DefaultPath, "path to tailbond.json")
	_ = fs.Parse(args)
	store, err := config.Load(*cfgPath)
	if err != nil {
		fatal(err)
	}
	m := &sysproxy.Mac{RecordPath: filepath.Join(store.Get().StateDir, "phone-hub-proxy.json")}
	r, ok := m.Active()
	if !ok {
		return
	}
	if err := m.Restore(context.Background()); err != nil {
		fatal(err)
	}
	fmt.Printf("put back the proxy settings of %q\n", r.Service)
}

func cmdBackup(args []string) {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	cfgPath := fs.String("config", config.DefaultPath, "path to tailbond.json")
	out := fs.String("out", "", "write the encrypted backup here")
	passFile := fs.String("passphrase-file", "", "read the passphrase from this file (else TB_BACKUP_PASSPHRASE, else a prompt)")
	plain := fs.Bool("plain", false, "print a summary without keys or passwords instead")
	_ = fs.Parse(args)
	store, err := config.Load(*cfgPath)
	if err != nil {
		fatal(err)
	}
	cfg := store.Get()
	uci := uciExport()
	if *plain {
		p, err := backup.Collect(*cfgPath, cfg.StateDir, uci, cfg)
		if err != nil {
			fatal(err)
		}
		b, _ := jsonIndent(backup.Summary(cfg, p))
		fmt.Println(string(b))
		return
	}
	if *out == "" {
		fatal(errors.New("backup: --out FILE (or --plain)"))
	}
	pass := passphrase(*passFile, true)
	host, _ := os.Hostname()
	data, err := agent.MakeBackup(*cfgPath, cfg.StateDir, uci, pass, backup.Header{Role: cfg.Role, Platform: platform(), Host: host, AgentVersion: version}, cfg)
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*out, data, 0o600); err != nil {
		fatal(err)
	}
	fmt.Println("wrote", *out)
}

func cmdRestore(args []string) {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	cfgPath := fs.String("config", config.DefaultPath, "path to tailbond.json")
	passFile := fs.String("passphrase-file", "", "read the passphrase from this file (else TB_BACKUP_PASSPHRASE, else a prompt)")
	yes := fs.Bool("yes", false, "don't ask before replacing the settings")
	_ = fs.Parse(args)
	if fs.NArg() != 1 {
		fatal(errors.New("restore FILE"))
	}
	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		fatal(err)
	}
	store, err := config.Load(*cfgPath)
	if err != nil {
		fatal(err)
	}
	pass := passphrase(*passFile, false)
	h, pt, err := backup.Open(data, pass)
	if err != nil {
		fatal(err)
	}
	p, next, err := backup.Decode(pt)
	if err != nil {
		fatal(err)
	}
	cur := store.Get()
	if next.Role != cur.Role {
		fatal(fmt.Errorf("this is a %s backup; this machine is a %s (change role first)", next.Role, cur.Role))
	}
	fmt.Printf("Backup of %s (%s, %s), made %s by tailbond %s.\n", h.Host, h.Role, h.Platform, h.Created, h.AgentVersion)
	for _, c := range backup.Changes(cur, next) {
		fmt.Printf("  %s: %s -> %s\n", c.What, c.From, c.To)
	}
	if !*yes && !confirm("Replace this machine's tailbond settings with the backup? [y/N] ") {
		fmt.Println("nothing changed")
		return
	}
	host, _ := os.Hostname()
	rollback, err := agent.RestoreBackup(store, p, next, pass, uciExport(), backup.Header{Role: cur.Role, Platform: platform(), Host: host, AgentVersion: version}, platform() == "openwrt")
	if err != nil {
		fatal(err)
	}
	fmt.Println("restored; the previous settings are in", rollback)
	fmt.Println("restart tailbond now: systemctl restart tailbond (OpenWrt: /etc/init.d/tailbond restart)")
}

func passphrase(file string, twice bool) string {
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			fatal(err)
		}
		return strings.TrimRight(string(b), "\r\n")
	}
	if p := os.Getenv("TB_BACKUP_PASSPHRASE"); p != "" {
		return p
	}
	p := prompt("Passphrase: ")
	if twice && prompt("Again: ") != p {
		fatal(errors.New("the passphrases differ"))
	}
	return p
}

// prompt reads a line without echo when stdin is a terminal (stty).
func prompt(msg string) string {
	fmt.Fprint(os.Stderr, msg)
	off := exec.Command("stty", "-echo")
	off.Stdin = os.Stdin
	echoOff := off.Run() == nil
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if echoOff {
		on := exec.Command("stty", "echo")
		on.Stdin = os.Stdin
		_ = on.Run()
		fmt.Fprintln(os.Stderr)
	}
	return strings.TrimRight(line, "\r\n")
}

func confirm(msg string) bool {
	fmt.Fprint(os.Stderr, msg)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "y")
}

func platform() string {
	if _, err := os.Stat("/etc/openwrt_release"); err == nil {
		return "openwrt"
	}
	if _, err := os.Stat("/etc/fedora-release"); err == nil {
		return "fedora"
	}
	return "linux"
}

func uciExport() string {
	if platform() != "openwrt" {
		return ""
	}
	out, err := exec.Command("uci", "export", "tailbond").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "tailbondd:", err)
	os.Exit(1)
}

func jsonIndent(v any) ([]byte, error) { return json.MarshalIndent(v, "", "  ") }
