package main

import (
	"database/sql"
	"fmt"
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
	err := database.InitDB("properties.db")
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		return
	}

	err = database.UpdateExistingDB("properties.db")
	if err != nil {
		fmt.Printf("Error updating existing database: %v\n", err)
		return
	}

	db, err := sql.Open("sqlite3", "properties.db")
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	// Create a new bot instance
	b, err := bot.New(token)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Start the bot
	b.Start()
}
