// Package zstoredhttp provides the HTTP server for zstored.
package zstoredhttp

import (
	"net/http"

	"gopkg.in/mistifyio/go-zfs.v1"
)

const (
	// storageAPI is the path prefix for the storage provisioning API
	storageAPI = "/v1/storage/"
)

// NewServeMux returns a http.Handler for the zstored HTTP server.
func NewServeMux(zpool *zfs.Zpool) http.Handler {
	// Set up HTTP handlers
	mux := http.NewServeMux()
	//   - Storage provisioning API
	mux.Handle(storageAPI, &StorageContext{
		zpool: zpool,
	})

	return mux
}
