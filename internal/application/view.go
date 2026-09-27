package application

import (
	"sort"
)

type ProjectInfo struct {
	ID, Name, Root, Kind string
	Available            bool
	Configured           bool
	Environments         []string
}
type SharedInfo struct {
	ID, Name string
	Uses     []Usage
}
type View struct {
	Version        int
	Revision       uint64
	Projects       []ProjectInfo
	Shared         []SharedInfo
	Entries        []Entry
	SelectionError string
}

func (s *Service) View(id, scope string) (View, error) {
	state, e := s.State()
	if e != nil {
		return View{}, e
	}
	view := View{Version: state.Version, Revision: state.Revision, Projects: []ProjectInfo{}, Shared: []SharedInfo{}, Entries: []Entry{}}
	for id, p := range state.Projects {
		info := ProjectInfo{ID: id, Name: p.Name, Root: p.Root, Kind: p.Kind, Environments: []string{"local"}}
		if p.Root != "" {
			actual, e := directoryIdentity(p.Root)
			info.Available = e == nil && actual.ID == id
		}
		info.Configured = len(p.Base) > 0
		for name, env := range p.Environments {
			if len(env.Bindings) > 0 {
				info.Configured = true
			}
			if name != "local" {
				info.Environments = append(info.Environments, name)
			}
		}
		sort.Strings(info.Environments[1:])
		view.Projects = append(view.Projects, info)
	}
	sort.Slice(view.Projects, func(i, j int) bool {
		a, b := view.Projects[i], view.Projects[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.ID < b.ID
	})
	for id, item := range state.Shared {
		view.Shared = append(view.Shared, SharedInfo{ID: id, Name: item.Name, Uses: usages(state, id)})
	}
	sort.Slice(view.Shared, func(i, j int) bool {
		a, b := view.Shared[i], view.Shared[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.ID < b.ID
	})
	if id != "" {
		chosen, e := effective(state, id, scope)
		if e != nil {
			view.SelectionError = e.Error()
			return view, nil
		}
		for key, item := range chosen {
			b := item.binding
			view.Entries = append(view.Entries, Entry{Key: key, Kind: b.Kind, Source: item.source, SharedID: b.SharedID, SharedName: state.Shared[b.SharedID].Name, Inherited: scope != "base" && item.source == "base"})
		}
		sort.Slice(view.Entries, func(i, j int) bool { return view.Entries[i].Key < view.Entries[j].Key })
	}
	return view, nil
}
