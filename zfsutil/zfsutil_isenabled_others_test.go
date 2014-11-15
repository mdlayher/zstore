// +build !linux

package zfsutil

import (
	"testing"
)

// TestOthersIsEnabled verifies that IsEnabled always returns false and an error
// on non-Linux operating systems.
func TestOthersIsEnabled(t *testing.T) {
	ok, err := IsEnabled()
	if ok || err != ErrNotImplemented {
		t.Fatalf("IsEnabled() should return (%v, %v), but returned (%v, %v)", false, ErrNotImplemented, ok, err)
	}
}
