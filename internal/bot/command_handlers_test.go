package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"imitation_project/internal/database"
	"strings"
	"testing"
	"time"
)

// MockBotAPI2 is a mock implementation of the BotAPI interface for testing purposes
type MockBotAPI2 struct {
	messages            []tgbotapi.MessageConfig
	editedMessages      []tgbotapi.EditMessageReplyMarkupConfig
	editedTexts         []tgbotapi.EditMessageTextConfig
	answerCallbacks     []tgbotapi.CallbackConfig
	performSearchCalled bool
	performSearchParams struct {
		ChatID int64
		Prefs  database.UserPreferences
	}
}

// GetUpdatesChan mocks the method to get updates channel
func (m *MockBotAPI2) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return make(tgbotapi.UpdatesChannel)
}

// Send mocks the method to send messages
func (m *MockBotAPI2) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	switch v := c.(type) {
	case tgbotapi.MessageConfig:
		m.messages = append(m.messages, v)
	case tgbotapi.EditMessageReplyMarkupConfig:
		m.editedMessages = append(m.editedMessages, v)
	case tgbotapi.EditMessageTextConfig:
		m.editedTexts = append(m.editedTexts, v)
	}
	return tgbotapi.Message{}, nil
}

// Request mocks the method to make API requests
func (m *MockBotAPI2) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	switch v := c.(type) {
	case tgbotapi.CallbackConfig:
		m.answerCallbacks = append(m.answerCallbacks, v)
	}
	return &tgbotapi.APIResponse{}, nil
}

// PerformSearch mocks the method to perform a search
func (m *MockBotAPI2) PerformSearch(chatID int64, prefs database.UserPreferences) {
	m.performSearchCalled = true
	m.performSearchParams.ChatID = chatID
	m.performSearchParams.Prefs = prefs
}

// MessageSent checks if a specific message was sent
func (m *MockBotAPI2) MessageSent(chatID int64, text string) bool {
	for _, msg := range m.messages {
		if msg.ChatID == chatID && strings.Contains(msg.Text, text) {
			return true
		}
	}
	return false
}

// MessageEdited checks if a message was edited
func (m *MockBotAPI2) MessageEdited(chatID int64, messageID int) bool {
	for _, edit := range m.editedMessages {
		if edit.ChatID == chatID && edit.MessageID == messageID {
			return true
		}
	}
	for _, edit := range m.editedTexts {
		if edit.ChatID == chatID && edit.MessageID == messageID {
			return true
		}
	}
	return false
}

// CallbackAnswered checks if a callback was answered
func (m *MockBotAPI2) CallbackAnswered(callbackID string) bool {
	for _, answer := range m.answerCallbacks {
		if answer.CallbackQueryID == callbackID {
			return true
		}
	}
	return false
}

// TestHandleStartCommand tests the handleStartCommand function
func TestHandleStartCommand(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI2{}
	bot := &Bot{
		api:   mockAPI,
		db:    db,
		state: make(map[int64]*UserState),
	}

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: 123,
		},
		From: &tgbotapi.User{
			FirstName: "Test",
		},
	}

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))

	bot.handleStartCommand(message)

	// Check if a message was sent
	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != 123 {
			t.Errorf("Expected message to be sent to chat ID 123, but was sent to %d", sentMessage.ChatID)
		}
		if sentMessage.Text == "" {
			t.Error("Expected non-empty message text")
		}
		if !strings.Contains(sentMessage.Text, "Welcome, Test!") {
			t.Error("Expected welcome message to contain user's first name")
		}
		if !strings.Contains(sentMessage.Text, "Bath area") {
			t.Error("Expected welcome message to mention Bath area")
		}
	}
}

// TestHandleHelpCommand tests the handleHelpCommand function
func TestHandleHelpCommand(t *testing.T) {
	mockAPI := &MockBotAPI2{}
	bot := &Bot{
		api:   mockAPI,
		state: make(map[int64]*UserState),
	}

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: 123,
		},
	}

	bot.handleHelpCommand(message)

	// Check if a message was sent
	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != 123 {
			t.Errorf("Expected message to be sent to chat ID 123, but was sent to %d", sentMessage.ChatID)
		}
		if sentMessage.Text == "" {
			t.Error("Expected non-empty message text")
		}
		expectedCommands := []string{"/start", "/search", "/save_preferences", "/view_preferences", "/clear_preferences", "/help", "/saved"}
		for _, cmd := range expectedCommands {
			if !strings.Contains(sentMessage.Text, cmd) {
				t.Errorf("Expected help message to contain %s command", cmd)
			}
		}
	}
}

