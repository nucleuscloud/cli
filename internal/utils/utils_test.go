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
	assert.False(t, IsValidRuntime(""))
	assert.False(t, IsValidRuntime("not_supported"))
	assert.False(t, IsValidRuntime("fastapi"))
	assert.True(t, IsValidRuntime("go"))
	assert.True(t, IsValidRuntime("nodejs"))
	assert.True(t, IsValidRuntime("python"))
	assert.True(t, IsValidRuntime("docker"))
	assert.True(t, IsValidRuntime("ruby"))
	assert.True(t, IsValidRuntime("java"))
	assert.True(t, IsValidRuntime("dotnet"))
}
