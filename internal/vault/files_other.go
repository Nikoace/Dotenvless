//go:build !windows

package vault

import "errors"

func lockFile(path string) (func(), error) { return nil, errors.New("Dotenvless requires Windows") }
func replaceFile(from, to string) error    { return errors.New("Dotenvless requires Windows") }
