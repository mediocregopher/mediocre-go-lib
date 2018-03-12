// Package mcrypto contains general purpose functionality related to
// cryptography, notably related to unique identifiers, signing/verifying data,
// and encrypting/decrypting data
package mcrypto

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
)

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
	exSigV0     = "0t" // t for time
	uniqueSigV0 = "0q" // q for uni"q"ue
	encryptedV0 = "0n" // n for "n"-crypted, harharhar
)

func stripPrefix(s, prefix string) (string, bool) {
	trimmed := strings.TrimPrefix(s, prefix)
	return trimmed, len(trimmed) < len(s)
}

func prefixReader(r io.Reader, prefix []byte) io.Reader {
	b := make([]byte, 0, len(prefix)+hex.EncodedLen(strconv.IntSize)+2)
	buf := bytes.NewBuffer(b)
	fmt.Fprintf(buf, "%x\n", len(prefix))
	buf.Write(prefix)
	buf.WriteByte('\n')
	return io.MultiReader(buf, r)
}
