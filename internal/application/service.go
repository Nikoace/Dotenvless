package application

import (
	"context"
	"crypto/sha256"
	"dotenvless/internal/project"
	"dotenvless/internal/vault"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Service struct {
	Store *vault.Manager
	Path  string
}
type Candidate struct {
	project.Identity
	Kind string
}

func New(path string, p vault.Protector) *Service {
	return &Service{Store: vault.NewManager(path, p), Path: path}
}
func (s *Service) State() (*vault.Snapshot, error) {
	state, e := s.Store.Read()
	if e != nil {
		return nil, e
	}
	for id, p := range state.Projects {
		if p.Root != "" {
			sum := sha256.Sum256([]byte(p.Root))
			if hex.EncodeToString(sum[:]) != id {
				return nil, vault.ErrCorrupt
			}
			if e = s.outside(p.Root); e != nil {
				return nil, e
			}
		}
	}
	return state, nil
}
func directoryIdentity(dir string) (project.Identity, error) {
	absolute, e := filepath.Abs(dir)
	if e != nil {
		return project.Identity{}, errors.New("cannot resolve project directory")
	}
	root, e := project.CanonicalDir(absolute)
	if e != nil {
		return project.Identity{}, errors.New("cannot resolve project directory")
	}
	info, e := os.Stat(root)
	if e != nil || !info.IsDir() {
		return project.Identity{}, errors.New("project directory is unavailable")
	}
	sum := sha256.Sum256([]byte(root))
	return project.Identity{Name: filepath.Base(root), Root: root, ID: hex.EncodeToString(sum[:])}, nil
}
func contains(root, path string) bool {
	rel, e := filepath.Rel(root, path)
	return e == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
func (s *Service) outside(root string) error {
	// Resolve the existing prefix of the storage path, including directory junctions.
	absolute, e := filepath.Abs(s.Path)
	if e != nil {
		return errors.New("cannot resolve vault directory")
	}
	dir := filepath.Dir(absolute)
	existing := dir
	for {
		info, e := os.Stat(existing)
		if e == nil {
			if !info.IsDir() {
				return errors.New("vault directory is invalid")
			}
			break
		}
		if !errors.Is(e, os.ErrNotExist) {
			return errors.New("cannot inspect vault directory")
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return errors.New("cannot resolve vault directory")
		}
		existing = parent
	}
	physical, e := project.CanonicalDir(existing)
	if e != nil {
		return errors.New("cannot resolve vault directory")
	}
	suffix, e := filepath.Rel(existing, dir)
	if e != nil {
		return errors.New("cannot resolve vault directory")
	}
	path := filepath.Join(physical, suffix, filepath.Base(absolute))
	canonicalRoot := root
	if _, e := os.Stat(root); e == nil {
		canonicalRoot, e = project.CanonicalDir(root)
		if e != nil {
			return errors.New("cannot resolve project directory")
		}
	}
	if contains(canonicalRoot, path) {
		return errors.New("vault must be outside the project directory")
	}
	return nil
}
func (s *Service) Register(dir string, revision uint64) (project.Identity, error) {
	identity, e := project.Discover(dir)
	kind := "git"
	if errors.Is(e, project.ErrNotGit) {
		identity, e = directoryIdentity(dir)
		kind = "directory"
	}
	if e != nil {
		return project.Identity{}, e
	}
	if e = s.outside(identity.Root); e != nil {
		return project.Identity{}, e
	}
	e = s.Store.Update(revision, func(state *vault.Snapshot) error {
		for _, p := range state.Projects {
			if p.Root != "" {
				if e := s.outside(p.Root); e != nil {
					return e
				}
			}
		}
		p, ok := state.Projects[identity.ID]
		if !ok {
			id, e := vault.NewID()
			if e != nil {
				return e
			}
			p = vault.ProjectRecord{Base: map[string]vault.Binding{}, Environments: map[string]vault.Environment{"local": {ID: id, Bindings: map[string]vault.Binding{}}}}
		}
		p.Name = identity.Name
		p.Root = identity.Root
		p.Kind = kind
		state.Projects[identity.ID] = p
		return nil
	})
	return identity, e
}
func (s *Service) FindProject(start string) (project.Identity, error) {
	p, e := project.Discover(start)
	if e == nil {
		return p, nil
	}
	if !errors.Is(e, project.ErrNotGit) {
		return project.Identity{}, e
	}
	dir, e := directoryIdentity(start)
	if e != nil {
		return project.Identity{}, e
	}
	state, e := s.State()
	if e != nil {
		return project.Identity{}, e
	}
	found := project.Identity{}
	for id, p := range state.Projects {
		if p.Kind != "directory" || p.Root == "" || !contains(p.Root, dir.Root) {
			continue
		}
		actual, e := directoryIdentity(p.Root)
		if e != nil || actual.ID != id {
			continue
		}
		if len(p.Root) > len(found.Root) {
			found = project.Identity{ID: id, Root: p.Root, Name: p.Name}
		}
	}
	if found.ID == "" {
		return project.Identity{}, vault.ErrNotInitialized
	}
	return found, nil
}
func (s *Service) RemoveProject(id string, revision uint64) error {
	return s.Store.Update(revision, func(state *vault.Snapshot) error {
		if _, ok := state.Projects[id]; !ok {
			return vault.ErrNotInitialized
		}
		delete(state.Projects, id)
		return nil
	})
}
func Discover(ctx context.Context, root string) ([]Candidate, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	start, e := directoryIdentity(root)
	if e != nil {
		return nil, e
	}
	// Discovery has no file-content scan and does not follow linked directories.
	skip := map[string]bool{".git": true, "node_modules": true, "vendor": true, ".venv": true, "venv": true, "bin": true, "build": true, "dist": true, "target": true, ".cache": true, ".tools": true}
	result := []Candidate{}
	seen := map[string]bool{}
	e = filepath.WalkDir(start.Root, func(path string, entry fs.DirEntry, walkErr error) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if walkErr != nil {
			return errors.New("directory scan incomplete: an entry is inaccessible")
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		info, e := entry.Info()
		if e != nil {
			return errors.New("directory scan incomplete")
		}
		if directoryLink(info) {
			return filepath.SkipDir
		}
		if path != start.Root && skip[strings.ToLower(entry.Name())] {
			return filepath.SkipDir
		}
		if _, e := os.Lstat(filepath.Join(path, ".git")); errors.Is(e, os.ErrNotExist) {
			return nil
		} else if e != nil {
			return errors.New("directory scan incomplete")
		}
		p, e := project.Discover(path)
		if e != nil {
			return errors.New("directory scan encountered an invalid Git marker")
		}
		if !seen[p.ID] {
			result = append(result, Candidate{Identity: p, Kind: "git"})
			seen[p.ID] = true
		}
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].Root < result[j].Root })
	return result, e
}
