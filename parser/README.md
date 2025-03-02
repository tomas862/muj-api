# Parser

responsible for nomenclature related data parsing and storing to database

## Running parser

go run . -type=nomenclature -file=./files/nomenclatures/Nomenclature\ LT.xlsx -chunk=1000

## To create new parser

```go
package main

import (
    "database/sql"
    "fmt"
    // Import other packages as needed
)

// MeasuresEntry represents a single row from the measures Excel file
type MeasuresEntry struct {
    // Define fields specific to measures entries
}

// MeasuresParser implements the Parser interface for measures files
type MeasuresParser struct{}

// Implement all required interface methods
func (p *MeasuresParser) Initialize() error {
    // Initialization specific to measures parser
    return nil
}

func (p *MeasuresParser) MapRow(cells []string) (interface{}, error) {
    // Map Excel row to MeasuresEntry
    return MeasuresEntry{}, nil
}

func (p *MeasuresParser) ProcessEntry(entryInterface interface{}) error {
    // Process a measures entry before adding to batch
    return nil
}

func (p *MeasuresParser) SaveEntries(db *sql.DB, entriesInterface []interface{}) (int, error) {
    // Save measures entries to database
    return 0, nil
}
```

### use it in main.go

```go
func createParser(parserType string) (Parser, error) {
    switch parserType {
    case "nomenclature":
        return &NomenclatureParser{}, nil
    case "measures":
        return &MeasuresParser{}, nil
    default:
        return nil, fmt.Errorf("unknown parser type: %s", parserType)
    }
}
```
