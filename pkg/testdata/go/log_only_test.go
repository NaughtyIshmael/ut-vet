package example

import (
	"fmt"
	"log"
	"testing"
)

// SHOULD TRIGGER: log-only-test — uses fmt.Println only
func TestFmtPrintOnly(t *testing.T) {
	result := 1 + 2
	fmt.Println("result:", result)
}

// SHOULD TRIGGER: log-only-test — uses log.Println only
func TestLogPrintOnly(t *testing.T) {
	result := 1 + 2
	log.Println("result:", result)
}

// SHOULD TRIGGER: log-only-test — uses t.Log only
func TestTLogOnly(t *testing.T) {
	t.Log("this test does nothing useful")
}

// SHOULD NOT TRIGGER: has logging AND an assertion
func TestLogWithAssertion(t *testing.T) {
	result := 1 + 2
	t.Logf("result: %d", result)
	if result != 3 {
		t.Errorf("expected 3, got %d", result)
	}
}
