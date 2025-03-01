package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"muj/database"
)

// NomenclatureData represents the combined data from both tables
type NomenclatureData struct {
    ID              int
    GoodsCode       string
    StartDate       string
    EndDate         *string
    HierPos         int
    Indent          int
    Description     string
    Language        string
    DescrStartDate  string
    SectionName     string
    Children        []*NomenclatureData `json:"children,omitempty"`
}

// HierarchicalStructure stores nomenclature data in a hierarchical format
type HierarchicalStructure map[string]*NomenclatureData

func getCategoryKey(goodsCode string, hierPos int) string {
    switch hierPos {
    case 2:
        if len(goodsCode) >= 2 {
            return goodsCode[:2]
        }
    case 4:
        if len(goodsCode) >= 4 {
            return goodsCode[:4]
        }
    case 6:
        if len(goodsCode) >= 6 {
            return goodsCode[:6]
        }
    case 8:
        if len(goodsCode) >= 8 {
            return goodsCode[:8]
        }
    case 10:
        return goodsCode
    }
    return goodsCode // Default case
}

func main() {
    // Connect to database using the database package
    db, err := database.Connect()
    if (err != nil) {
        log.Fatal(err)
    }
    defer db.Close()

    // Define the chunk size
    chunkSize := 9
    offset := 0
    
    // Map to store all items by their category key
    allItems := make(HierarchicalStructure)
    
    // Map to store all items by their ID for efficient lookups
    itemsById := make(map[int]*NomenclatureData)

    for {
        // Select records from the database in chunks
        rows, err := db.Query(`
            SELECT ni.id, ni.goods_code, ni.start_date, ni.end_date, ni.hier_pos, ni.indent, 
                   nd.description, nd.language, nd.descr_start_date, sd.name as section_name
            FROM nomenclature_items ni
            JOIN nomenclature_descriptions nd ON ni.id = nd.nomenclature_item_id
            JOIN section_chapter_mapping scm ON 
                CAST(SUBSTRING(ni.goods_code, 1, 2) AS INTEGER) = scm.chapter_id
            JOIN section_descriptions sd ON 
                scm.section_number = sd.section_number AND
                nd.language = sd.language
            ORDER BY ni.id
            LIMIT $1 OFFSET $2
        `, chunkSize, offset)
        if err != nil {
            log.Fatal(err)
        }

        // Check if there are no more rows to process
        if offset > 9 {
            break
        }

        // fake data limitation
        if offset <= 9 {
            // Iterate over the rows and process the data
            for rows.Next() {
                var data NomenclatureData
                var endDate sql.NullString

                err := rows.Scan(&data.ID, &data.GoodsCode, &data.StartDate, &endDate, &data.HierPos, &data.Indent, &data.Description, &data.Language, &data.DescrStartDate, &data.SectionName)
                if err != nil {
                    log.Fatal(err)
                }

                if endDate.Valid {
                    data.EndDate = &endDate.String
                } else {
                    data.EndDate = nil
                }

                // Create a new instance to store in our maps
                dataCopy := data
                dataPtr := &dataCopy
                
                // Store in itemsById map for easy referencing
                itemsById[data.ID] = dataPtr
                
                // Create a category key based on hier_pos and goods_code
                categoryKey := getCategoryKey(data.GoodsCode, data.HierPos)
                
                // Store in our hierarchical structure
                allItems[categoryKey] = dataPtr
            }
        }

        // Close the rows
        rows.Close()
        // Increment the offset for the next chunk
        offset += chunkSize
    }

    // Build the hierarchy
    buildHierarchy(allItems)

    // Create a map to track which items are already children of other items
    isChild := make(map[int]bool)
    
    // Mark all items that are children of other items
    for _, item := range allItems {
        for _, child := range item.Children {
            isChild[child.ID] = true
        }
    }

    // Extract only the true top-level items (those that aren't children of any other item)
    topLevelItems := []*NomenclatureData{}
    for _, item := range allItems {
        if !isChild[item.ID] {
            topLevelItems = append(topLevelItems, item)
        }
    }

    // Output the final hierarchical structure
    jsonData, err := json.MarshalIndent(topLevelItems, "", "  ")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(jsonData))
}

