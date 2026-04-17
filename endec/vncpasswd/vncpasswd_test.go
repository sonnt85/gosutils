package vncpasswd

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// dumpByteSlice returns a hex+ASCII dump of b, laid out like `hexdump -C`.
// Returned as a string so callers can route it through t.Log instead of
// writing to stdout from inside a test.
func dumpByteSlice(b []byte) string {
	var sb strings.Builder
	var a [16]byte
	n := (len(b) + 15) &^ 15
	for i := 0; i < n; i++ {
		if i%16 == 0 {
			fmt.Fprintf(&sb, "%4d", i)
		}
		if i%8 == 0 {
			sb.WriteByte(' ')
		}
		if i < len(b) {
			fmt.Fprintf(&sb, " %02X", b[i])
		} else {
			sb.WriteString("   ")
		}
		if i >= len(b) {
			a[i%16] = ' '
		} else if b[i] < 32 || b[i] > 126 {
			a[i%16] = '.'
		} else {
			a[i%16] = b[i]
		}
		if i%16 == 15 {
			fmt.Fprintf(&sb, "  %s\n", string(a[:]))
		}
	}
	return sb.String()
}

func TestVncpasswd(t *testing.T) {
	b := VncEncryptPasswd("hatuanson")
	s := VncEncryptPasswdToHexString("nongthon")
	t.Logf("hex: %s", s)
	t.Logf("dump:\n%s", dumpByteSlice(b))

	b, err := hex.DecodeString("F61E24D88C63963D")
	require.Nil(t, err)
	if s, ok := VncDecryptPasswd(b); ok {
		t.Logf("decode: %s", s)
	} else {
		t.Error("VncDecryptPasswd failed to decode known-good input")
	}
}
