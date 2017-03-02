package util

import (
	"testing"
	"io/ioutil"
	"github.com/stretchr/testify/assert"
	"strings"
)

func TestVersion(t *testing.T) {
	dat, _ := ioutil.ReadFile("../VERSION")

	str := strings.Split(string(dat), " ")
	assert.Equal(t, Version, str[0], "Version should be equal")
}