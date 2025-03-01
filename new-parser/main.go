package main

import (
	"database/sql"
	"fmt"
	"log"
	"muj/database"
	"muj/utils"
	"strconv"
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
	HierPos          int    // Hier. Pos.
	HierarchyPath         string    // Hier. Path
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
    xl, err := xlsxreader.OpenFile(utils.GetAbsolutePath("./files/nomenclatures/Nomenclature EN.xlsx"))
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
    
    // Add counters
    totalProcessed := 0
    totalInserted := 0
    totalErrors := 0

	// Add rowNumber counter
    rowNumber := 0

	for row := range xl.ReadRows(xl.Sheets[0]) {
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
            totalErrors++
            continue
        }

		hierPath, err := getHierarchyPath(entry.GoodsCode, entry.HierPos)

        if err != nil {
            log.Printf("Error getting hierarchy path for row %d: %v", rowNumber, err)
            totalErrors++
            continue
        }

		entry.HierarchyPath = hierPath
		entries = append(entries, entry)
        
        rowCount++
        totalProcessed++

		if rowCount >= chunkSize {
            insertedCount, err := insertEntries(db, entries)
            if err != nil {
                log.Fatalf("Failed to insert entries: %v", err)
            }
            totalInserted += insertedCount
            entries = []NomenclatureEntry{}
            rowCount = 0
        }
	}

	    fmt.Printf("Parsed %d entries\n", len(entries))

    // After processing all entries in main()
    if len(entries) > 0 {
        insertedCount, err := insertEntries(db, entries)
        if err != nil {
            log.Fatalf("Failed to insert entries: %v", err)
        }
        totalInserted += insertedCount
    }
    
    // Print summary statistics
    fmt.Println("\n*** Import Summary ***")
    fmt.Printf("Total rows processed: %d\n", totalProcessed)
    fmt.Printf("Total entries inserted/updated: %d\n", totalInserted)
    fmt.Printf("Total errors: %d\n", totalErrors)
    fmt.Println("Import process completed!")
}

// getCategoryKey returns the category key based on the goods code and hier_pos
func getHierarchyPath(goodsCode string, hierPos int) (string, error) {
    if goodsCode == "" {
        return goodsCode, fmt.Errorf("invalid goods code")
    }
    
    if hierPos <= 0 || hierPos > 10 || hierPos%2 != 0 {
        return "", fmt.Errorf("invalid hierarchy position: %d", hierPos)
    }
    
    if len(goodsCode) < hierPos {
        return "", fmt.Errorf("goods code '%s' too short for hierarchy position %d", goodsCode, hierPos)
    }
    
    var result strings.Builder
    
    // Iterate through the code in steps of 2 up to hierPos
    for i := 2; i <= hierPos; i += 2 {
        if i > 2 {
            result.WriteString(".")
        }
        result.WriteString(goodsCode[:i])
    }
    
    return result.String(), nil
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

		// Parse hier_pos
        hierPos, err := strconv.Atoi(row[4])
        if err != nil {
            hierPosFloat, floatErr := strconv.ParseFloat(row[4], 64)
            if floatErr != nil {
                return entry, fmt.Errorf("invalid Hier. Pos. format: %v", err)
            }
            hierPos = int(hierPosFloat)
        }
        
        // Validate hier_pos
        if hierPos != 2 && hierPos != 4 && hierPos != 6 && hierPos != 8 && hierPos != 10 {
            return entry, fmt.Errorf("invalid Hier. Pos. value: %d", hierPos)
        }
        entry.HierPos = hierPos

        // Parse indent
        indent := countDashes(row[5])
        if indent < 0 || indent > 12 {
            return entry, fmt.Errorf("invalid Indent value: %d", indent)
        }
        entry.Indent = indent

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

func insertEntries(db *sql.DB, entries []NomenclatureEntry) (int, error) {
    tx, err := db.Begin()
    if err != nil {
        return 0, fmt.Errorf("failed to begin transaction: %v", err)
    }
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()
    
    // Prepare statements for both tables
    itemStmt, err := tx.Prepare(`
        INSERT INTO nomenclatures 
        (goods_code, start_date, end_date, hierarchy_path, indent) 
        VALUES ($1, $2, $3, $4::ltree, $5)
        ON CONFLICT (goods_code, start_date, end_date) 
        DO UPDATE SET hierarchy_path = $4::ltree, indent = $5, updated_at = NOW()
        RETURNING id
    `)

    if err != nil {
        tx.Rollback()
        return 0, fmt.Errorf("failed to prepare item statement: %v", err)
    }
    defer itemStmt.Close()
    
    descStmt, err := tx.Prepare(`
        INSERT INTO nomenclature_descriptions
        (nomenclature_id, language, description, descr_start_date)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (nomenclature_id, language)
        DO UPDATE SET description = $3, descr_start_date = $4, updated_at = NOW()
    `)
    if err != nil {
        tx.Rollback()
        return 0, fmt.Errorf("failed to prepare description statement: %v", err)
    }
    defer descStmt.Close()
    
    // Track successful inserts/updates
    successCount := 0
    
    // Process entries
    for _, entry := range entries {
        // Insert into nomenclature_items and get the ID
        var itemID int
        err = itemStmt.QueryRow(
            entry.GoodsCode,
            entry.StartDate,
            entry.EndDate,
            entry.HierarchyPath,
            entry.Indent,
        ).Scan(&itemID)
        
        if err != nil {
            tx.Rollback()
            return successCount, fmt.Errorf("failed to insert item %s: %v", entry.GoodsCode, err)
        }
        
        // Insert into nomenclature_descriptions
        _, err = descStmt.Exec(
            itemID,
            entry.Language,
            entry.Description,
            entry.DescrStartDate,
        )
        
        if err != nil {
            tx.Rollback()
            return successCount, fmt.Errorf("failed to insert description for item %s: %v", entry.GoodsCode, err)
        }
        
        successCount++
    }
    
    err = tx.Commit()
    if err != nil {
        return 0, err
    }
    
    return successCount, nil
}