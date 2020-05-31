package bufcopy

import (
	"bytes"
	"io"
	"testing"
)

var str = bytes.Repeat([]byte("ABC"), 1000)

func TestBufCopy(t *testing.T) {
	buf := New()
	src := bytes.NewReader(str)
	var dst bytes.Buffer
	written, err := buf.Copy(&dst, src)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("written: %d", written)
	if l := (int64)(dst.Len()); l != written {
		t.Fatal("incorrect content")
	}
}

func BenchmarkBufCopy(b *testing.B) {
	buf := New()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			src := bytes.NewReader(str)
			var dst bytes.Buffer
			_, err := buf.Copy(&dst, src)
			if err != nil {
				b.Error(err)
			}
		}
	})
}

func BenchmarkIoCopy(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			src := bytes.NewReader(str)
			var dst bytes.Buffer
			_, err := io.Copy(&dst, src)
			if err != nil {
				b.Error(err)
			}
		}
	})
}
