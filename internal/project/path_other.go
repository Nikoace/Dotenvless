//go:build !windows

package project

import "errors"

func canonicalDir(path string) (string, error) { return "", errors.New("Dotenvless requires Windows") }
