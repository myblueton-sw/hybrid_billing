//go:build !windows

package decisionledger

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestUnixOpeningRefusesUnsafeFiles(t *testing.T) {
	for name, prepare := range map[string]func(t *testing.T, o Options){
		"symlinked file": func(t *testing.T, o Options) {
			real := filepath.Join(o.Dir, "elsewhere")
			os.Rename(ledgerPath(o), real)
			os.Symlink(real, ledgerPath(o))
		},
		"hard-linked file": func(t *testing.T, o Options) { os.Link(ledgerPath(o), filepath.Join(o.Dir, "second-name")) },
		"group-readable":   func(t *testing.T, o Options) { os.Chmod(ledgerPath(o), 0o644) },
		"open directory":   func(t *testing.T, o Options) { os.Chmod(filepath.Join(o.Dir, DirName), 0o755) },
		"symlinked directory": func(t *testing.T, o Options) {
			dir := filepath.Join(o.Dir, DirName)
			os.Rename(dir, dir+"-real")
			os.Symlink(dir+"-real", dir)
		},
		"directory is a file": func(t *testing.T, o Options) {
			dir := filepath.Join(o.Dir, DirName)
			os.RemoveAll(dir)
			os.WriteFile(dir, nil, 0o600)
		},
	} {
		t.Run(name, func(t *testing.T) {
			o := options(t)
			existing(t, &o)
			prepare(t, o)
			if _, err := Open(o); !errors.Is(err, ErrUnsafe) {
				t.Errorf("%v", err)
			}
		})
	}
}

func TestUnixPlatformEdges(t *testing.T) {
	o := options(t)
	existing(t, &o)
	p := platform{}
	if _, err := p.openFile(ledgerPath(o), true); err == nil || errors.Is(err, ErrUnsafe) {
		t.Errorf("exclusive create over an existing file %v", err)
	}
	h, err := p.openFile(ledgerPath(o), false)
	if err != nil {
		t.Fatal(err)
	}
	h.Close()
	if err := p.lock(h); err == nil || errors.Is(err, ErrLocked) {
		t.Errorf("lock on a closed handle %v", err)
	}
	if err := p.syncDir(filepath.Join(o.Dir, "absent")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("sync of a missing directory %v", err)
	}
	fifo := filepath.Join(o.Dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	if h, err := p.openFile(fifo, false); !errors.Is(err, ErrUnsafe) {
		t.Errorf("a named pipe %v", err)
		if h != nil {
			h.Close()
		}
	}
}
