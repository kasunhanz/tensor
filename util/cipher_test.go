package util

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestDecrypt(t *testing.T) {
	expected := "Hello World"
	cryptvalue := Cipher(expected)
	actual := Decipher(cryptvalue)

	assert.Equal(t, expected, string(actual))
}
