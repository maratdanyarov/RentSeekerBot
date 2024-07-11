// Package bot provides the core functionality for the Telegram bot.
package bot

import (
	"database/sql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"imitation_project/internal/database"
	"log"
	"strconv"
	"strings"
	"sync"
)

// Bot represents the Telegram bot instance.
type Bot struct {
	api   *tgbotapi.BotAPI
	state map[int64]*UserState
	mu    sync.Mutex
}

// UserState represents the current state of a user's interaction with the bot.
type UserState struct {
	Stage       string
	Preferences map[string]string
}

// New creates a new instance of the Bot.
// It takes a Telegram bot token as input and returns a new Bot instance and any error encountered.
func New(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		api:   api,
		state: make(map[int64]*UserState),
	}, nil
}

// Start begins the bot's operation.
// It sets up the update channel and enters the main event loop to process updates.
func (b *Bot) Start() {
	log.Printf("Authorised an account %s", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	// Main event loop
	for update := range updates {
		if update.Message != nil {
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			b.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

// getUserState retrieves the current state for a user.
func (b *Bot) getUserState(userID int64) *UserState {
	b.mu.Lock()
	defer b.mu.Unlock()

	if state, exists := b.state[userID]; exists {
		return state
	}
	state := &UserState{
		Stage:       "initial",
		Preferences: make(map[string]string),
	}
	b.state[userID] = state
	return state
}

// updateUserState updates the state for a user.
func (b *Bot) updateUserState(userID int64, state *UserState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state[userID] = state
}

// handleMessage processes incoming messages.
// It distinguishes between commands and regular messages and routes them accordingly.
func (b *Bot) handleMessage(message *tgbotapi.Message) {
	if message.IsCommand() {
		b.handleCommand(message)
	} else {
		b.handleRegularMessage(message)
	}
}

// handleCommand processes bot commands.
// It routes different commands to their respective handler functions.
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		b.handleStartCommand(message)
	case "help":
		b.handleHelpCommand(message)
	case "search":
		b.handleSearchCommand(message)
	default:
		b.sendMessage(message.Chat.ID, "Unknown command. Type /help for available commands.", nil)
	}
}

// sendMessage sends a text message to the specified chat.
// It takes the chat ID and the message text as input.
func (b *Bot) sendMessage(chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = markup
	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func (b *Bot) searchProperties(preferences map[string]string) ([]database.Property, error) {
	filters := make(map[string]interface{})

	if v, ok := preferences["property_type"]; ok && v != "" {
		filters["type"] = v
	}
	if v, ok := preferences["bedrooms"]; ok && v != "" {
		if v == "studio" {
			filters["bedrooms"] = 0
		} else {
			bedrooms, err := strconv.Atoi(v)
			if err == nil {
				filters["bedrooms"] = bedrooms
			}
		}
	}
	if v, ok := preferences["price_range"]; ok && v != "" {
		minPrice, maxPrice := parsePriceRange(v)
		filters["min_price"] = minPrice
		filters["max_price"] = maxPrice
	}
	if v, ok := preferences["location"]; ok && v != "" {
		filters["location"] = v
	}

	if v, ok := preferences["furnished"]; ok {
		furnished := strings.ToLower(v) == "furnished"
		filters["furnished"] = furnished
	}

	db, err := sql.Open("sqlite3", "properties.db")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	return database.GetProperties(db, filters)
}

func parsePriceRange(priceRange string) (int, int) {
	parts := strings.Split(priceRange, "-")
	if len(parts) != 2 {
		return 0, 1000000
	}

	// Trim spaces and parse to integers
	min, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		min = 0
	}
	max, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		max = 1000000
	}

	if min > max {
		min, max = max, min
	}

	return min, max
}
