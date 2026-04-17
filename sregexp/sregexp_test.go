package sregexp

import (
	"testing"
)

func TestSregex(t *testing.T) {
	str := "1.2.3.4:2024-ss"
	if !New("^[0-9,.:]+-ss").MatchString(str) {
		t.Errorf("expected %q to match ^[0-9,.:]+-ss", str)
	}
}
