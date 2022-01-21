package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidName(t *testing.T) {
	assert.False(t, isValidName(""))
	assert.False(t, isValidName("1-abc"))
	assert.False(t, isValidName("---"))
	assert.False(t, isValidName("abc_abc"))
	assert.False(t, isValidName("-abc-abc"))
	assert.True(t, isValidName("abc-abc"))
	assert.True(t, isValidName("abc-abc-"))
	assert.True(t, isValidName("abc-123"))
	assert.True(t, isValidName("abc-123-"))
	assert.True(t, isValidName("still-wood"))
	assert.True(t, isValidName("evserv2-fastapi"))
}