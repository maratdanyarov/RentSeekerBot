// Package bot provides the core functionality for the Telegram bot.
package bot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"imitation_project/internal/database"
	"log"
	"strings"
	"time"
)

// handleSearchCommand processes the /search command.
func (b *Bot) handleSearchCommand(message *tgbotapi.Message) {
	prefs, err := database.GetUserPreferences(b.db, message.From.ID)
	if err == nil && !prefs.LastSearch.IsZero() {
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
		// User has saved preferences
		prefsMsg := fmt.Sprintf("You have saved preferences:\n"+
			"Property Type: %s\n"+
			"Price Range: £%d - £%d\n"+
			"Bedrooms: %v\n"+
			"Furnished: %v\n"+
			"Location: %s\n\n"+
			"Would you like to use these preferences or start a new search?",
			strings.Join(propertyTypes, ", "), prefs.MinPrice, prefs.MaxPrice, strings.Join(bedrooms, ", "), prefs.Furnished, prefs.Location)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Use Saved Preferences", "use_saved_prefs"),
				tgbotapi.NewInlineKeyboardButtonData("Start New Search", "start_new_search"),
			),
		)

		b.sendMessage(message.Chat.ID, prefsMsg, keyboard)
	} else {
		// No saved preferences or error retrieving them, start new search
		b.startNewSearch(message.Chat.ID)
	}
}

// startNewSearch initiates a new property search process for the user.
// It sends an introductory message and starts the preference collection process.
func (b *Bot) startNewSearch(chatID int64) {
	searchText := `Let's start your property search."
					I'll ask you a series of questions to understand your preferences.`
	b.sendMessage(chatID, searchText, nil)

	// Start the preference collection process
	b.askPropertyType(chatID)
}

// presentSearchResults displays the search results to the user in a staged manner.
// It simulates a real-world scenario where properties are found over time.
func (b *Bot) presentSearchResults(chatID int64, properties []database.Property) {
	if len(properties) == 0 {
		b.sendMessage(chatID, "Sorry, no properties match your criteria. Try adjusting your preferences and searching again.", nil)
		return
	}

	// Present first 3 properties (or less if there are fewer than 3)
	numInitialProperties := min(3, len(properties))
	b.presentMultipleProperties(chatID, properties[:numInitialProperties], false) // Note the 'false' here

	if len(properties) > 3 {
		b.sendMessage(chatID, "As the bot is in testing mode, please assume that several hours have passed. So, a few moments later…", nil)
		time.Sleep(5 * time.Second)

		// Present next 2 properties (or less if there are fewer than 5 total)
		numNextProperties := min(2, len(properties)-3)
		b.presentMultipleProperties(chatID, properties[3:3+numNextProperties], false) // Note the 'false' here

		if len(properties) > 5 {
			b.sendMessage(chatID, "As the bot is in testing mode, please assume that one day has passed. So, a few moments later…", nil)
			time.Sleep(5 * time.Second)

			b.sendMessage(chatID, "🔔 Alert: New property found matching your criteria!", nil)
			time.Sleep(time.Second)

			// Present the last property
			b.presentMultipleProperties(chatID, properties[len(properties)-1:], false) // Note the 'false' here
		}
	}

	b.sendMessage(chatID, "That's all the properties I found matching your criteria. Would you like to start a new search?", nil)
}

// presentProperty displays a single property to the user.
// It formats the property information and sends it along with photos if available.
func (b *Bot) presentProperty(prop database.Property, isSaved bool) (string, tgbotapi.InlineKeyboardMarkup) {
	message := fmt.Sprintf(
		"🏠 %s\n"+
			"💰 £%d per month\n"+
			"🛏 %d bedrooms\n"+
			"📍 %s\n"+
			"🔑 %s\n\n"+
			"📝 Description: %s\n\n"+
			"🔗 <a href=\"%s\">View on website</a>",
		prop.Type,
		prop.PricePerMonth,
		prop.Bedrooms,
		prop.Location,
		propertyFurnished(prop.Furnished),
		prop.Description,
		prop.WebLink)

	var keyboard tgbotapi.InlineKeyboardMarkup
	if isSaved {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Delete Listing", fmt.Sprintf("delete:%d", prop.ID)),
			),
		)
	} else {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Save Listing", fmt.Sprintf("save:%d", prop.ID)),
			),
		)
	}

	return message, keyboard
}

// propertyFurnished converts boolean to "Furnished" or "Unfurnished"
func propertyFurnished(furnished bool) string {
	if furnished {
		return "Furnished"
	}
	return "Unfurnished"
}

// presentMultipleProperties displays multiple properties to the user.
// It formats the property information and sends it along with photos if available.
func (b *Bot) presentMultipleProperties(chatID int64, properties []database.Property, isSaved bool) {
	for _, prop := range properties {
		message, keyboard := b.presentProperty(prop, isSaved)

		if len(prop.PhotoURLs) > 0 {
			media := make([]interface{}, len(prop.PhotoURLs))
			for i, photoURL := range prop.PhotoURLs {
				media[i] = tgbotapi.NewInputMediaPhoto(tgbotapi.FileURL(photoURL))
			}

			mediaGroup := tgbotapi.NewMediaGroup(chatID, media)
			_, err := b.api.Send(mediaGroup)
			if err != nil {
				log.Printf("Error sending media group: %v", err)
			}
		}

		msg := tgbotapi.NewMessage(chatID, message)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = keyboard
		_, err := b.api.Send(msg)
		if err != nil {
			log.Printf("Error sending message: %v", err)
		}

		// Add a delay between messages to avoid hitting rate limits
		time.Sleep(time.Second)
	}
}
