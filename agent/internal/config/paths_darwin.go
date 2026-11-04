//go:build darwin

package config

// macOS locations (packaging/macos, docs/macos.md). The tunnel is a utun
// device: mqvpn asks for "utun" and the kernel picks the unit (the agent
// reads the real name from mqvpn's log). The dashboard listens on this
// Mac only.
const (
	DefaultPath     = "/usr/local/etc/tailbond/tailbond.json"
	DefaultStateDir = "/usr/local/var/lib/tailbond"
	DefaultHooksDir = "/usr/local/lib/tailbond/hooks"
	DefaultMqvpnBin = "/usr/local/bin/mqvpn"
	DefaultTunName  = "utun"
	DefaultListen   = "127.0.0.1:8077"
)
