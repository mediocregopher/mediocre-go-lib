// Package mtest contains types and functions which are useful when writing
// tests
package mtest

import (
	crand "crypto/rand"
	"encoding/hex"
	"math/rand"
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
