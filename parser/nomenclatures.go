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
	/**
	    The goods code is a structured 10-digit code, the first six digits of which contain the code defined by
	 	the Harmonised Commodity Description and Coding System (HS). The first two digits of the HS
		code represent the Chapter. There are 99 Chapters grouped according to material and use. The HS
		codes are revised every 5 years.
		Chapter 77 is currently unused and reserved for future use.
		Chapter 99 contains special Combined Nomenclature codes that are used for certain specific
		movements of goods.
		The HS codes are broken down at 8-digit level into the Combined Nomenclature (CN). The
		Combined Nomenclature is revised and published every year based on the HS codes.
		As the Combined Nomenclature is not sufficiently detailed to support the Union tariff and
		commercial legislation, the 8-digit codes can be broken down into 10-digit codes called "TARIC
		codes". These codes are created at any time, according to the legislative needs.
		The structure is therefore the following:
		- Chapter (digits 1-2);
		- HS (1-6);
		- CN (7-8);
		- TARIC (9-10).
		In the TARIC database, the goods codes are suffixed by a 2-digit code called the product line suffix
		(10, 20, 30… or 80). The product line suffix is a technical code that is necessary to build the
		structure of the goods code nomenclature in a proper sequence.
		If the suffix is different than "80", this means the goods code is an intermediary code that only serves
		as a heading for sub-products; those codes are not declarable codes in Customs.
		If the suffix is "80", this means that the goods code represent actual classified goods or groups of
		goods. This does not mean per se that the goods code can be declared in the SAD or in the EUCDM.
		A goods code can only be declared if the suffix is "80" and if it is not broken down into goods codes
		of lower level.
	*/
    GoodsCode      string     // 10-digit goods codes + 2-digit suffix;
    StartDate      time.Time  // Validity start date of the codes;
    EndDate        *time.Time // Validity end date of the codes (can be empty);
    Language       string     // Language codes of the descriptions;
	/**
	    Hierarchical level of the code.
		The codes in the TARIC database are always 10 digit long because they are padded with pairs
		of zeroes (00). The zero-padding has no effect on the level of the codes. The level of the code
		is defined by the right-most pair of digits which is different than 00.
		Example
		0702 00 00 00 (tomatoes) is of level 4.
		0702 00 00 07 (cherry tomatoes) is of level 10.
		The product line suffix is ignored to determine the level of a code.
	*/
    HierPos        int
    HierarchyPath  string     // Postgres Ltree type constructed hierarchy from goods code for easier access of the date later on.
	/**
		The indentation of the description in the nomenclature, represented by a number of dashes
		(indents). The indentation of goods can evolve independently of the goods itself if the goods
		are moved in the hierarchical structure without being redefined.;
	*/
    Indent         int
	/**
	    Description of the codes in all TARIC languages for the description period (column H).
		Goods codes are associated to description periods that define a period of time during which
		the description of the goods remains unchanged. A unique description period is defined for all
		languages
	*/ 
    Description    string
    DescrStartDate time.Time  // Description start date: first day of validity of the description period.
}

// NomenclatureParser implements the Parser interface for nomenclature files
type NomenclatureParser struct {
    BaseExcelParser // Embed the BaseExcelParser to inherit ReadRows
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

    entry.HierPos = hierPos

    // Parse indent
    indent := countDashes(cells[5])
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
func (p *NomenclatureParser) ProcessEntry(entryInterface *interface{}) error {
    // First, dereference the pointer to get the actual interface{} value
    entryValue := *entryInterface
    
    // Now we can perform type assertions on the actual interface value
    switch entry := entryValue.(type) {
    case *NomenclatureEntry:
        // It's already a pointer to NomenclatureEntry
        hierPath, err := getHierarchyPath(entry.GoodsCode, entry.HierPos)
        if err != nil {
            return fmt.Errorf("error getting hierarchy path: %v", err)
        }
        entry.HierarchyPath = hierPath
        
    case NomenclatureEntry:
        // It's a value type, need to create a pointer and update the interface
        newEntry := entry // Create a copy
        hierPath, err := getHierarchyPath(newEntry.GoodsCode, newEntry.HierPos)
        if err != nil {
            return fmt.Errorf("error getting hierarchy path: %v", err)
        }
        newEntry.HierarchyPath = hierPath
        
        // Update the original interface pointer to point to our new entry
        *entryInterface = newEntry
        
    default:
        return fmt.Errorf("unexpected entry type: %T", entryValue)
    }
    
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