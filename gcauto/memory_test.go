package gcauto

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNonZero(t *testing.T) {
	if TotalMemory() == 0 {
		t.Fatal("TotalMemory returned 0")
	}
	if FreeMemory() == 0 {
		t.Fatal("FreeMemory returned 0")
	}
}

func TestTotalMemory(t *testing.T) {
	fmt.Printf("Total system memory: %d\n", TotalMemory())
}
func TestFreeMemory(t *testing.T) {
	fmt.Printf("Free system memory: %d\n", FreeMemory())
}

func TestMem(t *testing.T) {
	is := assert.New(t)
	const mb = 1024 * 1024

	heap := make([]byte, 100*mb+1)
	inuse := readMemoryInuse()
	t.Logf("mem inuse: %d MB", inuse/mb)
	is.GreaterOrEqual(inuse, uint64(100*mb))
	heap[0] = 0
}
