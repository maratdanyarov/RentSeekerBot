package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"imitation_project/internal/database"
	"strings"
	"testing"
	"time"
)

// MockBotAPI3 is a mock implementation of the BotAPI interface for testing purposes
type MockBotAPI3 struct {
	messages []tgbotapi.MessageConfig
}

// GetUpdatesChan mocks the method to get updates channel
func (m *MockBotAPI3) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return make(tgbotapi.UpdatesChannel)
}

// Send mocks the method to send messages and stores them for later verification
func (m *MockBotAPI3) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	msg, ok := c.(tgbotapi.MessageConfig)
	if ok {
		m.messages = append(m.messages, msg)
	}
	return tgbotapi.Message{}, nil
}

// Request mocks the method to make API requests
func (m *MockBotAPI3) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return &tgbotapi.APIResponse{}, nil
}

// TestHandleSearchCommand2 tests the handleSearchCommand function with existing preferences
func TestHandleSearchCommand2(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI3{}
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

	propertyTypes, _ := json.Marshal(map[string]bool{"Apartment": true})
	bedrooms, _ := json.Marshal(map[string]bool{"2": true})
	furnished, _ := json.Marshal(map[string]bool{"Furnished": true})
	lastSearch := time.Now()

	rows := sqlmock.NewRows([]string{"user_id", "property_type", "min_price", "max_price", "bedrooms", "furnished", "location", "last_search"}).
		AddRow(456, propertyTypes, 1000, 2000, bedrooms, furnished, "Bath", lastSearch)

	mock.ExpectQuery("SELECT (.+) FROM user_preferences").WithArgs(message.From.ID).WillReturnRows(rows)

	bot.handleSearchCommand(message)

	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != 123 {
			t.Errorf("Expected message to be sent to chat ID 123, but was sent to %d", sentMessage.ChatID)
		}
		if !strings.Contains(sentMessage.Text, "You have saved preferences") {
			t.Errorf("Expected message about existing preferences, got: %s", sentMessage.Text)
		}
	}
}

// TestPresentSearchResults tests the presentSearchResults function with a single property
func TestPresentSearchResults(t *testing.T) {
	mockAPI := &MockBotAPI3{}
	bot := &Bot{
		api: mockAPI,
	}

	properties := []database.Property{
		{
			ID:            1,
			Type:          "Apartment",
			PricePerMonth: 1500,
			Bedrooms:      2,
			Furnished:     true,
			Location:      "Bath",
			Description:   "Nice apartment",
			PhotoURLs:     []string{"http://example.com/photo1.jpg"},
			WebLink:       "http://example.com/property1",
		},
	}

	bot.presentSearchResults(123, properties)

	expectedMessageCount := 2 // Property details + final message
	if len(mockAPI.messages) != expectedMessageCount {
		t.Errorf("Expected %d messages to be sent, but got %d", expectedMessageCount, len(mockAPI.messages))
	}

	// Check the content of the property message
	propertyMessage := mockAPI.messages[0]
	expectedContent := []string{"Apartment", "£1500", "2 bedrooms", "Bath", "Furnished", "Nice apartment", "http://example.com/property1"}
	for _, content := range expectedContent {
		if !strings.Contains(propertyMessage.Text, content) {
			t.Errorf("Expected property message to contain '%s', but it didn't. Message: %s", content, propertyMessage.Text)
		}
	}

	// Check the content of the final message
	finalMessage := mockAPI.messages[1]
	expectedFinalContent := "That's all the properties I found matching your criteria. Would you like to start a new search?"
	if !strings.Contains(finalMessage.Text, expectedFinalContent) {
		t.Errorf("Expected final message to contain '%s', but it didn't. Message: %s", expectedFinalContent, finalMessage.Text)
	}
}

// TestHandleSearchCommandNoPreferences tests the handleSearchCommand function when no preferences are found
func TestHandleSearchCommandNoPreferences(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI3{}
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

	mock.ExpectQuery("SELECT (.+) FROM user_preferences").WithArgs(message.From.ID).WillReturnError(sql.ErrNoRows)

	bot.handleSearchCommand(message)

	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != 123 {
			t.Errorf("Expected message to be sent to chat ID 123, but was sent to %d", sentMessage.ChatID)
		}
		if !strings.Contains(sentMessage.Text, "Let's start your property search") {
			t.Errorf("Expected message about starting a new search, got: %s", sentMessage.Text)
		}
	}
}

