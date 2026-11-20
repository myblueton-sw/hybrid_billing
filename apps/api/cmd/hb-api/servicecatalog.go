package main

import (
	"errors"
	"io"
	"os"

	"github.com/myblueton-sw/hybrid_billing/modules/servicecatalog"
)

var errServiceCatalog = errors.New("service catalog configuration could not be loaded")

type catalogFileFlag struct {
	path     string
	supplied bool
	invalid  bool
}

func (f *catalogFileFlag) String() string { return "" }

func (f *catalogFileFlag) Set(path string) error {
	if f.supplied || path == "" {
		f.invalid = true
	}
	f.supplied = true
	f.path = path
	return nil
}

func (f *catalogFileFlag) load() ([]byte, error) {
	if f.invalid {
		return nil, errServiceCatalog
	}
	if !f.supplied {
		return nil, nil
	}
	info, err := os.Stat(f.path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > servicecatalog.MaxBytes {
		return nil, errServiceCatalog
	}
	file, err := os.Open(f.path)
	if err != nil {
		return nil, errServiceCatalog
	}
	info, err = file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > servicecatalog.MaxBytes {
		file.Close()
		return nil, errServiceCatalog
	}
	data, readErr := io.ReadAll(io.LimitReader(file, servicecatalog.MaxBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(data) == 0 || len(data) > servicecatalog.MaxBytes {
		return nil, errServiceCatalog
	}
	return data, nil
}
