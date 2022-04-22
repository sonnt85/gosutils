// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !plan9

package filelock

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// HasExec reports whether the current system can start new processes
// using os.StartProcess or (more commonly) exec.Command.
func HasExec() bool {
	switch runtime.GOOS {
	case "js", "ios":
		return false
	}
	return true
}

// MustHaveExec checks that the current system can start new processes
// using os.StartProcess or (more commonly) exec.Command.
// If not, MustHaveExec calls t.Skip with an explanation.
func MustHaveExec(t testing.TB) {
	if !HasExec() {
		t.Skipf("skipping test: cannot exec subprocess on %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

func testlock(t *testing.T, f *os.File) {
	t.Helper()
	err := Lock(f)
	t.Logf("Lock(fd %d) = %v", f.Fd(), err)
	if err != nil {
		t.Fail()
	}
}

func rLock(t *testing.T, f *os.File) {
	t.Helper()
	err := RLock(f)
	t.Logf("RLock(fd %d) = %v", f.Fd(), err)
	if err != nil {
		t.Fail()
	}
}

func testunlock(t *testing.T, f *os.File) {
	t.Helper()
	err := Unlock(f)
	t.Logf("Unlock(fd %d) = %v", f.Fd(), err)
	if err != nil {
		t.Fail()
	}
}

func mustTempFile(t *testing.T) (f *os.File, remove func()) {
	t.Helper()

	base := filepath.Base(t.Name())
	f, err := os.CreateTemp("", base)
	if err != nil {
		t.Fatalf(`os.CreateTemp("", %q) = %v`, base, err)
	}
	t.Logf("fd %d = %s", f.Fd(), f.Name())

	return f, func() {
		f.Close()
		os.Remove(f.Name())
	}
}

func mustOpen(t *testing.T, name string) *os.File {
	t.Helper()

	f, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("os.Open(%q) = %v", name, err)
	}

	t.Logf("fd %d = os.Open(%q)", f.Fd(), name)
	return f
}

const (
	quiescent            = 10 * time.Millisecond
	probablyStillBlocked = 10 * time.Second
)

func mustBlock(t *testing.T, op string, f *os.File) (wait func(*testing.T)) {
	t.Helper()

	desc := fmt.Sprintf("%s(fd %d)", op, f.Fd())

	done := make(chan struct{})
	go func() {
		t.Helper()
		switch op {
		case "Lock":
			testlock(t, f)
		case "RLock":
			rLock(t, f)
		default:
			panic("invalid op: " + op)
		}
		close(done)
	}()

	select {
	case <-done:
		t.Fatalf("%s unexpectedly did not block", desc)
		return nil

	case <-time.After(quiescent):
		t.Logf("%s is blocked (as expected)", desc)
		return func(t *testing.T) {
			t.Helper()
			select {
			case <-time.After(probablyStillBlocked):
				t.Fatalf("%s is unexpectedly still blocked", desc)
			case <-done:
			}
		}
	}
}

func TestLockExcludesLock(t *testing.T) {
	t.Parallel()

	f, remove := mustTempFile(t)
	defer remove()

	other := mustOpen(t, f.Name())
	defer other.Close()

	testlock(t, f)
	lockOther := mustBlock(t, "Lock", other)
	testunlock(t, f)
	lockOther(t)
	testunlock(t, other)
}

func TestLockExcludesRLock(t *testing.T) {
	t.Parallel()

	f, remove := mustTempFile(t)
	defer remove()

	other := mustOpen(t, f.Name())
	defer other.Close()

	testlock(t, f)
	rLockOther := mustBlock(t, "RLock", other)
	testunlock(t, f)
	rLockOther(t)
	testunlock(t, other)
}

func TestRLockExcludesOnlyLock(t *testing.T) {
	t.Parallel()

	f, remove := mustTempFile(t)
	defer remove()
	rLock(t, f)

	f2 := mustOpen(t, f.Name())
	defer f2.Close()

	doUnlockTF := false
	switch runtime.GOOS {
	case "aix", "solaris":
		// When using POSIX locks (as on Solaris), we can't safely read-lock the
		// same inode through two different descriptors at the same time: when the
		// first descriptor is closed, the second descriptor would still be open but
		// silently unlocked. So a second RLock must block instead of proceeding.
		lockF2 := mustBlock(t, "RLock", f2)
		testunlock(t, f)
		lockF2(t)
	default:
		rLock(t, f2)
		doUnlockTF = true
	}

	other := mustOpen(t, f.Name())
	defer other.Close()
	lockOther := mustBlock(t, "Lock", other)

	testunlock(t, f2)
	if doUnlockTF {
		testunlock(t, f)
	}
	lockOther(t)
	testunlock(t, other)
}

func TestLockNotDroppedByExecCommand(t *testing.T) {
	MustHaveExec(t)

	f, remove := mustTempFile(t)
	defer remove()

	testlock(t, f)

	other := mustOpen(t, f.Name())
	defer other.Close()

	// Some kinds of file locks are dropped when a duplicated or forked file
	// descriptor is unlocked. Double-check that the approach used by os/exec does
	// not accidentally drop locks.
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	if err := cmd.Run(); err != nil {
		t.Fatalf("exec failed: %v", err)
	}

	lockOther := mustBlock(t, "Lock", other)
	testunlock(t, f)
	lockOther(t)
	testunlock(t, other)
}
