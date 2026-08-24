//go:build unix

package cdpclient

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestChromerpcBaseFromCmd(t *testing.T) {
	tmp := "/var/folders/xx/T"
	cases := []struct {
		cmd  string
		want string
	}{
		{"/Chrome --headless=new --user-data-dir=/var/folders/xx/T/chromerpc-abc123/user-data --disk-cache-dir=x about:blank", "/var/folders/xx/T/chromerpc-abc123"},
		{"/Chrome --user-data-dir=/var/folders/xx/T/chromerpc-deadbeef/user-data", "/var/folders/xx/T/chromerpc-deadbeef"},
		{"/Chrome --user-data-dir=/some/other/profile/user-data", ""}, // not a chromerpc dir
		{"/Chrome --headless=new about:blank", ""},                    // no user-data-dir
	}
	for _, c := range cases {
		if got := chromerpcBaseFromCmd(c.cmd, tmp); got != c.want {
			t.Errorf("chromerpcBaseFromCmd(%q) = %q, want %q", c.cmd, got, c.want)
		}
	}
}

// TestSweepOrphansReapsStaleDirs verifies the disk-side of the sweep: a stale
// chromerpc-* tree with no live owner is removed, while a brand-new one (which
// might be a launch in progress) is preserved.
func TestSweepOrphansReapsStaleDirs(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TMPDIR", root) // os.TempDir() honors TMPDIR on unix, uncached

	stale := filepath.Join(root, tempDirPrefix+"staleid")
	fresh := filepath.Join(root, tempDirPrefix+"freshid")
	other := filepath.Join(root, "unrelated-dir")
	for _, d := range []string{stale, fresh, other} {
		if err := os.MkdirAll(filepath.Join(d, "user-data"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	// Age the stale dir past the 30s cutoff; leave fresh at "now".
	old := time.Now().Add(-5 * time.Minute)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}

	SweepOrphans()

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale chromerpc dir should have been removed, stat err = %v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("fresh chromerpc dir should be preserved, stat err = %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("unrelated dir must never be touched, stat err = %v", err)
	}
}
