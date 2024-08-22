// Package bot provides the core functionality for the Telegram bot.
package bot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"imitation_project/internal/database"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	stageAwaitingPriceRange   = "awaiting_price_range"
	stageAwaitingLocation     = "awaiting_location"
	stageAwaitingPropertyType = "awaiting_property_type"
	stageAwaitingBedrooms     = "awaiting_bedrooms"
	stageAwaitingFurnished    = "awaiting_furnished"
)

// handleStartCommand processes the /start command.
// It sends a welcome message to the user with an overview of the bot's functionality.
func (b *Bot) handleStartCommand(message *tgbotapi.Message) {
	welcomeText := fmt.Sprintf(
		`🌟 Welcome, %s!

I’m delighted to assist you in finding your perfect home. 
Please note that currently, the bot is in testing mode, and the property search is restricted to the Bath area. 
To provide you with the best possible service, I will begin by asking a few questions to understand your preferences and tailor the property search to meet your specific needs within Bath.

Let’s get started on finding your ideal home in Bath!`,
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
// It handles various user interactions based on the callback data received.
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	// Retrieve the current state for the user
	state := b.getUserState(int64(query.From.ID))
	// Split the callback data into parts
	data := strings.Split(query.Data, ":")

	// Handle different callback actions based on the first part of the data
	switch data[0] {
	case "use_saved_prefs":
		// Use saved preferences to perform a search
		prefs, err := database.GetUserPreferences(b.db, query.From.ID)
		if err != nil {
			b.sendMessage(query.Message.Chat.ID, "Error retrieving saved preferences. Starting new search.", nil)
			b.startNewSearch(query.Message.Chat.ID)
		} else {
			// Directly perform the search without sending a message
			b.performSearch(query.Message.Chat.ID, prefs)
		}
	case "start_new_search":
		b.startNewSearch(query.Message.Chat.ID)
	case "start_preferences":
		b.askPropertyType(query.Message.Chat.ID)
	case "property_type":
		if data[1] == "done" {
			b.askBedrooms(query.Message.Chat.ID)
		} else {
			updateMultiSelectOption(state.Preferences.PropertyTypes, data[1])
			keyboard := createMultiSelectKeyboard(state.Preferences.PropertyTypes, "property_type")
			b.editMessageReplyMarkup(query.Message.Chat.ID, query.Message.MessageID, keyboard)
		}
	case "bedrooms":
		if data[1] == "done" {
			b.askPriceRange(query.Message.Chat.ID)
		} else {
			// Toggle the selected state
			state.Preferences.BedroomOptions[data[1]] = !state.Preferences.BedroomOptions[data[1]]

			// Update the keyboard
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(getButtonText("Studio", state.Preferences.BedroomOptions["Studio"]), "bedrooms:Studio"),
					tgbotapi.NewInlineKeyboardButtonData(getButtonText("1", state.Preferences.BedroomOptions["1"]), "bedrooms:1"),
					tgbotapi.NewInlineKeyboardButtonData(getButtonText("2", state.Preferences.BedroomOptions["2"]), "bedrooms:2"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(getButtonText("3", state.Preferences.BedroomOptions["3"]), "bedrooms:3"),
					tgbotapi.NewInlineKeyboardButtonData(getButtonText("4", state.Preferences.BedroomOptions["4"]), "bedrooms:4"),
					tgbotapi.NewInlineKeyboardButtonData(getButtonText("5+", state.Preferences.BedroomOptions["5+"]), "bedrooms:5+"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Done", "bedrooms:done"),
				),
			)

			editMsg := tgbotapi.NewEditMessageReplyMarkup(query.Message.Chat.ID, query.Message.MessageID, keyboard)
			b.api.Send(editMsg)
		}
	case "furnished":
		if data[1] == "done" {
			b.askLocation(query.Message.Chat.ID)
		} else {
			// Toggle the selected state
			state.Preferences.FurnishedOptions[data[1]] = !state.Preferences.FurnishedOptions[data[1]]

			// Update the keyboard
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(getButtonText("Furnished", state.Preferences.FurnishedOptions["Furnished"]), "furnished:Furnished"),
					tgbotapi.NewInlineKeyboardButtonData(getButtonText("Unfurnished", state.Preferences.FurnishedOptions["Unfurnished"]), "furnished:Unfurnished"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Done", "furnished:done"),
				),
			)

			editMsg := tgbotapi.NewEditMessageReplyMarkup(query.Message.Chat.ID, query.Message.MessageID, keyboard)
			b.api.Send(editMsg)
		}
	case "location":
		if data[1] == "Bath" {
			state.Preferences.Location = "Bath"
			state.Stage = "showing_summary" // Update the state to showing_summary
			b.updateUserState(query.From.ID, state)
			b.showSummary(query.Message.Chat.ID)
		}
	case "save":
		if len(data) != 2 {
			b.answerCallbackQuery(query.ID, "Invalid save request")
			return
		}
		propertyID, err := strconv.Atoi(data[1])
		if err != nil {
			b.answerCallbackQuery(query.ID, "Invalid property ID")
			return
		}
		err = database.SaveListing(b.db, int64(query.From.ID), propertyID)
		if err != nil {
			b.answerCallbackQuery(query.ID, "Error saving listing")
			return
		}

		// Update the message to reflect that the listing has been saved
		editMsg := tgbotapi.NewEditMessageText(
			query.Message.Chat.ID,
			query.Message.MessageID,
			query.Message.Text+"\n\n✅ Saved",
		)
		editMsg.ParseMode = "HTML"

		// Create the keyboard markup and assign its address to ReplyMarkup
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Saved ✅", "noop"),
			),
		)
		editMsg.ReplyMarkup = &keyboard

		_, err = b.api.Send(editMsg)
		if err != nil {
			log.Printf("Error updating message: %v", err)
		}

		b.answerCallbackQuery(query.ID, "Listing saved successfully!")
	case "noop":
		// Do nothing for the "Saved ✅" button
		b.answerCallbackQuery(query.ID, "")
	case "delete":
		if len(data) != 2 {
			b.answerCallbackQuery(query.ID, "Invalid delete request")
			return
		}
		propertyID, err := strconv.Atoi(data[1])
		if err != nil {
			b.answerCallbackQuery(query.ID, "Invalid property ID")
			return
		}
		err = database.DeleteSavedListing(b.db, int64(query.From.ID), propertyID)
		if err != nil {
			b.answerCallbackQuery(query.ID, "Error deleting listing")
			return
		}

		// Update the message to reflect that the listing has been deleted
		editMsg := tgbotapi.NewEditMessageText(
			query.Message.Chat.ID,
			query.Message.MessageID,
			query.Message.Text+"\n\n❌ Deleted",
		)
		editMsg.ParseMode = "HTML"

		_, err = b.api.Send(editMsg)
		if err != nil {
			log.Printf("Error updating message: %v", err)
		}

		b.answerCallbackQuery(query.ID, "Listing deleted successfully!")
	}

	b.updateUserState(int64(query.From.ID), state)
	b.answerCallbackQuery(query.ID, "")
}

