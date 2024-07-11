package database

import (
	"database/sql"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	"net/url"
	"strings"
)

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

var db *sql.DB

func InitDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
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
		return err
	}

	return nil
}

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

func validatePhotoURLs(urls []string) []string {
	var validURLs []string
	for _, u := range urls {
		if isValidURL(u) {
			validURLs = append(validURLs, u)
		}
	}
	return validURLs
}

func isValidURL(urlString string) bool {
	u, err := url.Parse(urlString)
	return err == nil && u.Scheme != "" && u.Host != "" &&
		(strings.HasPrefix(u.Scheme, "http") || strings.HasPrefix(u.Scheme, "https"))
}

func GetProperties(db *sql.DB, filters map[string]interface{}) ([]Property, error) {
	query := "SELECT id, type, price_per_month, bedrooms, furnished, location, description, photo_urls, web_link FROM properties WHERE 1=1"
	var args []interface{}

	if v, ok := filters["type"]; ok {
		query += " AND type = ?"
		args = append(args, v)
	}
	if v, ok := filters["min_price"]; ok {
		query += " AND price_per_month >= ?"
		args = append(args, v)
	}
	if v, ok := filters["max_price"]; ok {
		query += "AND price_per_month <= ?"
		args = append(args, v)
	}
	if v, ok := filters["bedrooms"]; ok {
		query += " AND bedrooms = ?"
		args = append(args, v)
	}
	if v, ok := filters["location"]; ok {
		query += " AND location LIKE ?"
		args = append(args, "%"+v.(string)+"%")
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var properties []Property
	for rows.Next() {
		var p Property
		var photoURLsJSON string
		err := rows.Scan(&p.ID, &p.Type, &p.PricePerMonth, &p.Bedrooms, &p.Furnished, &p.Location, &p.Description, &photoURLsJSON, &p.WebLink)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(photoURLsJSON), &p.PhotoURLs)
		if err != nil {
			return nil, err
		}

		properties = append(properties, p)
	}

	return properties, nil
}

func CloseDB() {
	if db != nil {
		db.Close()
	}
}

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
