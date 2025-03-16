package main

import (
	"context"
	"database/sql"
	"log"
	"muj/database"
	"muj/utils"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/joho/godotenv"
	"github.com/typesense/typesense-go/v3/typesense"
	"github.com/typesense/typesense-go/v3/typesense/api"
	"github.com/typesense/typesense-go/v3/typesense/api/pointer"
	"golang.org/x/text/unicode/norm"
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
	Id 		   	   string                `json:"id"`
    GoodsCode      string              `json:"goods_code"`
    DescriptionEn    string   `json:"description_en"`
    DescriptionLt    string   `json:"description_lt"`
    DescriptionLtNormalized string `json:"description_lt_normalized"`
	CategoryCodes  []string `json:"category_codes"`
    CategoriesEn     []string `json:"categories_en"`
    CategoriesLt     []string `json:"categories_lt"`
    CategoriesLtNormalized []string `json:"categories_lt_normalized"`
    RankBoost      int       `json:"rank_boost"`
    // CategoriesPath map[string]string   `json:"categories_path"`
	// CategoryCodesPath string `json:"category_codes_path"`
}

// removeDiacritics removes diacritical marks from a string
func removeDiacritics(input string) string {
	// Normalize to decomposed form (NFD), so accents become separate characters
	t := norm.NFD.String(input)
	
	// Filter out non-spacing marks (accents, diacritics)
	var b strings.Builder
	for _, r := range t {
		if unicode.IsMark(r) {
			continue // Skip diacritics
		}
		b.WriteRune(r)
	}

	return b.String()
}

func main () {
	// Load environment variables from the same directory as this file
	if err := godotenv.Load(filepath.Join(utils.GetAbsolutePath(".env"))); err != nil {
		log.Fatal("Error loading .env file")
	}
	
	// Connect to database using the database package
	db, err := database.Connect()
	if (err != nil) {
		log.Fatal(err)
	}
	defer db.Close()

	// Define the chunk size
    chunkSize := 1000
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
        if !rows.Next() {
            break
        }

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

		// Close the rows
        rows.Close()
        // Increment the offset for the next chunk
        offset += chunkSize
	}
    
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
						Id:			 strconv.Itoa(entry.ID),
						GoodsCode:      entry.GoodsCode,
						DescriptionEn:   "",
						DescriptionLt:   "",
                        DescriptionLtNormalized: "",
						CategoryCodes: 	[]string{},
						CategoriesEn:     []string{},
						CategoriesLt:     []string{},
                        CategoriesLtNormalized: []string{},
						RankBoost:      0, // Default boost value
						// CategoriesPath: make(map[string]string),
						// CategoryCodesPath: "",
					}
				}

				// Add this language's description
				if language == "EN" {
					result.DescriptionEn = entry.Description
				} else if language == "LT" {
					result.DescriptionLt = entry.Description
                    result.DescriptionLtNormalized = removeDiacritics(entry.Description)
				}
				
				// Set rank boost based on isLeaf value
				if entry.IsLeaf != nil && *entry.IsLeaf {
					result.RankBoost = 10 // Higher value for leaf nodes
				}
				
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
				if language == "EN" {
					result.CategoriesEn = categories
				} else if language == "LT" {
					result.CategoriesLt = categories
                    result.CategoriesLtNormalized = make([]string, len(categories))
                    for i, category := range categories {
                        result.CategoriesLtNormalized[i] = removeDiacritics(category)
                    }
				}
				// result.CategoriesPath[language] = strings.Join(categories, " > ")
				result.CategoryCodes = categoryCodes
				// result.CategoryCodesPath = strings.Join(categoryCodes, " > ")
				
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

	client := typesense.NewClient(
	    typesense.WithServer(os.Getenv("TYPESENSE_HOST")),
	    typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")),)
	
	client.Collection("nomenclatures").Delete(context.Background())

	schema := &api.CollectionSchema{
		Name: "nomenclatures",
		Fields: []api.Field{
			{
				Name: "goods_code",
				Type: "string",
			},
			{
				Name: "description_en",
				Type: "string",
				Locale: pointer.String("en"),
			},
			{
				Name: "description_lt",
				Type: "string",
				Locale: pointer.String("lt"),
			},
            {
                Name: "description_lt_normalized",
                Type: "string",
                Locale: pointer.String("lt"),
            },
			{
				Name: "category_codes",
				Type: "string[]",
				Facet: pointer.True(),
			},
			{
				Name: "categories_en",
				Type: "string[]",
				Facet: pointer.True(),
				Locale: pointer.String("en"),
			},
			{
				Name: "categories_lt",
				Type: "string[]",
				Facet: pointer.True(),
				Locale: pointer.String("lt"),
			},
            {
                Name: "categories_lt_normalized",
				Type: "string[]",
				Facet: pointer.True(),
				Locale: pointer.String("lt"),
            },
			{
				Name: "rank_boost",
				Type: "int32",
			},
		},
	}

	_, err = client.Collections().Create(context.Background(), schema)

	if err != nil {
		log.Fatal(err)
	}

	chunkSize = 1000
    rowCount := 0
    totalRecords := len(results)
    log.Printf("Starting import of %d records to Typesense", totalRecords)
    
    for i := 0; i < len(results); i++ {
        rowCount++
        if (rowCount >= chunkSize || i == len(results)-1) {
            start := i - rowCount + 1
            end := i + 1  // exclusive end
            
            log.Printf("Importing batch %d-%d of %d records (%d%%)", 
                start+1, end, totalRecords, end*100/totalRecords)
            
            action := api.Create
            params := &api.ImportDocumentsParams{
                Action:    &action,  // Create a pointer to the constant
                BatchSize: pointer.Int(rowCount),
            }

            // Convert the slice to []interface{}
            documents := make([]interface{}, rowCount)
            for j := 0; j < rowCount; j++ {
                documents[j] = results[start+j]
            }

            importResult, err := client.Collection("nomenclatures").Documents().Import(context.Background(), documents, params)
            
            if err != nil {
                log.Fatal(err)
            }
            
            // Log success/failure counts
            successCount := 0
            for _, doc := range importResult {
                if doc.Success {
                    successCount++
                } else {
                    log.Printf("Error importing document: %s", doc.Error)
                }
            }
            
            log.Printf("Batch import completed: %d successful, %d failed", 
                successCount, rowCount-successCount)

            rowCount = 0
        }
    }

    // No need for a separate final batch handling as it's now included in the main loop
    
    log.Printf("Import process completed: %d total records processed", totalRecords)
}