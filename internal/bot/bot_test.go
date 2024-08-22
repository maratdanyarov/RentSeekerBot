package bot

import (
	"github.com/DATA-DOG/go-sqlmock"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
	"strings"
	"testing"
)

// MockBotAPI is a mock implementation of the BotAPI interface
type MockBotAPI struct{}

// GetUpdatesChan mocks the method to get updates channel
func (m *MockBotAPI) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return make(tgbotapi.UpdatesChannel)
}

// Send mocks the method to send messages
func (m *MockBotAPI) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	return tgbotapi.Message{}, nil
}

// Request mocks the method to make API requests
func (m *MockBotAPI) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return &tgbotapi.APIResponse{}, nil
}

// TestNew tests the New function that creates a new Bot instance
func TestNew(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI{}
	botUserName := "testbot"
	b := New(mockAPI, db, botUserName)

	if b == nil {
		t.Error("New() returned nil")
	}

	if b.api != mockAPI {
		t.Error("New() did not set the api field correctly")
	}

	if b.db != db {
		t.Error("New() did not set the db field correctly")
	}

	if b.state == nil {
		t.Error("New() did not initialize the state map")
	}

	if b.botUserName != botUserName {
		t.Errorf("New() did not set the botUserName correctly. Got %s, want %s", b.botUserName, botUserName)
	}
}

// TestGetUserState tests the getUserState method
func TestGetUserState(t *testing.T) {
	b := &Bot{
		state: make(map[int64]*UserState),
	}

	userID := int64(123)
	state := b.getUserState(userID)

	if state == nil {
		t.Errorf("getUserState() returned nil")
	}

	if state.Stage != "initial" {
		t.Errorf("getUserState() returned state with incorrect stage: got %v, want %v", state.Stage, "initial")
	}

	if state.Preferences == nil {
		t.Errorf("getUserState() returned state with nil Preferences")
	}
}

// TestUpdateUserState tests the updateUserState method
func TestUpdateUserState(t *testing.T) {
	b := &Bot{
		state: make(map[int64]*UserState),
	}

	userID := int64(123)
	newState := &UserState{
		Stage: "test_stage",
		Preferences: &SearchPreferences{
			PropertyTypes: map[string]bool{"Apartment": true},
		},
	}

	b.updateUserState(userID, newState)

	if b.state[userID] != newState {
		t.Errorf("updateUserState() did not update the state correctly")
	}
}

// TestStartNewSearch tests the startNewSearch method
func TestStartNewSearch(t *testing.T) {
	mockAPI := &MockBotAPI3{}
	bot := &Bot{
		api:   mockAPI,
		state: make(map[int64]*UserState),
	}

	bot.startNewSearch(123)

	if len(mockAPI.messages) != 2 {
		t.Errorf("Expected 2 messages to be sent, but got %d", len(mockAPI.messages))
	} else {
		// Check first message
		firstMessage := mockAPI.messages[0]
		if firstMessage.ChatID != 123 {
			t.Errorf("Expected first message to be sent to chat ID 123, but was sent to %d", firstMessage.ChatID)
		}
		expectedContent := "Let's start your property search"
		if !strings.Contains(firstMessage.Text, expectedContent) {
			t.Errorf("Expected first message to contain '%s', but it didn't. Message: %s", expectedContent, firstMessage.Text)
		}

		// Check second message
		secondMessage := mockAPI.messages[1]
		if secondMessage.ChatID != 123 {
			t.Errorf("Expected second message to be sent to chat ID 123, but was sent to %d", secondMessage.ChatID)
		}
		expectedContent = "Select property type(s):"
		if !strings.Contains(secondMessage.Text, expectedContent) {
			t.Errorf("Expected second message to contain '%s', but it didn't. Message: %s", expectedContent, secondMessage.Text)
		}
		if secondMessage.ReplyMarkup == nil {
			t.Error("Expected reply markup (keyboard) to be set for the second message, but it wasn't")
		}
	}

	state, exists := bot.state[123]
	if !exists {
		t.Error("Expected user state to be created, but it wasn't")
	} else if state.Stage != "awaiting_property_type" {
		t.Errorf("Expected user state stage to be 'awaiting_property_type', but got '%s'", state.Stage)
	}
}

// TestAskPropertyType tests the askPropertyType method
func TestAskPropertyType(t *testing.T) {
	mockAPI := &MockBotAPI3{}
	bot := &Bot{
		api:   mockAPI,
		state: make(map[int64]*UserState),
	}

	bot.askPropertyType(123)

	if len(mockAPI.messages) != 1 {
		t.Errorf("Expected 1 message to be sent, but got %d", len(mockAPI.messages))
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != 123 {
			t.Errorf("Expected message to be sent to chat ID 123, but was sent to %d", sentMessage.ChatID)
		}
		expectedContent := "Select property type(s):"
		if !strings.Contains(sentMessage.Text, expectedContent) {
			t.Errorf("Expected message to contain '%s', but it didn't. Message: %s", expectedContent, sentMessage.Text)
		}
		if sentMessage.ReplyMarkup == nil {
			t.Error("Expected reply markup (keyboard) to be set, but it wasn't")
		}
	}

	state, exists := bot.state[123]
	if !exists {
		t.Error("Expected user state to be created, but it wasn't")
	} else if state.Stage != "awaiting_property_type" {
		t.Errorf("Expected user state stage to be 'awaiting_property_type', but got '%s'", state.Stage)
	}
}

