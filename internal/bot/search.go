// Package bot provides the core functionality for the Telegram bot.
package bot

import (
	"fmt"
	"imitation_project/internal/database"
	"log"
	"strconv"
	"strings"
)

// searchProperties performs a property search based on the given preferences.
// It builds filters, executes the search, and applies filter relaxation if no results are found.
func (b *Bot) searchProperties(preferences *SearchPreferences) ([]database.Property, error) {
	filters := b.buildFilters(preferences)
	log.Printf("Initial search with filters: %+v", filters)

	properties, err := database.GetProperties(b.db, filters)
	if err != nil {
		return nil, fmt.Errorf("error searching properties: %w", err)
	}

	if len(properties) == 0 {
		log.Println("No properties found, relaxing filters")
		relaxationSteps := []string{"bedrooms", "types", "price", "furnished"}

		for _, step := range relaxationSteps {
			delete(filters, step)
			log.Printf("Relaxing %s filter. New filters: %+v", step, filters)

			properties, err = database.GetProperties(b.db, filters)
			if err != nil {
				return nil, fmt.Errorf("error searching properties with relaxed filters: %w", err)
			}

			if len(properties) > 0 {
				break
			}
		}
	}

	log.Printf("Found %d properties", len(properties))
	return properties, nil
}

// buildFilters constructs a map of filters based on the given search preferences.
func (b *Bot) buildFilters(preferences *SearchPreferences) map[string]interface{} {
	filters := make(map[string]interface{})

	if types := getSelectedOptions(preferences.PropertyTypes); len(types) > 0 {
		filters["types"] = types
	}

	if bedrooms := getSelectedBedroomOptions(preferences.BedroomOptions); len(bedrooms) > 0 {
		filters["bedrooms"] = bedrooms
	}

	if preferences.PriceRange != "" {
		min, max := parsePriceRange(preferences.PriceRange)
		if min > 0 {
			filters["min_price"] = min
		}
		if max > 0 {
			filters["max_price"] = max
		}
	}

	if furnishedOptions := getSelectedOptions(preferences.FurnishedOptions); len(furnishedOptions) > 0 {
		if len(furnishedOptions) == 1 {
			filters["furnished"] = furnishedOptions[0] == "Furnished"
		}
	}

	filters["location"] = "Bath"

	return filters
}

// getSelectedOptions returns a slice of strings for options that are selected (true) in the given map.
func getSelectedOptions(options map[string]bool) []string {
	var selected []string
	for option, isSelected := range options {
		if isSelected {
			selected = append(selected, option)
		}
	}
	return selected
}

// getSelectedBedroomOptions converts selected bedroom options to a slice of integers.
// It handles the "Studio" option as 0 and parses other options as integers.
func getSelectedBedroomOptions(options map[string]bool) []int {
	var selected []int
	for option, isSelected := range options {
		if isSelected {
			if option == "Studio" {
				selected = append(selected, 0) // Use 0 for studio apartments
			} else {
				if num, err := strconv.Atoi(option); err == nil {
					selected = append(selected, num)
				}
			}
		}
	}
	return selected
}

// parsePriceRange converts a price range string (e.g., "500-1000") to minimum and maximum integer values.
// If parsing fails, it returns default values of 0 for min and 1000000 for max.
func parsePriceRange(priceRange string) (int, int) {
	parts := strings.Split(priceRange, "-")
	if len(parts) != 2 {
		return 0, 1000000
	}

	// Trim spaces and parse to integers
	min, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		min = 0
	}
	max, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		max = 1000000
	}

	if min > max {
		min, max = max, min
	}

	return min, max
}