// editMessageReplyMarkup updates the reply markup (inline keyboard) of an existing message.
// It's used to refresh the keyboard after a user interaction.
func (b *Bot) editMessageReplyMarkup(chatID int64, messageID int, keyboard tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, keyboard)
	_, err := b.api.Send(edit)
	if err != nil {
		log.Printf("Error editing message: %v", err)
	}
}

// answerCallbackQuery sends a response to a callback query.
// This is required by the Telegram Bot API to acknowledge that the query was received and processed.
func (b *Bot) answerCallbackQuery(callbackQueryID string, text string) {
	callback := tgbotapi.NewCallback(callbackQueryID, text)
	_, err := b.api.Request(callback)
	if err != nil {
		log.Printf("Error answering callback query: %v", err)
	}
}

// handleHelpCommand processes the /help command.
// It sends a message to the user with a list of available commands and their descriptions.
func (b *Bot) handleHelpCommand(message *tgbotapi.Message) {
	helpText := `Welcome to RentSeekerBot!

Below are the commands you can use to interact with the bot:

	1.	/start - Initiates the bot and displays a welcome message.

	2.	/search- Starts a property search using your saved preferences if they are available. You will be prompted to provide details if no preferences are saved.
	
	3.	/save_preferences - Saves your current search preferences for future use. This includes details such as property type, price range, number of bedrooms, furnishing status, and location.
	
	4.	/view_preferences - Displays your currently saved search preferences.
	
	5.	/clear_preferences - Clears all your saved search preferences.
	
	6.	/help - Provides information about all available commands and their usage.

 	7.  /saved - View all your saved property listings. To save a listing, use the "Save Listing" button that appears below each property listing.
	

If you need further assistance or have any questions, please do not hesitate to contact our support team. Thank you for using RentSeekerBot!
`

	b.sendMessage(message.Chat.ID, helpText, nil)
}

