package example

import "testing"

// SHOULD NOT TRIGGER any rule — these are well-written tests.

func TestAddition(t *testing.T) {
	result := Add(1, 2)
	if result != 3 {
		t.Errorf("Add(1, 2) = %d, want 3", result)
	}
}

func TestSubtraction(t *testing.T) {
	result := Subtract(5, 3)
	if result != 2 {
		t.Fatalf("Subtract(5, 3) = %d, want 2", result)
	}
}
