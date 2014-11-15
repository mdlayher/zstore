// Package zfsutil provides ZFS utility operations for the zstore project.
package zfsutil

import (
	"errors"
	"fmt"

	"gopkg.in/mistifyio/go-zfs.v1"
)

const (
	// linuxDevZFS is the name of the Linux ZFS virtual device
	linuxDevZFS = "/dev/zfs"

	// ZpoolName is the name of the ZFS zpool which zstored manages
	ZpoolName = "zstore"

	// ZpoolOnline is the status reported when a zpool is online and healthy
	// TODO(mdlayher): replace with zfs.ZpoolOnline constant once new stable
	// TODO(mdlayher): go-zfs release is tagged
	ZpoolOnline = "ONLINE"
)

var (
	// ErrNotImplemented is returned when zstore functionality is not implemented
	// on the current operating system.
	ErrNotImplemented = errors.New("not implemented")
)

// IsZFSPermissionDenied determines if an input error is caused by the current
// user not having permission to manipulate the ZFS virtual device.
func IsZFSPermissionDenied(err error) bool {
	// Check for ZFS error
	zErr, ok := err.(*zfs.Error)
	if !ok {
		// Not a ZFS error at all
		return false
	}

	// Check for specific error string from stderr
	return zErr.Stderr == fmt.Sprintf("Unable to open %s: Permission denied.\n", linuxDevZFS)
}

// IsZpoolNotExists determines if an input error is caused by the necessary
// zpool not existing when zstored is run.
func IsZpoolNotExists(err error) bool {
	// Check for ZFS error
	zErr, ok := err.(*zfs.Error)
	if !ok {
		// Not a ZFS error at all
		return false
	}

	// Check for specific error string from stderr
	return zErr.Stderr == fmt.Sprintf("cannot open '%s': no such pool\n", ZpoolName)
}

// Zpool returns the designated zpool for zstored operations.
func Zpool() (*zfs.Zpool, error) {
	return zfs.GetZpool(ZpoolName)
}
