package dotenv

import (
	"bytes"
	"dotenvless/internal/vault"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxFileBytes = 1 << 20

func Parse(reader io.Reader) (values map[string][]byte, err error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxFileBytes+1))
	if err != nil {
		return nil, errors.New("cannot read dotenv source")
	}
	defer clear(data)
	if len(data) > maxFileBytes || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return nil, errors.New("dotenv source must be UTF-8 without NUL and at most 1 MiB")
	}
	text := strings.TrimPrefix(string(data), "\ufeff")
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	values = map[string][]byte{}
	defer func() {
		if err != nil {
			for _, value := range values {
				clear(value)
			}
			values = nil
		}
	}()
	for index := 0; index < len(lines); index++ {
		number := index + 1
		line := strings.TrimLeftFunc(lines[index], unicode.IsSpace)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") || strings.HasPrefix(line, "export\t") {
			line = strings.TrimLeftFunc(line[6:], unicode.IsSpace)
		}
		equals := strings.IndexByte(line, '=')
		if equals < 1 {
			return values, syntaxError(number)
		}
		key, keyErr := vault.NormalizeKey(strings.TrimSpace(line[:equals]))
		if keyErr != nil {
			return values, syntaxError(number)
		}
		if _, exists := values[key]; exists {
			return values, fmt.Errorf("duplicate dotenv key at line %d", number)
		}
		raw := strings.TrimLeft(line[equals+1:], " \t")
		var value []byte
		if len(raw) > 0 && (raw[0] == '\'' || raw[0] == '"') {
			var parseErr error
			value, index, parseErr = quoted(lines, index, raw)
			if parseErr != nil {
				return values, syntaxError(number)
			}
		} else {
			for i, c := range raw {
				if c == '#' && (i == 0 || raw[i-1] == ' ' || raw[i-1] == '\t') {
					raw = raw[:i]
					break
				}
			}
			value = []byte(strings.TrimSpace(raw))
		}
		if err := vault.ValidateValue(value); err != nil {
			clear(value)
			return values, fmt.Errorf("invalid dotenv value at line %d", number)
		}
		values[key] = value
	}
	return values, nil
}
func syntaxError(line int) error { return fmt.Errorf("invalid dotenv syntax at line %d", line) }
func quoted(lines []string, index int, raw string) (value []byte, last int, err error) {
	quote := raw[0]
	position := 1
	defer func() {
		if err != nil {
			clear(value)
			value = nil
		}
	}()
	for {
		if position == len(raw) {
			index++
			if index >= len(lines) {
				return value, index, errors.New("unclosed quote")
			}
			value = append(value, '\n')
			raw = lines[index]
			position = 0
			continue
		}
		c := raw[position]
		position++
		if c == quote {
			tail := strings.TrimSpace(raw[position:])
			if tail != "" && !strings.HasPrefix(tail, "#") {
				return value, index, errors.New("unexpected trailing content")
			}
			return value, index, nil
		}
		if c == '\\' && quote == '"' {
			if position >= len(raw) {
				return value, index, errors.New("incomplete escape")
			}
			next := raw[position]
			position++
			switch next {
			case 'n':
				c = '\n'
			case 'r':
				c = '\r'
			case 't':
				c = '\t'
			case '\\', '"':
				c = next
			default:
				return value, index, errors.New("unsupported escape")
			}
		}
		value = append(value, c)
		if len(value) > vault.MaxSecretBytes {
			return value, index, errors.New("value too long")
		}
	}
}
