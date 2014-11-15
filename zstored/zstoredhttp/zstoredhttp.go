// Package zstoredhttp provides the HTTP server for zstored.
package zstoredhttp

import (
	"net/http"
)

// NewServeMux returns a http.Handler for the zstored HTTP server.
func NewServeMux() http.Handler {
	return http.NewServeMux()
}
