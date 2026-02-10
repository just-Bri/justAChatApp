package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

type User struct {
	ID           int
	Username     string
	PasswordHash string
	Salt         string
}

// Session store (in-memory for this simple app, but could be DB)
var sessions = make(map[string]int) // cookie value -> user_id
var sessionMutex sync.Mutex

func hashPassword(password string, salt string) string {
	h := sha256.New()
	h.Write([]byte(password + salt))
	return hex.EncodeToString(h.Sum(nil))
}

func generateSalt() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})
}

func getAuthenticatedUserID(r *http.Request) (int, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return 0, false
	}

	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	userID, ok := sessions[cookie.Value]
	return userID, ok
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getAuthenticatedUserID(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		// In a real app, you might want to add userID to request context
		// But for this simple one, we'll just re-auth where needed or pass it
		_ = userID
		next(w, r)
	}
}
