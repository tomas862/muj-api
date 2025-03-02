package main

import (
    "database/sql"
    "flag"
    "fmt"
    "log"
    "muj/database"
    "muj/utils"
    "os"

    "github.com/thedatashed/xlsxreader"
)

// ParserConfig holds configuration for the parser
type ParserConfig struct {
    ParserType string
    FilePath   string
    ChunkSize  int
}

// Parser interface that all parsers must implement
type Parser interface {
    Initialize() error
    MapRow([]string) (interface{}, error)  // Maps Excel row to a specific entry type
    ProcessEntry(interface{}) error        // Performs any processing on an entry before it's added to the batch
    SaveEntries(db *sql.DB, entries []interface{}) (int, error)  // Saves a batch of entries to the database
}

func main() {
    // Parse command line arguments
    parserType := flag.String("type", "nomenclature", "Type of parser to use")
    filePath := flag.String("file", "", "Path to the file to parse. E.g ./files/nomenclatures/Nomenclature EN.xlsx")
    chunkSize := flag.Int("chunk", 1000, "Size of chunks to process")
    flag.Parse()

    // Validate arguments
    if *filePath == "" {
        fmt.Println("Error: File path is required")
        flag.Usage()
        os.Exit(1)
    }

    // Create parser configuration
    config := ParserConfig{
        ParserType: *parserType,
        FilePath:   *filePath,
        ChunkSize:  *chunkSize,
    }

    // Connect to database
    db, err := database.Connect()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Get the appropriate parser based on type
    parser, err := createParser(config.ParserType)
    if err != nil {
        log.Fatal(err)
    }

    // Initialize the parser
    if err := parser.Initialize(); err != nil {
        log.Fatal(err)
    }

    // Common file reading and chunking logic
    totalProcessed, totalInserted, totalErrors := readAndProcessFile(db, parser, config)

    // Print summary statistics
    fmt.Println("\n*** Import Summary ***")
    fmt.Printf("Parser type: %s\n", config.ParserType)
    fmt.Printf("File: %s\n", config.FilePath)
    fmt.Printf("Total rows processed: %d\n", totalProcessed)
    fmt.Printf("Total entries inserted/updated: %d\n", totalInserted)
    fmt.Printf("Total errors: %d\n", totalErrors)
    fmt.Println("Import process completed!")
}

// createParser returns the appropriate parser based on the type
func createParser(parserType string) (Parser, error) {
    switch parserType {
    case "nomenclature":
        return &NomenclatureParser{}, nil
    // Add more parser types here
    default:
        return nil, fmt.Errorf("unknown parser type: %s", parserType)
    }
}

// readAndProcessFile handles the common logic of reading an Excel file and processing entries
func readAndProcessFile(db *sql.DB, parser Parser, config ParserConfig) (int, int, int) {
    // Create an instance of the reader by opening the target file
    xl, err := xlsxreader.OpenFile(utils.GetAbsolutePath(config.FilePath))
    if err != nil {
        log.Fatal(err)
    }
    defer xl.Close()
    
    // Check if the sheet exists
    if len(xl.Sheets) == 0 {
        log.Fatal("No sheets found in the Excel file")
    }

    // Initialize counters and batch
    totalProcessed := 0
    totalInserted := 0
    totalErrors := 0
    rowCount := 0
    entries := make([]interface{}, 0, config.ChunkSize)
    firstRow := true
    rowNumber := 0

    // Process rows
    for row := range xl.ReadRows(xl.Sheets[0]) {
        rowNumber++
        
        // Skip the header row
        if firstRow {
            firstRow = false
            continue
        }

        // Initialize cells with empty strings and map by column
        cells := make([]string, 30) // Larger capacity for different parsers
        for _, cell := range row.Cells {
            colIndex := int(cell.Column[0] - 'A') // Convert column letter to index (A=0, B=1, etc.)
            if colIndex >= 0 && colIndex < len(cells) {
                cells[colIndex] = cell.Value
            }
        }

        // Parse the row using the specific parser
        entry, err := parser.MapRow(cells)
        if err != nil {
            log.Printf("Error parsing row %d: %v", rowNumber, err)
            totalErrors++
            continue
        }

        // Process the entry (e.g., calculate derived fields)
        if err := parser.ProcessEntry(entry); err != nil {
            log.Printf("Error processing row %d: %v", rowNumber, err)
            totalErrors++
            continue
        }

        entries = append(entries, entry)
        rowCount++
        totalProcessed++

        // Save in batches when we reach the chunk size
        if rowCount >= config.ChunkSize {
            insertedCount, err := parser.SaveEntries(db, entries)
            if err != nil {
                log.Fatalf("Failed to insert entries: %v", err)
            }
            totalInserted += insertedCount
            entries = entries[:0] // Clear the slice while keeping capacity
            rowCount = 0
        }
    }

    // Save any remaining entries
    if len(entries) > 0 {
        insertedCount, err := parser.SaveEntries(db, entries)
        if err != nil {
            log.Fatalf("Failed to insert entries: %v", err)
        }
        totalInserted += insertedCount
    }

    return totalProcessed, totalInserted, totalErrors
}