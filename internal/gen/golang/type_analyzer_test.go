package golang

import (
	"testing"
)

func TestIsLeafRecord(t *testing.T) {
	tests := []struct {
		name       string
		fieldTypes map[string]string
		wantLeaf   bool
	}{
		{
			name: "all primitives - Coord",
			fieldTypes: map[string]string{
				"X": "int64",
				"Y": "int64",
			},
			wantLeaf: true,
		},
		{
			name: "mixed primitives - Camera",
			fieldTypes: map[string]string{
				"X":    "float64",
				"Y":    "float64",
				"Zoom": "float64",
			},
			wantLeaf: true,
		},
		{
			name: "with bool and string",
			fieldTypes: map[string]string{
				"Name":    "string",
				"Active":  "bool",
				"Counter": "int64",
			},
			wantLeaf: true,
		},
		{
			name: "with nested pointer - Entity",
			fieldTypes: map[string]string{
				"Pos": "*Coord",
				"Vel": "*Coord",
			},
			wantLeaf: false,
		},
		{
			name: "with nested value type - not leaf",
			fieldTypes: map[string]string{
				"Pos": "Coord",
				"Vel": "Coord",
			},
			wantLeaf: false,
		},
		{
			name: "with slice",
			fieldTypes: map[string]string{
				"Name":  "string",
				"Items": "[]int64",
			},
			wantLeaf: false,
		},
		{
			name: "with pointer slice",
			fieldTypes: map[string]string{
				"Npcs": "[]*NPC",
			},
			wantLeaf: false,
		},
		{
			name: "with map",
			fieldTypes: map[string]string{
				"Data": "map[string]interface{}",
			},
			wantLeaf: false,
		},
		{
			name: "with interface{}",
			fieldTypes: map[string]string{
				"Value": "interface{}",
			},
			wantLeaf: false,
		},
		{
			name:       "empty record",
			fieldTypes: map[string]string{},
			wantLeaf:   true, // Vacuously true - no non-primitive fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLeafRecord(tt.fieldTypes)
			if got != tt.wantLeaf {
				t.Errorf("IsLeafRecord() = %v, want %v", got, tt.wantLeaf)
			}
		})
	}
}

func TestAnalyzeRecordType(t *testing.T) {
	tests := []struct {
		name         string
		fieldTypes   map[string]string
		threshold    int
		wantCategory TypeCategory
		wantLeaf     bool
	}{
		{
			name: "small leaf record - value",
			fieldTypes: map[string]string{
				"X": "int64",
				"Y": "int64",
			},
			threshold:    4,
			wantCategory: TypeCategoryValue,
			wantLeaf:     true,
		},
		{
			name: "at threshold - value",
			fieldTypes: map[string]string{
				"X": "int64",
				"Y": "int64",
				"Z": "int64",
				"W": "int64",
			},
			threshold:    4,
			wantCategory: TypeCategoryValue,
			wantLeaf:     true,
		},
		{
			name: "over threshold - pointer",
			fieldTypes: map[string]string{
				"A": "int64",
				"B": "int64",
				"C": "int64",
				"D": "int64",
				"E": "int64",
			},
			threshold:    4,
			wantCategory: TypeCategoryPointer,
			wantLeaf:     true, // Still leaf, just too big
		},
		{
			name: "non-leaf - always pointer",
			fieldTypes: map[string]string{
				"Pos": "*Coord",
			},
			threshold:    10, // Even with high threshold
			wantCategory: TypeCategoryPointer,
			wantLeaf:     false,
		},
		{
			name: "threshold 0 - all pointers",
			fieldTypes: map[string]string{
				"X": "int64",
			},
			threshold:    0,
			wantCategory: TypeCategoryPointer,
			wantLeaf:     true,
		},
		{
			name: "threshold 3 - small value",
			fieldTypes: map[string]string{
				"X": "int64",
				"Y": "int64",
				"Z": "int64",
			},
			threshold:    3,
			wantCategory: TypeCategoryValue,
			wantLeaf:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New("test")
			g.SetValueThreshold(tt.threshold)

			gotCategory, gotLeaf := g.AnalyzeRecordType(tt.fieldTypes)
			if gotCategory != tt.wantCategory {
				t.Errorf("AnalyzeRecordType() category = %v, want %v", gotCategory, tt.wantCategory)
			}
			if gotLeaf != tt.wantLeaf {
				t.Errorf("AnalyzeRecordType() isLeaf = %v, want %v", gotLeaf, tt.wantLeaf)
			}
		})
	}
}

func TestRegisterRecordTypeWithAnalysis(t *testing.T) {
	g := New("test")
	g.SetValueThreshold(4)

	// Register a leaf record
	g.RegisterRecordTypeWithAnalysis("Coord", []string{"X", "Y"}, map[string]string{
		"X": "int64",
		"Y": "int64",
	})

	info := g.recordTypes["Coord"]
	if info == nil {
		t.Fatal("expected Coord to be registered")
	}
	if info.Category != TypeCategoryValue {
		t.Errorf("expected TypeCategoryValue, got %v", info.Category)
	}
	if !info.IsLeaf {
		t.Error("expected IsLeaf = true")
	}
	if info.FieldCount != 2 {
		t.Errorf("expected FieldCount = 2, got %d", info.FieldCount)
	}

	// Register a non-leaf record
	g.RegisterRecordTypeWithAnalysis("Entity", []string{"Pos", "Vel"}, map[string]string{
		"Pos": "*Coord",
		"Vel": "*Coord",
	})

	info = g.recordTypes["Entity"]
	if info == nil {
		t.Fatal("expected Entity to be registered")
	}
	if info.Category != TypeCategoryPointer {
		t.Errorf("expected TypeCategoryPointer, got %v", info.Category)
	}
	if info.IsLeaf {
		t.Error("expected IsLeaf = false")
	}
}

func TestSetValueThreshold(t *testing.T) {
	g := New("test")

	// Default threshold
	if g.GetValueThreshold() != 4 {
		t.Errorf("expected default threshold 4, got %d", g.GetValueThreshold())
	}

	// Set to 0
	g.SetValueThreshold(0)
	if g.GetValueThreshold() != 0 {
		t.Errorf("expected threshold 0, got %d", g.GetValueThreshold())
	}

	// Negative treated as 0
	g.SetValueThreshold(-5)
	if g.GetValueThreshold() != 0 {
		t.Errorf("expected negative threshold to be 0, got %d", g.GetValueThreshold())
	}

	// Set to positive
	g.SetValueThreshold(8)
	if g.GetValueThreshold() != 8 {
		t.Errorf("expected threshold 8, got %d", g.GetValueThreshold())
	}
}
