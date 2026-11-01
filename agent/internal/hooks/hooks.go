// Package hooks runs the platform hook executables from hooks_dir
// (docs/architecture.md, "Platform hooks").
package hooks

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

const (
	ServerUp   = "server-up"
	ServerDown = "server-down"
	ClientUp   = "client-up"
	ClientDown = "client-down"
	// Captive portal sign-in on one WAN (client role): LAN devices reach
	// that WAN directly until the portal lets the router through.
	PortalOpen  = "portal-open"
	PortalClose = "portal-close"
)

type Logger func(src, line string)

type Runner struct {
	Dir     string
	Log     Logger
	Timeout time.Duration
}

// Run executes <dir>/<name> and, if present, every executable in <dir>/<name>.d/
// in lexical order. Missing hooks are not an error.
func (r *Runner) Run(name string, env map[string]string) {
	if r.Dir == "" {
		return
	}
	var cands []string
	if fi, err := os.Stat(filepath.Join(r.Dir, name)); err == nil && !fi.IsDir() {
		cands = append(cands, filepath.Join(r.Dir, name))
	}
	if ents, err := os.ReadDir(filepath.Join(r.Dir, name+".d")); err == nil {
		var extra []string
		for _, e := range ents {
			if !e.IsDir() {
				extra = append(extra, filepath.Join(r.Dir, name+".d", e.Name()))
			}
		}
		sort.Strings(extra)
		cands = append(cands, extra...)
	}
	for _, path := range cands {
		r.exec(name, path, env)
	}
}

func (r *Runner) exec(name, path string, env map[string]string) {
	fi, err := os.Stat(path)
	if err != nil || fi.Mode()&0o111 == 0 {
		return
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TB_HOOK="+name)
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		cmd.Env = append(cmd.Env, k+"="+env[k])
	}
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	r.log("running " + path)
	err = cmd.Run()
	sc := bufio.NewScanner(&out)
	for sc.Scan() {
		r.log(name + ": " + sc.Text())
	}
	if err != nil {
		r.log(name + ": " + filepath.Base(path) + " failed: " + err.Error())
	}
}

func (r *Runner) log(line string) {
	if r.Log != nil {
		r.Log("hook", line)
	}
}
