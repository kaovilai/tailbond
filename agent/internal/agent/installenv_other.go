//go:build !darwin

package agent

// defaultInstallEnv marks an install by packaging/linux/install.sh.
const defaultInstallEnv = "/etc/tailbond/install.env"
