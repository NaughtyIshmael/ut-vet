package example

import (
	"testing"
)

// SHOULD TRIGGER: no-assertion — no assertion calls at all
func TestNoAssertion(t *testing.T) {
	x := 1 + 2
	_ = x
}

// SHOULD TRIGGER: empty-test — completely empty body
func TestEmptyBody(t *testing.T) {
}

// SHOULD TRIGGER: empty-test — only comments
func TestOnlyComments(t *testing.T) {
	// TODO: implement this test
	// another comment
}

// SHOULD TRIGGER: log-only-test — only logging, no assertion
func TestLogOnly(t *testing.T) {
	t.Log("starting test")
	x := 1 + 2
	t.Logf("result: %d", x)
}

// SHOULD NOT TRIGGER: has a real assertion
func TestWithAssertion(t *testing.T) {
	x := 1 + 2
	if x != 3 {
		t.Errorf("expected 3, got %d", x)
	}
}

// SHOULD NOT TRIGGER: uses t.Fatal
func TestWithFatal(t *testing.T) {
	x := 1 + 2
	if x != 3 {
		t.Fatalf("expected 3, got %d", x)
	}
}

// SHOULD NOT TRIGGER: uses t.Fail
func TestWithFail(t *testing.T) {
	x := 1 + 2
	if x != 3 {
		t.Fail()
	}
}
