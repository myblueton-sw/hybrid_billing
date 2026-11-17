//go:build !windows

package decisionledger

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

type platform struct{}

func ownerOnly(info fs.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.Mode().Perm()&0o077 == 0 && int(st.Uid) == os.Getuid()
}

func (platform) openFile(path string, create bool) (handle, error) {
	flags := os.O_RDWR | os.O_APPEND | syscall.O_NOFOLLOW
	if create {
		flags |= os.O_CREATE | os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0o600)
	if errors.Is(err, syscall.ELOOP) {
		return nil, ErrUnsafe
	}
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || !ownerOnly(info) || info.Sys().(*syscall.Stat_t).Nlink != 1 {
		f.Close()
		return nil, ErrUnsafe
	}
	return f, nil
}

func (platform) lock(h handle) error {
	err := syscall.Flock(int(h.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return ErrLocked
	}
	return err
}

func (platform) checkDir(info fs.FileInfo) error {
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() || !ownerOnly(info) {
		return ErrUnsafe
	}
	return nil
}

func (platform) syncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

func (platform) rename(from, to string) error { return os.Rename(from, to) }
