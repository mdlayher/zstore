package zstoredhttp

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/mdlayher/zstore/storage"
	"github.com/mdlayher/zstore/storage/zfsutil"
)

var (
	// errInvalidSize is returned when an invalid size slug is selected
	// for volume creation or resizing.
	errInvalidSize = errors.New("invalid size slug")
)

// StorageRequest is a struct which represents a valid request to
// the storage API.
type StorageRequest struct {
	Size string `json:"size"`
}

// StorageResponse is a struct which represents a response from the
// storage API.
type StorageResponse struct {
	Volumes []*Volume `json:"volumes"`
}

// Volume is the JSON representation of a block storage volume.
type Volume struct {
	Name string `json:"name"`
	Size uint64 `json:"size"`
}

// StorageHandlerFunc is a function which accepts a volume name and HTTP
// request, and returns a HTTP status code, body, and server error.
type StorageHandlerFunc func(string, *http.Request) (int, []byte, error)

// StorageContext provides shared members required for zstored storage
// HTTP handlers.
type StorageContext struct {
	pool storage.Pool
}

// ServeHTTP delegates requests to the Context to the correct handlers.
func (c *StorageContext) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Generate volume name based upon information from input HTTP request
	name, err := c.volumeName(r)
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Map of HTTP methods to the appropriate StorageHandlerFunc
	methodFnMap := map[string]StorageHandlerFunc{
		"DELETE": c.destroyVolume,
		"GET":    c.getVolumeMetadata,
		"PUT":    c.createVolume,
	}

	// Check for a valid StorageHandlerFunc, 405 if none found
	fn, ok := methodFnMap[r.Method]
	if !ok {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve code, body, and server error from StorageHandlerFunc invocation
	code, body, err := fn(name, r)
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Return necessary code and body
	w.WriteHeader(code)
	w.Write(body)
}

// destroyVolume is a StorageHandlerFunc which destroys a volume via
// the HTTP server.
func (c *StorageContext) destroyVolume(name string, r *http.Request) (int, []byte, error) {
	// Check for a volume with the specified name
	volume, err := c.pool.Volume(name)
	if err != nil {
		// If volume does not exist, 404
		if err == storage.ErrVolumeNotExists {
			return http.StatusNotFound, nil, nil
		}

		// Any other errors
		return http.StatusInternalServerError, nil, err
	}

	// Destroy the volume, and all recursive volumes
	if err := volume.Destroy(); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// Return HTTP 204 on success
	return http.StatusNoContent, nil, nil
}

// getVolumeMetadata is a StorageHandlerFunc which returns metadata for a
// volume from the HTTP server.
func (c *StorageContext) getVolumeMetadata(name string, r *http.Request) (int, []byte, error) {
	// Check for a volume with the specified name
	volume, err := c.pool.Volume(name)
	if err != nil {
		// If volume does not exist, 404
		if err == storage.ErrVolumeNotExists {
			return http.StatusNotFound, nil, nil
		}

		// Any other errors
		return http.StatusInternalServerError, nil, err
	}

	// Return JSON representation of volume
	body, err := json.Marshal(&StorageResponse{
		Volumes: []*Volume{
			&Volume{
				Name: path.Base(volume.Name()),
				Size: volume.Size(),
			},
		},
	})
	return http.StatusOK, body, err
}

// createVolume is a StorageHandlerFunc which handles new volume creation
// for the HTTP server.
func (c *StorageContext) createVolume(name string, r *http.Request) (int, []byte, error) {
	// Ensure request name is bucketed to pool, unique hash, and volume name
	if len(strings.Split(name, "/")) != 3 {
		return http.StatusMethodNotAllowed, nil, nil
	}

	// Check for a volume with the specified name
	_, err := c.pool.Volume(name)
	if err == nil {
		// If no error, one already exists, so return 409
		return http.StatusConflict, nil, nil
	}

	// For any other errors, return server error
	if err != storage.ErrVolumeNotExists {
		return http.StatusInternalServerError, nil, err
	}

	// Parse volume size from HTTP request
	size, err := storageSize(r)
	if err != nil {
		// Check for invalid storage size slug
		if err == errInvalidSize {
			return http.StatusBadRequest, []byte(fmt.Sprintf("%s", zfsutil.Slugs())), nil
		}

		// Any other error
		return http.StatusInternalServerError, nil, err
	}

	// Generate a volume with the specified name and size
	volume, err := c.pool.CreateVolume(name, size)
	if err != nil {
		// Check for out of space error, return 503
		if err == storage.ErrPoolOutOfSpace {
			return http.StatusServiceUnavailable, nil, nil
		}

		return http.StatusInternalServerError, nil, err
	}

	// Return JSON representation of volume
	body, err := json.Marshal(&Volume{
		Name: path.Base(volume.Name()),
		Size: volume.Size(),
	})
	return http.StatusCreated, body, err
}

// volumeName uses HTTP server context and the current request to create a
// volume name specific to this client.
func (c *StorageContext) volumeName(r *http.Request) (string, error) {
	// Retrieve IP address from HTTP request
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}

	// Create a bucketed storage volume name which is limited to the
	// zstored pool, a MD5'd IP address, and the user-specified
	// volume name
	return filepath.Join(
		c.pool.Name(),
		fmt.Sprintf("%x", md5.Sum([]byte(host))),
		// Strip API path prefix
		path.Base(r.URL.Path[len(storageAPI):]),
	), nil
}

// storageSize returns a uint64 volume size after reading an input HTTP request
// and parsing a size slug from the request.
func storageSize(r *http.Request) (uint64, error) {
	// Decode HTTP request body into StorageRequest
	sr := new(StorageRequest)
	if err := json.NewDecoder(r.Body).Decode(sr); err != nil {
		// If no request body, return invalid size
		if err == io.EOF {
			return 0, errInvalidSize
		}

		return 0, err
	}

	// Check if slug is valid, return size
	size, ok := zfsutil.SlugSize(sr.Size)
	if !ok {
		return 0, errInvalidSize
	}

	return uint64(size), nil
}
