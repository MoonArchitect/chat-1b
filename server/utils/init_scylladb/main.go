package main

import (
	"log"

	"github.com/gocql/gocql"
	"github.com/scylladb/gocqlx"
)

func main() {
	config := gocql.NewCluster("localhost")
	sess, err := gocqlx.WrapSession(config.CreateSession())
	if err != nil {
		log.Fatalf("error creating sess: %v", err)
	}

	create_keyspace := `
CREATE KEYSPACE main_db
WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};
	`

	create_users_table := `
CREATE TABLE main_db.users (
	user_id UUID, 
	chats MAP<UUID, BIGINT>, 
	PRIMARY KEY (user_id)
);
`
	create_chats_table := `
CREATE TABLE main_db.chats (
	chat_id UUID, 
	users SET<UUID>, 
	PRIMARY KEY (chat_id)
);
`
	// test difference between using user_id vs chat_id first in primary key

	create_messages_table := `
CREATE TABLE main_db.messages (
	msg_id UUID,
	chat_id UUID,
	user_id UUID,
	created_at_micro BIGINT,
	text TEXT,
	PRIMARY KEY (chat_id, created_at_micro)
) WITH CLUSTERING ORDER BY (created_at_micro ASC);
`

	err = sess.ExecStmt(create_keyspace)
	if err != nil {
		log.Fatalf("error create_keyspace: %v", err)
	}
	err = sess.ExecStmt(create_chats_table)
	if err != nil {
		log.Fatalf("error create_chat_table: %v", err)
	}
	err = sess.ExecStmt(create_users_table)
	if err != nil {
		log.Fatalf("error create_users_table: %v", err)
	}
	err = sess.ExecStmt(create_messages_table)
	if err != nil {
		log.Fatalf("error create_messages_table: %v", err)
	}
}
