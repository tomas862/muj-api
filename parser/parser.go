package main

import (
	"fmt"
	"log"
	"muj/database"
	"muj/utils"

	"github.com/thedatashed/xlsxreader"
)

func main() {
	// Connect to database using the database package
	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create an instance of the reader by opening a target file
    xl, err := xlsxreader.OpenFile(utils.GetAbsolutePath("./Nomenclature EN.xlsx"))
    if err != nil {
        log.Fatal(err)
    }
    defer xl.Close()

    // Check if the sheet exists
    if len(xl.Sheets) == 0 {
        log.Fatal("No sheets found in the Excel file")
    }

    // Iterate on the rows of data in chunks of 1000
    chunkSize := 1000
    rowCount := 0
    for row := range xl.ReadRows(xl.Sheets[0]) {
        fmt.Println(row)
        rowCount++
        if rowCount >= chunkSize {
            break
        }
    }
}
