package supervisor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// A process named in PidFile and running Spec.Bin is stopped; anything else
// named there (a reused pid) is left alone.
func TestStopOrphan(t *testing.T) {
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("no sleep binary")
	}
	pidFile := filepath.Join(t.TempDir(), "mqvpn.pid")
	// each process is reaped as soon as it exits, as launchd would
	start := func() (*exec.Cmd, chan struct{}) {
		c := exec.Command(sleep, "30")
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() { _, _ = c.Process.Wait(); close(done) }()
		t.Cleanup(func() { _ = c.Process.Kill(); <-done })
		return c, done
	}

	other, _ := start()
	_ = os.WriteFile(pidFile, []byte(strconv.Itoa(other.Process.Pid)), 0o644)
	New(Spec{Bin: "/usr/bin/mqvpn", PidFile: pidFile}, NewRing(10)).stopOrphan()
	if syscall.Kill(other.Process.Pid, 0) != nil {
		t.Fatal("stopped a process that isn't running Spec.Bin")
	}

	orphan, gone := start()
	_ = os.WriteFile(pidFile, []byte(strconv.Itoa(orphan.Process.Pid)), 0o644)
	New(Spec{Bin: sleep, PidFile: pidFile}, NewRing(10)).stopOrphan()
	select {
	case <-gone:
	case <-time.After(6 * time.Second):
		t.Fatal("the orphan is still running")
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatal("pid file not removed")
	}
}
