package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if _, err := os.Stat("../db-data/sqlite-database.db"); err == nil {
		panic(fmt.Errorf("DB file already exists, remove it first before initializing the db"))
	}

	_, err := os.Create("../db-data/sqlite-database.db")
	if err != nil {
		panic(fmt.Errorf("failed to create db file: %w", err))
	}

	db, err := sql.Open("sqlite3", "../db-data/sqlite-database.db")
	if err != nil {
		panic(fmt.Errorf("failed to open sqlite db file: %w", err))
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE chats(
		chat_id 				TEXT NOT NULL,
		user_id					TEXT NOT NULL,
		PRIMARY KEY (chat_id, user_id)
	);`)
	if err != nil {
		panic(fmt.Errorf("failed to create table in db: %w", err))
	}

	_, err = db.Exec(`CREATE TABLE messages(
		msg_id 				TEXT PRIMARY KEY NOT NULL,
		chat_id				TEXT NOT NULL,
		user_id				TEXT NOT NULL,
		created_at			INTEGER,
		text				TEXT 	NOT NULL
	);`)
	if err != nil {
		panic(fmt.Errorf("failed to create table in db: %w", err))
	}

	fmt.Println("DB initialized!")
}
