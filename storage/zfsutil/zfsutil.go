// Package zfsutil provides ZFS utility operations for the zstore project.
package zfsutil

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/mistifyio/go-zfs.v2"
)

const (
	// fevZFS is the name of the FreeBSD or Linux ZFS virtual device
	devZFS = "/dev/zfs"

	// ZpoolName is the name of the ZFS zpool which zstored manages.
	ZpoolName = "zstore"
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
	return zErr.Stderr == fmt.Sprintf("Unable to open %s: Permission denied.\n", devZFS)
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

// IsDatasetNotExists determines if an input error is caused by the necessary
// ZFS dataset not existing.
func IsDatasetNotExists(err error) bool {
	// Check for ZFS error
	zErr, ok := err.(*zfs.Error)
	if !ok {
		// Not a ZFS error at all
		return false
	}

	// Check for tail end of error string
	return strings.Contains(zErr.Stderr, "dataset does not exist\n")
}

// IsOutOfSpace determines if an input error is caused by the zpool being too
// full to process a volume creation request.
func IsOutOfSpace(err error) bool {
	// Check for ZFS error
	zErr, ok := err.(*zfs.Error)
	if !ok {
		// Not a ZFS error at all
		return false
	}

	// Check for tail end of error string
	return strings.Contains(zErr.Stderr, "out of space\n")
}

// Zpool returns the designated zpool for zstored operations.
func Zpool() (*zfs.Zpool, error) {
	return zfs.GetZpool(ZpoolName)
}
