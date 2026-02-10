package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() {
	// Simple connection string - in a real app, use env vars
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		fmt.Println("Warning: DATABASE_URL is empty, falling back to localhost")
		connStr = "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable"
	} else {
		fmt.Println("Attempting to connect to database using DATABASE_URL environment variable")
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Cannot connect to database:", err)
	}

	createTables()
}

func createTables() {
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		salt TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	messageTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id SERIAL PRIMARY KEY,
		user_id INTEGER REFERENCES users(id),
		content TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(userTable)
	if err != nil {
		log.Fatal("Error creating users table:", err)
	}

	_, err = db.Exec(messageTable)
	if err != nil {
		log.Fatal("Error creating messages table:", err)
	}

	fmt.Println("Database tables initialized")
}
