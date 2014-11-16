// Package zstoredhttp provides the HTTP server for zstored.
package zstoredhttp

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

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
	mux.Handle("/", c)

	return mux
}

// Context provides shared members required for zstored HTTP handlers.
type Context struct {
	zpool *zfs.Zpool

	zpath string
}

// ServeHTTP delegates requests to the Context to the correct handlers.
func (c *Context) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Generate file path using zpool name and URL path
	c.zpath = filepath.Join(string(os.PathSeparator), c.zpool.Name, r.URL.Path)

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

// GetHandler serves a file from the HTTP server.
func (c *Context) GetHandler(w http.ResponseWriter, r *http.Request) {
	// Attempt to open file from generated path
	f, err := os.Open(c.zpath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	// Stat file to determine its properties
	stat, err := f.Stat()
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Report any directories as 404
	if stat.IsDir() {
		http.NotFound(w, r)
		return
	}

	// Serve file stream
	http.ServeContent(w, r, c.zpath, stat.ModTime(), f)
}

// PutHandler handles file uploads to the HTTP server.
func (c *Context) PutHandler(w http.ResponseWriter, r *http.Request) {
	// Attempt to create necessary directories
	if err := os.MkdirAll(filepath.Dir(c.zpath), 0644); err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Attempt to create file using generated path
	f, err := os.Create(c.zpath)
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Read multipart file by its key
	upload, _, err := r.FormFile("file")
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Copy file from upload stream
	if _, err := io.Copy(f, upload); err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Reply no content
	w.WriteHeader(http.StatusNoContent)
}
