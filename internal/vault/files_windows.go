package vault

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"time"
)

func lockFile(path string) (func(), error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	handle := windows.Handle(file.Fd())
	overlap := new(windows.Overlapped)
	deadline := time.Now().Add(5 * time.Second)
	for {
		err = windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, overlap)
		if err == nil {
			break
		}
		if !errors.Is(err, windows.ERROR_LOCK_VIOLATION) || time.Now().After(deadline) {
			file.Close()
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return func() { _ = windows.UnlockFileEx(handle, 0, 1, 0, overlap); _ = file.Close() }, nil
}
func replaceFile(from, to string) error {
	source, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(source, target, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
