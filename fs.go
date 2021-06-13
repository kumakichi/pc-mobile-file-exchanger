package main

import (
	"bytes"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	toParentPatch = "<a style='top:0;left:0;position:fixed;z-index:9999;' href=window.location.href+/../>toParent/</a></br><a href=window.location.href+/../../>toParent2/</a></br>"
)

var (
	htmlTagReg = regexp.MustCompile(`(<html[^>]*>)`)
)

type suffixDirFS string

func (dir suffixDirFS) Open(name string) (fs.File, error) {
	f, err := udfOpen(string(dir)+"/"+name, filterSuffix)
	if err != nil {
		return nil, err
	}
	return f, nil
}

type suffixFile struct {
	*os.File
	fileSuffix   string
	filterSuffix string
}

func udfOpen(name, filterSuffix string) (*suffixFile, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	return &suffixFile{f, filepath.Ext(name), filterSuffix}, nil
}

type sizeFileInfo struct {
	os.FileInfo
}

func (s sizeFileInfo) Size() int64 {
	if filepath.Ext(s.Name()) == ".html" && flag.Lookup(patchHtmlName).Value.String() == "true" {
		return s.FileInfo.Size() + int64(len(toParentPatch))
	}
	return s.FileInfo.Size()
}

func (f *suffixFile) Stat() (fs.FileInfo, error) {
	fi, err := f.File.Stat()
	return sizeFileInfo{fi}, err
}

func (f *suffixFile) Read(b []byte) (int, error) {
	n, err := f.File.Read(b)
	length := len(b)
	if f.fileSuffix == ".html" && flag.Lookup(patchHtmlName).Value.String() == "true" {
		tmp := b
		hTags := htmlTagReg.FindSubmatch(b)
		if len(hTags) == 2 {
			tmp = bytes.Replace(b, hTags[1], append(hTags[1], []byte(toParentPatch)...), 1)[:length]
			f.File.Seek(int64(-len(toParentPatch)), os.SEEK_CUR)

		}
		tmp = bytes.Replace(tmp, []byte("<base href="), []byte("<bbse href="), 1) // ignore base href
		for i := 0; i < length; i++ {
			b[i] = tmp[i]
		}
	}

	return n, err
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
			if f.filterSuffix != "" && ss[len(ss)-1] != f.filterSuffix {
				continue
			}
		}
		newEntries = append(newEntries, entry)
	}
	return newEntries, nil
}
