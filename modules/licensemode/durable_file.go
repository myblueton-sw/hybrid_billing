package licensemode

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

const maxCheckpointBytes = 64 << 20

type durableFile struct {
	dir     string
	lock    *os.File
	fault   func(string) bool
	initial bool
}

func privateRegular(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return ErrStorage
	}
	return nil
}

func openDurableFile(dir string, create bool) (*durableFile, error) {
	if create {
		if err := os.Mkdir(dir, 0700); err != nil {
			return nil, errors.Join(ErrStorage, err)
		}
		parent, err := os.Open(filepath.Dir(dir))
		if err != nil {
			return nil, ErrStorage
		}
		if err = errors.Join(parent.Sync(), parent.Close()); err != nil {
			return nil, ErrStorage
		}
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, ErrStorage
	}
	lock, err := os.OpenFile(filepath.Join(dir, "owner"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, errors.Join(ErrOwned, err)
	}
	s := &durableFile{dir: dir, lock: lock, initial: create}
	if err = lock.Sync(); err == nil {
		err = s.syncDir()
	}
	if err != nil {
		_ = s.close()
		return nil, ErrStorage
	}
	path := filepath.Join(dir, "state.json")
	if create {
		if _, err = os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			_ = s.close()
			return nil, ErrStorage
		}
	} else if err = privateRegular(path); err != nil {
		_ = s.close()
		return nil, ErrStorage
	}
	return s, nil
}
func (s *durableFile) fails(step string) bool { return s.fault != nil && s.fault(step) }
func (s *durableFile) syncDir() error {
	dir, err := os.Open(s.dir)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}
func (s *durableFile) read() ([]byte, error) {
	path := filepath.Join(s.dir, "state.json")
	if err := privateRegular(path); err != nil {
		return nil, ErrStorage
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, ErrStorage
	}
	data, err := io.ReadAll(io.LimitReader(file, maxCheckpointBytes+1))
	err = errors.Join(err, file.Close())
	if err != nil || len(data) > maxCheckpointBytes {
		return nil, ErrCheckpoint
	}
	return data, nil
}
func (s *durableFile) write(data []byte) error {
	if len(data) > maxCheckpointBytes || s.fails("create") {
		return ErrStorage
	}
	path := filepath.Join(s.dir, "state.json")
	if err := privateRegular(path); err != nil && !(s.initial && errors.Is(err, os.ErrNotExist)) {
		return ErrStorage
	}
	temp, err := os.CreateTemp(s.dir, ".state-")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if s.fails("write") {
		_ = temp.Close()
		return ErrStorage
	}
	n, err := temp.Write(data)
	if n != len(data) {
		err = errors.Join(err, io.ErrShortWrite)
	}
	if err == nil {
		if s.fails("sync") {
			err = ErrStorage
		} else {
			err = temp.Sync()
		}
	}
	err = errors.Join(err, temp.Close())
	if s.fails("close") {
		err = ErrStorage
	}
	if err != nil {
		return err
	}
	if s.fails("rename") {
		return ErrStorage
	}
	if err = os.Rename(name, path); err != nil {
		return err
	}
	s.initial = false
	if s.fails("directory_sync") {
		return ErrStorage
	}
	return s.syncDir()
}
func (s *durableFile) close() error {
	if s.lock == nil {
		return nil
	}
	err := s.lock.Close()
	s.lock = nil
	if err != nil {
		return errors.Join(ErrStorage, err)
	}
	if err = os.Remove(filepath.Join(s.dir, "owner")); err != nil {
		return errors.Join(ErrStorage, err)
	}
	if err = s.syncDir(); err != nil {
		return errors.Join(ErrStorage, err)
	}
	return nil
}