// TestAskPriceRange tests the askPriceRange method
func TestAskPriceRange(t *testing.T) {
	mockAPI := &MockBotAPI3{}
	bot := &Bot{
		api:   mockAPI,
		state: make(map[int64]*UserState),
	}

	bot.askPriceRange(123)

	if len(mockAPI.messages) != 1 {
		t.Errorf("Expected 1 message to be sent, but got %d", len(mockAPI.messages))
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != 123 {
			t.Errorf("Expected message to be sent to chat ID 123, but was sent to %d", sentMessage.ChatID)
		}
		expectedContent := "Let me know the price range for the monthly rent in GBP"
		if !strings.Contains(sentMessage.Text, expectedContent) {
			t.Errorf("Expected message to contain '%s', but it didn't. Message: %s", expectedContent, sentMessage.Text)
		}
	}

	state, exists := bot.state[123]
	if !exists {
		t.Error("Expected user state to be created, but it wasn't")
	} else if state.Stage != "awaiting_price_range" {
		t.Errorf("Expected user state stage to be 'awaiting_price_range', but got '%s'", state.Stage)
	}
}

// TestValidatePriceRange tests the validatePriceRange method
func TestValidatePriceRange(t *testing.T) {
	bot := &Bot{}

	// Test cases for price range validation
	testCases := []struct {
		input    string
		expected bool
	}{
		{"1000 - 2000", true},
		{"500-1500", true},
		{"1000", false},
		{"1000 - ", false},
		{" - 2000", false},
		{"abc - def", false},
		{"1000 - 500", true},
	}

	for _, tc := range testCases {
		result := bot.validatePriceRange(tc.input)
		if result != tc.expected {
			t.Errorf("validatePriceRange(%s) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}

// TestAskBedrooms tests the askBedrooms method
func TestAskBedrooms(t *testing.T) {
	mockAPI := &MockBotAPI3{}
	bot := &Bot{
		api:   mockAPI,
		state: make(map[int64]*UserState),
	}

	bot.askBedrooms(123)

	if len(mockAPI.messages) != 1 {
		t.Errorf("Expected 1 message to be sent, but got %d", len(mockAPI.messages))
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != 123 {
			t.Errorf("Expected message to be sent to chat ID 123, but was sent to %d", sentMessage.ChatID)
		}
		expectedContent := "Select the number of bedrooms"
		if !strings.Contains(sentMessage.Text, expectedContent) {
			t.Errorf("Expected message to contain '%s', but it didn't. Message: %s", expectedContent, sentMessage.Text)
		}
		if sentMessage.ReplyMarkup == nil {
			t.Error("Expected reply markup (keyboard) to be set, but it wasn't")
		}
	}

	state, exists := bot.state[123]
	if !exists {
		t.Error("Expected user state to be created, but it wasn't")
	} else if state.Stage != "awaiting_bedrooms" {
		t.Errorf("Expected user state stage to be 'awaiting_bedrooms', but got '%s'", state.Stage)
	}
}

// TestAskFurnished tests the askFurnished method
func TestAskFurnished(t *testing.T) {
	mockAPI := &MockBotAPI3{}
	bot := &Bot{
		api:   mockAPI,
		state: make(map[int64]*UserState),
	}

	bot.askFurnished(123)

	if len(mockAPI.messages) != 1 {
		t.Errorf("Expected 1 message to be sent, but got %d", len(mockAPI.messages))
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != 123 {
			t.Errorf("Expected message to be sent to chat ID 123, but was sent to %d", sentMessage.ChatID)
		}
		expectedContent := "Do you want to search for furnished or unfurnished accommodation?"
		if !strings.Contains(sentMessage.Text, expectedContent) {
			t.Errorf("Expected message to contain '%s', but it didn't. Message: %s", expectedContent, sentMessage.Text)
		}
		if sentMessage.ReplyMarkup == nil {
			t.Error("Expected reply markup (keyboard) to be set, but it wasn't")
		}
	}

	state, exists := bot.state[123]
	if !exists {
		t.Error("Expected user state to be created, but it wasn't")
	} else if state.Stage != "awaiting_furnished" {
		t.Errorf("Expected user state stage to be 'awaiting_furnished', but got '%s'", state.Stage)
	}
}

// TestAskLocation tests the askLocation method
func TestAskLocation(t *testing.T) {
	mockAPI := &MockBotAPI3{}
	bot := &Bot{
		api:   mockAPI,
		state: make(map[int64]*UserState),
	}

	bot.askLocation(123)

	if len(mockAPI.messages) != 1 {
		t.Errorf("Expected 1 message to be sent, but got %d", len(mockAPI.messages))
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != 123 {
			t.Errorf("Expected message to be sent to chat ID 123, but was sent to %d", sentMessage.ChatID)
		}
		expectedContent := "The bot is in testing mode, so the search area is restricted to Bath"
		if !strings.Contains(sentMessage.Text, expectedContent) {
			t.Errorf("Expected message to contain '%s', but it didn't. Message: %s", expectedContent, sentMessage.Text)
		}
		if sentMessage.ReplyMarkup == nil {
			t.Error("Expected reply markup (keyboard) to be set, but it wasn't")
		}
	}

	state, exists := bot.state[123]
	if !exists {
		t.Error("Expected user state to be created, but it wasn't")
	} else if state.Stage != "awaiting_location" {
		t.Errorf("Expected user state stage to be 'awaiting_location', but got '%s'", state.Stage)
	}
}
