// Package bot provides the core functionality for the Telegram bot.
package bot

import (
	"database/sql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"sync"
)

// BotAPI is an interface that wraps the methods we use from tgbotapi.BotAPI
type BotAPI interface {
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

// Bot represents the Telegram bot instance.
// It handles user interactions, maintains user states,
// and interfaces with the database for property searches.
type Bot struct {
	api         BotAPI
	db          *sql.DB
	state       map[int64]*UserState
	mu          sync.Mutex
	botUserName string
}

// SearchPreferences represents the user's search criteria for properties.
// It encapsulates various options that a user can select to filter their property search.
type SearchPreferences struct {
	PropertyTypes    map[string]bool
	BedroomOptions   map[string]bool
	FurnishedOptions map[string]bool
	PriceRange       string
	Location         string
}

// UserState represents the current state of a user's interaction with the bot.
type UserState struct {
	Stage       string
	Preferences *SearchPreferences
}

// New creates a new instance of the Bot.
// It takes a Telegram bot token as input and returns a new Bot instance and any error encountered.
func New(api BotAPI, db *sql.DB, botUserName string) *Bot {
	return &Bot{
		api:         api,
		db:          db,
		state:       make(map[int64]*UserState),
		botUserName: botUserName,
	}
}

// Start begins the bot's operation.
// It sets up the update channel and enters the main event loop to process updates.
func (b *Bot) Start() {
	log.Printf("Authorised an account %s", b.botUserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	// Main event loop
	for update := range updates {
		if update.Message != nil {
			go b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			go b.handleCallbackQuery(update.CallbackQuery)
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
		Preferences: NewFlexibleSearchPreferences(),
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
	case "save_preferences":
		b.handleSavePreferences(message)
	case "view_preferences":
		b.handleViewPreferences(message)
	case "clear_preferences":
		b.handleClearPreferences(message)
	case "saved":
		b.handleViewSavedListings(message)
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

// NewFlexibleSearchPreferences creates and returns a new SearchPreferences struct.
// It initializes the PropertyTypes, BedroomOptions, and FurnishedOptions maps.
func NewFlexibleSearchPreferences() *SearchPreferences {
	return &SearchPreferences{
		PropertyTypes:    make(map[string]bool),
		BedroomOptions:   make(map[string]bool),
		FurnishedOptions: make(map[string]bool),
	}
}

// createMultiSelectKeyboard generates an inline keyboard markup for Telegram bot.
// It creates a keyboard with multiple selectable options and a "Done" button.
func createMultiSelectKeyboard(options map[string]bool, prefix string) tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for option, selected := range options {
		var text string
		if selected {
			text = "✅ " + option
		} else {
			text = "☐ " + option
		}
		button := tgbotapi.NewInlineKeyboardButtonData(text, prefix+":"+option)
		keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(button))
	}

	doneButton := tgbotapi.NewInlineKeyboardButtonData("Done", prefix+":done")
	keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(doneButton))

	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

// updateMultiSelectOption toggles the selected state of an option in a multi-select map.
// If the option was previously selected, it becomes unselected, and vice versa.
func updateMultiSelectOption(options map[string]bool, option string) {
	if options == nil {
		options = make(map[string]bool)
	}
	options[option] = !options[option]
}
