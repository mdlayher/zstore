// Command zstored provides a prototype, ZFS-based, block storage provisioning daemon.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/mdlayher/zstore/zfsutil"
	"github.com/mdlayher/zstore/zstored/zstoredhttp"

	"github.com/stretchr/graceful"
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
	log.Printf("starting [os: %s_%s] [pid: %d]", runtime.GOOS, runtime.GOARCH, os.Getpid())

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
		log.Fatal("ZFS kernel module not loaded, exiting")
	}

	// Ensure that the necessary zpool is already in place, since building a zpool
	// may be too complicated or risky to do on program startup
	zpool, err := zfsutil.Zpool()
	if err != nil {
		// Check for permission denied
		if zfsutil.IsZFSPermissionDenied(err) {
			log.Fatalf("permission denied to ZFS virtual device, exiting")
		}

		// Check for zpool not exists
		if zfsutil.IsZpoolNotExists(err) {
			log.Fatalf("required zpool %q does not exist, exiting", zfsutil.ZpoolName)
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
		log.Fatalf("zpool %q unhealthy, status: %q; exiting", zpool.Name, zpool.Health)
	}

	// Receive errors from HTTP server
	httpErrC := make(chan error, 1)
	go func() {
		// Configure HTTP server
		httpServer := graceful.Server{
			Timeout: 10 * time.Second,
			Server: &http.Server{
				Addr:    host,
				Handler: zstoredhttp.NewServeMux(zpool),
			},
		}

		// Start listening on HTTP
		log.Println("HTTP listening:", httpServer.Server.Addr)
		httpErrC <- httpServer.ListenAndServe()
	}()

	// Check for HTTP server errors
	if err := <-httpErrC; err != nil {
		// Ignore error when shutting down
		if !strings.Contains(err.Error(), "use of closed network connection") {
			log.Fatalln("HTTP server error:", err)
		}
	}

	log.Println("graceful shutdown complete")
}
