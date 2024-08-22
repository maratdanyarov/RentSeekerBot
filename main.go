package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("Failed to create bot API: %v", err)
	}

	b := bot.New(api, db, api.Self.UserName)
	b.Start()
}
