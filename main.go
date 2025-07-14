package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// Create directory for uploaded files
	if err := os.MkdirAll("uploads", 0755); err != nil {
		log.Fatal("Failed to create uploads directory:", err)
	}

	// Create static directory if it doesn't exist
	if err := os.MkdirAll("static", 0755); err != nil {
		log.Fatal("Failed to create static directory:", err)
	}

	// Setup routes
	http.HandleFunc("/", HomeHandler)
	http.HandleFunc("/upload", UploadHandler)

	// Serve static files (CSS, JS, images)
	http.Handle("/www/", http.StripPrefix("/www/", http.FileServer(http.Dir("www/"))))

	log.Println("Server started on port :8080")
	log.Println("Open http://localhost:8080 in your browser")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server startup error:", err)
	}
}
