// Package bot provides the core functionality for the Telegram bot.
package bot

import (
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestGetSelectedOptions tests the getSelectedOptions function to ensure it correctly
// filters and returns selected options from a map.
func TestGetSelectedOptions(t *testing.T) {
	testCases := []struct {
		name     string
		options  map[string]bool
		expected []string
	}{
		{
			name:     "AllSelected",
			options:  map[string]bool{"A": true, "B": true, "C": true},
			expected: []string{"A", "B", "C"},
		},
		{
			name:     "SomeSelected",
			options:  map[string]bool{"A": true, "B": false, "C": true},
			expected: []string{"A", "C"},
		},
		{
			name:     "NoneSelected",
			options:  map[string]bool{"A": false, "B": false, "C": false},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getSelectedOptions(tc.options)
			// Sort both slices before comparing
			sort.Strings(result)
			sort.Strings(tc.expected)
			if !equalStringSlices(result, tc.expected) {
				t.Errorf("getSelectedOptions() = %v; want %v", result, tc.expected)
			}
		})
	}
}

// TestGetSelectedBedroomOptions tests the getSelectedBedroomOptions function to ensure
// it correctly converts selected bedroom options to integer values.
func TestGetSelectedBedroomOptions(t *testing.T) {
	testCases := []struct {
		name     string
		options  map[string]bool
		expected []int
	}{
		{
			name:     "MixedOptions",
			options:  map[string]bool{"Studio": true, "1": true, "2": false, "3": true},
			expected: []int{0, 1, 3},
		},
		{
			name:     "OnlyStudio",
			options:  map[string]bool{"Studio": true, "1": false, "2": false},
			expected: []int{0},
		},
		{
			name:     "NoSelection",
			options:  map[string]bool{"Studio": false, "1": false, "2": false},
			expected: []int{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getSelectedBedroomOptions(tc.options)
			// Sort both slices before comparing
			sort.Ints(result)
			sort.Ints(tc.expected)
			if !equalIntSlices(result, tc.expected) {
				t.Errorf("getSelectedBedroomOptions() = %v; want %v", result, tc.expected)
			}
		})
	}
}

// TestParsePriceRange tests the parsePriceRange function to ensure it correctly
// parses and returns the minimum and maximum prices from a string input.
func TestParsePriceRange(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectedMin int
		expectedMax int
	}{
		{"ValidRange", "500-1000", 500, 1000},
		{"ReversedRange", "1000-500", 500, 1000},
		{"SingleNumber", "500", 0, 1000000},
		{"InvalidFormat", "abc-def", 0, 1000000},
		{"EmptyString", "", 0, 1000000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			min, max := parsePriceRange(tc.input)
			// Check if the returned min and max match the expected values
			if min != tc.expectedMin || max != tc.expectedMax {
				t.Errorf("parsePriceRange(%s) = (%d, %d); want (%d, %d)",
					tc.input, min, max, tc.expectedMin, tc.expectedMax)
			}
		})
	}
}

// equalIntSlices is a helper function to compare two slices of integers,
// treating nil slices and empty slices as equal.
func equalIntSlices(a, b []int) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// equalStringSlices is a helper function to compare two slices of strings,
// treating nil slices and empty slices as equal.
func equalStringSlices(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBuildFilters(t *testing.T) {
	bot := &Bot{} // Create a minimal Bot instance for testing

	testCases := []struct {
		name        string
		preferences *SearchPreferences
		expected    map[string]interface{}
	}{
		{
			name: "AllPreferences",
			preferences: &SearchPreferences{
				PropertyTypes:    map[string]bool{"Flat": true, "House": false},
				BedroomOptions:   map[string]bool{"1": true, "2": true, "Studio": false},
				FurnishedOptions: map[string]bool{"Furnished": true, "Unfurnished": false},
				PriceRange:       "500-1000",
				Location:         "Bath",
			},
			expected: map[string]interface{}{
				"types":     []string{"Flat"},
				"bedrooms":  []int{1, 2},
				"min_price": 500,
				"max_price": 1000,
				"location":  "Bath",
			},
		},
		{
			name: "NoPreferences",
			preferences: &SearchPreferences{
				PropertyTypes:    map[string]bool{},
				BedroomOptions:   map[string]bool{},
				FurnishedOptions: map[string]bool{},
				PriceRange:       "",
				Location:         "Bath",
			},
			expected: map[string]interface{}{
				"location": "Bath",
			},
		},
		// Add more test cases here
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := bot.buildFilters(tc.preferences)

			// Compare the result with the expected output
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("buildFilters() = %v; want %v", result, tc.expected)
			}
		})
	}
}

func TestGetUserState(t *testing.T) {
	bot := &Bot{
		state: make(map[int64]*UserState),
	}

	testCases := []struct {
		name     string
		userID   int64
		expected string
	}{
		{"NewUser", 123, "initial"},
		{"ExistingUser", 456, "custom"},
	}

	// Set up an existing user
	bot.state[456] = &UserState{Stage: "custom"}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := bot.getUserState(tc.userID)
			if state.Stage != tc.expected {
				t.Errorf("getUserState(%d).Stage = %s; want %s", tc.userID, state.Stage, tc.expected)
			}
		})
	}
}

func TestUpdateUserState(t *testing.T) {
	bot := &Bot{
		state: make(map[int64]*UserState),
	}

	userID := int64(123)
	newState := &UserState{Stage: "updated"}

	bot.updateUserState(userID, newState)

	if bot.state[userID] != newState {
		t.Errorf("updateUserState() did not update the state correctly")
	}
}

// TestValidatePriceRange tests the validatePriceRange function
func TestValidatePriceRange(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"ValidRange", "500-1000", true},
		{"InvalidFormat", "500_1000", false},
		{"SingleNumber", "500", false},
		{"EmptyString", "", false},
		{"NonNumeric", "abc-def", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validatePriceRange(tc.input)
			if result != tc.expected {
				t.Errorf("validatePriceRange(%s) = %v; want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func validatePriceRange(input string) bool {
	parts := strings.Split(input, "-")
	if len(parts) != 2 {
		return false
	}

	for _, part := range parts {
		if _, err := strconv.Atoi(strings.TrimSpace(part)); err != nil {
			return false
		}
	}

	return true
}
