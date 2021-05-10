package progressbar

import (
	"fmt"
	"testing"
	"time"
)

type winsize struct {
	ws_row    uint16
	ws_col    uint16
	ws_xpixel uint16
	ws_ypixel uint16
}

func TestT2(t *testing.T) {
	for i := 1; i <= 1000; i++ {
		time.Sleep(time.Millisecond * 10)
		Show(float32(i) / 1000.00)
	}

	fmt.Println()
}
