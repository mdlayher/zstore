package storage

import (
	"sort"
	"strconv"
)

// Common size constants for volume creation and resizing.
const (
	MB = 1 * 1024 * 1024
	GB = 1024 * MB
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
