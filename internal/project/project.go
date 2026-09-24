package project

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Identity struct{ Name, Root, ID string }

// Discover resolves the physical working directory before walking to the nearest
// Git marker. Windows EvalSymlinks also restores filesystem casing and drive case.
func Discover(start string) (Identity, error) {
	absolute, err := filepath.Abs(start)
	if err != nil {
		return Identity{}, errors.New("cannot resolve working directory")
	}
	root, err := canonicalDir(absolute)
	if err != nil {
		return Identity{}, errors.New("cannot resolve working directory")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return Identity{}, errors.New("working directory is not a directory")
	}
	for {
		marker := filepath.Join(root, ".git")
		info, err := os.Stat(marker)
		switch {
		case err == nil:
			if !info.IsDir() {
				if err := validateGitFile(marker, root); err != nil {
					return Identity{}, err
				}
			}
			sum := sha256.Sum256([]byte(root))
			return Identity{Name: filepath.Base(root), Root: root, ID: hex.EncodeToString(sum[:])}, nil
		case !errors.Is(err, os.ErrNotExist):
			return Identity{}, errors.New("cannot read Git project marker")
		}
		parent := filepath.Dir(root)
		if parent == root {
			return Identity{}, errors.New("not inside a Git project")
		}
		root = parent
	}
}

func validateGitFile(marker, root string) error {
	invalid := errors.New("invalid Git project marker")
	file, err := os.Open(marker)
	if err != nil {
		return invalid
	}
	defer file.Close()
	const maxMarker = 16 * 1024
	data, err := io.ReadAll(io.LimitReader(file, maxMarker+1))
	if err != nil || len(data) > maxMarker {
		return invalid
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir: ") {
		return invalid
	}
	path := strings.TrimSpace(strings.TrimPrefix(line, "gitdir: "))
	if path == "" || strings.ContainsAny(path, "\r\n\x00") {
		return invalid
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return invalid
	}
	return nil
}
