package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidName(t *testing.T) {
	assert.False(t, IsValidName(""))
	assert.False(t, IsValidName("1-abc"))
	assert.False(t, IsValidName("---"))
	assert.False(t, IsValidName("abc_abc"))
	assert.False(t, IsValidName("-abc-abc"))
	assert.True(t, IsValidName("abc-abc"))
	assert.True(t, IsValidName("abc-abc-"))
	assert.True(t, IsValidName("abc-123"))
	assert.True(t, IsValidName("abc-123-"))
	assert.True(t, IsValidName("still-wood"))
	assert.True(t, IsValidName("evserv2-fastapi"))
}

// Service Types
func TestValidateRuntime(t *testing.T) {
	assert.False(t, isValidRuntime(""))
	assert.False(t, isValidRuntime("not_supported"))
	assert.False(t, isValidRuntime("fastapi"))
	assert.True(t, isValidRuntime("go"))
	assert.True(t, isValidRuntime("nodejs"))
	assert.True(t, isValidRuntime("python"))
}
