//go:build unix

package cdpclient

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// tempDirPrefix is the basename prefix of the isolated per-process directory
// tree Launch creates under os.TempDir() (see Launch). It is also how the
// orphan sweep recognizes a Chrome that belongs to chromerpc.
const tempDirPrefix = "chromerpc-"

// setProcessGroup makes the launched Chrome the leader of a new process group,
// so its whole tree (main process + renderer/GPU/utility helpers) can be
// signalled at once with a negative-pid kill. Without this, killing only the
// main pid leaves the helpers — and on abnormal parent death the entire tree —
// running and reparented to init.
func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessGroup kills the entire process group led by pid. Falls back to
// killing pid alone if the group signal fails. Safe with pid <= 0 (no-op).
func killProcessGroup(pid int) {
	if pid <= 0 {
		return
	}
	// Negative pid targets the process group whose leader is pid (set up by
	// setProcessGroup). This reaches every Chrome helper, not just the main.
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		// Group may not exist (e.g. Setpgid unsupported); kill the bare pid.
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
}

var sweepOnce sync.Once

// sweepOrphansOnce runs SweepOrphans exactly once per process, lazily, the first
// time Chrome is launched. It bounds the blast radius of any past leak without
// requiring every caller to remember to sweep.
func sweepOrphansOnce() {
	sweepOnce.Do(func() {
		if n := SweepOrphans(); n > 0 {
			log.Printf("launcher: swept %d orphaned chromerpc Chrome process(es)", n)
		}
	})
}

// SweepOrphans finds Chrome processes that were launched by chromerpc (their
// --user-data-dir lives under <TMPDIR>/chromerpc-*) but are now orphaned —
// reparented to init (ppid 1) because the process that launched them died
// without cleaning up — kills their process groups, and removes stale
// chromerpc-* temp directory trees no live process still owns. It returns the
// number of orphaned Chrome processes it killed.
//
// This is the only defense against a launcher that was SIGKILLed (uncatchable):
// signal handlers and deferred Cleanup never run in that case, so the orphans
// can only be reaped after the fact, on the next launch.
func SweepOrphans() int {
	tmp := os.TempDir()
	marker := "--user-data-dir=" + filepath.Join(tmp, tempDirPrefix)

	// One `ps` snapshot: pid, ppid, full command. Portable across macOS/Linux.
	out, err := exec.Command("ps", "-Ao", "pid=,ppid=,command=").Output()
	if err != nil {
		return 0
	}

	live := map[string]bool{} // chromerpc temp base dirs with a live owner
	killed := 0
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if !strings.Contains(line, marker) {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		ppid, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil {
			continue
		}
		base := chromerpcBaseFromCmd(line, tmp)
		// ppid==1 means the launching process is gone: a true orphan. A Chrome
		// still owned by a live launcher (ppid!=1) is left alone.
		if ppid == 1 {
			killProcessGroup(pid)
			killed++
			// Its dir is now reapable; do not mark it live.
			continue
		}
		if base != "" {
			live[base] = true
		}
	}

	// Remove chromerpc-* temp trees that no live process owns and that are not
	// brand-new (avoid racing a launch mid-flight).
	if entries, err := os.ReadDir(tmp); err == nil {
		cutoff := time.Now().Add(-30 * time.Second)
		for _, e := range entries {
			if !strings.HasPrefix(e.Name(), tempDirPrefix) {
				continue
			}
			base := filepath.Join(tmp, e.Name())
			if live[base] {
				continue
			}
			if info, err := e.Info(); err == nil && info.ModTime().After(cutoff) {
				continue // possibly a launch in progress
			}
			_ = os.RemoveAll(base)
		}
	}
	return killed
}

// chromerpcBaseFromCmd extracts the <TMPDIR>/chromerpc-<id> base directory from
// a Chrome command line's --user-data-dir=<base>/user-data argument.
func chromerpcBaseFromCmd(cmdline, tmp string) string {
	const flag = "--user-data-dir="
	i := strings.Index(cmdline, flag)
	if i < 0 {
		return ""
	}
	rest := cmdline[i+len(flag):]
	if j := strings.IndexByte(rest, ' '); j >= 0 {
		rest = rest[:j]
	}
	// rest is <base>/user-data; the base is its parent.
	base := filepath.Dir(rest)
	if strings.HasPrefix(filepath.Base(base), tempDirPrefix) {
		return base
	}
	return ""
}