// handleRegularMessage processes non-command messages.
func (b *Bot) handleRegularMessage(message *tgbotapi.Message) {
	state := b.getUserState(int64(message.From.ID))

	switch state.Stage {
	case stageAwaitingPriceRange:
		if b.validatePriceRange(message.Text) {
			state.Preferences.PriceRange = message.Text
			b.askFurnished(message.Chat.ID)
		} else {
			b.sendMessage(message.Chat.ID, "Invalid price range format. Please use the format: min - max (e.g., 1200 - 1800)", nil)
			return
		}
	case stageAwaitingLocation:
		log.Printf("Handling awaiting_location state")
		state.Preferences.Location = message.Text
		log.Printf("Updated location to: %s", state.Preferences.Location)
		b.updateUserState(message.From.ID, state)
		b.showSummary(message.Chat.ID)
	default:
		log.Printf("Unhandled state: %s", state.Stage)
		b.sendMessage(message.Chat.ID, "I'm sorry, I didn't understand that. Please use the provided buttons or follow the instructions.", nil)
	}

	b.updateUserState(int64(message.From.ID), state)
}

// askPropertyType asks the user to select a property type.
func (b *Bot) askPropertyType(chatID int64) {
	state := b.getUserState(chatID)
	if state.Preferences == nil {
		state.Preferences = NewFlexibleSearchPreferences()
	}
	if len(state.Preferences.PropertyTypes) == 0 {
		state.Preferences.PropertyTypes["Flat"] = false
		state.Preferences.PropertyTypes["House"] = false
	}

	keyboard := createMultiSelectKeyboard(state.Preferences.PropertyTypes, "property_type")
	b.sendMessage(chatID, "🏠 Select property type(s):", keyboard)
	state.Stage = stageAwaitingPropertyType
	b.updateUserState(chatID, state)
}

// askPriceRange asks the user to input a price range.
func (b *Bot) askPriceRange(chatID int64) {
	state := b.getUserState(chatID)
	state.Stage = stageAwaitingPriceRange
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
	// Trim spaces from each part
	min := strings.TrimSpace(parts[0])
	max := strings.TrimSpace(parts[1])

	// Check if both parts are valid numbers
	_, err1 := strconv.Atoi(min)
	_, err2 := strconv.Atoi(max)

	return err1 == nil && err2 == nil
}