// TestHandleSearchCommand tests the handleSearchCommand function
func TestHandleSearchCommand(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI2{}
	bot := &Bot{
		api:   mockAPI,
		db:    db,
		state: make(map[int64]*UserState),
	}

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: 123,
		},
		From: &tgbotapi.User{
			ID: 456,
		},
	}

	// Mock the database query for user preferences
	mock.ExpectQuery("SELECT (.+) FROM user_preferences").WillReturnRows(sqlmock.NewRows([]string{"user_id", "property_type", "min_price", "max_price", "bedrooms", "furnished", "location", "last_search"}))

	bot.handleSearchCommand(message)

	// Check if a message was sent
	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != 123 {
			t.Errorf("Expected message to be sent to chat ID 123, but was sent to %d", sentMessage.ChatID)
		}
		if !strings.Contains(sentMessage.Text, "Let's start your property search") {
			t.Error("Expected message to start a new search when no preferences are found")
		}
	}
}

// TestHandleSavePreferences tests the handleSavePreferences function
func TestHandleSavePreferences(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI2{}
	bot := &Bot{
		api:   mockAPI,
		db:    db,
		state: make(map[int64]*UserState),
	}

	userID := int64(123)
	chatID := int64(456)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: chatID,
		},
		From: &tgbotapi.User{
			ID: userID,
		},
	}

	// Set up a mock state for the user
	bot.state[userID] = &UserState{
		Preferences: &SearchPreferences{
			PropertyTypes:    map[string]bool{"Apartment": true},
			BedroomOptions:   map[string]bool{"2": true},
			PriceRange:       "1000-2000",
			Location:         "Bath",
			FurnishedOptions: map[string]bool{"Furnished": true},
		},
	}

	// Expect the INSERT OR REPLACE query
	mock.ExpectExec("INSERT OR REPLACE INTO user_preferences").WithArgs(
		userID,
		sqlmock.AnyArg(), // property_type JSON
		1000,
		2000,
		sqlmock.AnyArg(), // bedrooms JSON
		sqlmock.AnyArg(), // furnished JSON
		"Bath",
		sqlmock.AnyArg(), // last_search timestamp
	).WillReturnResult(sqlmock.NewResult(1, 1))

	bot.handleSavePreferences(message)

	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != chatID {
			t.Errorf("Expected message to be sent to chat ID %d, but was sent to %d", chatID, sentMessage.ChatID)
		}
		if !strings.Contains(sentMessage.Text, "preferences have been saved successfully") {
			t.Error("Expected success message for saving preferences")
		}
	}
}

// TestHandleViewPreferences tests the handleViewPreferences function
func TestHandleViewPreferences(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI2{}
	bot := &Bot{
		api:   mockAPI,
		db:    db,
		state: make(map[int64]*UserState),
	}

	userID := int64(123)
	chatID := int64(456)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: chatID,
		},
		From: &tgbotapi.User{
			ID: userID,
		},
	}

	propertyTypesJSON, _ := json.Marshal(map[string]bool{"Apartment": true})
	bedroomsJSON, _ := json.Marshal(map[string]bool{"2": true})
	furnishedJSON, _ := json.Marshal(map[string]bool{"Furnished": true})
	lastSearch := time.Now()

	rows := sqlmock.NewRows([]string{"user_id", "property_type", "min_price", "max_price", "bedrooms", "furnished", "location", "last_search"}).
		AddRow(userID, propertyTypesJSON, 1000, 2000, bedroomsJSON, furnishedJSON, "Bath", lastSearch)

	mock.ExpectQuery("SELECT (.+) FROM user_preferences").WithArgs(userID).WillReturnRows(rows)

	bot.handleViewPreferences(message)

	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != chatID {
			t.Errorf("Expected message to be sent to chat ID %d, but was sent to %d", chatID, sentMessage.ChatID)
		}
		expectedContent := []string{
			"Your saved preferences:",
			"Property Type: Apartment",
			"Price Range: £1000 - £2000",
			"Bedrooms: 2",
			"Furnished: map[Furnished:true]",
			"Location: Bath",
		}
		for _, content := range expectedContent {
			if !strings.Contains(sentMessage.Text, content) {
				t.Errorf("Expected preferences message to contain '%s', but it didn't. Message: %s", content, sentMessage.Text)
			}
		}
	}
}

