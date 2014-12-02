// +build zfs

package zfsutil

import (
	"testing"
)

// TestIntegrationZpool verifies that Zpool returns the zstore zpool when
// ZFS is enabled and the pool exists.
func TestIntegrationZpool(t *testing.T) {
	// Check for the zpool
	zpool, err := Zpool()
	if err != nil {
		// If permission is denied, skip test
		if IsZFSPermissionDenied(err) {
			t.Skipf("permission denied to ZFS virtual device, skipping integration test")
		}

		// If the zpool does not exist, skip test
		if IsZpoolNotExists(err) {
			t.Skipf("zpool %q does not exist, skipping integration test")
		}

		// Fail test on other errors
		t.Fatal(err)
	}

	// Verify name
	if zpool.Name != ZpoolName {
		t.Fatalf("unexpected zpool name: %v != %v", zpool.Name, ZpoolName)
	}
}
