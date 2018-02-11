package mtest

import (
	. "testing"

	"github.com/stretchr/testify/assert"
)

func TestRandBytes(t *T) {
	var prev []byte
	for i := 0; i < 10000; i++ {
		curr := RandBytes(16)
		assert.Len(t, curr, 16)
		assert.NotEqual(t, prev, curr)
		prev = curr
	}
}

func TestRandHex(t *T) {
	// RandHex is basically a wrapper of RandBytes, so we don't have to test it
	// much
	assert.Len(t, RandHex(16), 16)
}
