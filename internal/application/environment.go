package application

import (
	"dotenvless/internal/vault"
	"errors"
	"sort"
)

type Entry struct {
	Key, Kind, Source, SharedID, SharedName string
	Inherited                               bool
}
type Resolved struct {
	Values  map[string]string
	Removed []string
}
type selected struct {
	binding         vault.Binding
	scopeID, source string
}

func (s *Service) update(revision uint64, f func(*vault.Snapshot) error) error {
	return s.Store.Update(revision, func(state *vault.Snapshot) error {
		for _, p := range state.Projects {
			if p.Root != "" {
				if e := s.outside(p.Root); e != nil {
					return e
				}
			}
		}
		return f(state)
	})
}
func layer(state *vault.Snapshot, id, scope string) (map[string]vault.Binding, string, error) {
	p, ok := state.Projects[id]
	if !ok {
		return nil, "", vault.ErrNotInitialized
	}
	if scope == "base" {
		return p.Base, "base", nil
	}
	env, ok := p.Environments[scope]
	if !ok {
		return nil, "", errors.New("environment does not exist")
	}
	return env.Bindings, env.ID, nil
}
func effective(state *vault.Snapshot, id, scope string) (map[string]selected, error) {
	p, ok := state.Projects[id]
	if !ok {
		return nil, vault.ErrNotInitialized
	}
	bindings, sid, e := layer(state, id, scope)
	if e != nil {
		return nil, e
	}
	out := map[string]selected{}
	for k, b := range p.Base {
		out[k] = selected{b, "base", "base"}
	}
	if scope != "base" {
		for k, b := range bindings {
			out[k] = selected{b, sid, scope}
		}
	}
	return out, nil
}
func (s *Service) Set(id, scope, key string, value []byte, revision uint64) error {
	key, e := vault.NormalizeKey(key)
	if e != nil {
		return e
	}
	if e = vault.ValidateValue(value); e != nil {
		return e
	}
	return s.update(revision, func(state *vault.Snapshot) error {
		entries, sid, e := layer(state, id, scope)
		if e != nil {
			return e
		}
		cipher, e := s.Store.Seal(value, "project", id, sid, key)
		if e != nil {
			return e
		}
		entries[key] = vault.Binding{Kind: "value", Cipher: cipher}
		return nil
	})
}
func (s *Service) Unset(id, scope, key string, revision uint64) error {
	key, e := vault.NormalizeKey(key)
	if e != nil {
		return e
	}
	return s.update(revision, func(state *vault.Snapshot) error {
		entries, _, e := layer(state, id, scope)
		if e != nil {
			return e
		}
		if _, ok := entries[key]; !ok {
			return errors.New("key is not defined in this layer; inherited keys must be changed in base or disabled in the client")
		}
		delete(entries, key)
		return nil
	})
}
func (s *Service) Disable(id, scope, key string, revision uint64) error {
	if scope == "base" {
		return errors.New("base variables cannot be disabled")
	}
	key, e := vault.NormalizeKey(key)
	if e != nil {
		return e
	}
	return s.update(revision, func(state *vault.Snapshot) error {
		entries, _, e := layer(state, id, scope)
		if e != nil {
			return e
		}
		entries[key] = vault.Binding{Kind: "disabled"}
		return nil
	})
}
func (s *Service) BindShared(id, scope, key, shared string, revision uint64) error {
	key, e := vault.NormalizeKey(key)
	if e != nil {
		return e
	}
	return s.update(revision, func(state *vault.Snapshot) error {
		entries, _, e := layer(state, id, scope)
		if e != nil {
			return e
		}
		if _, ok := state.Shared[shared]; !ok {
			return errors.New("shared item does not exist")
		}
		entries[key] = vault.Binding{Kind: "shared", SharedID: shared}
		return nil
	})
}
func (s *Service) Entries(id, scope string) ([]Entry, uint64, error) {
	state, e := s.State()
	if e != nil {
		return nil, 0, e
	}
	chosen, e := effective(state, id, scope)
	if e != nil {
		return nil, 0, e
	}
	out := []Entry{}
	for key, item := range chosen {
		b := item.binding
		out = append(out, Entry{Key: key, Kind: b.Kind, Source: item.source, SharedID: b.SharedID, SharedName: state.Shared[b.SharedID].Name, Inherited: scope != "base" && item.source == "base"})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, state.Revision, nil
}
func (s *Service) openSelected(state *vault.Snapshot, id, key string, item selected) ([]byte, error) {
	switch item.binding.Kind {
	case "value":
		if state.Version == 1 {
			return s.Store.OpenLegacy(id, key, item.binding.Cipher)
		}
		return s.Store.Open(item.binding.Cipher, "project", id, item.scopeID, key)
	case "shared":
		shared, ok := state.Shared[item.binding.SharedID]
		if !ok {
			return nil, errors.New("shared reference is missing; repair it before running")
		}
		return s.Store.Open(shared.Cipher, "shared", item.binding.SharedID)
	default:
		return nil, vault.ErrMissing
	}
}
func (s *Service) Resolve(id, scope string) (Resolved, error) {
	if scope == "base" {
		return Resolved{}, errors.New("select an environment to run")
	}
	state, e := s.State()
	if e != nil {
		return Resolved{}, e
	}
	chosen, e := effective(state, id, scope)
	if e != nil {
		return Resolved{}, e
	}
	result := Resolved{Values: map[string]string{}, Removed: []string{}}
	keys := make([]string, 0, len(chosen))
	for key := range chosen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		item := chosen[key]
		if item.binding.Kind == "disabled" {
			result.Removed = append(result.Removed, key)
			continue
		}
		value, e := s.openSelected(state, id, key, item)
		if e != nil {
			clear(result.Values)
			return Resolved{}, e
		}
		result.Values[key] = string(value)
		clear(value)
	}
	return result, nil
}
func (s *Service) Reveal(id, scope, key string) ([]byte, error) {
	key, e := vault.NormalizeKey(key)
	if e != nil {
		return nil, e
	}
	state, e := s.State()
	if e != nil {
		return nil, e
	}
	chosen, e := effective(state, id, scope)
	if e != nil {
		return nil, e
	}
	item, ok := chosen[key]
	if !ok {
		return nil, vault.ErrMissing
	}
	return s.openSelected(state, id, key, item)
}
func (s *Service) CreateEnvironment(id, name, source string, revision uint64) error {
	if !vault.ValidEnvironment(name) || source == "base" {
		return errors.New("invalid environment name or copy source")
	}
	return s.update(revision, func(state *vault.Snapshot) error {
		p, ok := state.Projects[id]
		if !ok {
			return vault.ErrNotInitialized
		}
		if _, ok := p.Environments[name]; ok {
			return errors.New("environment already exists")
		}
		sid, e := vault.NewID()
		if e != nil {
			return e
		}
		env := vault.Environment{ID: sid, Bindings: map[string]vault.Binding{}}
		if source != "" {
			original, ok := p.Environments[source]
			if !ok {
				return errors.New("source environment does not exist")
			}
			for key, b := range original.Bindings {
				if b.Kind == "value" {
					value, e := s.Store.Open(b.Cipher, "project", id, original.ID, key)
					if e != nil {
						return e
					}
					cipher, e := s.Store.Seal(value, "project", id, sid, key)
					clear(value)
					if e != nil {
						return e
					}
					b.Cipher = cipher
				}
				env.Bindings[key] = b
			}
		}
		p.Environments[name] = env
		return nil
	})
}
func (s *Service) RenameEnvironment(id, old, name string, revision uint64) error {
	if old == "local" || !vault.ValidEnvironment(name) {
		return errors.New("invalid rename; local is protected")
	}
	return s.update(revision, func(state *vault.Snapshot) error {
		p, ok := state.Projects[id]
		if !ok {
			return vault.ErrNotInitialized
		}
		env, ok := p.Environments[old]
		if !ok {
			return errors.New("environment does not exist")
		}
		if _, ok := p.Environments[name]; ok {
			return errors.New("environment already exists")
		}
		delete(p.Environments, old)
		p.Environments[name] = env
		return nil
	})
}
func (s *Service) DeleteEnvironment(id, name string, revision uint64) error {
	if name == "local" || name == "base" {
		return errors.New("base and local are protected")
	}
	return s.update(revision, func(state *vault.Snapshot) error {
		p, ok := state.Projects[id]
		if !ok {
			return vault.ErrNotInitialized
		}
		if _, ok := p.Environments[name]; !ok {
			return errors.New("environment does not exist")
		}
		delete(p.Environments, name)
		return nil
	})
}
