// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lockedfile

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	// "github.com/sonnt85/errors"
	"github.com/sonnt85/gosutils/lockedfile/internal/filelock"

	"github.com/sonnt85/gosutils/endec"
)

// A Mutex provides mutual exclusion within and across processes by locking a
// well-known file. Such a file generally guards some other part of the
// filesystem: for example, a Mutex file in a directory might guard access to
// the entire tree rooted in that directory.
//
// Mutex does not implement sync.Locker: unlike a sync.Mutex, a lockedfile.Mutex
// can fail to lock (e.g. if there is a permission error in the filesystem).
//
// Like a sync.Mutex, a Mutex may be included as a field of a larger struct but
// must not be copied after first use. The Path field must be set before first
// use and must not be change thereafter.
type Mutex struct {
	Path string       // The path to the well-known lock file. Must be non-empty.
	mu   sync.RWMutex // A redundant mutex. The race detector doesn't know about file locking, so in tests we may need to lock something that it understands.
}

// MutexAt returns a new Mutex with Path set to the given non-empty path.
func MutexAt(path string) *Mutex {
	// if path == "" {
	// 	panic("lockedfile.MutexAt: path must be non-empty")
	// }
	return &Mutex{Path: path}
}

func (mu *Mutex) String() string {
	return fmt.Sprintf("lockedfile.Mutex(%s)", mu.Path)
}

// Lock attempts to lock the Mutex.
//
// If successful, Lock returns a non-nil unlock function: it is provided as a
// return-value instead of a separate method to remind the caller to check the
// accompanying error. (See https://golang.org/issue/20803.)
func (mu *Mutex) Lock() (f *File, unlock func(), err error) {
	if mu.Path == "" {
		return f, nil, MISSING_FILE
	}
	mu.mu.Lock()

	// We could use either O_RDWR or O_WRONLY here. If we choose O_RDWR and the
	// file at mu.Path is write-only, the call to OpenFile will fail with a
	// permission error. That's actually what we want: if we add an RLock method
	// in the future, it should call OpenFile with O_RDONLY and will require the
	// files must be readable, so we should not let the caller make any
	// assumptions about Mutex working with write-only files.
	f, err = OpenFile(mu.Path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		mu.mu.Unlock()
		return f, nil, err
	}

	return f, func() {
		f.Close()
		filelock.Unlock(f)
		mu.mu.Unlock()
	}, nil
}

func (mu *Mutex) TryLock() (f *File, unlock func(), err error) {
	if mu.Path == "" {
		return f, nil, MISSING_FILE
	}
	if !mu.mu.TryLock() {
		return f, nil, errors.New("can not trylock")
	}
	// We could use either O_RDWR or O_WRONLY here. If we choose O_RDWR and the
	// file at mu.Path is write-only, the call to OpenFile will fail with a
	// permission error. That's actually what we want: if we add an RLock method
	// in the future, it should call OpenFile with O_RDONLY and will require the
	// files must be readable, so we should not let the caller make any
	// assumptions about Mutex working with write-only files.vd
	f, err = OpenFile(mu.Path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		mu.mu.Unlock()
		return f, nil, err
	}

	return f, func() {
		f.Close()
		filelock.Unlock(f)
		mu.mu.Unlock()
	}, nil
}

func (mu *Mutex) RLock() (f *File, unlock func(), err error) {
	if mu.Path == "" {
		return f, nil, MISSING_FILE
	}
	mu.mu.RLock()

	f, err = OpenFile(mu.Path, os.O_RDONLY, 0666)
	if err != nil {
		mu.mu.Unlock()
		return f, nil, err
	}

	return f, func() {
		f.Close()
		filelock.Unlock(f)
		mu.mu.RUnlock()
	}, nil
}

func (mu *Mutex) TryRLock() (f *File, unlock func(), err error) {
	if mu.Path == "" {
		return f, nil, MISSING_FILE
		// return nil, errors.New("lockedfile.Mutex: missing Path during Lock")
	}

	if !mu.mu.TryRLock() {
		return f, nil, errors.New("can not trylock")
	}

	f, err = OpenFile(mu.Path, os.O_RDONLY, 0666)
	if err != nil {
		mu.mu.Unlock()
		return f, nil, err
	}

	return f, func() {
		f.Close()
		filelock.Unlock(f)
		mu.mu.RUnlock()
	}, nil
}

func (mu *Mutex) RLockTimeout(timeout time.Duration, intervalCheck ...time.Duration) (f *File, unlock func(), err error) {
	// if timeout == 0 {
	// 	return mu.RLock()
	// }
	timeoutAt := time.Now().Add(timeout)
	timeoutns := timeout.Nanoseconds()
	stepSleep := time.Duration(endec.RandRangeInt64(timeoutns/50, timeoutns/10))
	if len(intervalCheck) != 0 {
		stepSleep = intervalCheck[0]
	}
	for {
		f, unlock, err = mu.TryRLock()
		if err != nil {
			if timeoutns != 0 && time.Now().After(timeoutAt) {
				err = fmt.Errorf("Timeout")
				return
			} else {
				runtime.Gosched()
				time.Sleep(stepSleep)
			}
		} else {
			return
		}
	}
}

func (mu *Mutex) LockTimeout(timeout time.Duration, intervalCheck ...time.Duration) (f *File, unlock func(), err error) {
	// if timeout == 0 {
	// return mu.Lock()
	// }
	timeoutAt := time.Now().Add(timeout)
	timeoutns := timeout.Nanoseconds()
	stepSleep := time.Duration(endec.RandRangeInt64(timeoutns/50, timeoutns/10))
	//stepSleep := timeout / time.Nanosecond / 10
	if len(intervalCheck) != 0 {
		stepSleep = intervalCheck[0]
	}
	for {
		f, unlock, err = mu.TryLock()
		if err != nil {
			if timeoutns != 0 && time.Now().After(timeoutAt) {
				err = errors.Join(err, errors.New("Timeout "+timeout.String()))
				// err = fmt.Errorf("Timeout %s", timeout.String())
				return
			} else {
				runtime.Gosched()
				time.Sleep(stepSleep)
			}
		} else {
			return
		}
	}
}

func Lock(path string) (f *File, unlock func(), err error) {
	mu := MutexAt(path)
	return mu.Lock()
}

func LockTimeout(path string, timeout time.Duration, intervalCheck ...time.Duration) (f *File, nlock func(), err error) {
	mu := MutexAt(path)
	return mu.LockTimeout(timeout, intervalCheck...)
}

func RLockTimeout(path string, timeout time.Duration, intervalCheck ...time.Duration) (f *File, unlock func(), err error) {
	mu := MutexAt(path)
	return mu.RLockTimeout(timeout, intervalCheck...)
}
