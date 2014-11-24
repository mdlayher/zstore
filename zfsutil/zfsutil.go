// Package zfsutil provides ZFS utility operations for the zstore project.
package zfsutil

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
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

// Common size constants for volume creation and resizing.
const (
	MB = 1 * 1024 * 1024
	GB = 1024 * MB
)

var (
	// ErrNotImplemented is returned when zstore functionality is not implemented
	// on the current operating system.
	ErrNotImplemented = errors.New("not implemented")
)

// storageSizeMap is a map of available slugs and int64 sizes
var storageSizeMap = map[string]int64{
	"256M": 256 * MB,
	"512M": 512 * MB,
	"1G":   1 * GB,
	"2G":   2 * GB,
	"4G":   4 * GB,
	"8G":   8 * GB,
}

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

// Slugs returns a sorted list of all available size slugs which zstore
// considers valid.
func Slugs() []string {
	// Retrieve all slugs from map; order is currently undefined
	var slugs []string
	for k := range storageSizeMap {
		slugs = append(slugs, k)
	}

	// Sort slugs using custom sort type
	sort.Sort(bySizeSlug(slugs))
	return slugs
}

// SlugSize checks if an input slug string is a valid size constant for
// zstore, and returns the size if possible.
func SlugSize(slug string) (int64, bool) {
	size, ok := storageSizeMap[slug]
	return size, ok
}

// Zpool returns the designated zpool for zstored operations.
func Zpool() (*zfs.Zpool, error) {
	return zfs.GetZpool(ZpoolName)
}

// bySizeSlug implements sort.Interface, for use in sorting size slugs which
// contain both an integer and a byte suffix, such as 256M, 1G, 2T, etc.
type bySizeSlug []string

// Len returns the length of the collection.
func (s bySizeSlug) Len() int {
	return len(s)
}

// Swap swaps to values by their index.
func (s bySizeSlug) Swap(i int, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less compares each size slug using both its integer value and its byte suffix.
func (s bySizeSlug) Less(i int, j int) bool {
	// Map known byte suffixes to precedence values, for easy
	// comparison
	suffixPrecedence := map[string]int{
		"B": 0,
		"K": 1,
		"M": 2,
		"G": 3,
		"T": 4,
		"P": 5,
		// It's fairly likely these won't be used, but why not?
		"E": 6,
		"Z": 7,
		"Y": 8,
	}

	// Capture the suffix character for the elements at indices
	// i and j
	iSuffix := s[i][len(s[i])-1:]
	jSuffix := s[j][len(s[j])-1:]

	// Ensure both suffix characters are present in map
	iPrecedence, iOK := suffixPrecedence[iSuffix]
	jPrecedence, jOK := suffixPrecedence[jSuffix]
	if !iOK || !jOK {
		panic("unknown size slug suffix")
	}

	// If i has a smaller suffix than j, i is less
	// Conversely, if j has a smaller suffix than i, j is less
	if iPrecedence < jPrecedence {
		return true
	} else if jPrecedence < iPrecedence {
		return false
	}

	// If both have the same suffix, we must compare the integers
	if iPrecedence == jPrecedence {
		// Retrieve int64 value of each integer
		iValue, err := strconv.ParseUint(s[i][:len(s[i])-1], 10, 64)
		if err != nil {
			panic(err)
		}
		jValue, err := strconv.ParseUint(s[j][:len(s[j])-1], 10, 64)
		if err != nil {
			panic(err)
		}

		return iValue < jValue
	}

	// Ensure sort always returns
	panic("cannot sort by size slug")
}
