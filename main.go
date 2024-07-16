package main

import (
	"imitation_project/internal/bot"
	"imitation_project/internal/config"
	"imitation_project/internal/database"
	"log"
)

func main() {
	config.LoadConfig()

	token := config.GetEnv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN must be set")
	}

	db, err := database.InitDB("properties.db")
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer db.Close()

	err = database.UpdateExistingDB("properties.db")
	if err != nil {
		log.Printf("Error updating existing database: %v", err)
	}

	b, err := bot.New(token, db)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	b.Start()
}
