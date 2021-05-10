package progressbar

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

type WinSize struct {
	Ws_row    uint16 // rows, in characters
	Ws_col    uint16 // columns, in characters
	Ws_xpixel uint16 // horizontal size, pixels
	Ws_ypixel uint16 // vertical size, pixels
}

// see: http://www.delorie.com/djgpp/doc/libc/libc_495.html
func GetWinSize() (*WinSize, error) {
	ws := &WinSize{}

	_, _, err := syscall.Syscall(
		uintptr(syscall.SYS_IOCTL),
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)

	if err != 0 {
		return nil, err
	}

	return ws, nil
}

// clear whole line and move cursor to leftmost of line
func Clear() {
	fmt.Print("\033[2K\033[0G")
}

func repeat(str string, count int) string {
	var out string

	for i := 0; i < count; i++ {
		out += str
	}

	return out
}

const (
	remain = 5
)

var (
	mutex = &sync.Mutex{}
)

func Show(percent float32) error {
	var (
		ws   *WinSize
		err  error
		ps   string
		half bool

		num   string
		pg    string
		space string

		pgl int
		l   int
	)

	mutex.Lock()
	defer mutex.Unlock()

	if ws, err = GetWinSize(); err != nil {
		return err
	}

	num = fmt.Sprintf("%.2f%%", percent*100)

	// 2 = two |
	// 7 = len(" 99.00%")
	pgl = int(ws.Ws_col) - remain - 2 - 7

	// if the third decimal is not zero
	half = int(percent*1000)%10 != 0

	// remove the third decimal
	percent = percent * 100 / 100

	count := percent * float32(pgl)
	pg = repeat("=", int(count))

	if half {
		pg += "-"
	}

	l = pgl - len(pg)
	if l > 0 {
		space = repeat(" ", l)
	}

	ps = pg + space

	Clear()

	if int(percent) == 1 {
		fmt.Print(fmt.Sprintf("|%s| %s\n", ps, num))
	} else {
		fmt.Print(fmt.Sprintf("|%s| %s", ps, num))
	}

	return nil
}
