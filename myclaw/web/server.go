package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

// Start starts the HTTP server on the given port and blocks until it fails.
// It is intended to be run in its own goroutine.
func Start(port string) error {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("preparing static files: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(sub)))

	addr := ":" + port
	fmt.Printf("Web UI: http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}