// TestHandleClearPreferences tests the handleClearPreferences function
func TestHandleClearPreferences(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI2{}
	bot := &Bot{
		api:   mockAPI,
		db:    db,
		state: make(map[int64]*UserState),
	}

	userID := int64(123)
	chatID := int64(456)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: chatID,
		},
		From: &tgbotapi.User{
			ID: userID,
		},
	}

	mock.ExpectExec("DELETE FROM user_preferences").WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 1))

	bot.handleClearPreferences(message)

	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != chatID {
			t.Errorf("Expected message to be sent to chat ID %d, but was sent to %d", chatID, sentMessage.ChatID)
		}
		if !strings.Contains(sentMessage.Text, "preferences have been cleared successfully") {
			t.Error("Expected success message for clearing preferences")
		}
	}
}

// TestHandleViewSavedListings tests the handleViewSavedListings function
func TestHandleViewSavedListings(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI2{}
	bot := &Bot{
		api:   mockAPI,
		db:    db,
		state: make(map[int64]*UserState),
	}

	userID := int64(123)
	chatID := int64(456)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: chatID,
		},
		From: &tgbotapi.User{
			ID: userID,
		},
	}

	rows := sqlmock.NewRows([]string{"id", "type", "price_per_month", "bedrooms", "furnished", "location", "description", "photo_urls", "web_link"}).
		AddRow(1, "Apartment", 1500, 2, true, "Bath", "Nice apartment", "[]", "http://example.com/property1")

	mock.ExpectQuery("SELECT (.+) FROM properties (.+) JOIN saved_listings").WithArgs(userID).WillReturnRows(rows)

	bot.handleViewSavedListings(message)

	if len(mockAPI.messages) < 2 {
		t.Error("Expected at least two messages to be sent")
	} else {
		introMessage := mockAPI.messages[0]
		if introMessage.ChatID != chatID {
			t.Errorf("Expected message to be sent to chat ID %d, but was sent to %d", chatID, introMessage.ChatID)
		}
		if !strings.Contains(introMessage.Text, "Here are your saved listings") {
			t.Error("Expected introduction message for saved listings")
		}

		propertyMessage := mockAPI.messages[1]
		expectedContent := []string{"Apartment", "1500", "2", "Bath", "Nice apartment", "http://example.com/property1"}
		for _, content := range expectedContent {
			if !strings.Contains(propertyMessage.Text, content) {
				t.Errorf("Expected property message to contain %s", content)
			}
		}
	}
}

// TestHandleViewPreferencesNoPreferences tests the handleViewPreferences function when no preferences are saved
func TestHandleViewPreferencesNoPreferences(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI2{}
	bot := &Bot{
		api:   mockAPI,
		db:    db,
		state: make(map[int64]*UserState),
	}

	userID := int64(123)
	chatID := int64(456)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: chatID,
		},
		From: &tgbotapi.User{
			ID: userID,
		},
	}

	mock.ExpectQuery("SELECT (.+) FROM user_preferences").WithArgs(userID).WillReturnError(sql.ErrNoRows)

	bot.handleViewPreferences(message)

	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != chatID {
			t.Errorf("Expected message to be sent to chat ID %d, but was sent to %d", chatID, sentMessage.ChatID)
		}
		if !strings.Contains(sentMessage.Text, "You haven't saved any preferences yet") {
			t.Errorf("Expected message about no saved preferences, got: %s", sentMessage.Text)
		}
	}
}

// TestHandleSavePreferencesError tests the handleSavePreferences function when an error occurs
func TestHandleSavePreferencesError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI2{}
	bot := &Bot{
		api:   mockAPI,
		db:    db,
		state: make(map[int64]*UserState),
	}

	userID := int64(123)
	chatID := int64(456)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: chatID,
		},
		From: &tgbotapi.User{
			ID: userID,
		},
	}

	bot.state[userID] = &UserState{
		Preferences: &SearchPreferences{
			PropertyTypes:    map[string]bool{"Apartment": true},
			BedroomOptions:   map[string]bool{"2": true},
			PriceRange:       "1000-2000",
			Location:         "Bath",
			FurnishedOptions: map[string]bool{"Furnished": true},
		},
	}

	mock.ExpectExec("INSERT OR REPLACE INTO user_preferences").WithArgs(
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
	).WillReturnError(errors.New("database error"))

	bot.handleSavePreferences(message)

	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != chatID {
			t.Errorf("Expected message to be sent to chat ID %d, but was sent to %d", chatID, sentMessage.ChatID)
		}
		if !strings.Contains(sentMessage.Text, "Sorry, there was an error saving your preferences") {
			t.Errorf("Expected error message for saving preferences, got: %s", sentMessage.Text)
		}
	}
}

