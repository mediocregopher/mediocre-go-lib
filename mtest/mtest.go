// Package mtest contains types and functions which are useful when writing
// tests
package mtest

import (
	crand "crypto/rand"
	"encoding/hex"
	"math/rand"
	"reflect"
	"time"
)

// Rand is a public instance of rand.Rand, seeded with the current
// nano-timestamp
var Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

// RandBytes returns n random bytes
func RandBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := crand.Read(b); err != nil {
		panic(err)
	}
	return b
}

// RandHex returns a random hex string which is n characters long
func RandHex(n int) string {
	b := RandBytes(hex.DecodedLen(n))
	return hex.EncodeToString(b)
}

// RandElement returns a random element from the given slice.
//
// If a weighting function is given then that function is used to weight each
// element of the slice relative to the others, based on whatever metric and
// scale is desired.  The weight function must be able to be called more than
// once on each element.
func RandElement(slice interface{}, weight func(i int) uint64) interface{} {
	v := reflect.ValueOf(slice)
	l := v.Len()

	if weight == nil {
		return v.Index(Rand.Intn(l)).Interface()
	}

	var totalWeight uint64
	for i := 0; i < l; i++ {
		totalWeight += weight(i)
	}

	target := Rand.Int63n(int64(totalWeight))
	for i := 0; i < l; i++ {
		w := int64(weight(i))
		target -= w
		if target < 0 {
			return v.Index(i).Interface()
		}
	}
	panic("should never get here, perhaps the weighting function is inconsistent?")
}
