package application

import (
	"dotenvless/internal/dotenv"
	"dotenvless/internal/vault"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type ImportPreview struct {
	Keys, Conflicts []string
	Revision        uint64
	mu              sync.Mutex
	owner           *Service
	project, scope  string
	values          map[string][]byte
	closed          bool
}

func (p *ImportPreview) Close() { p.mu.Lock(); defer p.mu.Unlock(); p.closeLocked() }
func (p *ImportPreview) closeLocked() {
	for _, v := range p.values {
		clear(v)
	}
	clear(p.values)
	p.closed = true
}
func (s *Service) PreviewImport(id, scope string, r io.Reader) (*ImportPreview, error) {
	state, e := s.State()
	if e != nil {
		return nil, e
	}
	if state.Version == 1 {
		return nil, vault.ErrMigration
	}
	chosen, e := effective(state, id, scope)
	if e != nil {
		return nil, e
	}
	values, e := dotenv.Parse(r)
	if e != nil {
		return nil, e
	}
	p := &ImportPreview{owner: s, project: id, scope: scope, Revision: state.Revision, values: values, Keys: []string{}, Conflicts: []string{}}
	for key := range values {
		p.Keys = append(p.Keys, key)
		if _, ok := chosen[key]; ok {
			p.Conflicts = append(p.Conflicts, key)
		}
	}
	sort.Strings(p.Keys)
	sort.Strings(p.Conflicts)
	return p, nil
}
func (s *Service) CommitImport(p *ImportPreview, overwrite bool) error {
	if p == nil {
		return errors.New("import preview is unavailable")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.owner != s {
		return errors.New("import preview expired or was cancelled")
	}
	e := s.update(p.Revision, func(state *vault.Snapshot) error {
		chosen, e := effective(state, p.project, p.scope)
		if e != nil {
			return e
		}
		entries, sid, e := layer(state, p.project, p.scope)
		if e != nil {
			return e
		}
		for key := range p.values {
			if _, ok := chosen[key]; ok && !overwrite {
				return errors.New("import conflicts with existing keys; use --overwrite explicitly")
			}
		}
		protected := map[string]vault.Binding{}
		for key, value := range p.values {
			cipher, e := s.Store.Seal(value, "project", p.project, sid, key)
			if e != nil {
				return e
			}
			protected[key] = vault.Binding{Kind: "value", Cipher: cipher}
		}
		for key, b := range protected {
			entries[key] = b
		}
		return nil
	})
	if e == nil || errors.Is(e, vault.ErrConflict) {
		p.closeLocked()
	}
	return e
}
func (s *Service) Verify(id, scope string) error {
	state, e := s.State()
	if e != nil {
		return e
	}
	chosen, e := effective(state, id, scope)
	if e != nil {
		return e
	}
	for key, item := range chosen {
		if item.binding.Kind == "disabled" {
			continue
		}
		value, e := s.openSelected(state, id, key, item)
		clear(value)
		if e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) Example(id, scope, root string) error {
	p, e := directoryIdentity(root)
	if e != nil {
		return e
	}
	if p.ID != id {
		return errors.New("project directory no longer matches its identity")
	}
	entries, _, e := s.Entries(id, scope)
	if e != nil {
		return e
	}
	keys := []string{}
	for _, entry := range entries {
		if entry.Kind != "disabled" {
			keys = append(keys, entry.Key)
		}
	}
	return WriteExample(root, keys)
}
func WriteExample(root string, keys []string) error {
	var content strings.Builder
	for _, key := range keys {
		if n, e := vault.NormalizeKey(key); e != nil || n != key {
			return errors.New("invalid example key")
		}
		content.WriteString(key + "=\n")
	}
	path := filepath.Join(root, ".env.example")
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if errors.Is(e, os.ErrExist) {
		return errors.New(".env.example already exists; merge it manually")
	}
	if e != nil {
		return errors.New("cannot create .env.example")
	}
	success := false
	defer func() {
		_ = f.Close()
		if !success {
			_ = os.Remove(path)
		}
	}()
	if _, e = f.WriteString(content.String()); e != nil {
		return errors.New("cannot write .env.example")
	}
	if f.Sync() != nil {
		return errors.New("cannot sync .env.example")
	}
	if f.Close() != nil {
		return errors.New("cannot close .env.example")
	}
	success = true
	return nil
}