// TestHandleSearchCommandWithExistingPreferences tests the handleSearchCommand function with existing preferences
func TestHandleSearchCommandWithExistingPreferences(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI2{}
	bot := &Bot{
		api:   mockAPI,
		db:    db,
		state: make(map[int64]*UserState),
	}

	userID := int64(123)
	chatID := int64(456)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: chatID,
		},
		From: &tgbotapi.User{
			ID: userID,
		},
	}

	propertyTypesJSON, _ := json.Marshal(map[string]bool{"Apartment": true})
	bedroomsJSON, _ := json.Marshal(map[string]bool{"2": true})
	furnishedJSON, _ := json.Marshal(map[string]bool{"Furnished": true})
	lastSearch := time.Now()

	rows := sqlmock.NewRows([]string{"user_id", "property_type", "min_price", "max_price", "bedrooms", "furnished", "location", "last_search"}).
		AddRow(userID, propertyTypesJSON, 1000, 2000, bedroomsJSON, furnishedJSON, "Bath", lastSearch)

	mock.ExpectQuery("SELECT (.+) FROM user_preferences").WithArgs(userID).WillReturnRows(rows)

	bot.handleSearchCommand(message)

	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != chatID {
			t.Errorf("Expected message to be sent to chat ID %d, but was sent to %d", chatID, sentMessage.ChatID)
		}
		if !strings.Contains(sentMessage.Text, "You have saved preferences") {
			t.Errorf("Expected message about existing preferences, got: %s", sentMessage.Text)
		}
		if !strings.Contains(sentMessage.Text, "Would you like to use these preferences or start a new search?") {
			t.Errorf("Expected message to ask about using existing preferences, got: %s", sentMessage.Text)
		}
	}
}

// TestHandleInvalidCommand tests the handling of an invalid command
func TestHandleInvalidCommand(t *testing.T) {
	mockAPI := &MockBotAPI2{}
	bot := &Bot{
		api:   mockAPI,
		state: make(map[int64]*UserState),
	}

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{
			ID: 123,
		},
		Text: "/invalidcommand",
	}

	bot.handleCommand(message)

	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != 123 {
			t.Errorf("Expected message to be sent to chat ID 123, but was sent to %d", sentMessage.ChatID)
		}
		if !strings.Contains(sentMessage.Text, "Unknown command") {
			t.Errorf("Expected unknown command message, got: %s", sentMessage.Text)
		}
	}
}

