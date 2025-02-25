package main

import (
    "database/sql"
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
}

func main() {
    // Connect to database using the database package
    db, err := database.Connect()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Define the chunk size
    chunkSize := 1000
    offset := 0

    for {
        // Select records from the database in chunks
        rows, err := db.Query(`
            SELECT ni.id, ni.goods_code, ni.start_date, ni.end_date, ni.hier_pos, ni.indent, nd.description, nd.language, nd.descr_start_date
            FROM nomenclature_items ni
            JOIN nomenclature_descriptions nd ON ni.id = nd.nomenclature_item_id
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

        // Iterate over the rows and print the data
        for rows.Next() {
            var data NomenclatureData
            var endDate sql.NullString

            err := rows.Scan(&data.ID, &data.GoodsCode, &data.StartDate, &endDate, &data.HierPos, &data.Indent, &data.Description, &data.Language, &data.DescrStartDate)
            if err != nil {
                log.Fatal(err)
            }

            if endDate.Valid {
                data.EndDate = &endDate.String
            } else {
                data.EndDate = nil
            }

            fmt.Printf("%+v\n", data)
        }

        // Close the rows
        rows.Close()

        // Increment the offset for the next chunk
        offset += chunkSize
    }
}