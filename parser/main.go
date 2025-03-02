package main

import (
    "database/sql"
    "flag"
    "fmt"
    "log"
    "muj/database"
)

// ParserConfig holds configuration for the parser
type ParserConfig struct {
	ParserType string // Type of parser to use
	FilePath   string // Optional path to the file to parse (can be empty if parser uses other data sources)
	ChunkSize  int    // Size of chunks to process
}

// RowData represents a single row of data from any source
type RowData interface{}

// Parser interface that all parsers must implement
type Parser interface {
    ReadRows(config ParserConfig) (<-chan RowData, error)  // Returns a channel of rows from the data source
    MapRow(RowData) (interface{}, error)  // Maps row data to a specific entry type
    ProcessEntry(*interface{}) error        // Performs any processing on an entry before it's added to the batch
    SaveEntries(db *sql.DB, entries []interface{}) (int, error)  // Saves a batch of entries to the database
}

func main() {
    // Parse command line arguments
    parserType := flag.String("type", "nomenclature", "Type of parser to use")
    filePath := flag.String("file", "", "Path to the file to parse. E.g ./files/nomenclatures/Nomenclature EN.xlsx")
    chunkSize := flag.Int("chunk", 1000, "Size of chunks to process")
    flag.Parse()

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

    // Common file reading and chunking logic
    totalProcessed, totalInserted, totalErrors := readAndProcessFile(db, parser, config)

    // Print summary statistics
    fmt.Println("\n*** Import Summary ***")
    fmt.Printf("Parser type: %s\n", config.ParserType)
    fmt.Printf("Path: %s\n", config.FilePath)
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
    case "declarable_codes":
		return &DeclarableCodesParser{}, nil
    default:
        return nil, fmt.Errorf("unknown parser type: %s", parserType)
    }
}

// readAndProcessFile handles the common logic of reading data and processing entries
func readAndProcessFile(db *sql.DB, parser Parser, config ParserConfig) (int, int, int) {
    // Initialize counters and batch
    totalProcessed := 0
    totalInserted := 0
    totalErrors := 0
    rowCount := 0
    entries := make([]interface{}, 0, config.ChunkSize)
    rowNumber := 0

    // Get channel of rows from the parser
    rowsChan, err := parser.ReadRows(config)
    if err != nil {
        log.Fatalf("Failed to read file: %v", err)
    }

    // Process rows
    for row := range rowsChan {
        rowNumber++

        // Parse the row using the specific parser
        entry, err := parser.MapRow(row)
        if err != nil {
            log.Printf("Error parsing row %d: %v", rowNumber, err)
            totalErrors++
            continue
        }

        // Process the entry (e.g., calculate derived fields)
        if err := parser.ProcessEntry(&entry); err != nil {
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