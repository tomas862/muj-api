package main

import "testing"

func TestBuildTaricPath(t *testing.T) {
	tests := []struct {
		name          string
		categoryCodes []string
		goodsCode     string
		expected      string
	}{
		{
			name:          "Empty category codes",
			categoryCodes: []string{},
			goodsCode:     "0101210000",
			expected:      "",
		},
		{
			name:          "Single section code",
			categoryCodes: []string{"1"},
			goodsCode:     "0101210000",
			expected:      "1 > 01",
		},
		{
			name:          "Section and chapter",
			categoryCodes: []string{"1", "01"},
			goodsCode:     "0101210000",
			expected:      "1 > 01 > 0101",
		},
		{
			name:          "Up to heading",
			categoryCodes: []string{"1", "01", "0101"},
			goodsCode:     "0101210000",
			expected:      "1 > 01 > 0101 > 010121",
		},
		{
			name:          "With space in goods code",
			categoryCodes: []string{"1", "01"},
			goodsCode:     "0101210000 80",
			expected:      "1 > 01 > 0101",
		},
		{
			name:          "Next level already in path",
			categoryCodes: []string{"1", "01", "0101"},
			goodsCode:     "0101000000",
			expected:      "1 > 01 > 0101",
		},
		{
			name:          "Short goods code",
			categoryCodes: []string{"1", "01"},
			goodsCode:     "01",
			expected:      "1 > 01",
		},
		{
			name:          "Different section example",
			categoryCodes: []string{"3", "03"},
			goodsCode:     "0306310000",
			expected:      "3 > 03 > 0306",
		},
		{
			name:          "Category path longer than code hierarchy",
			categoryCodes: []string{"1", "01", "0101", "010121", "01012100"},
			goodsCode:     "0101210000",
			expected:      "1 > 01 > 0101 > 010121 > 01012100",
		},
		{
			name:          "Goods code shorter than expected",
			categoryCodes: []string{"1", "01"},
			goodsCode:     "010",
			expected:      "1 > 01",
		},
		{
			name:          "Goods code with non-digit characters",
			categoryCodes: []string{"1", "01"},
			goodsCode:     "0101A210000",
			expected:      "1 > 01 > 0101",
		},
		{
			name:          "Deep category hierarchy with exact match",
			categoryCodes: []string{"1", "01", "0101", "010121"},
			goodsCode:     "0101210000",
			expected:      "1 > 01 > 0101 > 010121",
		},
		{
			name:          "Goods code shorter than chapter level",
			categoryCodes: []string{"1", "01"},
			goodsCode:     "0",
			expected:      "1",
		},
		{
			name:          "Category code does not match start of goods code",
			categoryCodes: []string{"2", "02"},
			goodsCode:     "0101210000",
			expected:      "2 > 02",
		},
		{
			name: "Actual code",
			categoryCodes: []string{"1"},
			goodsCode: "0300000000 80",
			expected: "1 > 03",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildTaricPath(tt.categoryCodes, tt.goodsCode)
			if result != tt.expected {
				t.Errorf("BuildTaricPath(%v, %s) = %s; want %s",
					tt.categoryCodes, tt.goodsCode, result, tt.expected)
			}
		})
	}
} 