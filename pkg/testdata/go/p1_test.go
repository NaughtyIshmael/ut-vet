package example

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SHOULD TRIGGER: error-not-checked — error assigned to _
func TestSaveIgnoreError(t *testing.T) {
	_, _ = Save(entity)
	assert.True(t, true)
}

// SHOULD TRIGGER: error-not-checked — error var not checked
func TestSaveNoCheck(t *testing.T) {
	result, err := Save(entity)
	_ = err
	assert.NotNil(t, result)
}

// SHOULD NOT TRIGGER: error is checked
func TestSaveChecked(t *testing.T) {
	result, err := Save(entity)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// SHOULD TRIGGER: zero-value-input — all args are zero-values
func TestCreateUserZero(t *testing.T) {
	user, err := CreateUser("", 0, false)
	require.NoError(t, err)
	assert.NotNil(t, user)
}

// SHOULD NOT TRIGGER: meaningful inputs
func TestCreateUserReal(t *testing.T) {
	user, err := CreateUser("Alice", 30, true)
	require.NoError(t, err)
	assert.Equal(t, "Alice", user.Name)
}

// SHOULD TRIGGER: only-nil-check — only checks error, ignores result
func TestGetUserOnlyErr(t *testing.T) {
	_, err := GetUser(42)
	assert.NoError(t, err)
}

// SHOULD NOT TRIGGER: checks both error and result
func TestGetUserFull(t *testing.T) {
	user, err := GetUser(42)
	require.NoError(t, err)
	assert.Equal(t, "Alice", user.Name)
}
