package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	initDB()
	defer db.Close()

	// Static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Routes
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/", authMiddleware(handleIndex))
	http.HandleFunc("/send", authMiddleware(handleSendMessage))
	http.HandleFunc("/events", authMiddleware(handleEvents))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
