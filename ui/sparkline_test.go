package ui

import (
	"testing"
	"unicode/utf8"
)

func runeLen(s string) int { return utf8.RuneCountInString(s) }

func TestSparklineEmpty(t *testing.T) {
	if got := sparkline(nil, 8); runeLen(got) != 8 {
		t.Fatalf("empty sparkline len = %d, want 8", runeLen(got))
	}
	for _, r := range sparkline(nil, 8) {
		if r != ' ' {
			t.Fatalf("empty sparkline should be blank space, got %q", r)
		}
	}
}

func TestSparklineMonotonic(t *testing.T) {
	up := sparkline([]float64{1, 2, 3, 4, 5, 6, 7, 8}, 8)
	down := sparkline([]float64{8, 7, 6, 5, 4, 3, 2, 1}, 8)
	if up == down {
		t.Fatalf("rising series rendered same as falling: %q vs %q", up, down)
	}
}

func TestSparklineFlat(t *testing.T) {
	flat := sparkline([]float64{5, 5, 5, 5}, 4)
	if runeLen(flat) != 4 {
		t.Fatalf("flat sparkline len = %d, want 4", runeLen(flat))
	}
}

func TestSparklineShortSeries(t *testing.T) {
	// fewer points than columns must not panic and must fill the width
	if got := sparkline([]float64{3}, 8); runeLen(got) != 8 {
		t.Fatalf("short series len = %d, want 8", runeLen(got))
	}
}