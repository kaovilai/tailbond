//go:build darwin

package agent

import (
	"context"
	"os/exec"
	"time"
)

func readPower(ctx context.Context) *powerInfo {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	a, err := exec.CommandContext(ctx, "pmset", "-g", "assertions").Output()
	if err != nil {
		return nil
	}
	b, _ := exec.CommandContext(ctx, "pmset", "-g", "batt").Output()
	p := &powerInfo{Awake: parsePmsetAssertions(string(a))}
	p.Laptop, p.OnBattery = parsePmsetBatt(string(b))
	return p
}
