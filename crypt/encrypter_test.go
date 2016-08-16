package crypt

import (
	"testing"
)

func TestDecrypt(t *testing.T) {
	expected := "Hello World"
	cryptvalue := Encrypt(expected)
	actual := Decrypt(cryptvalue)

	if actual != expected {
		t.Errorf("Decrypt(%s): expected %s, actual %s", expected, expected, actual)
	}
}
