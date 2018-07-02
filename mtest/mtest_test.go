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

func TestRandElement(t *T) {
	slice := []uint64{1, 2, 3} // values are also each value's weight
	total := func() uint64 {
		var t uint64
		for i := range slice {
			t += slice[i]
		}
		return t
	}()
	m := map[uint64]uint64{}

	iterations := 100000
	for i := 0; i < iterations; i++ {
		el := RandElement(slice, func(i int) uint64 { return slice[i] }).(uint64)
		m[el]++
	}

	for i := range slice {
		t.Logf("%d -> %d (%f)", slice[i], m[slice[i]], float64(m[slice[i]])/float64(iterations))
	}

	assertEl := func(i int) {
		el, elF := slice[i], float64(slice[i])
		gotRatio := float64(m[el]) / float64(iterations)
		expRatio := elF / float64(total)
		diff := (gotRatio - expRatio) / expRatio
		if diff > 0.1 || diff < -0.1 {
			t.Fatalf("ratio of element %d is off: got %f, expected %f (diff:%f)", el, gotRatio, expRatio, diff)
		}
	}

	for i := range slice {
		assertEl(i)
	}
}
