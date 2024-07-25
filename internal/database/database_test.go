// Package database provides functionality for interacting with the SQLite database.
package database

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

// TestInitDB tests the InitDB function to ensure it correctly initializes the database
// and creates the necessary tables.
func TestInitDB(t *testing.T) {
	// Use a temporary file for the test database
	tmpfile, err := os.CreateTemp("", "test.db")
	if err != nil {
		t.Fatalf("Could not create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Initialize the database
	db, err := InitDB(tmpfile.Name())
	if err != nil {
		t.Fatalf("InitDB() failed: %v", err)
	}
	defer db.Close()

	// Check if tables were created
	tables := []string{"properties", "user_preferences"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s was not created", table)
		}
	}
}

// TestAddAndGetProperty tests the AddProperty and GetProperties functions to ensure
// they correctly add a property to the database and retrieve it.
func TestAddAndGetProperty(t *testing.T) {
	// Set up test database
	tmpfile, _ := os.CreateTemp("", "test.db")
	defer os.Remove(tmpfile.Name())
	db, _ := InitDB(tmpfile.Name())
	defer db.Close()

	// Test property
	testProp := Property{
		Type:          "Flat",
		PricePerMonth: 1000,
		Bedrooms:      2,
		Furnished:     true,
		Location:      "Bath",
		Description:   "A nice flat",
		PhotoURLs:     []string{"http://example.com/photo.jpg"},
		WebLink:       "http://example.com/property",
	}

	// Add property
	err := AddProperty(db, testProp)
	if err != nil {
		t.Fatalf("AddProperty() failed: %v", err)
	}

	// Get property
	filters := map[string]interface{}{
		"types":     []string{"Flat"},
		"min_price": 500,
		"max_price": 1500,
		"bedrooms":  []int{2},
		"location":  "Bath",
	}
	properties, err := GetProperties(db, filters)
	if err != nil {
		t.Fatalf("GetProperties() failed: %v", err)
	}

	if len(properties) != 1 {
		t.Fatalf("Expected 1 property, got %d", len(properties))
	}

	// Compare retrieved property with the original
	retrievedProp := properties[0]
	if retrievedProp.Type != testProp.Type ||
		retrievedProp.PricePerMonth != testProp.PricePerMonth ||
		retrievedProp.Bedrooms != testProp.Bedrooms ||
		retrievedProp.Furnished != testProp.Furnished ||
		retrievedProp.Location != testProp.Location ||
		retrievedProp.Description != testProp.Description ||
		retrievedProp.PhotoURLs[0] != testProp.PhotoURLs[0] ||
		retrievedProp.WebLink != testProp.WebLink {
		t.Errorf("Retrieved property does not match added property")
	}
}

// TestUserPreferences tests the SaveUserPreferences, GetUserPreferences, and DeleteUserPreferences
// functions to ensure they correctly handle user preferences in the database.
func TestUserPreferences(t *testing.T) {
	// Set up test database
	tmpfile, _ := os.CreateTemp("", "test.db")
	defer os.Remove(tmpfile.Name())
	db, _ := InitDB(tmpfile.Name())
	defer db.Close()

	// Test user preferences
	testPrefs := UserPreferences{
		UserID:         123,
		PropertyTypes:  map[string]bool{"Flat": true, "House": false},
		BedroomOptions: map[string]bool{"2": true, "3": true},
		MinPrice:       1000,
		MaxPrice:       2000,
		Location:       "Bath",
		Furnished:      map[string]bool{"Furnished": true},
		LastSearch:     time.Now(),
	}

	// Save preferences
	err := SaveUserPreferences(db, testPrefs)
	if err != nil {
		t.Fatalf("SaveUserPreferences() failed: %v", err)
	}

	// Get preferences
	retrievedPrefs, err := GetUserPreferences(db, 123)
	if err != nil {
		t.Fatalf("GetUserPreferences() failed: %v", err)
	}

	// Compare retrieved preferences with the original
	if retrievedPrefs.UserID != testPrefs.UserID ||
		!mapEqual(retrievedPrefs.PropertyTypes, testPrefs.PropertyTypes) ||
		!mapEqual(retrievedPrefs.BedroomOptions, testPrefs.BedroomOptions) ||
		retrievedPrefs.MinPrice != testPrefs.MinPrice ||
		retrievedPrefs.MaxPrice != testPrefs.MaxPrice ||
		retrievedPrefs.Location != testPrefs.Location ||
		!mapEqual(retrievedPrefs.Furnished, testPrefs.Furnished) {
		t.Errorf("Retrieved preferences do not match saved preferences")
	}

	// Delete preferences
	err = DeleteUserPreferences(db, 123)
	if err != nil {
		t.Fatalf("DeleteUserPreferences() failed: %v", err)
	}

	// Try to get deleted preferences
	_, err = GetUserPreferences(db, 123)
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

// mapEqual is a helper function to compare two maps of string to bool.
func mapEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}
	return true
}
