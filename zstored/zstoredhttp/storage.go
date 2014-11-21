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

	"github.com/mdlayher/zstore/zfsutil"

	"gopkg.in/mistifyio/go-zfs.v1"
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

// StorageHandlerFunc is a function which accepts a volume name and HTTP
// request, and returns a HTTP status code, body, and server error.
type StorageHandlerFunc func(string, *http.Request) (int, []byte, error)

// Volume is the JSON representation of a block storage volume.
type Volume struct {
	Name string `json:"name"`
	Size uint64 `json:"size"`
}

// StorageContext provides shared members required for zstored storage
// HTTP handlers.
type StorageContext struct {
	zpool *zfs.Zpool
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
		"DELETE": c.DestroyVolume,
		"GET":    c.GetVolumeMetadata,
		"PUT":    c.CreateVolume,
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

// DestroyVolume is a StorageHandlerFunc which destroys a volume via
// the HTTP server.
func (c *StorageContext) DestroyVolume(name string, r *http.Request) (int, []byte, error) {
	// Check for a dataset which contains the specified name
	zvol, err := zfs.GetDataset(name)
	if err != nil {
		// If dataset does not exist, 404
		if zfsutil.IsDatasetNotExists(err) {
			return http.StatusNotFound, nil, nil
		}

		// Any other errors
		return http.StatusInternalServerError, nil, err
	}

	// Ensure that returned dataset is a volume
	if zvol.Type != zfsutil.DatasetVolume {
		return http.StatusNotFound, nil, nil
	}

	// Destroy the volume, and all recursive volumes
	if err := zvol.Destroy(true); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// Return HTTP 204 on success
	return http.StatusNoContent, nil, nil
}

// GetVolumeMetadata is a StorageHandlerFunc which returns metadata for a
// volume from the HTTP server.
func (c *StorageContext) GetVolumeMetadata(name string, r *http.Request) (int, []byte, error) {
	// Check for a dataset which contains the specified name
	zvol, err := zfs.GetDataset(name)
	if err != nil {
		// If dataset does not exist, 404
		if zfsutil.IsDatasetNotExists(err) {
			return http.StatusNotFound, nil, nil
		}

		// Any other errors
		return http.StatusInternalServerError, nil, err
	}

	// Ensure that returned dataset is a volume
	if zvol.Type != zfsutil.DatasetVolume {
		return http.StatusNotFound, nil, nil
	}

	// Return JSON representation of volume
	body, err := json.Marshal(&Volume{
		Name: name,
		Size: zvol.Avail,
	})
	return http.StatusOK, body, err
}

// CreateVolume is a StorageHandlerFunc which handles new volume creation
// for the HTTP server.
func (c *StorageContext) CreateVolume(name string, r *http.Request) (int, []byte, error) {
	// Check for a dataset which contains the specified name
	_, err := zfs.GetDataset(name)
	if err == nil {
		// If no error, one already exists, so return 409
		return http.StatusConflict, nil, nil
	}

	// For any other errors, return server error
	if !zfsutil.IsDatasetNotExists(err) {
		return http.StatusInternalServerError, nil, err
	}

	// Parse volume size from HTTP request
	size, err := storageSize(r)
	if err != nil {
		// Check for invalid storage size slug
		if err == errInvalidSize {
			return http.StatusBadRequest, nil, nil
		}

		// Any other error
		return http.StatusInternalServerError, nil, err
	}

	// Generate a volume with the specified name and size
	zvol, err := zfs.CreateVolume(name, size, nil)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// Return JSON representation of volume
	body, err := json.Marshal(&Volume{
		Name: name,
		Size: zvol.Avail,
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
	// zstored zpool, a MD5'd IP address, and the user-specified
	// volume name
	return filepath.Join(
		c.zpool.Name,
		fmt.Sprintf("%x", md5.Sum([]byte(host))),
		path.Base(r.URL.Path),
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

	// Common size constants for volume creation and resizing.
	const (
		MB = 1 * 1024 * 1024
		GB = 1024 * MB
	)

	// Map of available slugs and uint64 sizes
	storageSizeMap := map[string]uint64{
		"256M": 256 * MB,
		"512M": 512 * MB,
		"1G":   1 * GB,
		"2G":   2 * GB,
		"4G":   4 * GB,
		"8G":   8 * GB,
	}

	// Look up to check if size slug is valid
	size, ok := storageSizeMap[sr.Size]
	if !ok {
		return 0, errInvalidSize
	}

	// Return size
	return size, nil
}