// TestValidatePriceRangeEdgeCases tests the validatePriceRange function with various edge cases
func TestValidatePriceRangeEdgeCases(t *testing.T) {
	bot := &Bot{}
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid range", "1000-2000", true},
		{"Valid range with spaces", " 1000 - 2000 ", true},
		{"Invalid format", "1000 2000", false},
		{"Single number", "1000", false},
		{"Empty string", "", false},
		{"Non-numeric", "abc-def", false},
		{"Negative numbers", "-1000--500", false}, // Depending on your implementation
		{"Zero in range", "0-1000", true},
		{"Reversed range", "2000-1000", true}, // Depending on your implementation
		{"Very large numbers", "1000000000-2000000000", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := bot.validatePriceRange(tc.input)
			if result != tc.expected {
				t.Errorf("validatePriceRange(%s) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestShowSummaryWithDifferentPreferences tests the showSummary function with different preference configurations
func TestShowSummaryWithDifferentPreferences(t *testing.T) {
	testCases := []struct {
		name        string
		preferences *SearchPreferences
		expected    []string
	}{
		{
			name: "All preferences set",
			preferences: &SearchPreferences{
				PropertyTypes:    map[string]bool{"Apartment": true, "House": true},
				BedroomOptions:   map[string]bool{"2": true, "3": true},
				PriceRange:       "1000-2000",
				Location:         "Bath",
				FurnishedOptions: map[string]bool{"Furnished": true},
			},
			expected: []string{"Apartment", "House", "1000-2000", "2", "3", "Furnished", "Bath"},
		},
		{
			name: "Minimal preferences",
			preferences: &SearchPreferences{
				PropertyTypes: map[string]bool{"Apartment": true},
				PriceRange:    "1000-2000",
				Location:      "Bath",
			},
			expected: []string{"Apartment", "1000-2000", "Bath"},
		},
		{
			name: "No furnished preference",
			preferences: &SearchPreferences{
				PropertyTypes:  map[string]bool{"House": true},
				BedroomOptions: map[string]bool{"4": true},
				PriceRange:     "2000-3000",
				Location:       "Bath",
			},
			expected: []string{"House", "2000-3000", "4", "Bath"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAPI := &MockBotAPI2{}
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			bot := &Bot{
				api:   mockAPI,
				db:    db,
				state: make(map[int64]*UserState),
			}

			userID := int64(123)
			chatID := int64(123)

			bot.state[userID] = &UserState{
				Stage:       "showing_summary",
				Preferences: tc.preferences,
			}

			// Mock the database query for property search
			rows := sqlmock.NewRows([]string{"id", "type", "price_per_month", "bedrooms", "furnished", "location", "description", "photo_urls", "web_link"})
			mock.ExpectQuery("SELECT (.+) FROM properties").WillReturnRows(rows)

			bot.showSummary(chatID)

			if len(mockAPI.messages) < 1 {
				t.Errorf("Expected at least 1 message to be sent, but got %d", len(mockAPI.messages))
			} else {
				summaryMessage := mockAPI.messages[0]
				for _, expectedContent := range tc.expected {
					if !strings.Contains(summaryMessage.Text, expectedContent) {
						t.Errorf("Expected summary message to contain '%s', but it didn't. Message: %s", expectedContent, summaryMessage.Text)
					}
				}
			}
		})
	}
}

// TestHandleRegularMessageWithDifferentStates tests the handleRegularMessage function with different user states
func TestHandleRegularMessageWithDifferentStates(t *testing.T) {
	testCases := []struct {
		name          string
		initialState  string
		messageText   string
		expectedState string
		expectedReply string
		setupMock     func(mock sqlmock.Sqlmock)
	}{
		{
			name:          "Awaiting price range",
			initialState:  "awaiting_price_range",
			messageText:   "1000-2000",
			expectedState: "awaiting_furnished",
			expectedReply: "Do you want to search for furnished or unfurnished accommodation?",
			setupMock:     func(mock sqlmock.Sqlmock) {}, // No database interaction expected
		},
		{
			name:          "Awaiting location",
			initialState:  "awaiting_location",
			messageText:   "Bath",
			expectedState: "awaiting_location",
			expectedReply: "Great! Here's a summary of your preferences:",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Mock the search query
				rows := sqlmock.NewRows([]string{"id", "type", "price_per_month", "bedrooms", "furnished", "location", "description", "photo_urls", "web_link"}).
					AddRow(1, "Apartment", 1500, 2, true, "Bath", "Nice apartment", "[]", "http://example.com/property1")
				mock.ExpectQuery("SELECT (.+) FROM properties").WillReturnRows(rows)
			},
		},
		{
			name:          "Invalid state",
			initialState:  "invalid_state",
			messageText:   "Some message",
			expectedState: "invalid_state",
			expectedReply: "I'm sorry, I didn't understand that.",
			setupMock:     func(mock sqlmock.Sqlmock) {}, // No database interaction expected
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAPI := &MockBotAPI2{}
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			// Setup mock expectations
			tc.setupMock(mock)

			bot := &Bot{
				api:   mockAPI,
				db:    db,
				state: make(map[int64]*UserState),
			}

			userID := int64(123)
			chatID := int64(123)

			bot.state[userID] = &UserState{
				Stage:       tc.initialState,
				Preferences: &SearchPreferences{},
			}

			message := &tgbotapi.Message{
				Chat: &tgbotapi.Chat{
					ID: chatID,
				},
				From: &tgbotapi.User{
					ID: userID,
				},
				Text: tc.messageText,
			}

			bot.handleRegularMessage(message)

			if bot.state[userID].Stage != tc.expectedState {
				t.Errorf("Expected state to be '%s', but got '%s'", tc.expectedState, bot.state[userID].Stage)
			}

			// Ensure all database expectations were met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled database expectations: %s", err)
			}
		})
	}
}

// TestHandleCallbackQueryWithVariousData tests the handleCallbackQuery function with various callback data
func TestHandleCallbackQueryWithVariousData(t *testing.T) {
	testCases := []struct {
		name           string
		callbackData   string
		initialState   string
		expectedState  string
		expectedAction string
		setupMock      func(mock sqlmock.Sqlmock)
	}{
		{
			name:           "Use saved preferences",
			callbackData:   "use_saved_prefs",
			initialState:   "initial",
			expectedState:  "initial",
			expectedAction: "performSearch",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"user_id", "property_type", "min_price", "max_price", "bedrooms", "furnished", "location", "last_search"}).
					AddRow(123, "{}", 1000, 2000, "{}", "{}", "Bath", time.Now())
				mock.ExpectQuery("SELECT (.+) FROM user_preferences").WillReturnRows(rows)
				mock.ExpectQuery("SELECT (.+) FROM properties").WillReturnRows(sqlmock.NewRows([]string{"id", "type", "price_per_month", "bedrooms", "furnished", "location", "description", "photo_urls", "web_link"}))
			},
		},
		{
			name:           "Start new search",
			callbackData:   "start_new_search",
			initialState:   "initial",
			expectedState:  "awaiting_property_type",
			expectedAction: "sendMessage",
		},
		{
			name:           "Property type selection",
			callbackData:   "property_type:Apartment",
			initialState:   "awaiting_property_type",
			expectedState:  "awaiting_property_type",
			expectedAction: "editMessageReplyMarkup",
		},
		{
			name:           "Bedroom selection",
			callbackData:   "bedrooms:2",
			initialState:   "awaiting_bedrooms",
			expectedState:  "awaiting_bedrooms",
			expectedAction: "editMessageReplyMarkup",
		},
		{
			name:           "Furnished selection",
			callbackData:   "furnished:Furnished",
			initialState:   "awaiting_furnished",
			expectedState:  "awaiting_furnished",
			expectedAction: "editMessageReplyMarkup",
		},
		{
			name:           "Location selection",
			callbackData:   "location:Bath",
			initialState:   "awaiting_location",
			expectedState:  "showing_summary",
			expectedAction: "sendMessage",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM properties").WillReturnRows(sqlmock.NewRows([]string{"id", "type", "price_per_month", "bedrooms", "furnished", "location", "description", "photo_urls", "web_link"}))
			},
		},
		{
			name:           "Save listing",
			callbackData:   "save:123",
			initialState:   "showing_results",
			expectedState:  "showing_results",
			expectedAction: "editMessageText",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT OR IGNORE INTO saved_listings").WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:           "Delete listing",
			callbackData:   "delete:123",
			initialState:   "showing_saved",
			expectedState:  "showing_saved",
			expectedAction: "editMessageText",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM saved_listings").WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAPI := &MockBotAPI2{}
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			if tc.setupMock != nil {
				tc.setupMock(mock)
			}

			bot := &Bot{
				api:   mockAPI,
				db:    db,
				state: make(map[int64]*UserState),
			}

			userID := int64(123)
			chatID := int64(123)

			bot.state[userID] = &UserState{
				Stage: tc.initialState,
				Preferences: &SearchPreferences{
					PropertyTypes:    make(map[string]bool),
					BedroomOptions:   make(map[string]bool),
					FurnishedOptions: make(map[string]bool),
				},
			}

			callbackQuery := &tgbotapi.CallbackQuery{
				Data: tc.callbackData,
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{
						ID: chatID,
					},
					MessageID: 456,
				},
				From: &tgbotapi.User{
					ID: int64(userID),
				},
				ID: "query_id",
			}

			bot.handleCallbackQuery(callbackQuery)

			if bot.state[userID].Stage != tc.expectedState {
				t.Errorf("Expected state to be '%s', but got '%s'", tc.expectedState, bot.state[userID].Stage)
			}

			var actionPerformed string
			if len(mockAPI.messages) > 0 {
				actionPerformed = "sendMessage"
			} else if len(mockAPI.editedMessages) > 0 {
				actionPerformed = "editMessageReplyMarkup"
			} else if len(mockAPI.editedTexts) > 0 {
				actionPerformed = "editMessageText"
			}

			// Special case for performSearch action
			if tc.expectedAction == "performSearch" && actionPerformed == "" {
				// Check if performSearch was called (you might need to add a way to track this in your MockBotAPI2)
				if !mockAPI.performSearchCalled {
					t.Errorf("Expected performSearch to be called, but it wasn't")
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled database expectations: %s", err)
			}
		})
	}
}
