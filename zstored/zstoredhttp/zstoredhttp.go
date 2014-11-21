// Package zstoredhttp provides the HTTP server for zstored.
package zstoredhttp

import (
	"net/http"

	"gopkg.in/mistifyio/go-zfs.v1"
)

// NewServeMux returns a http.Handler for the zstored HTTP server.
func NewServeMux(zpool *zfs.Zpool) http.Handler {
	// Set up HTTP handlers
	mux := http.NewServeMux()
	//   - Storage provisioning API
	mux.Handle("/v1/storage/", &StorageContext{
		zpool: zpool,
	})

	return mux
}
