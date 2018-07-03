package mcrypto

import (
	. "testing"

	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/stretchr/testify/assert"
)

func TestKeyPair(t *T) {
	pub, priv := NewWeakKeyPair()

	// test signing/verifying
	str := mrand.Hex(512)
	sig := SignString(priv, str)
	assert.NoError(t, VerifyString(pub, sig, str))
}
