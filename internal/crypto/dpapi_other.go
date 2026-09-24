//go:build !windows

package crypto

import "errors"

func (DPAPI) Protect(value, context []byte) ([]byte, error) {
	return nil, errors.New("Dotenvless requires Windows")
}
func (DPAPI) Unprotect(value, context []byte) ([]byte, error) {
	return nil, errors.New("Dotenvless requires Windows")
}
