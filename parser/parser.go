package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"muj/database"

	"github.com/joho/godotenv"
)

func main() {
	// Get the directory of the current file
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// Load environment variables from the same directory as this file
	if err := godotenv.Load(filepath.Join(dir, ".env")); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Connect to database using the database package
	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Successfully connected to database!")
}
