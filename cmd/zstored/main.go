// Command zstored provides a prototype, ZFS-based, object storage daemon.
package main

import (
	"flag"
	"log"
	"os"
	"runtime"

	"github.com/mdlayher/zstore/zfsutil"
)

var (
	// host is the address to which the HTTP server is bound
	host string
)

func init() {
	flag.StringVar(&host, "host", ":5000", "HTTP server host")
}

func main() {
	// Parse CLI flags
	flag.Parse()

	// Set up logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetPrefix("zstored: ")
	log.Printf("starting [pid: %d]", os.Getpid())

	// Check if ZFS is enabled on this operating system
	ok, err := zfsutil.IsEnabled()
	if err != nil {
		// If not implemented, zstore currently does not run on the host
		// operating system
		if err == zfsutil.ErrNotImplemented {
			log.Fatalf("zstored currently does not run on the %q operating system", runtime.GOOS)
		}

		// All other errors
		log.Fatal(err)
	}

	// No error, but ZFS kernel module not loaded on this system
	if !ok {
		log.Fatal("ZFS kernel module not loaded")
	}

	// Ensure that the necessary zpool is already in place, since building a zpool
	// may be too complicated or risky to do on program startup
	zpool, err := zfsutil.Zpool()
	if err != nil {
		// Check for permission denied
		if zfsutil.IsZFSPermissionDenied(err) {
			log.Fatalf("permission denied to ZFS virtual device, please run as root")
		}

		// Check for zpool not exists
		if zfsutil.IsZpoolNotExists(err) {
			log.Fatalf("required zpool %q does not exist, please create the zpool", zfsutil.ZpoolName)
		}

		// All other errors
		log.Fatal(err)
	}

	// Calculate zpool statistics in gigabytes, percent full
	allocGB := float64(zpool.Allocated) / 1024 / 1024 / 1024
	totalGB := float64(zpool.Size) / 1024 / 1024 / 1024
	percent := int(float64(float64(zpool.Allocated)/float64(zpool.Size)) * 100)

	log.Printf("zpool: %s [%s] [%03.3f / %03.3f GB, %03d%%]", zpool.Name, zpool.Health, allocGB, totalGB, percent)

	// Ensure zpool is online
	if zpool.Health != zfsutil.ZpoolOnline {
		log.Fatalf("zpool %q unhealthy, status: %q; exiting now", zpool.Name, zpool.Health)
	}

	log.Println("listening", host)
}
