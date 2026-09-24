//go:build !windows

package cli

import (
	"errors"
	"os"
)

func readHidden(input *os.File) ([]byte, error) {
	return nil, errors.New("Dotenvless requires Windows")
}
