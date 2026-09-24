package config

import (
	"dotenvless/internal/project"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func VaultPath(projectRoot string) (string, error) {
	base := os.Getenv("APPDATA")
	if base == "" || !filepath.IsAbs(base) {
		return "", errors.New("APPDATA must be an absolute directory")
	}
	directory := filepath.Join(base, "dotenvless")
	existing := directory
	for {
		info, err := os.Stat(existing)
		if err == nil {
			if !info.IsDir() {
				return "", errors.New("vault directory is invalid")
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", errors.New("cannot inspect vault directory")
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", errors.New("cannot resolve vault directory")
		}
		existing = parent
	}
	physical, err := project.CanonicalDir(existing)
	if err != nil {
		return "", errors.New("cannot resolve vault directory")
	}
	remaining, err := filepath.Rel(existing, directory)
	if err != nil {
		return "", errors.New("cannot resolve vault directory")
	}
	directory = filepath.Join(physical, remaining)
	root, err := project.CanonicalDir(projectRoot)
	if err != nil {
		return "", errors.New("cannot resolve project directory")
	}
	rel, err := filepath.Rel(root, directory)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("vault must be outside the project directory")
	}
	path := filepath.Join(directory, "vault.dat")
	info, err := os.Lstat(path)
	if err == nil && !info.Mode().IsRegular() {
		return "", errors.New("vault must be a regular file")
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", errors.New("cannot inspect vault file")
	}
	return path, nil
}
