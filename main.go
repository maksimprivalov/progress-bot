package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"progress-bot/internal/bot"
	"progress-bot/internal/storage"
)

func main() {
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment varijabla nije postavljena")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "progress.db"
	}

	store, err := storage.Open(dbPath)
	if err != nil {
		log.Fatalf("otvaranje baze: %v", err)
	}

	// arise in the end of the program, after the bot is done running and the main function is about to exit. This ensures that the database connection is properly closed and any resources are released.
	defer store.Close()

	b, err := bot.New(token, store)
	if err != nil {
		log.Fatalf("kreiranje bota: %v", err)
	}

	b.Run()
}
