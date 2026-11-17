//go:build windows

package decisionledger

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
	"unsafe"
)

const (
	lockExclusive       = 0x2
	lockFailImmediately = 0x1
	lockViolation       = syscall.Errno(33)
)

var lockFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("LockFileEx")

type platform struct{}

func (platform) openFile(path string, create bool) (handle, error) {
	before, err := os.Lstat(path)
	switch {
	case create && err == nil:
		return nil, fs.ErrExist
	case !create && err != nil:
		return nil, err
	case err == nil && !before.Mode().IsRegular():
		return nil, ErrUnsafe
	}
	flags := os.O_RDWR
	if create {
		flags |= os.O_CREATE | os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		return nil, err
	}
	after, err := f.Stat()
	if err != nil || !after.Mode().IsRegular() || (!create && !os.SameFile(before, after)) {
		f.Close()
		return nil, ErrUnsafe
	}
	return f, nil
}

func (platform) lock(h handle) error {
	var overlapped syscall.Overlapped
	r, _, err := lockFileEx.Call(h.Fd(), lockExclusive|lockFailImmediately, 0, 0xFFFFFFFF, 0xFFFFFFFF, uintptr(unsafe.Pointer(&overlapped)))
	if r != 0 {
		return nil
	}
	if errors.Is(err, lockViolation) {
		return ErrLocked
	}
	return err
}

func (platform) checkDir(info fs.FileInfo) error {
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return ErrUnsafe
	}
	return nil
}

func (platform) syncDir(string) error { return nil }

func (platform) rename(from, to string) error { return os.Rename(from, to) }
