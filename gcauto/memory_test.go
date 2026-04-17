package gcauto

import (
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
	t.Logf("Total system memory: %d", TotalMemory())
}
func TestFreeMemory(t *testing.T) {
	t.Logf("Free system memory: %d", FreeMemory())
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
