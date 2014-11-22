// Package zfsutil provides ZFS utility operations for the zstore project.
package zfsutil

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/mistifyio/go-zfs.v1"
)

const (
	// fevZFS is the name of the FreeBSD or Linux ZFS virtual device
	devZFS = "/dev/zfs"

	// DatasetVolume is the type reported for a ZFS volume.
	// TODO(mdlayher): replace with zfs.DatasetVolume constant once new stable
	// TODO(mdlayher): go-zfs release is tagged
	DatasetVolume = "volume"

	// ZpoolName is the name of the ZFS zpool which zstored manages.
	ZpoolName = "zstore"

	// ZpoolOnline is the status reported when a zpool is online and healthy.
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

// SlugSize checks if an input slug string is a valid size constant for
// zstore, and returns the size if possible.
func SlugSize(slug string) (int64, bool) {
	// Common size constants for volume creation and resizing.
	const (
		MB = 1 * 1024 * 1024
		GB = 1024 * MB
	)

	// Map of available slugs and int64 sizes
	storageSizeMap := map[string]int64{
		"256M": 256 * MB,
		"512M": 512 * MB,
		"1G":   1 * GB,
		"2G":   2 * GB,
		"4G":   4 * GB,
		"8G":   8 * GB,
	}

	size, ok := storageSizeMap[slug]
	return size, ok
}

// Zpool returns the designated zpool for zstored operations.
func Zpool() (*zfs.Zpool, error) {
	return zfs.GetZpool(ZpoolName)
}
