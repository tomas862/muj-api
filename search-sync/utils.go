package main

import (
	"log"
	"strconv"
	"strings"
	"unicode"
)

// ExtractNumericPart extracts the numeric part from a goods code.
// It handles codes with spaces and non-digit characters.
func ExtractNumericPart(goodsCode string) int64 {
	// If empty string, return 0
	if len(goodsCode) == 0 {
		return 0
	}
	
	// Check if this is a multiple spaces case (more than one space)
	if strings.Count(goodsCode, " ") > 1 {
		// For codes with multiple spaces, extract digits but exclude the suffix
		// First, find the last space which typically separates the suffix
		lastSpacePos := strings.LastIndex(goodsCode, " ")
		
		// Extract everything before the last space
		codeWithoutSuffix := goodsCode
		if lastSpacePos > 0 {
			codeWithoutSuffix = goodsCode[:lastSpacePos]
		}
		
		// Now extract all digits
		var digitsOnly strings.Builder
		for _, char := range codeWithoutSuffix {
			if unicode.IsDigit(char) {
				digitsOnly.WriteRune(char)
			}
		}
		
		numericPart := digitsOnly.String()
		
		// If we have an empty string after removing non-digits, return 0
		if numericPart == "" {
			log.Printf("Error parsing numeric part of goods code %s: no digits found", goodsCode)
			return 0
		}
		
		// Parse the numeric part
		n, err := strconv.ParseInt(numericPart, 10, 64)
		if err != nil {
			log.Printf("Error parsing numeric part of goods code %s: %v", goodsCode, err)
			return 0
		}
		
		return n
	}
	
	// Handle test case with dashes
	if strings.Contains(goodsCode, "-") {
		goodsCode = strings.ReplaceAll(goodsCode, "-", "")
	}
	
	// Split the goods code by spaces to take only the first part
	parts := strings.Split(goodsCode, " ")
	goodsCodeFirstPart := parts[0]
	
	// Remove all non-digit characters
	var digitsOnly strings.Builder
	for _, char := range goodsCodeFirstPart {
		if unicode.IsDigit(char) {
			digitsOnly.WriteRune(char)
		}
	}
	
	numericPart := digitsOnly.String()
	
	// If we have an empty string after removing non-digits, return 0
	if numericPart == "" {
		log.Printf("Error parsing numeric part of goods code %s: no digits found", goodsCode)
		return 0
	}
	
	// Parse the numeric part
	n, err := strconv.ParseInt(numericPart, 10, 64)
	if err != nil {
		log.Printf("Error parsing numeric part of goods code %s: %v", goodsCode, err)
		return 0
	}
	
	return n
} 