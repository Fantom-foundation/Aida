// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

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
