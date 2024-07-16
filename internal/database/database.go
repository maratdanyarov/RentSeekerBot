package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/url"
	"strings"
	"time"
)

// Property represents a rental property with its attributes.
type Property struct {
	ID            int
	Type          string
	PricePerMonth int
	Bedrooms      int
	Furnished     bool
	Location      string
	Description   string
	PhotoURLs     []string
	WebLink       string
}

// UserPreferences represents a user's search preferences for rental properties.
type UserPreferences struct {
	UserID         int64
	PropertyTypes  map[string]bool
	BedroomOptions map[string]bool
	MinPrice       int
	MaxPrice       int
	Location       string
	Furnished      map[string]bool
	LastSearch     time.Time
}

var db *sql.DB // Global database connection

// InitDB initializes the database connection and creates necessary tables.
func InitDB(dbPath string) (*sql.DB, error) {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create the properties table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS properties (
		    id INTEGER PRIMARY KEY AUTOINCREMENT,
		    type TEXT,
			price_per_month INTEGER,
			bedrooms INTEGER,
			furnished BOOLEAN,
			location TEXT,
			description TEXT,
			photo_urls TEXT,
            web_link TEXT
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create the user_preferences table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_preferences (
		    user_id INTEGER PRIMARY KEY,
		    property_type TEXT,
		    min_price INTEGER,
		    max_price INTEGER,
		    bedrooms INTEGER,
		    furnished BOOLEAN,
		    location TEXT,
		    last_search TIMESTAMP
		)
	`)

	if err != nil {
		return nil, err
	}
	return db, nil
}

// AddProperty inserts a new property into the database.
func AddProperty(db *sql.DB, p Property) error {
	validPhotoURLs := validatePhotoURLs(p.PhotoURLs)
	photoURLsJSON, err := json.Marshal(validPhotoURLs)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
        INSERT INTO properties (type, price_per_month, bedrooms, furnished, location, description, photo_urls, web_link)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?)
    `, p.Type, p.PricePerMonth, p.Bedrooms, p.Furnished, p.Location, p.Description, string(photoURLsJSON), p.WebLink)

	return err
}

// validatePhotoURLs filters out invalid URLs from the given slice.
func validatePhotoURLs(urls []string) []string {
	var validURLs []string
	for _, u := range urls {
		if isValidURL(u) {
			validURLs = append(validURLs, u)
		}
	}
	return validURLs
}

// isValidURL checks if the given string is a valid URL.
func isValidURL(urlString string) bool {
	u, err := url.Parse(urlString)
	return err == nil && u.Scheme != "" && u.Host != "" &&
		(strings.HasPrefix(u.Scheme, "http") || strings.HasPrefix(u.Scheme, "https"))
}

// GetProperties retrieves properties from the database based on the provided filters.
// It constructs a dynamic SQL query to apply the filters and returns matching properties.
func GetProperties(db *sql.DB, filters map[string]interface{}) ([]Property, error) {
	query := "SELECT id, type, price_per_month, bedrooms, furnished, location, description, photo_urls, web_link FROM properties WHERE 1=1"
	var args []interface{}

	if v, ok := filters["types"].([]string); ok && len(v) > 0 {
		placeholders := make([]string, len(v))
		for i, t := range v {
			placeholders[i] = "LOWER(type) = LOWER(?)"
			args = append(args, t)
		}
		query += " AND (" + strings.Join(placeholders, " OR ") + ")"
	}

	if v, ok := filters["min_price"].(int); ok {
		query += " AND price_per_month >= ?"
		args = append(args, v)
	}
	if v, ok := filters["max_price"].(int); ok {
		query += " AND price_per_month <= ?"
		args = append(args, v)
	}
	if v, ok := filters["bedrooms"].([]int); ok && len(v) > 0 {
		placeholders := make([]string, len(v))
		for i, b := range v {
			placeholders[i] = "?"
			args = append(args, b)
		}
		query += " AND bedrooms IN (" + strings.Join(placeholders, ",") + ")"
	}
	if v, ok := filters["location"].(string); ok {
		query += " AND LOWER(location) = LOWER(?)"
		args = append(args, v)
	}
	if v, ok := filters["furnished"].(bool); ok {
		query += " AND furnished = ?"
		args = append(args, v)
	}

	log.Printf("Executing query: %s with args: %v", query, args)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	var properties []Property
	for rows.Next() {
		var p Property
		var photoURLsJSON string
		err := rows.Scan(&p.ID, &p.Type, &p.PricePerMonth, &p.Bedrooms, &p.Furnished, &p.Location, &p.Description, &photoURLsJSON, &p.WebLink)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		err = json.Unmarshal([]byte(photoURLsJSON), &p.PhotoURLs)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling photo URLs: %w", err)
		}

		properties = append(properties, p)
	}

	return properties, nil
}

// CloseDB closes the database connection.
func CloseDB() {
	if db != nil {
		db.Close()
	}
}

// UpdateExistingDB updates the existing database schema if necessary.
func UpdateExistingDB(dbPath string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Check and add photo_urls column if it doesn't exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('properties') WHERE name='photo_urls'").Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = db.Exec("ALTER TABLE properties ADD COLUMN photo_urls TEXT")
		if err != nil {
			return err
		}
	}

	// Check and add web_link column if it doesn't exist
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('properties') WHERE name='web_link'").Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = db.Exec("ALTER TABLE properties ADD COLUMN web_link TEXT")
		if err != nil {
			return err
		}
	}

	return nil
}

// SaveUserPreferences saves or updates a user's search preferences in the database.
func SaveUserPreferences(db *sql.DB, prefs UserPreferences) error {
	propertyTypesJSON, err := json.Marshal(prefs.PropertyTypes)
	if err != nil {
		return err
	}
	bedroomOptionsJSON, err := json.Marshal(prefs.BedroomOptions)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT OR REPLACE INTO user_preferences
		(user_id, property_type, min_price, max_price, bedrooms, furnished, location, last_search)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, prefs.UserID, propertyTypesJSON, prefs.MinPrice, prefs.MaxPrice, bedroomOptionsJSON, prefs.Furnished, prefs.Location, time.Now())
	return err
}

// GetUserPreferences retrieves a user's search preferences from the database.
func GetUserPreferences(db *sql.DB, userID int64) (UserPreferences, error) {
	var prefs UserPreferences
	var propertyTypesJSON, bedroomOptionsJSON []byte
	err := db.QueryRow(`
		SELECT user_id, property_type, min_price, max_price, bedrooms, furnished, location, last_search
		FROM user_preferences WHERE user_id = ?
	`, userID).Scan(&prefs.UserID, &propertyTypesJSON, &prefs.MinPrice, &prefs.MaxPrice, &bedroomOptionsJSON, &prefs.Furnished, &prefs.Location, &prefs.LastSearch)
	if err != nil {
		return prefs, err
	}

	err = json.Unmarshal(propertyTypesJSON, &prefs.PropertyTypes)
	if err != nil {
		return prefs, err
	}

	err = json.Unmarshal(bedroomOptionsJSON, &prefs.BedroomOptions)
	if err != nil {
		return prefs, err
	}

	return prefs, nil
}

// DeleteUserPreferences removes a user's search preferences from the database.
func DeleteUserPreferences(db *sql.DB, userID int64) error {
	_, err := db.Exec("DELETE FROM user_preferences WHERE user_id = ?", userID)
	return err
}
