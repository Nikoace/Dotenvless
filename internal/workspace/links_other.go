//go:build !windows

package workspace

import "errors"

func directoryLink(path string) (bool, error) {
	return false, errors.New("Dotenvless requires Windows")
}
