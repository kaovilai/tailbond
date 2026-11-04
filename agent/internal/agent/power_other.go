//go:build !darwin

package agent

import "context"

// readPower: only a Mac server is checked (Linux servers are always-on
// machines; a laptop running Linux is out of scope here).
func readPower(context.Context) *powerInfo { return nil }
