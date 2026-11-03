//go:build linux

package supervisor

import "syscall"

// procAttr: mqvpn gets its own process group and dies with the agent.
func procAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGTERM}
}
