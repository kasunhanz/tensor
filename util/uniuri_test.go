package util

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestUniqueNew(t *testing.T) {
	for i := 0; i < 10000; i++ {
		one := UniqueNew()
		two := UniqueNew()
		assert.NotEqual(t, one, two, "should not equal")
	}

	one := UniqueNew()
	for i := 0; i < 10000; i++ {
		two := UniqueNew()
		assert.NotEqual(t, one, two, "should not equal")
	}
}

func TestUniqueNewLen(t *testing.T) {
	for i := 0; i < 10000; i++ {
		one := UniqueNewLen(UUIDLen)
		two := UniqueNewLen(UUIDLen)
		assert.NotEqual(t, one, two, "should not equal")
	}

	one := UniqueNewLen(UUIDLen)
	for i := 0; i < 10000; i++ {
		two := UniqueNewLen(UUIDLen)
		assert.NotEqual(t, one, two, "should not equal")
	}
}