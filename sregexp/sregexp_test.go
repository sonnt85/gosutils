package sregexp

import (
	"fmt"
	"strings"
	"testing"
)

func TestSregex(t *testing.T) {
	str := "1.2.3.4:2024-ss"
	if New("^[0-9,.:]+-ss").MatchString(str) {
		fmt.Print(strings.Split(str, "-")[0])
	}
}
