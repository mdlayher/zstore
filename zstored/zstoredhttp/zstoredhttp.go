// Package zstoredhttp provides the HTTP server for zstored.
package zstoredhttp

import (
	"net/http"

	"github.com/mdlayher/zstore/storage"
)

const (
	// storageAPI is the path prefix for the storage provisioning API
	storageAPI = "/v1/storage/"
)

// NewServeMux returns a http.Handler for the zstored HTTP server.
func NewServeMux(pool storage.Pool) http.Handler {
	// Set up HTTP handlers
	mux := http.NewServeMux()
	//   - Storage provisioning API
	mux.Handle(storageAPI, &StorageContext{
		pool: pool,
	})

	return mux
}
