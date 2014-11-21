// +build !freebsd,!linux

package zfsutil

// IsEnabled always returns an error on non-Linux operating systems.
func IsEnabled() (bool, error) {
	return false, ErrNotImplemented
}
