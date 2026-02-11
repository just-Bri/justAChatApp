package main

import (
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"
)

type Message struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	templates = template.Must(template.ParseGlob("templates/*.html"))
	clients   = make(map[chan Message]bool)
	clientsMu sync.Mutex
)

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templates.ExecuteTemplate(w, "register.html", nil)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	salt := generateSalt()
	hash := hashPassword(password, salt)

	_, err := db.Exec("INSERT INTO users (username, password_hash, salt) VALUES ($1, $2, $3)", username, hash, salt)
	if err != nil {
		if r.Header.Get("HX-Request") != "" {
			fmt.Fprintf(w, `<div class="alert error">[!] system_err: user_exists</div>`)
			return
		}
		http.Error(w, "Username already exists", http.StatusBadRequest)
		return
	}

	if r.Header.Get("HX-Request") != "" {
		fmt.Fprintf(w, `<div class="alert success">[+] system_msg: registration_complete. <a href="/login" style="color: inherit">/login</a></div>`)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templates.ExecuteTemplate(w, "login.html", nil)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var user User
	err := db.QueryRow("SELECT id, password_hash, salt FROM users WHERE username = $1", username).Scan(&user.ID, &user.PasswordHash, &user.Salt)
	if err != nil || hashPassword(password, user.Salt) != user.PasswordHash {
		if r.Header.Get("HX-Request") != "" {
			fmt.Fprintf(w, `<div class="alert error">[!] system_err: invalid_credentials</div>`)
			return
		}
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	sessionID := generateSessionID()
	sessionMutex.Lock()
	sessions[sessionID] = user.ID
	sessionMutex.Unlock()

	setSessionCookie(w, sessionID)

	if r.Header.Get("HX-Request") != "" {
		w.Header().Set("HX-Redirect", "/")
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	userID, _ := getAuthenticatedUserID(r)

	var username string
	db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&username)

	rows, _ := db.Query(`
		SELECT m.id, u.username, m.content, m.created_at 
		FROM messages m 
		JOIN users u ON m.user_id = u.id 
		ORDER BY m.created_at DESC LIMIT 50`)
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		rows.Scan(&m.ID, &m.Username, &m.Content, &m.CreatedAt)
		messages = append(messages, m)
	}

	data := struct {
		Username string
		Messages []Message
	}{
		Username: username,
		Messages: messages,
	}

	templates.ExecuteTemplate(w, "index.html", data)
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := getAuthenticatedUserID(r)
	content := r.FormValue("content")

	var msg Message
	err := db.QueryRow(`
		INSERT INTO messages (user_id, content) VALUES ($1, $2) 
		RETURNING id, created_at`, userID, content).Scan(&msg.ID, &msg.CreatedAt)
	if err != nil {
		http.Error(w, "Error saving message", http.StatusInternalServerError)
		return
	}

	db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&msg.Username)
	msg.Content = content

	// Broadcast to SSE clients
	clientsMu.Lock()
	for client := range clients {
		client <- msg
	}
	clientsMu.Unlock()

	// Return empty or new message piece for HTMX if needed (SSE will update the list)
	w.WriteHeader(http.StatusOK)
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	messageChan := make(chan Message)
	clientsMu.Lock()
	clients[messageChan] = true
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, messageChan)
		clientsMu.Unlock()
		close(messageChan)
	}()

	for {
		select {
		case msg := <-messageChan:
			// HTMX SSE format: data: <content>
			html := fmt.Sprintf(`<div class="message"><span class="time">[%s]</span> <span class="prompt">%s:$</span> %s</div>`,
				msg.CreatedAt.Format("2006-01-02T15:04:05"), msg.Username, msg.Content)
			fmt.Fprintf(w, "event: newMessage\ndata: %s\n\n", html)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
	}
}
