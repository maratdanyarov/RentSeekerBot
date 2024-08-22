package bot

import (
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

// TestSearchProperties tests the searchProperties function with a basic scenario
func TestSearchProperties(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	bot := &Bot{
		db: db,
	}

	preferences := &SearchPreferences{
		PropertyTypes: map[string]bool{"Apartment": true},
		PriceRange:    "1000-2000",
		Location:      "Bath",
	}

	rows := sqlmock.NewRows([]string{"id", "type", "price_per_month", "bedrooms", "furnished", "location", "description", "photo_urls", "web_link"}).
		AddRow(1, "Apartment", 1500, 2, true, "Bath", "Nice apartment", "[]", "http://example.com")

	mock.ExpectQuery("SELECT (.+) FROM properties").WillReturnRows(rows)

	properties, err := bot.searchProperties(preferences)
	if err != nil {
		t.Errorf("searchProperties() returned an error: %v", err)
	}

	if len(properties) != 1 {
		t.Errorf("searchProperties() returned %d properties, want 1", len(properties))
	}

}

// TestBuildFilters tests the buildFilters function
func TestBuildFilters(t *testing.T) {
	bot := &Bot{}

	preferences := &SearchPreferences{
		PropertyTypes:    map[string]bool{"Apartment": true, "House": false},
		BedroomOptions:   map[string]bool{"2": true, "3": true},
		PriceRange:       "1000-2000",
		Location:         "Bath",
		FurnishedOptions: map[string]bool{"Furnished": true},
	}

	filters := bot.buildFilters(preferences)

	if len(filters) != 6 {
		t.Errorf("buildFilters() returned %d filters, want 6", len(filters))
	}

	// Check if all expected filters are present
	expectedFilters := map[string]bool{
		"types":     true,
		"bedrooms":  true,
		"min_price": true,
		"max_price": true,
		"location":  true,
		"furnished": true,
	}

	for key := range expectedFilters {
		if _, exists := filters[key]; !exists {
			t.Errorf("Expected filter '%s' not found in result", key)
		}
	}

	// Verify specific filter values
	if types, ok := filters["types"].([]string); !ok || len(types) != 1 || types[0] != "Apartment" {
		t.Errorf("Unexpected value for 'types' filter: %v", filters["types"])
	}

	if bedrooms, ok := filters["bedrooms"].([]int); !ok || len(bedrooms) != 2 || (bedrooms[0] != 2 && bedrooms[1] != 3) {
		t.Errorf("Unexpected value for 'bedrooms' filter: %v", filters["bedrooms"])
	}

	if minPrice, ok := filters["min_price"].(int); !ok || minPrice != 1000 {
		t.Errorf("Unexpected value for 'min_price' filter: %v", filters["min_price"])
	}

	if maxPrice, ok := filters["max_price"].(int); !ok || maxPrice != 2000 {
		t.Errorf("Unexpected value for 'max_price' filter: %v", filters["max_price"])
	}

	if location, ok := filters["location"].(string); !ok || location != "Bath" {
		t.Errorf("Unexpected value for 'location' filter: %v", filters["location"])
	}

	if furnished, ok := filters["furnished"].(bool); !ok || !furnished {
		t.Errorf("Unexpected value for 'furnished' filter: %v", filters["furnished"])
	}
}

// TestSearchPropertiesNoResults tests the searchProperties function when no results are found
func TestSearchPropertiesNoResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	bot := &Bot{
		db: db,
	}

	preferences := &SearchPreferences{
		PropertyTypes: map[string]bool{"Mansion": true},
		PriceRange:    "1000000-2000000",
		Location:      "Bath",
	}

	// Initial query
	mock.ExpectQuery("SELECT (.+) FROM properties").WillReturnRows(sqlmock.NewRows([]string{"id", "type", "price_per_month", "bedrooms", "furnished", "location", "description", "photo_urls", "web_link"}))

	// Queries for each relaxation step
	relaxationSteps := []string{"bedrooms", "types", "price", "furnished"}
	for range relaxationSteps {
		mock.ExpectQuery("SELECT (.+) FROM properties").WillReturnRows(sqlmock.NewRows([]string{"id", "type", "price_per_month", "bedrooms", "furnished", "location", "description", "photo_urls", "web_link"}))
	}

	properties, err := bot.searchProperties(preferences)
	if err != nil {
		t.Errorf("searchProperties() returned an error: %v", err)
	}

	if len(properties) != 0 {
		t.Errorf("searchProperties() returned %d properties, want 0", len(properties))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// TestSearchPropertiesWithRelaxation tests the searchProperties function with filter relaxation
func TestSearchPropertiesWithRelaxation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	bot := &Bot{
		db: db,
	}

	preferences := &SearchPreferences{
		PropertyTypes:    map[string]bool{"Apartment": true},
		BedroomOptions:   map[string]bool{"2": true},
		PriceRange:       "1000-2000",
		Location:         "Bath",
		FurnishedOptions: map[string]bool{"Furnished": true},
	}

	// First query returns no results
	mock.ExpectQuery("SELECT (.+) FROM properties").WillReturnRows(sqlmock.NewRows([]string{"id", "type", "price_per_month", "bedrooms", "furnished", "location", "description", "photo_urls", "web_link"}))

	// Second query (with relaxed filters) returns a result
	rows := sqlmock.NewRows([]string{"id", "type", "price_per_month", "bedrooms", "furnished", "location", "description", "photo_urls", "web_link"}).
		AddRow(1, "Apartment", 2500, 2, true, "Bath", "Nice apartment", "[]", "http://example.com")
	mock.ExpectQuery("SELECT (.+) FROM properties").WillReturnRows(rows)

	properties, err := bot.searchProperties(preferences)
	if err != nil {
		t.Errorf("searchProperties() returned an error: %v", err)
	}

	if len(properties) != 1 {
		t.Errorf("searchProperties() returned %d properties, want 1", len(properties))
	}
}

// TestSearchPropertiesError tests the searchProperties function when a database error occurs
func TestSearchPropertiesError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	bot := &Bot{
		db: db,
	}

	preferences := &SearchPreferences{
		PropertyTypes: map[string]bool{"Apartment": true},
		PriceRange:    "1000-2000",
		Location:      "Bath",
	}

	mock.ExpectQuery("SELECT (.+) FROM properties").WillReturnError(errors.New("database error"))

	_, err = bot.searchProperties(preferences)
	if err == nil {
		t.Error("searchProperties() did not return an error when one was expected")
	}
}

