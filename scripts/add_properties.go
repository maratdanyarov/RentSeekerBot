package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"imitation_project/internal/database"
	"os"
	"strconv"
	"strings"
)

func main() {
	db, err := sql.Open("sqlite3", "properties.db")
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	err = database.InitDB("properties.db")
	if err != nil {
		fmt.Printf("Error initialising database: %v\\n", err)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		var p database.Property

		fmt.Print("Enter property type (flat/house): ")
		p.Type, _ = reader.ReadString('\n')
		p.Type = strings.TrimSpace(p.Type)

		fmt.Print("Enter price per month: ")
		priceStr, _ := reader.ReadString('\n')
		p.PricePerMonth, _ = strconv.Atoi(strings.TrimSpace(priceStr))

		fmt.Print("Enter number of bedrooms: ")
		bedroomsStr, _ := reader.ReadString('\n')
		p.Bedrooms, _ = strconv.Atoi(strings.TrimSpace(bedroomsStr))

		fmt.Print("Is it furnished? (true/false): ")
		furnishedStr, _ := reader.ReadString('\n')
		p.Furnished, _ = strconv.ParseBool(strings.TrimSpace(furnishedStr))

		fmt.Print("Enter location: ")
		p.Location, _ = reader.ReadString('\n')
		p.Location = strings.TrimSpace(p.Location)

		fmt.Print("Enter description: ")
		p.Description, _ = reader.ReadString('\n')
		p.Description = strings.TrimSpace(p.Description)

		fmt.Print("Enter web link to property listing: ")
		p.WebLink, _ = reader.ReadString('\n')
		p.WebLink = strings.TrimSpace(p.WebLink)

		fmt.Println("Enter photo URLs (one per line, empty line to finish):")
		for {
			photoURL, _ := reader.ReadString('\n')
			photoURL = strings.TrimSpace(photoURL)
			if photoURL == "" {
				break
			}
			p.PhotoURLs = append(p.PhotoURLs, photoURL)
		}

		err := database.AddProperty(db, p)
		if err != nil {
			fmt.Printf("Error adding property: %v\n", err)
		} else {
			fmt.Println("Property added successfully!")
		}

		fmt.Print("Add another property? (y/n): ")
		answer, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			break
		}
	}

}
