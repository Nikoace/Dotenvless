package cli

import (
	"dotenvless/internal/vault"
	"errors"
	"io"
	"unicode/utf8"
)

// readSecretLine never writes input and never truncates a value silently.
func readSecretLine(reader io.Reader) (value []byte, err error) {
	defer func() {
		if err != nil {
			clear(value)
			value = nil
		}
	}()
	invalid := false
	var input [1]byte
	for {
		n, readErr := reader.Read(input[:])
		if n > 0 {
			c := input[0]
			switch c {
			case 3, 4:
				return value, errors.New("secret input cancelled")
			case '\r', '\n':
				if invalid || !utf8.Valid(value) {
					return value, errors.New("invalid secret input")
				}
				return value, nil
			case 8, 127:
				if len(value) > 0 {
					_, size := utf8.DecodeLastRune(value)
					clear(value[len(value)-size:])
					value = value[:len(value)-size]
				}
			default:
				if c < 32 || len(value) >= vault.MaxSecretBytes {
					invalid = true
					continue
				}
				value = append(value, c)
			}
		}
		if readErr != nil {
			return value, errors.New("secret input ended before newline")
		}
	}
}
