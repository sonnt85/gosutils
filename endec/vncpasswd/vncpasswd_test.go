package vncpasswd

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func dumpByteSlice(b []byte) {
	// os.Stdout
	var a [16]byte
	n := (len(b) + 15) &^ 15
	for i := 0; i < n; i++ {
		if i%16 == 0 {
			fmt.Printf("%4d", i)
		}
		if i%8 == 0 {
			fmt.Print(" ")
		}
		if i < len(b) {
			fmt.Printf(" %02X", b[i])
		} else {
			fmt.Print("   ")
		}
		if i >= len(b) {
			a[i%16] = ' '
		} else if b[i] < 32 || b[i] > 126 {
			a[i%16] = '.'
		} else {
			a[i%16] = b[i]
		}
		if i%16 == 15 {
			fmt.Printf("  %s\n", string(a[:]))
		}
	}
}
func TestVncpasswd(t *testing.T) {
	b := VncEncryptPasswd("hatuanson")
	s := VncEncryptPasswdToHexString("nongthon")
	fmt.Println(s)
	dumpByteSlice(b)
	b, err := hex.DecodeString("F61E24D88C63963D")
	require.Nil(t, err)
	// dumpByteSlice(b)
	s, ok := VncDecryptPasswd(b)
	if ok {
		fmt.Println("decode:", s)
	}
}
