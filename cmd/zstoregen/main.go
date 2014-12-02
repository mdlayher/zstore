// Command zstoregen provides a testing utility which can be used to quickly set
// up a temporary ZFS zpool for zstored, using temporary files.
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"runtime"

	"github.com/mdlayher/zstore/storage/zfsutil"

	"gopkg.in/mistifyio/go-zfs.v2"
)

var (
	// n is the number of temporary files to generate and add to the zpool
	n uint

	// s is the size slug which determines the size of each temporary file
	s string
)

func init() {
	flag.UintVar(&n, "n", 1, "number of temporary files to create for zpool")
	flag.StringVar(&s, "s", "256M", "size slug for each file to add to zpool")
}

func main() {
	// Parse CLI flags
	flag.Parse()

	// Set up logging
	log.SetPrefix("zstoregen: ")

	// Check if ZFS is enabled on this operating system
	ok, err := zfsutil.IsEnabled()
	if err != nil {
		// If not implemented, zstore currently does not run on the host
		// operating system
		if err == zfsutil.ErrNotImplemented {
			log.Fatalf("zstoregen currently does not run on the %q operating system", runtime.GOOS)
		}

		// All other errors
		log.Fatal(err)
	}

	// No error, but ZFS kernel module not loaded on this system
	if !ok {
		log.Fatal("ZFS kernel module not loaded, exiting")
	}

	// Check for valid size slug
	size, ok := zfsutil.SlugSize(s)
	if !ok {
		log.Fatalf("invalid size slug: %q [sizes: %s]", s, zfsutil.Slugs())
	}

	// Check if the zstore zpool already exists
	if _, err := zfsutil.Zpool(); err != nil && !zfsutil.IsZpoolNotExists(err) {
		// Check for permission denied
		if zfsutil.IsZFSPermissionDenied(err) {
			log.Fatalf("permission denied to ZFS virtual device, exiting")
		}

		// All other errors
		log.Fatal(err)
	}

	log.Printf("generating %d temporary files, size %s", n, s)

	// Generate n temporary files
	var tmpFiles []string
	for i := uint(0); i < n; i++ {
		// Make a temporary file
		f, err := ioutil.TempFile(os.TempDir(), zfsutil.ZpoolName)
		if err != nil {
			log.Fatal(err)
		}

		// Truncate to appropriate size from slug
		if err := f.Truncate(size); err != nil {
			log.Fatal(err)
		}

		// Close file
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}

		// Add to slice of files for use as vdevs
		tmpFiles = append(tmpFiles, f.Name())
		log.Printf("  - [%02d] %s", i, f.Name())
	}

	// Create the zstore zpool
	if _, err = zfs.CreateZpool(zfsutil.ZpoolName, nil, tmpFiles...); err != nil {
		log.Fatal(err)
	}

	log.Printf("created zpool %q [%d x %s]", zfsutil.ZpoolName, n, s)
}
