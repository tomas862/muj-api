package main

import (
	"testing"
)

func TestExtractNumericPart(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		// Normal cases
		{"Normal goods code", "4112000000", 4112000000},
		{"With leading zero", "0304810000", 304810000},
		
		// With spaces/suffixes
		{"With space and suffix", "4112000000 80", 4112000000},
		{"With space and suffix leading zero", "0304810000 80", 304810000},
		
		// Section codes
		{"Section code 1", "1", 1},
		{"Section code 21", "21", 21},
		
		// Edge cases
		{"Empty string", "", 0},
		{"Non-numeric", "ABC", 0},
		{"Mixed", "X123Y456", 123456},
		
		// Special formats
		{"With prefix letters", "CN0102909100", 102909100},
		{"With mixed alphanumeric", "0304-959-011 10", 304959011},
		{"Multiple spaces", "7606 129 291 80", 7606129291},
		
		// Real examples from dataset
		{"Real example 1", "0304530011 10", 304530011},
		{"Real example 2", "2204213290 80", 2204213290},
		{"Real example 3", "5512299000 80", 5512299000},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractNumericPart(tc.input)
			if result != tc.expected {
				t.Errorf("ExtractNumericPart(%q) = %d, expected %d", tc.input, result, tc.expected)
			}
		})
	}
} 