package mcrypto

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mtest"
	"github.com/stretchr/testify/assert"
)

func TestKeyPair(t *T) {
	pub, priv := NewWeakKeyPair()

	// test signing/verifying
	str := mtest.RandHex(512)
	sig := SignString(priv, str)
	assert.NoError(t, VerifyString(pub, sig, str))
}
