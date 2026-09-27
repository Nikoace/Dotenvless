//go:build !windows

package privacy

import "errors"

type unsupported struct{}

func NewNativeClipboard(handle func() uintptr) Clipboard { return unsupported{} }
func (unsupported) SetSensitive([]byte) (uint32, error)  { return 0, errors.New("Windows is required") }
func (unsupported) ClearOwned(uint32) error              { return nil }
