package crypto

import (
	"bytes"
	"errors"
	"golang.org/x/sys/windows"
	"runtime"
	"unsafe"
)

func blob(data []byte) windows.DataBlob {
	b := windows.DataBlob{Size: uint32(len(data))}
	if len(data) > 0 {
		b.Data = &data[0]
	}
	return b
}
func release(b *windows.DataBlob) {
	if b.Data != nil {
		clear(unsafe.Slice(b.Data, int(b.Size)))
		_, _ = windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(b.Data))))
	}
}
func (DPAPI) Protect(value, context []byte) ([]byte, error) {
	// A version byte gives empty secrets a nonempty DPAPI payload.
	payload := make([]byte, len(value)+1)
	payload[0] = 1
	copy(payload[1:], value)
	defer clear(payload)
	input, entropy := blob(payload), blob(context)
	var output windows.DataBlob
	defer release(&output)
	// Deliberately omit CRYPTPROTECT_LOCAL_MACHINE. No UI or descriptions.
	err := windows.CryptProtectData(&input, nil, &entropy, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output)
	runtime.KeepAlive(input)
	runtime.KeepAlive(context)
	if err != nil {
		return nil, errors.New("DPAPI protection failed")
	}
	return bytes.Clone(unsafe.Slice(output.Data, int(output.Size))), nil
}
func (DPAPI) Unprotect(value, context []byte) ([]byte, error) {
	input, entropy := blob(value), blob(context)
	var output windows.DataBlob
	defer release(&output)
	err := windows.CryptUnprotectData(&input, nil, &entropy, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output)
	runtime.KeepAlive(input)
	runtime.KeepAlive(context)
	if err != nil {
		return nil, errors.New("DPAPI decryption failed")
	}
	if output.Size < 1 || output.Data == nil || *output.Data != 1 {
		return nil, errors.New("unsupported DPAPI payload")
	}
	return bytes.Clone(unsafe.Slice(output.Data, int(output.Size))[1:]), nil
}
