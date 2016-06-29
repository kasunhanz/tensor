package crypt

import (
	"testing"
)

func TestDecrypt(t *testing.T) {
	expected := "Hello Word"
	cryptvalue := Encrypt(expected)
	actual := Decrypt(cryptvalue)

	if actual != expected {
		t.Errorf("Decrypt(%d): expected %d, actual %d", expected, expected, actual)
	}
}