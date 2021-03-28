package main

import (
	"io/fs"
	"os"
	"strings"
)

type suffixDirFS string

func (dir suffixDirFS) Open(name string) (fs.File, error) {
	f, err := udfOpen(string(dir)+"/"+name, suffix)
	if err != nil {
		return nil, err
	}
	return f, nil
}

type suffixFile struct {
	*os.File
	suffix string
}

func udfOpen(name, suffix string) (*suffixFile, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return &suffixFile{f, suffix}, nil
}

func (f suffixFile) ReadDir(count int) ([]fs.DirEntry, error) {
	entries, err := f.File.ReadDir(count)
	if err != nil {
		return nil, err
	}
	var newEntries []fs.DirEntry

	for _, entry := range entries {
		if !entry.IsDir() {
			ss := strings.Split(entry.Name(), ".")
			if f.suffix != "" && ss[len(ss)-1] != f.suffix {
				continue
			}
		}
		newEntries = append(newEntries, entry)
	}
	return newEntries, nil
}
