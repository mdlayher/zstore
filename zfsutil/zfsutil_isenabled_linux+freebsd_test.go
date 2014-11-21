// +build freebsd linux

package zfsutil

import (
	"os"
	"testing"
)

// TestLinuxIsEnabled verifies that IsEnabled properly detects the presence or
// absence of the ZFS virtual device on FreeBSD and Linux operating systems.
func TestLinuxIsEnabled(t *testing.T) {
	// Check function result immediately
	enabled, err := IsEnabled()
	if err != nil {
		t.Fatal(err)
	}

	// Check for ZFS virtual device
	_, err = os.Stat(devZFS)
	if os.IsNotExist(err) {
		if enabled {
			t.Fatalf("could not find %q, but IsEnabled returned true", devZFS)
		}

		return
	}

	// Fail test on errors other than "not exists"
	if err != nil {
		t.Fatal(err)
	}

	// Verify ZFS is enabled
	if !enabled {
		t.Fatalf("found %q, but IsEnabled returned false", devZFS)
	}
}
