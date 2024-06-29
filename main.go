package main

import (
	"imitation_project/internal/bot"
	"imitation_project/internal/config"
	"log"
)

func main() {
	config.LoadConfig()

	token := config.GetEnv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN must be set")
	}

	// Create a new bot instance
	b, err := bot.New(token)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Start the bot
	b.Start()
}