// TestHandleSearchCommandDatabaseError tests the handleSearchCommand function when a database error occurs
func TestHandleSearchCommandDatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mockAPI := &MockBotAPI3{}
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

	mock.ExpectQuery("SELECT (.+) FROM user_preferences").WithArgs(message.From.ID).WillReturnError(errors.New("database error"))

	bot.handleSearchCommand(message)

	if len(mockAPI.messages) == 0 {
		t.Error("Expected a message to be sent, but none was")
	} else {
		sentMessage := mockAPI.messages[0]
		if sentMessage.ChatID != 123 {
			t.Errorf("Expected message to be sent to chat ID 123, but was sent to %d", sentMessage.ChatID)
		}
		expectedContent := "Let's start your property search"
		if !strings.Contains(sentMessage.Text, expectedContent) {
			t.Errorf("Expected message to contain '%s', but it didn't. Message: %s", expectedContent, sentMessage.Text)
		}
	}
}

// TestPresentSearchResultsMultipleProperties tests the presentSearchResults function with multiple properties
func TestPresentSearchResultsMultipleProperties(t *testing.T) {
	mockAPI := &MockBotAPI3{}
	bot := &Bot{
		api: mockAPI,
	}

	properties := []database.Property{
		{
			ID:            1,
			Type:          "Apartment",
			PricePerMonth: 1500,
			Bedrooms:      2,
			Furnished:     true,
			Location:      "Bath",
			Description:   "Nice apartment",
			PhotoURLs:     []string{"http://example.com/photo1.jpg"},
			WebLink:       "http://example.com/property1",
		},
		{
			ID:            2,
			Type:          "House",
			PricePerMonth: 2000,
			Bedrooms:      3,
			Furnished:     false,
			Location:      "Bath",
			Description:   "Spacious house",
			PhotoURLs:     []string{"http://example.com/photo2.jpg"},
			WebLink:       "http://example.com/property2",
		},
	}

	bot.presentSearchResults(123, properties)

	expectedMessageCount := 3 // 2 properties + 1 final message
	if len(mockAPI.messages) != expectedMessageCount {
		t.Errorf("Expected %d messages to be sent, but got %d", expectedMessageCount, len(mockAPI.messages))
		return
	}

	// Check content of property messages
	for i, propertyMessage := range mockAPI.messages[:2] {
		property := properties[i]
		expectedContent := []string{
			property.Type,
			fmt.Sprintf("£%d", property.PricePerMonth),
			fmt.Sprintf("%d bedrooms", property.Bedrooms),
			property.Location,
			property.Description,
			property.WebLink,
		}
		for _, content := range expectedContent {
			if !strings.Contains(propertyMessage.Text, content) {
				t.Errorf("Expected property message to contain '%s', but it didn't. Message: %s", content, propertyMessage.Text)
			}
		}
	}

	// Check final message
	finalMessage := mockAPI.messages[len(mockAPI.messages)-1]
	expectedFinalContent := "That's all the properties I found matching your criteria"
	if !strings.Contains(finalMessage.Text, expectedFinalContent) {
		t.Errorf("Expected final message to contain '%s', but it didn't. Message: %s", expectedFinalContent, finalMessage.Text)
	}
}

// TestPresentSearchResultsNoProperties tests the presentSearchResults function when no properties are found
func TestPresentSearchResultsNoProperties(t *testing.T) {
	mockAPI := &MockBotAPI3{}
	bot := &Bot{
		api: mockAPI,
	}

	properties := []database.Property{}

	bot.presentSearchResults(123, properties)

	if len(mockAPI.messages) != 1 {
		t.Errorf("Expected 1 message to be sent, but got %d", len(mockAPI.messages))
	}

	sentMessage := mockAPI.messages[0]
	expectedContent := "Sorry, no properties match your criteria"
	if !strings.Contains(sentMessage.Text, expectedContent) {
		t.Errorf("Expected message to contain '%s', but it didn't. Message: %s", expectedContent, sentMessage.Text)
	}
}
