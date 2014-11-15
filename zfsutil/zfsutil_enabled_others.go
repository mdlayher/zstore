// +build !linux

package zfsutil

// Enabled always returns an error on non-Linux operating systems.
func Enabled() (bool, error) {
	return false, ErrNotImplemented
}
