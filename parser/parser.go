package main

import (
	"fmt"
	"log"
	"muj/database"
	"muj/utils"
	"strings"
	"time"

	"github.com/thedatashed/xlsxreader"
)

// NomenclatureEntry represents a single row from the nomenclature Excel file
type NomenclatureEntry struct {
	GoodsCode        string    // Goods code
	StartDate        time.Time // Start date
	EndDate          *time.Time // End date (pointer because this field might be empty)
	Language         string    // Language
	HierPos          string    // Hier. Pos.
	Indent           int       // Indent
	Description      string    // Description
	DescrStartDate   time.Time // Descr. start date
}

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
	entries := []NomenclatureEntry{}
	firstRow := true
	
	// Add rowNumber counter
	rowNumber := 0
	for row := range xl.ReadRows(xl.Sheets[0]) {
		rowNumber++ // Increment row counter
		
		// Skip the header row
		if firstRow {
			firstRow = false
			continue
		}

		// Initialize cells with empty strings
		cells := make([]string, 8)

		// Map cells based on their column letters
		for _, cell := range row.Cells {
			switch cell.Column {
			case "A": // Goods code
				cells[0] = cell.Value
			case "B": // Start date
				cells[1] = cell.Value
			case "C": // End date
				cells[2] = cell.Value
			case "D": // Language
				cells[3] = cell.Value
			case "E": // Hier. Pos.
				cells[4] = cell.Value
			case "F": // Indent
				cells[5] = cell.Value
			case "G": // Description
				cells[6] = cell.Value
			case "H": // Descr. start date
				cells[7] = cell.Value
			}
		}

		entry, err := parseNomenclatureRow(cells)
		if err != nil {
			log.Printf("Error parsing row %d: %v", rowNumber, err)
			continue
		}
		
		entries = append(entries, entry)
		fmt.Printf("Parsed: row %d %+v\n", rowNumber, entry) // Print the structured data
		
		rowCount++
		if rowCount >= chunkSize {
			// break
		}
	}
	
	fmt.Printf("Parsed %d entries\n", len(entries))
}

// Add this helper function to count dashes
func countDashes(s string) int {
    if s == "" {
        return 0
    }
    
    // Split by spaces to get individual dashes
    parts := strings.Split(strings.TrimSpace(s), " ")
    return len(parts)
}

// parseNomenclatureRow converts a row of Excel data into a structured NomenclatureEntry
func parseNomenclatureRow(row []string) (NomenclatureEntry, error) {
	entry := NomenclatureEntry{}
	
	// This is a basic implementation. You'll need to adapt this based on the exact
	// structure of your Excel file and handle possible parsing errors properly.
	if len(row) == 8 {
		entry.GoodsCode = row[0]
		
		// Parse dates - handle empty dates
		if row[1] != "" {
			startDate, err := time.Parse("02-01-2006", row[1])
			if err != nil {
				return entry, fmt.Errorf("invalid start date format: %v", err)
			}
			entry.StartDate = startDate
		}
		
		if row[2] != "" {
			endDate, err := time.Parse("02-01-2006", row[2])
			if err != nil {
				return entry, fmt.Errorf("invalid end date format: %v", err)
			}
			entry.EndDate = &endDate
		}
		
		entry.Language = row[3]
		entry.HierPos = row[4]
		entry.Indent = countDashes(row[5])
		entry.Description = row[6]
		
		// Parse description start date
		if row[7] != "" {
			descrStartDate, err := time.Parse("02-01-2006", row[7])
			if err != nil {
				return entry, fmt.Errorf("invalid description start date format: %v", err)
			}
			entry.DescrStartDate = descrStartDate
		}
	} else {
		return entry, fmt.Errorf("row does not have 8 columns")
	}
	
	return entry, nil
}
