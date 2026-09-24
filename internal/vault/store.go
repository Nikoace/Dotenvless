package vault

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const maxVaultBytes = 16 << 20
const MaxSecretBytes = 32 << 10

var (
	ErrNotInitialized = errors.New("project is not initialized; run dvl init")
	ErrCorrupt        = errors.New("vault is corrupt or uses an unsupported format")
	ErrCrypto         = errors.New("cannot protect or decrypt vault data")
	ErrMissing        = errors.New("secret key does not exist")
	keyPattern        = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

type Protector interface {
	Protect(value, context []byte) ([]byte, error)
	Unprotect(value, context []byte) ([]byte, error)
}
type Store struct {
	path      string
	protector Protector
	replace   func(string, string) error
}
type document struct {
	Version  int                          `json:"version"`
	Projects map[string]map[string][]byte `json:"projects"`
}

func New(path string, p Protector) *Store {
	return &Store{path: path, protector: p, replace: replaceFile}
}

func NormalizeKey(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", errors.New("invalid environment variable name")
	}
	return strings.ToUpper(key), nil
}
func ValidateValue(value []byte) error {
	if bytes.IndexByte(value, 0) >= 0 || len(value) > MaxSecretBytes || !utf8.Valid(value) {
		return errors.New("secret must be valid UTF-8 without NUL and at most 32 KiB")
	}
	return nil
}
func contextFor(project, key string) []byte {
	return []byte("dotenvless:1\x00" + project + "\x00" + key)
}
func (s *Store) Init(project string) error {
	if project == "" {
		return errors.New("invalid project identity")
	}
	return s.transaction(true, true, func(d *document) error {
		if _, ok := d.Projects[project]; !ok {
			d.Projects[project] = map[string][]byte{}
		}
		return nil
	})
}
func (s *Store) Set(project, key string, value []byte) error {
	key, err := NormalizeKey(key)
	if err != nil {
		return err
	}
	if err := ValidateValue(value); err != nil {
		return err
	}
	return s.transaction(false, true, func(d *document) error {
		entries, ok := d.Projects[project]
		if !ok {
			return ErrNotInitialized
		}
		if s.protector == nil {
			return ErrCrypto
		}
		encrypted, err := s.protector.Protect(value, contextFor(project, key))
		if err != nil || len(encrypted) == 0 {
			return ErrCrypto
		}
		entries[key] = encrypted
		return nil
	})
}
func (s *Store) Keys(project string) ([]string, error) {
	keys := []string{}
	err := s.transaction(false, false, func(d *document) error {
		entries, ok := d.Projects[project]
		if !ok {
			return ErrNotInitialized
		}
		for key := range entries {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return nil
	})
	return keys, err
}
func (s *Store) Values(project string) (map[string]string, error) {
	values := map[string]string{}
	err := s.transaction(false, false, func(d *document) error {
		entries, ok := d.Projects[project]
		if !ok {
			return ErrNotInitialized
		}
		if s.protector == nil {
			return ErrCrypto
		}
		for key, encrypted := range entries {
			plaintext, err := s.protector.Unprotect(encrypted, contextFor(project, key))
			if err != nil {
				return ErrCrypto
			}
			if err := ValidateValue(plaintext); err != nil {
				clear(plaintext)
				return ErrCorrupt
			}
			values[key] = string(plaintext)
			clear(plaintext)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return values, nil
}
func (s *Store) Unset(project, key string) error {
	key, err := NormalizeKey(key)
	if err != nil {
		return err
	}
	return s.transaction(false, true, func(d *document) error {
		entries, ok := d.Projects[project]
		if !ok {
			return ErrNotInitialized
		}
		if _, ok := entries[key]; !ok {
			return ErrMissing
		}
		delete(entries, key)
		return nil
	})
}
func (s *Store) transaction(allowNew, write bool, operation func(*document) error) error {
	if s.path == "" {
		return errors.New("vault path is not configured")
	}
	if allowNew {
		if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
			return errors.New("cannot create vault directory")
		}
	}
	unlock, err := lockFile(s.path + ".lock")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotInitialized
		}
		return errors.New("cannot lock vault; retry when it is available")
	}
	defer unlock()
	d, err := s.load(allowNew)
	if err != nil {
		return err
	}
	if err := operation(d); err != nil {
		return err
	}
	if !write {
		return nil
	}
	data, err := json.Marshal(d)
	if err != nil || len(data) > maxVaultBytes {
		return errors.New("vault exceeds the supported size")
	}
	return s.save(data)
}
func (s *Store) load(allowNew bool) (*document, error) {
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		if allowNew {
			return &document{Version: 1, Projects: map[string]map[string][]byte{}}, nil
		}
		return nil, ErrNotInitialized
	}
	if err != nil {
		return nil, errors.New("cannot read vault")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxVaultBytes+1))
	if err != nil {
		return nil, errors.New("cannot read vault")
	}
	if len(data) > maxVaultBytes {
		return nil, ErrCorrupt
	}
	tokens := json.NewDecoder(bytes.NewReader(data))
	if err := uniqueJSON(tokens, 0); err != nil {
		return nil, ErrCorrupt
	}
	if _, err := tokens.Token(); err != io.EOF {
		return nil, ErrCorrupt
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var d document
	if err := decoder.Decode(&d); err != nil || d.Version != 1 || d.Projects == nil {
		return nil, ErrCorrupt
	}
	for project, entries := range d.Projects {
		if project == "" || entries == nil {
			return nil, ErrCorrupt
		}
		for key, value := range entries {
			normalized, err := NormalizeKey(key)
			if err != nil || normalized != key || len(value) == 0 {
				return nil, ErrCorrupt
			}
		}
	}
	return &d, nil
}
func uniqueJSON(decoder *json.Decoder, depth int) error {
	if depth > 16 {
		return ErrCorrupt
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			token, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := token.(string)
			if !ok || seen[key] {
				return ErrCorrupt
			}
			seen[key] = true
			if err := uniqueJSON(decoder, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := uniqueJSON(decoder, depth+1); err != nil {
				return err
			}
		}
	default:
		return ErrCorrupt
	}
	_, err = decoder.Token()
	return err
}
func (s *Store) save(data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(s.path), ".vault-*")
	if err != nil {
		return errors.New("cannot prepare encrypted vault update")
	}
	temp := file.Name()
	defer os.Remove(temp)
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return errors.New("cannot write encrypted vault")
	}
	if err := file.Sync(); err != nil {
		return errors.New("cannot sync encrypted vault")
	}
	if err := file.Close(); err != nil {
		return errors.New("cannot close encrypted vault")
	}
	if err := s.replace(temp, s.path); err != nil {
		return errors.New("cannot replace encrypted vault; original retained")
	}
	return nil
}
