package cli

import (
	"errors"
	"golang.org/x/sys/windows"
	"golang.org/x/term"
	"os"
)

func readHidden(input *os.File) (value []byte, err error) {
	if input == nil || !term.IsTerminal(int(input.Fd())) {
		return nil, errors.New("interactive terminal required")
	}
	fd := int(input.Fd())
	previous, err := term.MakeRaw(fd)
	if err != nil {
		return nil, errors.New("cannot hide terminal input")
	}
	defer func() {
		flushErr := windows.FlushConsoleInputBuffer(windows.Handle(input.Fd()))
		restoreErr := term.Restore(fd, previous)
		if flushErr != nil || restoreErr != nil {
			clear(value)
			value = nil
			err = errors.New("cannot restore terminal input")
		}
	}()
	return readSecretLine(input)
}
