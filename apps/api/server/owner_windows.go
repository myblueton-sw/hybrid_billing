//go:build windows

package server

import "io/fs"

func owned(fs.FileInfo) bool { return true }
