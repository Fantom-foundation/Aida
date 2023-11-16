package utildb

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// mock server is tool to test update command

// main starts the mock server manually
func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: programname /path/to/aida-patches")
		return
	}

	// Get the base directory path from the command-line argument
	baseDir := os.Args[1]
	err := StartMockServer(baseDir)
	if err != nil {
		fmt.Printf("unable to start mock server; %v\n", err)
	}
	return
}

// StartMockServer starts the mock server
func StartMockServer(baseDir string) error {
	port := 8080

	// Create a custom handler to serve files based on the URL path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requestedPath := r.URL.Path
		filePath := filepath.Join(baseDir, requestedPath[1:]) // Remove the leading "/" in the URL path

		// Check if the file exists
		_, err := os.Stat(filePath)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		// Serve the requested file
		http.ServeFile(w, r, filePath)
	})

	// Start the HTTP server.
	fmt.Printf("Starting server on port %d...\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
