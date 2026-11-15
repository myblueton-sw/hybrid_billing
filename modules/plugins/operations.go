package plugins

import (
	"sort"
)

func viewInstallation(workspace string, i *installation) InstallationView {
	v := InstallationView{Workspace: workspace, Plugin: i.plugin, Version: i.version, State: i.state, Reason: i.reason, Revision: i.revision, Endpoints: []EndpointView{}}
	for _, e := range i.endpoints {
		v.Endpoints = append(v.Endpoints, *e)
	}
	sort.Slice(v.Endpoints, func(a, b int) bool { return v.Endpoints[a].Endpoint.ID < v.Endpoints[b].Endpoint.ID })
	if i.state == StateRemoved {
		v.Disposal = i.disposal
	}
	return v
}

func (r *Registry) installation(workspace, plugin string) (*workspace, *installation, error) {
	if !validID(workspace) || !validID(plugin) {
		return nil, nil, ErrInvalid
	}
	w := r.workspaces[workspace]
	if w == nil || w.installations[plugin] == nil {
		return nil, nil, ErrNotFound
	}
	return w, w.installations[plugin], nil
}

func (r *Registry) Disable(workspace, plugin string, actor Actor) (InstallationView, error) {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return InstallationView{}, err
	}
	_, i, err := r.installation(workspace, plugin)
	if err != nil {
		return InstallationView{}, err
	}
	switch i.state {
	case StateEnabled, StateStopped:
		i.state, i.reason = StateInstalled, ""
		i.revision++
	case StateRemoved:
		return InstallationView{}, refuse(ReasonNotAllowedState)
	}
	return viewInstallation(workspace, i), nil
}

func (r *Registry) Stop(workspace, plugin, reason string, actor Actor) (InstallationView, error) {
	r.enter()
	defer r.leave()
	if !actor.valid() || !validID(reason) {
		return InstallationView{}, ErrInvalid
	}
	w, i, err := r.installation(workspace, plugin)
	if err != nil {
		return InstallationView{}, err
	}
	switch {
	case actor.Kind == KindAssistant:
		return InstallationView{}, refuse(ReasonAINotAllowed)
	case actor.Kind == KindSystem && actor.ID != w.view.Runtime:
		return InstallationView{}, refuse(ReasonNotRuntime)
	}
	switch i.state {
	case StateEnabled:
		i.state, i.reason = StateStopped, reason
		i.revision++
	case StateInstalled, StateRemoved:
		return InstallationView{}, refuse(ReasonNotAllowedState)
	}
	return viewInstallation(workspace, i), nil
}

func (r *Registry) Installation(workspace, plugin string) (InstallationView, error) {
	r.enter()
	defer r.leave()
	_, i, err := r.installation(workspace, plugin)
	if err != nil {
		return InstallationView{}, err
	}
	return viewInstallation(workspace, i), nil
}

func (r *Registry) Workspace(id string) (Workspace, error) {
	r.enter()
	defer r.leave()
	if !validID(id) {
		return Workspace{}, ErrInvalid
	}
	w := r.workspaces[id]
	if w == nil {
		return Workspace{}, ErrNotFound
	}
	return w.view, nil
}

func copyMark(m *Mark) *Mark {
	if m == nil {
		return nil
	}
	out := *m
	return &out
}

func viewKey(k *key) KeyView {
	v := k.view
	v.Public = append([]byte(nil), v.Public...)
	v.Compromise = copyMark(k.compromise)
	return v
}

func (r *Registry) Publisher(id string) (PublisherView, error) {
	r.enter()
	defer r.leave()
	if !validID(id) {
		return PublisherView{}, ErrInvalid
	}
	p := r.publishers[id]
	if p == nil {
		return PublisherView{}, ErrNotFound
	}
	v := PublisherView{Publisher: p.view, Revocation: copyMark(p.revocation), Keys: []KeyView{}}
	for _, k := range p.keys {
		v.Keys = append(v.Keys, viewKey(k))
	}
	sort.Slice(v.Keys, func(a, b int) bool { return v.Keys[a].Version < v.Keys[b].Version })
	return v, nil
}

func (r *Registry) found(plugin, number string) (*version, error) {
	if _, ok := semver(number); !ok || !validID(plugin) {
		return nil, ErrInvalid
	}
	p := r.plugins[plugin]
	if p == nil || p.versions[number] == nil {
		return nil, ErrNotFound
	}
	return p.versions[number], nil
}

func viewVersion(v *version) VersionView {
	return VersionView{Manifest: copyManifest(v.manifest), Signature: append([]byte(nil), v.signature...), Vulnerable: copyMark(v.vulnerable)}
}

