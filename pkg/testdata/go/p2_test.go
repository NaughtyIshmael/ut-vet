package example

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// SHOULD TRIGGER: tautological-assert — comparing x to x
func TestSelfCompare(t *testing.T) {
	result := Compute()
	assert.Equal(t, result, result)
}

// SHOULD TRIGGER: dead-assertion — assertion after t.Fatal
func TestDeadCode(t *testing.T) {
	t.Fatal("always fails")
	assert.Equal(t, 1, 2)
}

// SHOULD TRIGGER: no-arrange — all nil args to NewHandler
func TestNilSetup(t *testing.T) {
	h := NewHandler(nil, nil)
	assert.NotNil(t, h)
}

// SHOULD NOT TRIGGER: meaningful setup
func TestGoodSetup(t *testing.T) {
	result := Compute()
	assert.Equal(t, 42, result)
}
