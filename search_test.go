package main

import "testing"

func TestSmartSearch(t *testing.T) {
	cases := []struct {
		query, category, term string
		min, max              float64
	}{
		{"беспроводная мышь до 10 тысяч тенге", "Mice", "wireless", 0, 10000},
		{"покажи мышь до десяти тысяч", "Mice", "", 0, 10000},
		{"mouse under ten thousand", "Mice", "", 0, 10000},
		{"mouse under 10 000 KZT", "Mice", "", 0, 10000},
		{"keyboard from 5000 to", "Keyboards", "to", 5000, 0},
		{"клавиатура от пяти тысяч до десяти тысяч", "Keyboards", "", 5000, 10000},
		{"SSD 500 GB", "Storage", "500", 0, 0},
		{"webcam below 15.5k", "Webcams", "", 0, 15500},
		{"монитор до 80 тысяч", "Monitors", "", 0, 80000},
		{"C270", "", "c270", 0, 0},
	}
	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			got, err := parseSearch(c.query)
			if err != nil {
				t.Fatal(err)
			}
			if got.Category != c.category {
				t.Fatalf("category %q", got.Category)
			}
			if c.min > 0 && (got.Min == nil || *got.Min != c.min) {
				t.Fatalf("min %v", got.Min)
			}
			if c.max > 0 && (got.Max == nil || *got.Max != c.max) {
				t.Fatalf("max %v", got.Max)
			}
			if c.term != "" {
				found := false
				for _, term := range got.Terms {
					if term == c.term {
						found = true
					}
				}
				if !found {
					t.Fatalf("terms %v", got.Terms)
				}
			}
		})
	}
}
func TestSmartSearchRejectsAmbiguousInput(t *testing.T) {
	for _, q := range []string{"мышь клавиатура", "mouse from 20000 under 1000", string(make([]rune, 301)), "mouse under 999999999999k"} {
		if _, err := parseSearch(q); err == nil {
			t.Fatalf("expected error for %q", q)
		}
	}
}
