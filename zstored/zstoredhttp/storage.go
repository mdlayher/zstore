package zstoredhttp

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"path"
	"path/filepath"

	"github.com/mdlayher/zstore/zfsutil"

	"gopkg.in/mistifyio/go-zfs.v1"
)

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

	name, err := c.VolumeName(r)
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	methodFnMap := map[string]StorageHandlerFunc{
		"GET": c.GetVolumeMetadata,
		"PUT": c.CreateVolume,
	}

	fn, ok := methodFnMap[r.Method]
	if !ok {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code, body, err := fn(name, r)
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(code)
	w.Write(body)
}

// VolumeName uses HTTP server context and the current request to create a
// volume name specific to this client.
func (c *StorageContext) VolumeName(r *http.Request) (string, error) {

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}

	return filepath.Join(
		c.zpool.Name,
		fmt.Sprintf("%x", md5.Sum([]byte(host))),
		path.Base(r.URL.Path),
	), nil
}

// GetVolumeMetadata is a StorageHandlerFunc which returns metadata for a
// volume from the HTTP server.
func (c *StorageContext) GetVolumeMetadata(name string, r *http.Request) (int, []byte, error) {

	zvol, err := zfs.GetDataset(name)
	if err != nil {

		if zfsutil.IsDatasetNotExists(err) {
			return http.StatusNotFound, nil, nil
		}

		return http.StatusInternalServerError, nil, err
	}

	if zvol.Type != zfsutil.DatasetVolume {
		return http.StatusNotFound, nil, nil
	}

	body, err := json.Marshal(&Volume{
		Name: name,
		Size: zvol.Avail,
	})
	return http.StatusOK, body, err
}

// CreateVolume is a StorageHandlerFunc which handles new volume creation
// for the HTTP server.
func (c *StorageContext) CreateVolume(name string, r *http.Request) (int, []byte, error) {

	_, err := zfs.GetDataset(name)
	if err == nil {

		return http.StatusConflict, nil, nil
	}
	if !zfsutil.IsDatasetNotExists(err) {
		return http.StatusInternalServerError, nil, err
	}

	zvol, err := zfs.CreateVolume(name, 1024*1024*32, nil)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	body, err := json.Marshal(&Volume{
		Name: name,
		Size: zvol.Avail,
	})
	return http.StatusCreated, body, err
}
