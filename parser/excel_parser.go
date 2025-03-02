package main

import (
    "fmt"
    "io/fs"
    "log"
    "muj/utils"
    "os"
    "path/filepath"
    "strings"

    "github.com/thedatashed/xlsxreader"
)

// ExcelRow contains data from an Excel row
type ExcelRow struct {
    Cells  []string
    Source string // Added to track which file the row came from
}

// BaseExcelParser provides common Excel file reading functionality
type BaseExcelParser struct{}

// ReadRows implements the common Excel file reading logic
func (p *BaseExcelParser) ReadRows(config ParserConfig) (<-chan RowData, error) {
    if config.FilePath == "" {
        return nil, fmt.Errorf("file path is required for Excel parser")
    }

    // Get absolute path
    absPath := utils.GetAbsolutePath(config.FilePath)
    
    // Check if path is a file or directory
    fileInfo, err := os.Stat(absPath)
    if err != nil {
        return nil, fmt.Errorf("failed to access path: %v", err)
    }

    rowsChan := make(chan RowData)

    // Handle based on whether it's a file or directory
    if fileInfo.IsDir() {
        // It's a directory - process all Excel files
        go processDirectory(absPath, rowsChan)
    } else {
        // It's a file - process the single file
        go processFile(absPath, rowsChan)
    }

    return rowsChan, nil
}

// processDirectory reads all Excel files from a directory and processes them
func processDirectory(dirPath string, rowsChan chan<- RowData) {
    // Remove the defer close here, since we'll handle closing differently
    // depending on whether we find files or not
    
    // Get all Excel files in the directory
    var excelFiles []string
    err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        
        // Skip subdirectories
        if d.IsDir() && path != dirPath {
            return filepath.SkipDir
        }
        
        // Check if file is an Excel file (.xlsx, .xls)
        if !d.IsDir() && isExcelFile(path) {
            excelFiles = append(excelFiles, path)
        }
        
        return nil
    })

    if err != nil {
        log.Printf("Error walking directory: %v", err)
        close(rowsChan) // Close here for error case
        return
    }

    // No files found case
    if len(excelFiles) == 0 {
        log.Printf("No Excel files found in directory: %s", dirPath)
        close(rowsChan) // Close here for empty directory case
        return
    }

    // Process each Excel file sequentially
    for i, file := range excelFiles {
        log.Printf("Processing Excel file %d/%d: %s", i+1, len(excelFiles), file)
        
        // Create a multiplexing channel for all files except the last
        if i < len(excelFiles)-1 {
            fileChan := make(chan RowData)
            
            // Start a goroutine to forward rows
            go func(f string) {
                // Forward all rows from this file to main channel
                for row := range fileChan {
                    rowsChan <- row
                }
            }(file)
            
            // Process file with its own channel
            processFileWithoutClosing(file, fileChan)
            close(fileChan) // Close the file's channel after processing
        } else {
            // For the last file, process and close the main channel when done
            processFileWithoutClosing(file, rowsChan)
            close(rowsChan) // Close main channel after last file
        }
    }
}

// processFile reads a single Excel file and sends rows to the channel, and closes the channel
func processFile(filePath string, rowsChan chan<- RowData) {
    processFileWithoutClosing(filePath, rowsChan)
    close(rowsChan)
}

// processFileWithoutClosing reads a single Excel file and sends rows to the channel without closing it
func processFileWithoutClosing(filePath string, rowsChan chan<- RowData) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Recovered from panic while processing file %s: %v", filePath, r)
        }
    }()

    // Extract filename for source tracking
    filename := filepath.Base(filePath)

    // Open the Excel file
    xl, err := xlsxreader.OpenFile(filePath)
    if err != nil {
        log.Printf("Failed to open Excel file %s: %v", filePath, err)
        return
    }
    defer xl.Close()
    
    // Check if the sheet exists
    if len(xl.Sheets) == 0 {
        log.Printf("No sheets found in the Excel file %s", filePath)
        return
    }

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

        rowsChan <- ExcelRow{
            Cells:  cells,
            Source: filename,
        }
    }
}

// isExcelFile checks if the file has an Excel extension
func isExcelFile(path string) bool {
    ext := strings.ToLower(filepath.Ext(path))
    return ext == ".xlsx" || ext == ".xls"
}