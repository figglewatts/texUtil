package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func getExeName() string {
	return filepath.Base(os.Args[0])
}

func closeWithErr(c io.Closer, label string, retErr *error) {
	if err := c.Close(); err != nil && *retErr == nil {
		*retErr = fmt.Errorf("closing %s: %w", label, err)
	}
}
