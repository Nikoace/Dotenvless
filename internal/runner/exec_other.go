//go:build !windows

package runner

import (
	"errors"
	"io"
)

func RunWithRemoved(args []string, secrets map[string]string, removed []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	return 1, errors.New("Dotenvless requires Windows")
}

func Run(args []string, secrets map[string]string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	return RunWithRemoved(args, secrets, nil, stdin, stdout, stderr)
}
