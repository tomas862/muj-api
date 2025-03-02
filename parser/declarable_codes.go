package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/lib/pq"
)


type DeclarableCodesEntry struct {
	GoodsCode      string     // 10-digit goods codes + 2-digit suffix;
	StartDate	  time.Time     // Validity start date of the nomenclature code;
	DeclStartDate time.Time     // Validity start date of the declarable code;
	Is_Leaf		  bool // Declarable codes in a customs declaration: "0" = non-declarable code; "1" = declarable code in customs.
}

// NomenclatureParser implements the Parser interface for nomenclature files
type DeclarableCodesParser struct {
    BaseExcelParser // Embed the BaseExcelParser to inherit ReadRows
}

func (p *DeclarableCodesParser) MapRow(rowData RowData) (interface{}, error) {
	row, ok := rowData.(ExcelRow)
    if !ok {
        return nil, fmt.Errorf("expected ExcelRow, got %T", rowData)
    }
    
    cells := row.Cells
    if len(cells) < 4 {
        return nil, fmt.Errorf("insufficient columns: need at least 4 columns")
    }

	entry := DeclarableCodesEntry{}
    entry.GoodsCode = cells[0]

    startDate, err := time.Parse("2006-01-02", cells[1])
    if err != nil {
        return nil, fmt.Errorf("invalid start date: %v", err)
    }

    entry.StartDate = startDate

    declStartDate, err := time.Parse("2006-01-02", cells[2])
    if err != nil {	
        return nil, fmt.Errorf("invalid declarable start date: %v", err)
    }

    entry.DeclStartDate = declStartDate

	isLeaf, err := strconv.ParseBool(cells[3])

	if err != nil {
		return nil, fmt.Errorf("invalid is leaf: %v", err)
	}

	entry.Is_Leaf = isLeaf

	return entry, nil
}

func (p *DeclarableCodesParser) ProcessEntry(entry *interface{}) error {
	// No processing needed for Declarable codes parser
	return nil
}

func (p *DeclarableCodesParser) SaveEntries(db *sql.DB, entriesInterface []interface{}) (int, error) {
	// Convert generic entries to NomenclatureEntry
    entries := make([]DeclarableCodesEntry, len(entriesInterface))
    for i, e := range entriesInterface {
        switch entry := e.(type) {
        case DeclarableCodesEntry:
            entries[i] = entry
        case *DeclarableCodesEntry:
            entries[i] = *entry
        default:
            return 0, fmt.Errorf("invalid entry type at index %d: %T", i, e)
        }
    }
    
    // Use the existing insertEntries function
    return insertDeclarableEntries(db, entries)
}

func insertDeclarableEntries(db *sql.DB, entries []DeclarableCodesEntry) (int, error) {
    tx, err := db.Begin()
    if err != nil {
        return 0, fmt.Errorf("failed to begin transaction: %v", err)
    }
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

	// Extract all goods codes into a slice
    goodsCodes := make([]string, len(entries))
    for i, entry := range entries {
        goodsCodes[i] = entry.GoodsCode
    }

	// Query existing nomenclature entries for these goods codes
    // Use a proper placeholder for each item in the list
    query := fmt.Sprintf(`
        SELECT id, goods_code FROM nomenclatures 
        WHERE goods_code = ANY($1)
    `)
    
    rows, err := tx.Query(query, pq.Array(goodsCodes))
    if err != nil {
        tx.Rollback()
        return 0, fmt.Errorf("failed to query existing nomenclatures: %v", err)
    }
    defer rows.Close()

	// Create a map of goods_code to ID for quick lookup
    existingCodes := make(map[string]int)
    for rows.Next() {
        var id int
        var code string
        if err := rows.Scan(&id, &code); err != nil {
            tx.Rollback()
            return 0, fmt.Errorf("failed to scan row: %v", err)
        }
        existingCodes[code] = id
    }

    // Prepare statements for both table
    itemStmt, err := tx.Prepare(`
        INSERT INTO nomenclature_declarable_codes
        (nomenclature_id, start_date, declarable_start_date, is_leaf, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
        ON CONFLICT (nomenclature_id) DO UPDATE SET start_date = $2, declarable_start_date = $3, is_leaf = $4, updated_at = NOW()
        RETURNING id
    `)

    if err != nil {
        tx.Rollback()
        return 0, fmt.Errorf("failed to prepare item statement: %v", err)
    }

    defer itemStmt.Close()

    // Track successful inserts/updates
    successCount := 0
    
    // Process entries
    for _, entry := range entries {
        // Get the nomenclature ID from the map
        nomenclatureID, exists := existingCodes[entry.GoodsCode]
        if !exists {
            log.Printf("no nomenclature found for goods code: %s", entry.GoodsCode)
            continue
        }
        
        // Insert into nomenclature_declarable_codes and get the ID
        var itemID int
        err = itemStmt.QueryRow(
            nomenclatureID,
            entry.StartDate,
            entry.DeclStartDate,
            entry.Is_Leaf,
        ).Scan(&itemID)
        
        if err != nil {
            tx.Rollback()
            return successCount, fmt.Errorf("failed to insert item %s: %v", entry.GoodsCode, err)
        }
        
        successCount++
    }
    
    err = tx.Commit()
    if err != nil {
        return 0, err
    }
    
    return successCount, nil
}



