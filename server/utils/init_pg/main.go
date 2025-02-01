package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go/aws"
	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "mysecretpassword"
	dbname   = "postgres"
)

func getAwsSecret() string {
	secretName := "rds!cluster-1302f671-a70b-4c48-812e-f29865805fc1"
	region := "us-east-2"

	config, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Fatal(err)
	}

	// Create Secrets Manager client
	svc := secretsmanager.NewFromConfig(config)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	result, err := svc.GetSecretValue(context.TODO(), input)
	if err != nil {
		// For a list of exceptions thrown, see
		// https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html
		log.Fatal(err.Error())
	}

	// Decrypts secret using the associated KMS key.
	var secretString string = *result.SecretString
	return secretString
}

func main() {
	// Connection string
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	// // Open connection to database
	db, err := sql.Open("postgres", psqlInfo)
	// pswd := getAwsSecret()
	// db, err := sqlx.Open("postgres", "host=database-1.cluster-cd0ck2iwiyfj.us-east-2.rds.amazonaws.com port=5432 user=postgres password="+pswd+" dbname=postgres sslmode=disable")
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
		CREATE INDEX idx_messages_chat_id_created_at ON messages (chat_id, created_at DESC NULLS LAST);
		CREATE INDEX idx_chats_user_id ON chats(user_id);
	`)

	if err != nil {
		log.Fatal("Error creating tables: ", err)
	}

	fmt.Println("Database initialized successfully!")
}
