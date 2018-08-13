// Package mrand implements extensions and conveniences for using the default
// math/rand package.
package mrand

import (
	"encoding/hex"
	"math/rand"
	"reflect"
	"time"
)

// Rand extends the default rand.Rand type with extra functionality.
type Rand struct {
	*rand.Rand
}

// Bytes returns n random bytes.
func (r Rand) Bytes(n int) []byte {
	b := make([]byte, n)
	if _, err := r.Read(b); err != nil {
		panic(err)
	}
	return b
}

// Hex returns a random hex string which is n characters long.
func (r Rand) Hex(n int) string {
	origN := n
	if n%2 == 1 {
		n++
	}
	b := r.Bytes(hex.DecodedLen(n))
	return hex.EncodeToString(b)[:origN]
}

// Element returns a random element from the given slice.
//
// If a weighting function is given then that function is used to weight each
// element of the slice relative to the others, based on whatever metric and
// scale is desired.  The weight function must be able to be called more than
// once on each element.
func (r Rand) Element(slice interface{}, weight func(i int) uint64) interface{} {
	v := reflect.ValueOf(slice)
	l := v.Len()

	if weight == nil {
		return v.Index(r.Intn(l)).Interface()
	}

	var totalWeight uint64
	for i := 0; i < l; i++ {
		totalWeight += weight(i)
	}

	target := r.Int63n(int64(totalWeight))
	for i := 0; i < l; i++ {
		w := int64(weight(i))
		target -= w
		if target < 0 {
			return v.Index(i).Interface()
		}
	}
	panic("should never get here, perhaps the weighting function is inconsistent?")
}

////////////////////////////////////////////////////////////////////////////////

// DefaultRand is an instance off Rand whose methods are directly exported by
// this package for convenience.
var DefaultRand = Rand{Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}

// Methods off DefaultRand exported to the top level of this package.
var (
	ExpFloat64  = DefaultRand.ExpFloat64
	Float32     = DefaultRand.Float32
	Float64     = DefaultRand.Float64
	Int         = DefaultRand.Int
	Int31       = DefaultRand.Int31
	Int31n      = DefaultRand.Int31n
	Int63       = DefaultRand.Int63
	Int63n      = DefaultRand.Int63n
	Intn        = DefaultRand.Intn
	NormFloat64 = DefaultRand.NormFloat64
	Perm        = DefaultRand.Perm
	Read        = DefaultRand.Read
	Seed        = DefaultRand.Seed
	Shuffle     = DefaultRand.Shuffle
	Uint32      = DefaultRand.Uint32
	Uint64      = DefaultRand.Uint64

	// extended methods
	Bytes   = DefaultRand.Bytes
	Hex     = DefaultRand.Hex
	Element = DefaultRand.Element
)
