//go:build !linux && !darwin

package sockbind

func bind(int, string, string) error { return nil }

// pins reports whether this platform pins sockets at all.
const pins = false