func (r *Registry) Version(plugin, number string) (VersionView, error) {
	r.enter()
	defer r.leave()
	v, err := r.found(plugin, number)
	if err != nil {
		return VersionView{}, err
	}
	return viewVersion(v), nil
}

func (r *Registry) publisherKey(publisher, version string) (*publisher, *key, error) {
	if !validID(publisher) || !validID(version) {
		return nil, nil, ErrInvalid
	}
	p := r.publishers[publisher]
	if p == nil || p.keys[version] == nil {
		return nil, nil, ErrNotFound
	}
	return p, p.keys[version], nil
}

func (r *Registry) RetireKey(publisher, version string, actor Actor) (KeyView, error) {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return KeyView{}, err
	}
	_, k, err := r.publisherKey(publisher, version)
	if err != nil {
		return KeyView{}, err
	}
	k.view.Retired = true
	return viewKey(k), nil
}

func (r *Registry) stopUnrunnable() {
	for _, w := range r.workspaces {
		for _, i := range w.installations {
			if i.state != StateEnabled {
				continue
			}
			if reason := r.runnable(w, r.plugins[i.plugin].versions[i.version]); reason != "" {
				i.state, i.reason = StateStopped, reason
				i.revision++
			}
		}
	}
}

func (r *Registry) slot(t Target) (**Mark, error) {
	switch t.Kind {
	case MarkVulnerable:
		v, err := r.found(t.Plugin, t.Version)
		if err != nil {
			return nil, err
		}
		return &v.vulnerable, nil
	case MarkCompromise:
		_, k, err := r.publisherKey(t.Publisher, t.Key)
		if err != nil {
			return nil, err
		}
		return &k.compromise, nil
	case MarkRevocation:
		if !validID(t.Publisher) {
			return nil, ErrInvalid
		}
		p := r.publishers[t.Publisher]
		if p == nil {
			return nil, ErrNotFound
		}
		return &p.revocation, nil
	}
	return nil, ErrInvalid
}

func (r *Registry) mark(t Target, reason string, actor Actor) (Mark, error) {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return Mark{}, err
	}
	if !validText(reason) {
		return Mark{}, ErrInvalid
	}
	slot, err := r.slot(t)
	if err != nil {
		return Mark{}, err
	}
	if *slot == nil {
		*slot = &Mark{By: actor.ID, Reason: reason}
		if t.Kind == MarkCompromise {
			r.publishers[t.Publisher].keys[t.Key].view.Retired = true
		}
		r.stopUnrunnable()
	}
	return **slot, nil
}

func (r *Registry) MarkVulnerable(plugin, version, reason string, actor Actor) (Mark, error) {
	return r.mark(Target{Kind: MarkVulnerable, Plugin: plugin, Version: version}, reason, actor)
}

func (r *Registry) CompromiseKey(publisher, key, reason string, actor Actor) (Mark, error) {
	return r.mark(Target{Kind: MarkCompromise, Publisher: publisher, Key: key}, reason, actor)
}

func (r *Registry) RevokePublisher(publisher, reason string, actor Actor) (Mark, error) {
	return r.mark(Target{Kind: MarkRevocation, Publisher: publisher}, reason, actor)
}

func (r *Registry) marked(t Target, actor Actor) (**Mark, error) {
	if err := actor.human(); err != nil {
		return nil, err
	}
	slot, err := r.slot(t)
	if err != nil {
		return nil, err
	}
	if *slot == nil {
		return nil, ErrNotFound
	}
	return slot, nil
}

func (r *Registry) ConfirmMark(t Target, actor Actor) (Mark, error) {
	r.enter()
	defer r.leave()
	slot, err := r.marked(t, actor)
	if err != nil {
		return Mark{}, err
	}
	m := *slot
	switch {
	case m.ConfirmedBy == actor.ID:
	case m.ConfirmedBy != "":
		return Mark{}, refuse(ReasonNotPending)
	case m.By == actor.ID:
		return Mark{}, refuse(ReasonSelfApproval)
	default:
		m.ConfirmedBy = actor.ID
	}
	return *m, nil
}

func (r *Registry) ClearMark(t Target, actor Actor) (Mark, error) {
	r.enter()
	defer r.leave()
	slot, err := r.marked(t, actor)
	if err != nil {
		return Mark{}, err
	}
	m := *slot
	switch {
	case m.By != actor.ID:
		return Mark{}, refuse(ReasonNotProposer)
	case m.ConfirmedBy != "":
		return Mark{}, refuse(ReasonNotPending)
	}
	*slot = nil
	return *m, nil
}
