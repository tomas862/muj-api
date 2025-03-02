package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"muj/database"
	"sort"
	"strings"
)

// NomenclatureData represents the combined data from both tables
type NomenclatureData struct {
    ID              int
    GoodsCode       string
    StartDate       string
    EndDate         *string
    HierarchyPath   string
    Indent          int
    Description     string
    Language        string
    DescrStartDate  string
    SectionName     string
	IsLeaf		 	*bool
}

// NomenclatureResult represents the structured result for a goods code with multi-language support
type NomenclatureResult struct {
    GoodsCode      string              `json:"goods_code"`
    Description    map[string]string   `json:"description"`
	CategoryCodes  []string `json:"category_codes"`
    Categories     map[string][]string `json:"categories"`
    CategoriesPath map[string]string   `json:"categories_path"`
	CategoryCodesPath string `json:"category_codes_path"`
}

func main () {
	// Connect to database using the database package
	db, err := database.Connect()
	if (err != nil) {
		log.Fatal(err)
	}
	defer db.Close()

	// Define the chunk size
    chunkSize := 9
    offset := 0

	// Create a map to store data indexed by hierarchy path
    hierarchyData := make(map[string]map[string][]NomenclatureData)

	for {	
        rows, err := db.Query(`
            SELECT ni.id, ni.goods_code, ni.start_date, ni.end_date, ni.hierarchy_path, ni.indent, 
                   nd.description, nd.language, nd.descr_start_date, sd.name as section_name,
				   dc.is_leaf
            FROM nomenclatures ni
            JOIN nomenclature_descriptions nd ON ni.id = nd.nomenclature_id
			LEFT JOIN nomenclature_declarable_codes dc ON ni.id = dc.nomenclature_id
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
				var isLeaf sql.NullBool

				err := rows.Scan(
					&data.ID,
					&data.GoodsCode,
					&data.StartDate,
					&endDate,
					&data.HierarchyPath,
					&data.Indent,
					&data.Description,
					&data.Language,
					&data.DescrStartDate,
					&data.SectionName,
					&isLeaf,
				)
				if err != nil {
					log.Fatal(err)
				}

				if endDate.Valid {
					data.EndDate = &endDate.String
				} else {
					data.EndDate = nil
				}

				if isLeaf.Valid {
					data.IsLeaf = &isLeaf.Bool
				} else {
					data.IsLeaf = nil
				}

				// Initialize the inner map if it doesn't exist
				if _, exists := hierarchyData[data.HierarchyPath]; !exists {
					hierarchyData[data.HierarchyPath] = make(map[string][]NomenclatureData)
				}

				// Add the data to the map, using HierarchyPath as the key
                hierarchyData[data.HierarchyPath][data.Language] = append(hierarchyData[data.HierarchyPath][data.Language], data)
			}
		}

		// Close the rows
        rows.Close()
        // Increment the offset for the next chunk
        offset += chunkSize
	}

	// Print the number of unique hierarchy paths found
    fmt.Printf("Found %d unique hierarchy paths\n", len(hierarchyData))
    
	// Create a map to store results by goodsCode
	resultMap := make(map[string]NomenclatureResult)

    // Now process each entry to build categories
    for _, entriesWithLanguage := range hierarchyData {
		for language, entries := range entriesWithLanguage {
			// First sort entries by indent to ensure proper nesting order
			sort.Slice(entries, func(i, j int) bool {
				return entries[i].Indent < entries[j].Indent
			})
			
			for i, entry := range entries {
				// Check if we already have an entry for this goods code
				result, exists := resultMap[entry.GoodsCode]
				if !exists {
					// Initialize a new result structure
					result = NomenclatureResult{
						GoodsCode:      entry.GoodsCode,
						Description:    make(map[string]string),
						CategoryCodes: 	[]string{},
						Categories:     make(map[string][]string),
						CategoriesPath: make(map[string]string),
						CategoryCodesPath: "",
					}
				}

				// Add this language's description
				result.Description[language] = entry.Description
				
				// Process categories for this language
				categories := []string{entry.SectionName}
				categoryCodes := []string{}
				
				segmentToCheck := ""
				pathSegments := strings.Split(entry.HierarchyPath, ".")

				for i, segment := range pathSegments {
					if i == len(pathSegments)-1 {
						break
					}

					categoryCodes = append(categoryCodes, segment)
					
					segmentToCheck += segment
					if _, exists := hierarchyData[segmentToCheck]; exists {
						if langData, ok := hierarchyData[segmentToCheck][language]; ok {
							for _, data := range langData {
								categories = append(categories, data.Description)
							}
						}
					}
					segmentToCheck += "."
				}

				if i > 0 && entries[i-1].Indent < entry.Indent {
					categories = append(categories, entries[i-1].Description)
				}

				// Store categories and path for this language
				result.Categories[language] = categories
				result.CategoriesPath[language] = strings.Join(categories, " > ")
				result.CategoryCodes = categoryCodes
				result.CategoryCodesPath = strings.Join(categoryCodes, " > ")
				
				// Update the map
				resultMap[entry.GoodsCode] = result
			}
		}
    }

	// Convert the map to a slice for output
	results := []NomenclatureResult{}

	for _, result := range resultMap {
		results = append(results, result)
	}

	// Output the results
	for _, result := range results {
		jsonResult, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Result: %s\n\n", jsonResult)
	}
}