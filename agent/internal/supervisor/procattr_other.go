//go:build !linux

package supervisor

import "syscall"

// procAttr: mqvpn gets its own process group. There is no parent-death
// signal outside Linux; Spec.PidFile covers an agent that died without
// stopping mqvpn (macOS: launchd restarts the agent, and mqvpn holds a
// single-instance lock).
func procAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
