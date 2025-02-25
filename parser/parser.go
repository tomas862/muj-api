package main

import (
	"fmt"
	"log"
	"muj/database"
)

func main() {
	// Connect to database using the database package
	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Successfully connected to database!")
}
