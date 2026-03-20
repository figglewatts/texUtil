package cmd

import (
	"fmt"
	"io"
)

func closeWithErr(c io.Closer, label string, retErr *error) {
	if err := c.Close(); err != nil && *retErr == nil {
		*retErr = fmt.Errorf("closing %s: %w", label, err)
	}
}
