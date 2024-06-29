package bot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"strings"
)

// handleStartCommand processes the /start command.
// It sends a welcome message to the user with an overview of the bot's functionality.
func (b *Bot) handleStartCommand(message *tgbotapi.Message) {
	welcomeText := fmt.Sprintf(
		`🌟 Hi %s! I'm here to assist you in finding your perfect home in Bath.
				I'll start by asking a few questions to tailor your search preferences.`,
		message.From.FirstName,
	)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Okay, let's go!", "start_preferences"),
		),
	)
	b.sendMessage(message.Chat.ID, welcomeText, keyboard)
}

// handleCallbackQuery processes callback queries from inline keyboards.
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	state := b.getUserState(int64(query.From.ID))

	switch query.Data {
	case "start_preferences":
		b.askPropertyType(query.Message.Chat.ID)
	case "flat", "house":
		state.Preferences["propery_type"] = query.Data
		b.askPriceRange(query.Message.Chat.ID)
	case "studio", "1", "2", "3", "4", "5":
		state.Preferences["bedrooms"] = query.Data
		b.askFurnished(query.Message.Chat.ID)
	case "furnished", "unfurnished":
		state.Preferences["furnished"] = query.Data
		b.askLocation(query.Message.Chat.ID)
	}

	b.updateUserState(int64(query.From.ID), state)
	b.api.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, ""))
}

func (b *Bot) handleHelpCommand(message *tgbotapi.Message) {
	helpText := `Here are the available commands:
				/start - Start the bot and see the welcome message
				/search - Begin a property search
				/help - Show this help message
				`

	b.sendMessage(message.Chat.ID, helpText, nil)
}

// handleSearchCommand processes the /search command.
func (b *Bot) handleSearchCommand(message *tgbotapi.Message) {
	searchText := `
					Let's start your property search in Bath.
					Please provide your preferences in the following format:
					Price range: [min]-[max]
					Number of rooms: [number]
					For example: "Price range: 500-1000, Number of rooms: 2"
					Or you can simply type "any" if you don't have specific preferences.
					`
	b.sendMessage(message.Chat.ID, searchText, nil)
}

// handleRegularMessage processes non-command messages.
func (b *Bot) handleRegularMessage(message *tgbotapi.Message) {
	state := b.getUserState(int64(message.From.ID))

	switch state.Stage {
	case "awaiting_price_range":
		if b.validatePriceRange(message.Text) {
			state.Preferences["price_range"] = message.Text
			b.askBedrooms(message.Chat.ID)
		} else {
			b.sendMessage(message.Chat.ID, "Invalid price range format. Please use the format: min - max (e.g., 1200 - 1800)", nil)
			return
		}
	case "awaiting_location":
		state.Preferences["location"] = message.Text
		b.showSummary(message.Chat.ID, state.Preferences)
	default:
		b.sendMessage(message.Chat.ID, "I'm sorry, I didn't understand that. Please use the provided buttons or follow the instructions.", nil)
		return
	}

	b.updateUserState(int64(message.From.ID), state)
}

// askPropertyType asks the user to select a property type.
func (b *Bot) askPropertyType(chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Flat", "flat"),
			tgbotapi.NewInlineKeyboardButtonData("House", "house"),
		),
	)
	b.sendMessage(chatID, "🏠 Select the property type:", keyboard)
}

// askPriceRange asks the user to input a price range.
func (b *Bot) askPriceRange(chatID int64) {
	state := b.getUserState(chatID)
	state.Stage = "awaiting_price_range"
	b.updateUserState(chatID, state)
	b.sendMessage(chatID, `💰 Let me know the price range for the monthly rent in GBP. 
								Format: min - max (e.g., 1200 - 1800)`, nil)
}

// validatePriceRange checks if the price range is in the correct format.
func (b *Bot) validatePriceRange(input string) bool {
	parts := strings.Split(input, "-")
	if len(parts) != 2 {
		return false
	}
	// TODO: add more validation here (e.g., checking if the values are numbers)
	return true
}

// askBedrooms asks the user to select the number of bedrooms.
func (b *Bot) askBedrooms(chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Studio", "studio"),
			tgbotapi.NewInlineKeyboardButtonData("1", "1"),
			tgbotapi.NewInlineKeyboardButtonData("2", "2"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("3", "3"),
			tgbotapi.NewInlineKeyboardButtonData("4", "4"),
			tgbotapi.NewInlineKeyboardButtonData("5+", "5"),
		),
	)
	b.sendMessage(chatID, "🛏 Select the number of bedrooms:", keyboard)
}

// askFurnished asks if the user wants furnished or unfurnished accommodation.
func (b *Bot) askFurnished(chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Furnished", "furnished"),
			tgbotapi.NewInlineKeyboardButtonData("Unfurnished", "unfurnished"),
		),
	)
	b.sendMessage(chatID, "🪑 Do you want to search for furnished or unfurnished accommodation?", keyboard)
}

// askLocation asks the user to input their preferred location.
func (b *Bot) askLocation(chatID int64) {
	state := b.getUserState(chatID)
	state.Stage = "awaiting_location"
	b.updateUserState(chatID, state)

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Bath"),
		),
	)

	keyboard.OneTimeKeyboard = true
	b.sendMessage(chatID, "📍Sorry, the bot is in testing mode so the search area is restricted to Bath", keyboard)
}

// showSummary displays a summary of the user's preferences.
func (b *Bot) showSummary(chatID int64, preferences map[string]string) {
	summary := fmt.Sprintf("Great! Here's a summary of your preferences:\n\n"+
		"🏠 Property Type: %s\n"+
		"💰 Price Range: %s\n"+
		"🛏 Bedrooms: %s\n"+
		"🪑 Furnished: %s\n"+
		"📍 Location: %s\n\n"+
		"I'll now search for properties matching these criteria. Please wait a moment.",
		preferences["property_type"],
		preferences["price_range"],
		preferences["bedrooms"],
		preferences["furnished"],
		preferences["location"])

	b.sendMessage(chatID, summary, nil)
	// TODO: add a function to perform the actual search
}
