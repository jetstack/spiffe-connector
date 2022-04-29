package main

import (
	"io/fs"
	"os"
)

// realFS represents the filesystem on disk
type realFS struct {
}

// Open implements fs.FS
func (r realFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}

// ReadFile implements fs.ReadFileFS
func (r realFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
