package project

import (
	"errors"
	"golang.org/x/sys/windows"
	"path/filepath"
	"strings"
)

func canonicalDir(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	name, err := windows.UTF16PtrFromString(absolute)
	if err != nil {
		return "", err
	}
	handle, err := windows.CreateFile(name, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)
	size := uint32(512)
	for size <= 32768 {
		buffer := make([]uint16, size)
		n, err := windows.GetFinalPathNameByHandle(handle, &buffer[0], size, 0)
		if err != nil {
			return "", err
		}
		if n >= size {
			size = n + 1
			continue
		}
		final := windows.UTF16ToString(buffer[:n])
		if strings.HasPrefix(final, `\\?\UNC\`) {
			final = `\\` + strings.TrimPrefix(final, `\\?\UNC\`)
		} else {
			final = strings.TrimPrefix(final, `\\?\`)
		}
		return filepath.EvalSymlinks(final)
	}
	return "", errors.New("directory path is too long")
}
