package fs

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	ToParentPatch = "<a style='top:0;left:0;position:fixed;z-index:9999;' href=window.location.href+/../>toParent/</a>"
)

var (
	PatchHTMLName = "pp"
	HTMLTagReg    = regexp.MustCompile(`(<html[^>]*>)`)
	filterSuffix  string
)

// SuffixDirFS is a custom filesystem that filters files by suffix
type SuffixDirFS string

// SetFilterSuffix sets the global filter suffix
func SetFilterSuffix(suffix string) {
	filterSuffix = suffix
}

// Open implements fs.FS interface
func (dir SuffixDirFS) Open(name string) (fs.File, error) {
	return UdfOpen(string(dir)+"/"+name, filterSuffix)
}

// SuffixFile is a custom file implementation with suffix filtering
type SuffixFile struct {
	*os.File
	FileSuffix   string
	FilterSuffix string
}

// UdfOpen opens a file with the specified filter suffix
func UdfOpen(name, filterSuffix string) (*SuffixFile, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open err: %w", err)
	}

	return &SuffixFile{f, filepath.Ext(name), filterSuffix}, nil
}

// SizeFileInfo is a custom FileInfo implementation with modified size
type SizeFileInfo struct {
	os.FileInfo
}

// Size returns the file size, possibly modified for HTML files
func (s SizeFileInfo) Size() int64 {
	if s.FileInfo == nil {
		return 0
	}

	name := s.Name()
	if name == "" {
		return 0
	}
	if filepath.Ext(name) == ".html" &&
		flag.Lookup(PatchHTMLName) != nil &&
		flag.Lookup(PatchHTMLName).Value.String() == "true" {
		return s.FileInfo.Size() + int64(len(ToParentPatch))
	}
	return s.FileInfo.Size()
}

// Stat returns file stats with modified size
func (f *SuffixFile) Stat() (fs.FileInfo, error) {
	if f.File == nil {
		return nil, os.ErrInvalid
	}

	fi, err := f.File.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat err: %w", err)
	}

	return SizeFileInfo{fi}, nil
}

// Read reads from the file with possible HTML modifications
func (f *SuffixFile) Read(b []byte) (int, error) {
	n, err := f.File.Read(b)
	if err != nil {
		return 0, fmt.Errorf("read err: %w", err)
	}
	length := len(b)
	if f.FileSuffix == ".html" &&
		flag.Lookup(PatchHTMLName) != nil &&
		flag.Lookup(PatchHTMLName).Value.String() == "true" {
		tmp := b
		hTags := HTMLTagReg.FindSubmatch(b)
		if len(hTags) == 2 {
			tmp = bytes.Replace(b, hTags[1], append(hTags[1], []byte(ToParentPatch)...), 1)[:length]
			_, err = f.File.Seek(int64(-len(ToParentPatch)), io.SeekCurrent)
			if err != nil {
				return 0, fmt.Errorf("seek err: %w", err)
			}
		}
		tmp = bytes.Replace(tmp, []byte("<base href="), []byte("<bbse href="), 1) // ignore base href

		for i := 0; i < length; i++ {
			b[i] = tmp[i]
		}
	}

	return n, err
}

// ReadDir reads directory entries with suffix filtering
func (f *SuffixFile) ReadDir(count int) ([]fs.DirEntry, error) {
	entries, err := f.File.ReadDir(count)
	if err != nil {
		return nil, fmt.Errorf("read dir err: %w", err)
	}

	var newEntries []fs.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			ss := strings.Split(entry.Name(), ".")
			if f.FilterSuffix != "" && ss[len(ss)-1] != f.FilterSuffix {
				continue
			}
		}
		newEntries = append(newEntries, entry)
	}
	return newEntries, nil
}

// CreateFilesystemHandler returns a custom filesystem handler
func CreateFilesystemHandler(rootDir, filterSuffix string) SuffixDirFS {
	SetFilterSuffix(filterSuffix)
	return SuffixDirFS(rootDir)
}