// TestGetSelectedOptions tests the getSelectedOptions function
func TestGetSelectedOptions(t *testing.T) {
	options := map[string]bool{
		"Option1": true,
		"Option2": false,
		"Option3": true,
	}

	selected := getSelectedOptions(options)

	if len(selected) != 2 {
		t.Errorf("getSelectedOptions() returned %d options, want 2", len(selected))
	}

	expectedOptions := []string{"Option1", "Option3"}
	for _, option := range expectedOptions {
		if !contains(selected, option) {
			t.Errorf("getSelectedOptions() did not return expected option: %s", option)
		}
	}
}

// TestGetSelectedBedroomOptions tests the getSelectedBedroomOptions function
func TestGetSelectedBedroomOptions(t *testing.T) {
	options := map[string]bool{
		"Studio": true,
		"1":      false,
		"2":      true,
		"3+":     true,
	}

	selected := getSelectedBedroomOptions(options)

	if len(selected) != 3 {
		t.Errorf("getSelectedBedroomOptions() returned %d options, want 3", len(selected))
	}

	expectedOptions := []int{0, 2, 3}
	for _, option := range expectedOptions {
		if !containsInt(selected, option) {
			t.Errorf("getSelectedBedroomOptions() did not return expected option: %d", option)
		}
	}
}

// TestParsePriceRange tests the parsePriceRange function with various inputs
func TestParsePriceRange(t *testing.T) {
	testCases := []struct {
		input   string
		minWant int
		maxWant int
	}{
		{"1000-2000", 1000, 2000}, // Normal case
		{"500-1500", 500, 1500},   // Normal case
		{"0-1000", 0, 1000},       // Zero minimum
		{"invalid", 0, 1000000},   // Invalid input
		{"1000", 0, 1000000},      // Missing maximum
		{"1000-", 1000, 1000000},  // Missing maximum
		{"-2000", 0, 2000},        // Missing minimum
	}

	for _, tc := range testCases {
		minGot, maxGot := parsePriceRange(tc.input)
		if minGot != tc.minWant || maxGot != tc.maxWant {
			t.Errorf("parsePriceRange(%s) = (%d, %d), want (%d, %d)", tc.input, minGot, maxGot, tc.minWant, tc.maxWant)
		}
	}
}

// contains is a helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// containsInt is a helper function to check if a slice contains an int
func containsInt(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
