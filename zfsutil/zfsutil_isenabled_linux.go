// +build linux

package zfsutil

import (
	"os"
)

// IsEnabled verifies that the Linux ZFS kernel module is loaded.
func IsEnabled() (bool, error) {
	// Verify that Linux ZFS kernel module is loaded by checking for ZFS
	// virtual device
	if _, err := os.Stat(linuxDevZFS); err != nil {
		// Module not loaded
		if os.IsNotExist(err) {
			return false, nil
		}

		// Other error
		return false, err
	}

	// Module loaded and ready
	return true, nil
}
