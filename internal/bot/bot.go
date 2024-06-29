// Package bot provides the core functionality for the Telegram bot.
package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
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

	updates, err := b.api.GetUpdatesChan(u)
	if err != nil {
		log.Fatalf("Failed to get updates channel: %v", err)
	}

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
	msg.ReplyMarkup = markup
	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}
