package thirdparty

import (
	"sort"
	"strings"
)

func (r *Registry) Component(id string) (ComponentView, error) {
	r.enter()
	defer r.leave()
	k := r.components[id]
	if k == nil {
		return ComponentView{}, ErrNotFound
	}
	return ComponentView{Component: copyChange(Change{Component: k.view}).Component, State: k.state, Reason: k.reason}, nil
}

func (r *Registry) ByName(name string) []ComponentView {
	r.enter()
	defer r.leave()
	out := []ComponentView{}
	ids := []string{}
	for id, k := range r.components {
		if strings.EqualFold(k.view.Name, name) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		k := r.components[id]
		out = append(out, ComponentView{Component: copyChange(Change{Component: k.view}).Component, State: k.state, Reason: k.reason})
	}
	return out
}

func (r *Registry) Release(version string) (ReleaseView, error) {
	r.enter()
	defer r.leave()
	v := r.releases[version]
	if v == nil {
		return ReleaseView{}, ErrNotFound
	}
	return copyRelease(*v), nil
}
