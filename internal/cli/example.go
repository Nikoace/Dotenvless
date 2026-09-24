package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func writeExample(root string, keys []string) error {
	var content strings.Builder
	for _, key := range keys {
		content.WriteString(key)
		content.WriteString("=\n")
	}
	path := filepath.Join(root, ".env.example")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if errors.Is(err, os.ErrExist) {
		return errors.New(".env.example already exists; merge it manually")
	}
	if err != nil {
		return errors.New("cannot create .env.example")
	}
	success := false
	defer func() {
		_ = file.Close()
		if !success {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.WriteString(content.String()); err != nil {
		return errors.New("cannot write .env.example")
	}
	if err := file.Sync(); err != nil {
		return errors.New("cannot sync .env.example")
	}
	if err := file.Close(); err != nil {
		return errors.New("cannot close .env.example")
	}
	success = true
	return nil
}
