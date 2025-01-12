package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "mysecretpassword"
	dbname   = "postgres"
)

func main() {
	// Connection string
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Open connection to database
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging the database: ", err)
	}

	// Create tables
	_, err = db.Exec(`
		DROP TABLE IF EXISTS messages;
		DROP TABLE IF EXISTS chats;

		CREATE TABLE chats (
			chat_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			PRIMARY KEY (chat_id, user_id)
		);

		CREATE TABLE messages (
			msg_id TEXT PRIMARY KEY NOT NULL,
			chat_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			created_at BIGINT,
			text TEXT NOT NULL
		);

		CREATE INDEX idx_messages_chat_id ON messages(chat_id);
		CREATE INDEX idx_messages_created_at ON messages(created_at);
		CREATE INDEX idx_chats_user_id ON chats(user_id);
	`)

	if err != nil {
		log.Fatal("Error creating tables: ", err)
	}

	fmt.Println("Database initialized successfully!")
}
