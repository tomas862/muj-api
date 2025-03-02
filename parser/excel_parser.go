package main

import (
    "fmt"
    "muj/utils"

    "github.com/thedatashed/xlsxreader"
)

// ExcelRow contains data from an Excel row
type ExcelRow struct {
    Cells []string
}

// BaseExcelParser provides common Excel file reading functionality
type BaseExcelParser struct{}

// ReadRows implements the common Excel file reading logic
func (p *BaseExcelParser) ReadRows(config ParserConfig) (<-chan RowData, error) {
    // Create an instance of the reader by opening the target file
    filePath := utils.GetAbsolutePath(config.FilePath)
    xl, err := xlsxreader.OpenFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to open Excel file: %v", err)
    }
    
    // Check if the sheet exists
    if len(xl.Sheets) == 0 {
        return nil, fmt.Errorf("no sheets found in the Excel file")
    }

    rowsChan := make(chan RowData)

    go func() {
        defer xl.Close()
        defer close(rowsChan)

        firstRow := true
        
        // Read rows from Excel and send to channel
        for row := range xl.ReadRows(xl.Sheets[0]) {
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

            rowsChan <- ExcelRow{Cells: cells}
        }
    }()

    return rowsChan, nil
}