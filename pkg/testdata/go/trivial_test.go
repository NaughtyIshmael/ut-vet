package example

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SHOULD TRIGGER: trivial-assertion — asserts a literal true
func TestTrivialTrue(t *testing.T) {
	assert.True(t, true)
}

// SHOULD TRIGGER: trivial-assertion — asserts equal literals
func TestTrivialEqualLiterals(t *testing.T) {
	assert.Equal(t, 1, 1)
}

// SHOULD TRIGGER: trivial-assertion — asserts equal string literals
func TestTrivialEqualStrings(t *testing.T) {
	require.Equal(t, "hello", "hello")
}

// SHOULD TRIGGER: trivial-assertion — asserts false is false
func TestTrivialFalse(t *testing.T) {
	assert.False(t, false)
}

// SHOULD NOT TRIGGER: asserts a variable
func TestRealAssertTrue(t *testing.T) {
	x := someFunc()
	assert.True(t, x)
}

// SHOULD NOT TRIGGER: asserts variable against literal
func TestRealAssertEqual(t *testing.T) {
	x := someFunc()
	assert.Equal(t, 42, x)
}

// SHOULD TRIGGER: no-assertion — has testify import but no assertion calls
func TestImportButNoAssert(t *testing.T) {
	_ = assert.ObjectsAreEqual // reference but no call
}
