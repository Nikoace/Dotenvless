package vault

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const AnyRevision = ^uint64(0)

var ErrMigration = errors.New("vault upgrade required; run dvl migrate")
var ErrConflict = errors.New("vault changed; refresh and retry")
var envPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

type Binding struct {
	Kind     string `json:"kind"`
	Cipher   []byte `json:"cipher,omitempty"`
	SharedID string `json:"sharedId,omitempty"`
}
type Environment struct {
	ID       string             `json:"id"`
	Bindings map[string]Binding `json:"bindings"`
}
type ProjectRecord struct {
	Name         string                 `json:"name"`
	Root         string                 `json:"root"`
	Kind         string                 `json:"kind"`
	Base         map[string]Binding     `json:"base"`
	Environments map[string]Environment `json:"environments"`
}
type SharedItem struct {
	Name   string `json:"name"`
	Cipher []byte `json:"cipher"`
}
type Snapshot struct {
	Version  int                      `json:"version"`
	Revision uint64                   `json:"revision"`
	Projects map[string]ProjectRecord `json:"projects"`
	Shared   map[string]SharedItem    `json:"shared"`
}
type Manager struct{ store *Store }

func NewManager(path string, p Protector) *Manager { return &Manager{New(path, p)} }
func EmptySnapshot() *Snapshot {
	return &Snapshot{Version: 2, Projects: map[string]ProjectRecord{}, Shared: map[string]SharedItem{}}
}
func NewID() (string, error) {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return "", errors.New("cannot create record identity")
	}
	return hex.EncodeToString(b), nil
}
func ValidName(s string) bool {
	return s != "" && len(s) <= 256 && utf8.ValidString(s) && !strings.ContainsFunc(s, unicode.IsControl)
}
func ValidEnvironment(s string) bool { return envPattern.MatchString(s) && s != "base" }
func (m *Manager) Read() (*Snapshot, error) {
	if m.store.path == "" {
		return nil, errors.New("vault path is not configured")
	}
	if _, e := os.Lstat(m.store.path); errors.Is(e, os.ErrNotExist) {
		return EmptySnapshot(), nil
	}
	unlock, e := lockFile(m.store.path + ".lock")
	if e != nil {
		return nil, errors.New("cannot lock vault; retry when it is available")
	}
	defer unlock()
	s, _, e := m.readLocked()
	return s, e
}
func (m *Manager) readLocked() (*Snapshot, []byte, error) {
	info, e := os.Lstat(m.store.path)
	if errors.Is(e, os.ErrNotExist) {
		return EmptySnapshot(), nil, nil
	}
	if e != nil || !info.Mode().IsRegular() {
		return nil, nil, errors.New("vault must be a readable regular file")
	}
	f, e := os.Open(m.store.path)
	if e != nil {
		return nil, nil, errors.New("cannot read vault")
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, maxVaultBytes+1))
	if e != nil || len(data) > maxVaultBytes {
		return nil, nil, ErrCorrupt
	}
	d := json.NewDecoder(bytes.NewReader(data))
	if uniqueJSON(d, 0) != nil {
		return nil, nil, ErrCorrupt
	}
	if _, e = d.Token(); e != io.EOF {
		return nil, nil, ErrCorrupt
	}
	var header struct {
		Version int `json:"version"`
	}
	if json.Unmarshal(data, &header) != nil {
		return nil, nil, ErrCorrupt
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if header.Version == 1 {
		var old document
		if decoder.Decode(&old) != nil || old.Projects == nil {
			return nil, nil, ErrCorrupt
		}
		s := EmptySnapshot()
		s.Version = 1
		for id, entries := range old.Projects {
			if !ValidName(id) || entries == nil {
				return nil, nil, ErrCorrupt
			}
			bindings := map[string]Binding{}
			for key, cipher := range entries {
				n, e := NormalizeKey(key)
				if e != nil || n != key || len(cipher) == 0 {
					return nil, nil, ErrCorrupt
				}
				bindings[key] = Binding{Kind: "value", Cipher: cipher}
			}
			s.Projects[id] = ProjectRecord{Kind: "unassociated", Base: map[string]Binding{}, Environments: map[string]Environment{"local": {ID: "legacy-local", Bindings: bindings}}}
		}
		return s, data, nil
	}
	var s Snapshot
	if header.Version != 2 || decoder.Decode(&s) != nil || s.Revision == 0 || validateSnapshot(&s) != nil {
		return nil, nil, ErrCorrupt
	}
	return &s, data, nil
}
func validateSnapshot(s *Snapshot) error {
	if s.Version != 2 || s.Projects == nil || s.Shared == nil {
		return ErrCorrupt
	}
	for id, p := range s.Projects {
		if !ValidName(id) || p.Base == nil || p.Environments == nil {
			return ErrCorrupt
		}
		if p.Kind != "git" && p.Kind != "directory" && p.Kind != "unassociated" {
			return ErrCorrupt
		}
		if p.Root != "" && !filepath.IsAbs(p.Root) {
			return ErrCorrupt
		}
		if p.Name != "" && !ValidName(p.Name) {
			return ErrCorrupt
		}
		if _, ok := p.Environments["local"]; !ok {
			return ErrCorrupt
		}
		if validateBindings(p.Base, false) != nil {
			return ErrCorrupt
		}
		ids := map[string]bool{"base": true}
		for name, env := range p.Environments {
			if !ValidEnvironment(name) || !ValidName(env.ID) || ids[env.ID] || validateBindings(env.Bindings, true) != nil {
				return ErrCorrupt
			}
			ids[env.ID] = true
		}
	}
	for id, item := range s.Shared {
		if !ValidName(id) || !ValidName(item.Name) || len(item.Cipher) == 0 {
			return ErrCorrupt
		}
	}
	return nil
}
func validateBindings(entries map[string]Binding, disabled bool) error {
	if entries == nil {
		return ErrCorrupt
	}
	for key, b := range entries {
		n, e := NormalizeKey(key)
		if e != nil || n != key {
			return ErrCorrupt
		}
		switch b.Kind {
		case "value":
			if len(b.Cipher) == 0 || b.SharedID != "" {
				return ErrCorrupt
			}
		case "shared":
			if !ValidName(b.SharedID) || len(b.Cipher) != 0 {
				return ErrCorrupt
			}
		case "disabled":
			if !disabled || len(b.Cipher) != 0 || b.SharedID != "" {
				return ErrCorrupt
			}
		default:
			return ErrCorrupt
		}
	}
	return nil
}
func (m *Manager) Update(expected uint64, f func(*Snapshot) error) error {
	if m.store.path == "" {
		return errors.New("vault path is not configured")
	}
	if e := os.MkdirAll(filepath.Dir(m.store.path), 0700); e != nil {
		return errors.New("cannot create vault directory")
	}
	unlock, e := lockFile(m.store.path + ".lock")
	if e != nil {
		return errors.New("cannot lock vault; retry when it is available")
	}
	defer unlock()
	s, _, e := m.readLocked()
	if e != nil {
		return e
	}
	if s.Version == 1 {
		return ErrMigration
	}
	if expected != AnyRevision && expected != s.Revision {
		return ErrConflict
	}
	if e = f(s); e != nil {
		return e
	}
	if s.Revision >= AnyRevision-1 {
		return errors.New("vault revision limit reached")
	}
	s.Revision++
	return m.save(s)
}
func (m *Manager) save(s *Snapshot) error {
	if validateSnapshot(s) != nil {
		return ErrCorrupt
	}
	data, e := json.Marshal(s)
	if e != nil || len(data) > maxVaultBytes {
		return errors.New("vault exceeds the supported size")
	}
	return m.store.save(data)
}
func contextV2(parts []string) []byte {
	data, _ := json.Marshal(parts)
	return append([]byte("dotenvless:2\x00"), data...)
}
func (m *Manager) Seal(value []byte, parts ...string) ([]byte, error) {
	if e := ValidateValue(value); e != nil {
		return nil, e
	}
	if m.store.protector == nil {
		return nil, ErrCrypto
	}
	cipher, e := m.store.protector.Protect(value, contextV2(parts))
	if e != nil || len(cipher) == 0 {
		return nil, ErrCrypto
	}
	return cipher, nil
}
func (m *Manager) Open(cipher []byte, parts ...string) ([]byte, error) {
	if m.store.protector == nil {
		return nil, ErrCrypto
	}
	value, e := m.store.protector.Unprotect(cipher, contextV2(parts))
	if e != nil {
		clear(value)
		return nil, ErrCrypto
	}
	if ValidateValue(value) != nil {
		clear(value)
		return nil, ErrCorrupt
	}
	return value, nil
}
func (m *Manager) OpenLegacy(project, key string, cipher []byte) ([]byte, error) {
	if m.store.protector == nil {
		return nil, ErrCrypto
	}
	value, e := m.store.protector.Unprotect(cipher, contextFor(project, key))
	if e != nil {
		clear(value)
		return nil, ErrCrypto
	}
	if ValidateValue(value) != nil {
		clear(value)
		return nil, ErrCorrupt
	}
	return value, nil
}
func (m *Manager) Migrate() (string, error) {
	if _, e := os.Lstat(m.store.path); errors.Is(e, os.ErrNotExist) {
		return "", ErrNotInitialized
	}
	unlock, e := lockFile(m.store.path + ".lock")
	if e != nil {
		return "", errors.New("cannot lock vault; retry when it is available")
	}
	defer unlock()
	s, original, e := m.readLocked()
	if e != nil {
		return "", e
	}
	if s.Version == 2 {
		return "", nil
	}
	backup := m.store.path + ".v1.bak"
	if info, e := os.Lstat(backup); e == nil {
		if !info.Mode().IsRegular() {
			return "", errors.New("migration backup already exists with different data")
		}
		b, e := os.ReadFile(backup)
		if e != nil || !bytes.Equal(b, original) {
			return "", errors.New("migration backup already exists with different data")
		}
	} else if errors.Is(e, os.ErrNotExist) {
		f, e := os.OpenFile(backup, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return "", errors.New("cannot create encrypted migration backup")
		}
		_, writeErr := f.Write(original)
		syncErr := f.Sync()
		closeErr := f.Close()
		if writeErr != nil || syncErr != nil || closeErr != nil {
			return "", errors.New("cannot finish encrypted migration backup")
		}
	} else {
		return "", errors.New("cannot inspect migration backup")
	}
	for project, p := range s.Projects {
		env := p.Environments["local"]
		id, e := NewID()
		if e != nil {
			return "", e
		}
		env.ID = id
		for key, b := range env.Bindings {
			value, e := m.OpenLegacy(project, key, b.Cipher)
			if e != nil {
				return "", e
			}
			cipher, e := m.Seal(value, "project", project, id, key)
			clear(value)
			if e != nil {
				return "", e
			}
			env.Bindings[key] = Binding{Kind: "value", Cipher: cipher}
		}
		p.Environments["local"] = env
		s.Projects[project] = p
	}
	s.Version = 2
	s.Revision = 1
	if e := m.save(s); e != nil {
		return "", e
	}
	return backup, nil
}
