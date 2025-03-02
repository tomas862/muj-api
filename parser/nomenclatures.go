package main

import (
    "database/sql"
    "fmt"
    "strconv"
    "strings"
    "time"
)

// NomenclatureEntry represents a single row from the nomenclature Excel file
type NomenclatureEntry struct {
    GoodsCode      string     // Goods code
    StartDate      time.Time  // Start date
    EndDate        *time.Time // End date (pointer because this field might be empty)
    Language       string     // Language
    HierPos        int        // Hier. Pos.
    HierarchyPath  string     // Hier. Path
    Indent         int        // Indent
    Description    string     // Description
    DescrStartDate time.Time  // Descr. start date
}

// NomenclatureParser implements the Parser interface for nomenclature files
type NomenclatureParser struct {
    BaseExcelParser // Embed the BaseExcelParser to inherit ReadRows
}

// Initialize prepares the parser for processing
func (p *NomenclatureParser) Initialize() error {
    // Any nomenclature specific initialization
    return nil
}

// MapRow now expects an ExcelRow and converts it to a NomenclatureEntry
func (p *NomenclatureParser) MapRow(rowData RowData) (interface{}, error) {
    row, ok := rowData.(ExcelRow)
    if !ok {
        return nil, fmt.Errorf("expected ExcelRow, got %T", rowData)
    }
    
    cells := row.Cells
    if len(cells) < 8 {
        return nil, fmt.Errorf("insufficient columns: need at least 8 columns")
    }
    
    entry := NomenclatureEntry{}
    entry.GoodsCode = cells[0]
    
    // Parse dates - handle empty dates
    if cells[1] != "" {
        startDate, err := time.Parse("02-01-2006", cells[1])
        if err != nil {
            return nil, fmt.Errorf("invalid start date format: %v", err)
        }
        entry.StartDate = startDate
    }

    if cells[2] != "" {
        endDate, err := time.Parse("02-01-2006", cells[2])
        if err != nil {
            return nil, fmt.Errorf("invalid end date format: %v", err)
        }
        entry.EndDate = &endDate
    }
    
    entry.Language = cells[3]

    // Parse hier_pos
    hierPos, err := strconv.Atoi(cells[4])
    if err != nil {
        hierPosFloat, floatErr := strconv.ParseFloat(cells[4], 64)
        if floatErr != nil {
            return nil, fmt.Errorf("invalid Hier. Pos. format: %v", err)
        }
        hierPos = int(hierPosFloat)
    }
    
    // Validate hier_pos
    if hierPos != 2 && hierPos != 4 && hierPos != 6 && hierPos != 8 && hierPos != 10 {
        return nil, fmt.Errorf("invalid Hier. Pos. value: %d", hierPos)
    }
    entry.HierPos = hierPos

    // Parse indent
    indent := countDashes(cells[5])
    if indent < 0 || indent > 12 {
        return nil, fmt.Errorf("invalid Indent value: %d", indent)
    }
    entry.Indent = indent

    entry.Description = cells[6]

    // Parse description start date
    if cells[7] != "" {
        descrStartDate, err := time.Parse("02-01-2006", cells[7])
        if err != nil {
            return nil, fmt.Errorf("invalid description start date format: %v", err)
        }
        entry.DescrStartDate = descrStartDate
    }

    return entry, nil
}

// ProcessEntry calculates additional fields for a nomenclature entry
func (p *NomenclatureParser) ProcessEntry(entryInterface interface{}) error {
    entry, ok := entryInterface.(*NomenclatureEntry)
    if !ok {
        entryValue := entryInterface.(NomenclatureEntry)
        entry = &entryValue
    }
    
    // Calculate hierarchy path
    hierPath, err := getHierarchyPath(entry.GoodsCode, entry.HierPos)
    if err != nil {
        return fmt.Errorf("error getting hierarchy path: %v", err)
    }
    entry.HierarchyPath = hierPath
    
    return nil
}

// SaveEntries saves a batch of nomenclature entries to the database
func (p *NomenclatureParser) SaveEntries(db *sql.DB, entriesInterface []interface{}) (int, error) {
    // Convert generic entries to NomenclatureEntry
    entries := make([]NomenclatureEntry, len(entriesInterface))
    for i, e := range entriesInterface {
        switch entry := e.(type) {
        case NomenclatureEntry:
            entries[i] = entry
        case *NomenclatureEntry:
            entries[i] = *entry
        default:
            return 0, fmt.Errorf("invalid entry type at index %d: %T", i, e)
        }
    }
    
    // Use the existing insertEntries function
    return insertEntries(db, entries)
}

func countDashes(s string) int {
    if s == "" {
        return 0
    }
    
    // Split by spaces to get individual dashes
    parts := strings.Split(strings.TrimSpace(s), " ")
    return len(parts)
}

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

// Reuse the insertEntries function from the original code
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
        ON CONFLICT (goods_code) 
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