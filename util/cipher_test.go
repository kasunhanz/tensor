package util

import (
	"testing"
)

func TestDecrypt(t *testing.T) {
	expected := "Hello World"
	cryptvalue := CipherEncrypt(expected)
	actual := CipherDecrypt(cryptvalue)

	if actual != expected {
		t.Errorf("Decrypt(%s): expected %s, actual %s", expected, expected, actual)
	}
}
