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

	// Matrix Client Delegation
	http.HandleFunc("/.well-known/matrix/client", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprint(w, `{"m.homeserver":{"base_url":"https://matrix.agaymergirl.com"}}`)
	})

	// Matrix Server Delegation (Federation)
	http.HandleFunc("/.well-known/matrix/server", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"m.server":"matrix.agaymergirl.com:443"}`)
	})

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
