//go:build !darwin

package config

// Linux (and OpenWrt) locations.
const (
	DefaultPath     = "/etc/tailbond/tailbond.json"
	DefaultStateDir = "/var/lib/tailbond"
	DefaultHooksDir = "/usr/lib/tailbond/hooks"
	DefaultMqvpnBin = "/usr/bin/mqvpn"
	DefaultTunName  = "tb0"
	DefaultListen   = ":8077"
)
