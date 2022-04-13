package cmd

import (
	"io/fs"
	"os"
)

// realFS represents the filesystem on disk
type realFS struct {
}

func (r realFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}
