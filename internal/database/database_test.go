package database

import (
	"database/sql"
	"os"
	"testing"
)

var testDB *sql.DB

// TestMain sets up the test environment by initializing an in-memory database,
// running all tests, and then closing the database connection.
func TestMain(m *testing.M) {

	var err error
	testDB, err = InitDB(":memory:")
	if err != nil {
		panic(err)
	}

	code := m.Run()

	testDB.Close()

	os.Exit(code)
}

// TestInitDB verifies that the database initialization creates the expected tables.
func TestInitDB(t *testing.T) {
	// InitDB is already called in TestMain, so we just need to check if tables exist
	var count int
	err := testDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('properties', 'user_preferences', 'saved_listings')").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query tables: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 tables, got %d", count)
	}
}

// TestAddAndGetProperty tests the addition of properties to the database
// and their subsequent retrieval using various filters.
func TestAddAndGetProperty(t *testing.T) {
	prop := Property{
		Type:          "Apartment",
		PricePerMonth: 1000,
		Bedrooms:      2,
		Furnished:     true,
		Location:      "Test Location",
		Description:   "Test Description",
		PhotoURLs:     []string{"http://example.com/photo1.jpg", "http://example.com/photo2.jpg"},
		WebLink:       "http://example.com/property",
	}

	err := AddProperty(testDB, prop)
	if err != nil {
		t.Fatalf("Failed to add property: %v", err)
	}

	// Retrieve the property
	properties, err := GetProperties(testDB, map[string]interface{}{
		"types": []string{"Apartment"},
	})
	if err != nil {
		t.Fatalf("Failed to get properties: %v", err)
	}

	if len(properties) != 1 {
		t.Fatalf("Expected 1 property, got %d", len(properties))
	}

	retrievedProp := properties[0]
	if retrievedProp.Type != prop.Type || retrievedProp.PricePerMonth != prop.PricePerMonth {
		t.Errorf("Retrieved property does not match added property")
	}

	// Test updating existing preferences
	updatedPrefs := UserPreferences{
		UserID: 12345,
		PropertyTypes: map[string]bool{
			"Apartment": false,
			"House":     true,
		},
		BedroomOptions: map[string]bool{
			"2": true,
			"3": true,
		},
		MinPrice: 1000,
		MaxPrice: 2000,
		Location: "Updated City",
		Furnished: map[string]bool{
			"Unfurnished": true,
		},
	}

	err = SaveUserPreferences(testDB, updatedPrefs)
	if err != nil {
		t.Fatalf("Failed to update user preferences: %v", err)
	}

	retrievedUpdatedPrefs, err := GetUserPreferences(testDB, 12345)
	if err != nil {
		t.Fatalf("Failed to get updated user preferences: %v", err)
	}

	if retrievedUpdatedPrefs.Location != "Updated City" || retrievedUpdatedPrefs.MinPrice != 1000 {
		t.Errorf("Retrieved updated preferences do not match")
	}
}

// TestSaveAndGetUserPreferences tests saving user preferences to the database,
// retrieving them, and updating existing preferences.
func TestSaveAndGetUserPreferences(t *testing.T) {
	prefs := UserPreferences{
		UserID: 12345,
		PropertyTypes: map[string]bool{
			"Apartment": true,
			"House":     false,
		},
		BedroomOptions: map[string]bool{
			"1": true,
			"2": true,
		},
		MinPrice: 500,
		MaxPrice: 1500,
		Location: "Test City",
		Furnished: map[string]bool{
			"Furnished": true,
		},
	}

	err := SaveUserPreferences(testDB, prefs)
	if err != nil {
		t.Fatalf("Failed to save user preferences: %v", err)
	}

	retrievedPrefs, err := GetUserPreferences(testDB, 12345)
	if err != nil {
		t.Fatalf("Failed to get user preferences: %v", err)
	}

	if retrievedPrefs.UserID != prefs.UserID || retrievedPrefs.MinPrice != prefs.MinPrice {
		t.Errorf("Retrieved preferences do not match saved preferences")
	}

	// Test saving duplicate listing
	err = SaveListing(testDB, 12345, 1)
	if err != nil {
		t.Fatalf("Failed to save duplicate listing: %v", err)
	}

	savedListings, err := GetSavedListings(testDB, 12345)
	if err != nil {
		t.Fatalf("Failed to get saved listings after duplicate save: %v", err)
	}

	if len(savedListings) != 1 {
		t.Errorf("Expected 1 saved listing after duplicate save, got %d", len(savedListings))
	}
}

