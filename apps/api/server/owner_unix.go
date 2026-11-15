//go:build !windows

package server

import (
	"io/fs"
	"os"
	"syscall"
)

func owned(info fs.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) == os.Getuid()
}
