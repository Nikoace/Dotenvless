//go:build !windows

package project

import "errors"

func CanonicalDir(path string) (string, error) { return "", errors.New("Dotenvless requires Windows") }
