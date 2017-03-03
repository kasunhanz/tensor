package util

import (
	"testing"
)

func TestDecrypt(t *testing.T) {
	expected := "Hello World"
	cryptvalue := Cipher(expected)
	actual := Decipher(cryptvalue)

	if actual != expected {
		t.Errorf("Decrypt(%s): expected %s, actual %s", expected, expected, actual)
	}
}