func buildHierarchy(items HierarchicalStructure) {
    // First pass: group items by their goods_code prefix
    prefixGroups := make(map[string][]*NomenclatureData)
    
    for _, item := range items {
        // Get the appropriate prefix based on hier_pos
        var prefix string
        switch item.HierPos {
        case 2:
            prefix = item.GoodsCode[:2]
        case 4:
            prefix = item.GoodsCode[:4]
        case 6:
            prefix = item.GoodsCode[:6]
        case 8:
            prefix = item.GoodsCode[:8]
        case 10:
            if len(item.GoodsCode) >= 8 {
                prefix = item.GoodsCode[:8]
            } else {
                prefix = item.GoodsCode
            }
        }
        
        prefixGroups[prefix] = append(prefixGroups[prefix], item)
    }
    
    // Second pass: assign children to parents based on indent and hier_pos
    for _, item := range items {
        if item.HierPos == 2 {
            // Top-level item, no parent to process
            continue
        }
        
        // Find the parent based on the hierarchy position
        var parentKey string
        
        switch item.HierPos {
        case 4:
            parentKey = item.GoodsCode[:2]
        case 6:
            parentKey = item.GoodsCode[:4]
        case 8:
            parentKey = item.GoodsCode[:6]
        case 10:
            parentKey = item.GoodsCode[:8]
        }
        
        // Get the potential parent
        if parent, exists := items[parentKey]; exists {
            // Add as child to parent
            parent.Children = append(parent.Children, item)
        }
    }
    
    // Third pass: handle cases where items have the same goods_code but different indents
    for _, group := range prefixGroups {
        if len(group) <= 1 {
            continue // No need to process single items
        }
        
        // Sort the group by indent value
        sortByIndent(group)
        
        // Process the group to establish parent-child relationships
        processIndentGroup(group)
    }
}

// sortByIndent sorts a slice of NomenclatureData pointers by their indent value
func sortByIndent(items []*NomenclatureData) {
    for i := 0; i < len(items)-1; i++ {
        for j := i + 1; j < len(items); j++ {
            if items[i].Indent > items[j].Indent {
                items[i], items[j] = items[j], items[i]
            }
        }
    }
}

// processIndentGroup establishes parent-child relationships within a group of items
// that share the same goods_code prefix but have different indent values
func processIndentGroup(items []*NomenclatureData) {
    if len(items) <= 1 {
        return
    }
    
    // Process each item and check if it should be a child of another item
    for i := 1; i < len(items); i++ {
        // Find the nearest item with a smaller indent
        parentIndex := -1
        for j := i - 1; j >= 0; j-- {
            if items[j].Indent < items[i].Indent {
                parentIndex = j
                break
            }
        }
        
        // If found a parent with smaller indent, make the current item its child
        if parentIndex != -1 {
            // Check if the item is already a child of this parent
            alreadyChild := false
            for _, child := range items[parentIndex].Children {
                if child.ID == items[i].ID {
                    alreadyChild = true
                    break
                }
            }
            
            // If not already a child, add it
            if !alreadyChild {
                // Remove from previous parent if exists (to avoid duplicates)
                for _, otherItem := range items {
                    if otherItem.ID != items[parentIndex].ID {
                        removeChild(otherItem, items[i].ID)
                    }
                }
                
                items[parentIndex].Children = append(items[parentIndex].Children, items[i])
            }
        }
    }
}

// removeChild removes a child with the given ID from the parent's children
func removeChild(parent *NomenclatureData, childID int) {
    if parent == nil || len(parent.Children) == 0 {
        return
    }
    
    newChildren := []*NomenclatureData{}
    for _, child := range parent.Children {
        if child.ID != childID {
            newChildren = append(newChildren, child)
        }
    }
    parent.Children = newChildren
}