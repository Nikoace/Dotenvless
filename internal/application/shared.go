package application

import (
	"dotenvless/internal/vault"
	"errors"
	"sort"
)

type Usage struct{ ProjectID, ProjectName, Scope, Key string }

func (s *Service) SaveShared(id, name string, value []byte, revision uint64) (string, error) {
	if !vault.ValidName(name) {
		return "", errors.New("invalid shared item name")
	}
	if e := vault.ValidateValue(value); e != nil {
		return "", e
	}
	fresh := id == ""
	if fresh {
		var e error
		id, e = vault.NewID()
		if e != nil {
			return "", e
		}
	}
	e := s.update(revision, func(state *vault.Snapshot) error {
		if !fresh {
			if _, ok := state.Shared[id]; !ok {
				return errors.New("shared item does not exist")
			}
		}
		cipher, e := s.Store.Seal(value, "shared", id)
		if e != nil {
			return e
		}
		state.Shared[id] = vault.SharedItem{Name: name, Cipher: cipher}
		return nil
	})
	return id, e
}
func (s *Service) RenameShared(id, name string, revision uint64) error {
	if !vault.ValidName(name) {
		return errors.New("invalid shared item name")
	}
	return s.update(revision, func(state *vault.Snapshot) error {
		item, ok := state.Shared[id]
		if !ok {
			return vault.ErrMissing
		}
		item.Name = name
		state.Shared[id] = item
		return nil
	})
}
func usages(state *vault.Snapshot, id string) []Usage {
	result := []Usage{}
	for pid, p := range state.Projects {
		collect := func(scope string, bindings map[string]vault.Binding) {
			for k, b := range bindings {
				if b.Kind == "shared" && b.SharedID == id {
					result = append(result, Usage{ProjectID: pid, ProjectName: p.Name, Scope: scope, Key: k})
				}
			}
		}
		collect("base", p.Base)
		for scope, env := range p.Environments {
			collect(scope, env.Bindings)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.ProjectID != b.ProjectID {
			return a.ProjectID < b.ProjectID
		}
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		return a.Key < b.Key
	})
	return result
}
func (s *Service) SharedUsage(id string) ([]Usage, error) {
	state, e := s.State()
	if e != nil {
		return nil, e
	}
	if _, ok := state.Shared[id]; !ok {
		return nil, vault.ErrMissing
	}
	return usages(state, id), nil
}
func (s *Service) DeleteShared(id string, revision uint64) error {
	return s.update(revision, func(state *vault.Snapshot) error {
		if _, ok := state.Shared[id]; !ok {
			return vault.ErrMissing
		}
		if len(usages(state, id)) != 0 {
			return errors.New("shared item is still referenced; remove its references first")
		}
		delete(state.Shared, id)
		return nil
	})
}
func (s *Service) RevealShared(id string) ([]byte, error) {
	state, e := s.State()
	if e != nil {
		return nil, e
	}
	item, ok := state.Shared[id]
	if !ok {
		return nil, vault.ErrMissing
	}
	return s.Store.Open(item.Cipher, "shared", id)
}
