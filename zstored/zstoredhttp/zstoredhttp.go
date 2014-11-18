// Package zstoredhttp provides the HTTP server for zstored.
package zstoredhttp

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"

	"github.com/mdlayher/zstore/zfsutil"

	"gopkg.in/mistifyio/go-zfs.v1"
)

// NewServeMux returns a http.Handler for the zstored HTTP server.
func NewServeMux(zpool *zfs.Zpool) http.Handler {
	// Build context for HTTP handlers
	c := &Context{
		zpool: zpool,
	}

	// Set up HTTP handlers
	mux := http.NewServeMux()
	mux.Handle("/block", c)

	return mux
}

// Volume is the JSON representation of a block storage volume.
type Volume struct {
	Name string `json:"name"`
	Size uint64 `json:"size"`
}

// Context provides shared members required for zstored HTTP handlers.
type Context struct {
	zpool *zfs.Zpool
}

// VolumeName uses HTTP server context and the current request to create a
// volume name specific to this client.
func (c *Context) VolumeName(r *http.Request) (string, error) {
	// Capture just IP address
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}

	// Compose zpool name and IP address in md5'd hex
	return filepath.Join(c.zpool.Name, fmt.Sprintf("%x", md5.Sum([]byte(host)))), nil
}

// ServeHTTP delegates requests to the Context to the correct handlers.
func (c *Context) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	// Serve file
	case "GET":
		c.GetHandler(w, r)
	// Upload file
	case "PUT":
		c.PutHandler(w, r)
	// Invalid HTTP method
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// GetHandler returns information about a volume from the HTTP server.
func (c *Context) GetHandler(w http.ResponseWriter, r *http.Request) {
	// Generate volume name from request and context
	name, err := c.VolumeName(r)
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Attempt to fetch ZFS volume dataset
	zvol, err := zfs.GetDataset(name)
	if err != nil {
		// Check if dataset does not exist, return 404
		if zfsutil.IsDatasetNotExists(err) {
			http.NotFound(w, r)
			return
		}

		// All other errors
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Ensure proper dataset type
	if zvol.Type != zfsutil.DatasetVolume {
		http.NotFound(w, r)
		return
	}

	// Wrap volume, return JSON
	volume := &Volume{
		Name: name,
		Size: zvol.Avail,
	}
	if err := json.NewEncoder(w).Encode(volume); err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

// PutHandler handles new volume creation for the HTTP server.
func (c *Context) PutHandler(w http.ResponseWriter, r *http.Request) {
	// Generate volume name from request and context
	name, err := c.VolumeName(r)
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Create new volume for this user
	// TODO(mdlayher): make this parameter tweakable via JSON body or HTTP header
	zvol, err := zfs.CreateVolume(name, 1024*1024*512, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Wrap volume, return JSON
	volume := &Volume{
		Name: name,
		Size: zvol.Avail,
	}
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(volume); err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
