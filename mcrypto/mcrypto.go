// Package mcrypto contains general purpose functionality related to
// cryptography, notably related to unique identifiers, signing/verifying data,
// and encrypting/decrypting data
package mcrypto

import (
	"strings"
)

// TODO rather than have the NewSignerVerifier methods, it might be better to
// have a Secret type, which implements Signer/Verifier. That way when there's
// Encrypter/Decrypter interfaces then Secret can implement those too, and
// PublicKey/PrivateKey can implement their respective ones. There'll be a nice
// symmetry there, rather than having NewEncrypterDecrypter functions.

// Instead of outputing opaque hex garbage, this package opts to add a prefix to
// the garbage. Each "type" of string returned has its own character which is
// not found in the hex range (0-9, a-f), and in addition each also has a
// version character prefixed as well, in case something wants to be changed
// going forward.
//
// We keep the constant prefices here to ensure there's no conflicts across
// string types in this package.
const (
	uuidV0      = "0u" // u for uuid
	sigV0       = "0s" // s for signature
	encryptedV0 = "0n" // n for "n"-crypted, harharhar
	pubKeyV0    = "0l" // b for pub"l"ic key
	privKeyV0   = "0v" // v for pri"v"ate key
)

func stripPrefix(s, prefix string) (string, bool) {
	trimmed := strings.TrimPrefix(s, prefix)
	return trimmed, len(trimmed) < len(s)
}