// askBedrooms asks the user to select the number of bedrooms.
func (b *Bot) askBedrooms(chatID int64) {
	state := b.getUserState(chatID)
	if len(state.Preferences.BedroomOptions) == 0 {
		state.Preferences.BedroomOptions = map[string]bool{
			"Studio": false,
			"1":      false,
			"2":      false,
			"3":      false,
			"4":      false,
			"5+":     false,
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(getButtonText("Studio", state.Preferences.BedroomOptions["Studio"]), "bedrooms:Studio"),
			tgbotapi.NewInlineKeyboardButtonData(getButtonText("1", state.Preferences.BedroomOptions["1"]), "bedrooms:1"),
			tgbotapi.NewInlineKeyboardButtonData(getButtonText("2", state.Preferences.BedroomOptions["2"]), "bedrooms:2"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(getButtonText("3", state.Preferences.BedroomOptions["3"]), "bedrooms:3"),
			tgbotapi.NewInlineKeyboardButtonData(getButtonText("4", state.Preferences.BedroomOptions["4"]), "bedrooms:4"),
			tgbotapi.NewInlineKeyboardButtonData(getButtonText("5+", state.Preferences.BedroomOptions["5+"]), "bedrooms:5+"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Done", "bedrooms:done"),
		),
	)

	b.sendMessage(chatID, "🛏 Select the number of bedrooms (you can select multiple options):", keyboard)
	state.Stage = stageAwaitingBedrooms
	b.updateUserState(chatID, state)
}

// getButtonText generates the display text for a button in the inline keyboard.
// It adds a checkmark emoji (✅) in front of the option text if it is selected.
func getButtonText(option string, isSelected bool) string {
	if isSelected {
		return "✅ " + option
	}
	return option
}

// askFurnished asks if the user wants furnished or unfurnished accommodation.
func (b *Bot) askFurnished(chatID int64) {
	state := b.getUserState(chatID)
	if len(state.Preferences.FurnishedOptions) == 0 {
		state.Preferences.FurnishedOptions = map[string]bool{
			"Furnished":   false,
			"Unfurnished": false,
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(getButtonText("Furnished", state.Preferences.FurnishedOptions["Furnished"]), "furnished:Furnished"),
			tgbotapi.NewInlineKeyboardButtonData(getButtonText("Unfurnished", state.Preferences.FurnishedOptions["Unfurnished"]), "furnished:Unfurnished"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Done", "furnished:done"),
		),
	)

	b.sendMessage(chatID, "🪑 Do you want to search for furnished or unfurnished accommodation? (You can select both)", keyboard)
	state.Stage = stageAwaitingFurnished
	b.updateUserState(chatID, state)

}

// askLocation asks the user to input their preferred location.
func (b *Bot) askLocation(chatID int64) {
	state := b.getUserState(chatID)
	state.Stage = stageAwaitingLocation
	b.updateUserState(chatID, state)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Bath", "location:Bath"),
		),
	)

	b.sendMessage(chatID, "📍 The bot is in testing mode, so the search area is restricted to Bath. Please confirm the location:", keyboard)
}

// showSummary displays a summary of the user's preferences.
func (b *Bot) showSummary(chatID int64) {
	state := b.getUserState(chatID)
	prefs := state.Preferences

	// Compile property types
	var propertyTypes []string
	for pType, selected := range prefs.PropertyTypes {
		if selected {
			propertyTypes = append(propertyTypes, pType)
		}
	}

	// Compile bedroom options
	var bedrooms []string
	for bedroom, selected := range prefs.BedroomOptions {
		if selected {
			bedrooms = append(bedrooms, bedroom)
		}
	}

	furnishedOptions := getSelectedOptions(prefs.FurnishedOptions)
	furnishedStatus := "Any"
	if len(furnishedOptions) == 1 {
		furnishedStatus = furnishedOptions[0]
	}

	summary := fmt.Sprintf("Great! Here's a summary of your preferences:\n\n"+
		"🏠 Property Type: %s\n"+
		"💰 Price Range: %s\n"+
		"🛏 Bedrooms: %s\n"+
		"🪑 Furnished: %s\n"+
		"📍 Location: %s\n\n"+
		"I'll now search for properties matching these criteria. Please wait a moment.",
		strings.Join(propertyTypes, ", "),
		prefs.PriceRange,
		strings.Join(bedrooms, ", "),
		furnishedStatus,
		prefs.Location)

	log.Printf("Sending summary message: %s", summary)
	b.sendMessage(chatID, summary, nil)

	// Perform the search
	properties, err := b.searchProperties(prefs)
	if err != nil {
		b.sendMessage(chatID, "Sorry, there was an error while searching for properties. Please try again later.", nil)
		return
	}
	if len(properties) == 0 {
		b.sendMessage(chatID, "Sorry, no properties match your criteria. Try adjusting your preferences and searching again.", nil)
		return
	}

	time.Sleep(5 * time.Second)
	b.presentSearchResults(chatID, properties)
}

// handleSavePreferences processes the user's request to save their current search preferences.
// It converts the current user state preferences to a format suitable for database storage
// and saves them using the database package.
func (b *Bot) handleSavePreferences(message *tgbotapi.Message) {
	state := b.getUserState(message.From.ID)
	prefs := state.Preferences

	dbPrefs := database.UserPreferences{
		UserID:         message.From.ID,
		PropertyTypes:  make(map[string]bool),
		BedroomOptions: make(map[string]bool),
		Location:       prefs.Location,
		Furnished:      make(map[string]bool),
	}

	// Copy PropertyTypes
	for pType, selected := range prefs.PropertyTypes {
		dbPrefs.PropertyTypes[pType] = selected
	}

	// Copy BedroomOptions
	for bedroom, selected := range prefs.BedroomOptions {
		dbPrefs.BedroomOptions[bedroom] = selected
	}

	// Parse price range
	if prefs.PriceRange != "" {
		parts := strings.Split(prefs.PriceRange, "-")
		if len(parts) == 2 {
			minPrice, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err == nil {
				dbPrefs.MinPrice = minPrice
			}
			maxPrice, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err == nil {
				dbPrefs.MaxPrice = maxPrice
			}
		}
	}
	err := database.SaveUserPreferences(b.db, dbPrefs)
	if err != nil {
		b.sendMessage(message.Chat.ID, "Sorry, there was an error saving your preferences.", nil)
	} else {
		b.sendMessage(message.Chat.ID, "Your preferences have been saved successfully!", nil)
	}
}

// handleViewPreferences retrieves and displays the user's saved preferences.
// If no preferences are saved, it informs the user.
func (b *Bot) handleViewPreferences(message *tgbotapi.Message) {
	prefs, err := database.GetUserPreferences(b.db, message.From.ID)
	if err != nil {
		b.sendMessage(message.Chat.ID, "You haven't saved any preferences yet.", nil)
		return
	}

	// Format PropertyTypes
	var propertyTypes []string
	for pType, selected := range prefs.PropertyTypes {
		if selected {
			propertyTypes = append(propertyTypes, pType)
		}
	}

	// Format BedroomOptions
	var bedrooms []string
	for bedroom, selected := range prefs.BedroomOptions {
		if selected {
			bedrooms = append(bedrooms, bedroom)
		}
	}

	prefsMsg := fmt.Sprintf("Your saved preferences:\n"+
		"Property Type: %s\n"+
		"Price Range: £%d - £%d\n"+
		"Bedrooms: %s\n"+
		"Furnished: %v\n"+
		"Location: %s",
		strings.Join(propertyTypes, ", "), prefs.MinPrice, prefs.MaxPrice, strings.Join(bedrooms, ", "), prefs.Furnished, prefs.Location)

	b.sendMessage(message.Chat.ID, prefsMsg, nil)
}

// handleClearPreferences removes all saved preferences for the user from the database.
func (b *Bot) handleClearPreferences(message *tgbotapi.Message) {
	err := database.DeleteUserPreferences(b.db, message.From.ID)
	if err != nil {
		b.sendMessage(message.Chat.ID, "Sorry, there was an error clearing your preferences.", nil)
	} else {
		b.sendMessage(message.Chat.ID, "Your preferences have been cleared successfully!", nil)
	}
}

// performSearch executes a property search based on the given user preferences.
// It converts the database UserPreferences to the internal SearchPreferences format,
// performs the search, and presents the results to the user.
func (b *Bot) performSearch(chatID int64, prefs database.UserPreferences) {
	// Convert UserPreferences to the format expected by searchProperties
	searchPrefs := &SearchPreferences{
		PropertyTypes:    prefs.PropertyTypes,
		BedroomOptions:   prefs.BedroomOptions,
		FurnishedOptions: prefs.Furnished,
		PriceRange:       fmt.Sprintf("%d-%d", prefs.MinPrice, prefs.MaxPrice),
		Location:         prefs.Location,
	}

	properties, err := b.searchProperties(searchPrefs)
	if err != nil {
		b.sendMessage(chatID, "Sorry, there was an error while searching for properties. Please try again later.", nil)
		return
	}
	if len(properties) == 0 {
		b.sendMessage(chatID, "Sorry, no properties match your criteria. Try adjusting your preferences and searching again.", nil)
		return
	}

	b.presentSearchResults(chatID, properties)
}

// handleViewSavedListings shows all saved property listings
func (b *Bot) handleViewSavedListings(message *tgbotapi.Message) {
	properties, err := database.GetSavedListings(b.db, message.From.ID)
	if err != nil {
		b.sendMessage(message.Chat.ID, "Sorry, there was an error retrieving your saved listings. Please try again.", nil)
		return
	}

	if len(properties) == 0 {
		b.sendMessage(message.Chat.ID, "You haven't saved any listings yet.", nil)
		return
	}

	b.sendMessage(message.Chat.ID, "Here are your saved listings:", nil)
	b.presentMultipleProperties(message.Chat.ID, properties, true)
}