// TestDeleteUserPreferences verifies that user preferences can be successfully
// deleted from the database.
func TestDeleteUserPreferences(t *testing.T) {
	userID := int64(12345)

	err := DeleteUserPreferences(testDB, userID)
	if err != nil {
		t.Fatalf("Failed to delete user preferences: %v", err)
	}

	_, err = GetUserPreferences(testDB, userID)
	if err == nil {
		t.Errorf("Expected error when getting deleted preferences, got nil")
	}
}

// TestSaveAndGetSavedListings tests the functionality of saving property listings
// for a user and retrieving those saved listings, including handling of duplicates.
func TestSaveAndGetSavedListings(t *testing.T) {
	userID := int64(12345)
	propertyID := 1

	err := SaveListing(testDB, userID, propertyID)
	if err != nil {
		t.Fatalf("Failed to save listing: %v", err)
	}

	savedListings, err := GetSavedListings(testDB, userID)
	if err != nil {
		t.Fatalf("Failed to get saved listings: %v", err)
	}

	if len(savedListings) != 1 {
		t.Errorf("Expected 1 saved listing, got %d", len(savedListings))
	}
}

// TestDeleteSavedListing verifies that a saved listing can be successfully
// deleted from the database.
func TestDeleteSavedListing(t *testing.T) {
	userID := int64(12345)
	propertyID := 1

	err := DeleteSavedListing(testDB, userID, propertyID)
	if err != nil {
		t.Fatalf("Failed to delete saved listing: %v", err)
	}

	savedListings, err := GetSavedListings(testDB, userID)
	if err != nil {
		t.Fatalf("Failed to get saved listings: %v", err)
	}

	if len(savedListings) != 0 {
		t.Errorf("Expected 0 saved listings after deletion, got %d", len(savedListings))
	}
}

// TestUpdateExistingDB checks if the database schema can be updated correctly,
// specifically testing the addition of new columns to existing tables.
func TestUpdateExistingDB(t *testing.T) {
	// Create a temporary database file
	tmpfile, err := os.CreateTemp("", "testdb")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Initialize the database
	db, err := InitDB(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer db.Close()

	// Update the existing database
	err = UpdateExistingDB(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to update existing database: %v", err)
	}

	// Verify that the new columns exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('properties') WHERE name IN ('photo_urls', 'web_link')").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query new columns: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 new columns, got %d", count)
	}
}

// TestValidatePhotoURLs tests the function that filters and validates photo URLs,
// ensuring only valid URLs are retained.
func TestValidatePhotoURLs(t *testing.T) {
	urls := []string{
		"http://example.com/photo1.jpg",
		"https://example.com/photo2.png",
		"ftp://invalid-url",
		"not-a-url",
	}

	validURLs := validatePhotoURLs(urls)

	if len(validURLs) != 2 {
		t.Errorf("Expected 2 valid URLs, got %d", len(validURLs))
	}

	if validURLs[0] != urls[0] || validURLs[1] != urls[1] {
		t.Errorf("Validated URLs do not match expected valid URLs")
	}
}

// TestIsValidURL tests the isValidURL function with various input URLsgit
func TestIsValidURL(t *testing.T) {
	testCases := []struct {
		url      string
		expected bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"ftp://example.com", false},
		{"not-a-url", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := isValidURL(tc.url)
		if result != tc.expected {
			t.Errorf("isValidURL(%s) = %v; want %v", tc.url, result, tc.expected)
		}
	}
}
