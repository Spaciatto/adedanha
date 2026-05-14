package domain

import "testing"

func TestValidLettersCount(t *testing.T) {
	// 26 letters - K, W, X, Y = 22
	if len(ValidLetters) != 22 {
		t.Errorf("Expected 22 valid letters, got %d", len(ValidLetters))
	}
}

func TestValidLettersExcluded(t *testing.T) {
	excluded := []string{"K", "W", "X", "Y"}
	letterSet := make(map[string]bool)
	for _, l := range ValidLetters {
		letterSet[l] = true
	}

	for _, ex := range excluded {
		if letterSet[ex] {
			t.Errorf("Letter %s should be excluded but is present", ex)
		}
	}
}

func TestValidLettersIncluded(t *testing.T) {
	expected := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "Z"}
	letterSet := make(map[string]bool)
	for _, l := range ValidLetters {
		letterSet[l] = true
	}

	for _, e := range expected {
		if !letterSet[e] {
			t.Errorf("Letter %s should be included but is missing", e)
		}
	}
}
