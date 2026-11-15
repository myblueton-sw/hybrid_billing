package installer

import (
	"sort"
	"time"
)

const (
	maxObserved    = 1024
	maxGranted     = 16
	previewTag     = "hybrid-billing/install-preview/v1"
	observationTag = "hybrid-billing/install-observation/v1"
)

func (o Observation) valid() bool {
	return o.Platform.valid() && validPorts(o.Ports, maxObserved) && len(o.Volumes) <= maxObserved && ascending(o.Volumes) &&
		len(o.Services) <= maxObserved && ascending(o.Services) && len(o.Granted) <= maxGranted && ascending(o.Granted)
}

func copyObservation(o Observation) Observation {
	o.Ports = append([]Port(nil), o.Ports...)
	o.Volumes = append([]string(nil), o.Volumes...)
	o.Services = append([]string(nil), o.Services...)
	o.Granted = append([]string(nil), o.Granted...)
	return o
}

func (o Observation) digest() string {
	var e encoder
	e.text(observationTag)
	e.text(o.Platform.OS)
	e.text(o.Platform.Arch)
	e.count(len(o.Ports))
	for _, p := range o.Ports {
		e.text(p.key())
	}
	for _, list := range [][]string{o.Volumes, o.Services, o.Granted} {
		e.count(len(list))
		for _, value := range list {
			e.text(value)
		}
	}
	return digestOf(e.out.Bytes())
}

func (r *Registry) probe(id string, actor Actor) (*site, error) {
	if !actor.valid() || !validID(id) {
		return nil, ErrInvalid
	}
	s := r.sites[id]
	if s == nil {
		return nil, ErrNotFound
	}
	if actor.Kind != KindSystem || actor.ID != s.view.Probe {
		return nil, refuse(ReasonNotProbe)
	}
	return s, nil
}

func (r *Registry) Observe(id string, o Observation, actor Actor) (uint64, error) {
	now := r.enter()
	defer r.leave()
	if !o.valid() {
		return 0, ErrInvalid
	}
	s, err := r.probe(id, actor)
	if err != nil {
		return 0, err
	}
	switch {
	case o.Platform != s.view.Platform:
		return 0, refuse(ReasonPlatformMismatch)
	case !o.Complete:
		return 0, refuse(ReasonIncomplete)
	}
	copied := copyObservation(o)
	s.observation, s.observed = &copied, now
	s.sequence++
	return s.sequence, nil
}

type layout struct {
	ports   map[string][]string
	volumes map[string]string
}

func (r *Registry) resolve(s *site, rel Release) layout {
	out := layout{ports: map[string][]string{}, volumes: map[string]string{}}
	matches := s.config.Version == rel.Version
	for _, svc := range rel.Services {
		for _, p := range svc.Ports {
			resolved := p
			if matches {
				for _, o := range s.config.Ports {
					if o.Service == svc.Name && o.Port == p {
						resolved.Number = o.Number
					}
				}
			}
			out.ports[svc.Name] = append(out.ports[svc.Name], resolved.key())
		}
	}
	for _, v := range rel.Volumes {
		out.volumes[v.Name] = v.Name
		if matches {
			for _, o := range s.config.Volumes {
				if o.Volume == v.Name {
					out.volumes[v.Name] = o.Host
				}
			}
		}
	}
	return out
}

func contains(list []string, value string) bool {
	index := sort.SearchStrings(list, value)
	return index < len(list) && list[index] == value
}

func (r *Registry) preview(s *site, rel *release, now time.Time, allowed map[string]bool) (PreviewView, error) {
	supported := false
	for _, p := range rel.release.Platforms {
		supported = supported || p == s.view.Platform
	}
	switch {
	case r.revoked(rel):
		return PreviewView{}, refuse(ReasonRevoked)
	case !supported:
		return PreviewView{}, refuse(ReasonUnsupported)
	case s.installed:
		return PreviewView{}, refuse(ReasonAlreadyInstalled)
	case s.observation == nil:
		return PreviewView{}, ErrNotFound
	case now.After(s.observed.Add(r.limits.ObservationLife)):
		return PreviewView{}, refuse(ReasonObservationStale)
	}
	o := s.observation
	used := map[string]bool{}
	for _, p := range o.Ports {
		used[p.key()] = true
	}
	plan := r.resolve(s, rel.release)
	v := PreviewView{Site: s.view.ID, Version: rel.release.Version, Services: []Item{}, Ports: []Item{}, Volumes: []Item{}, Data: []DataEffect{}, Permissions: []string{}, Conflicts: []Item{}}
	var extra []Item
	owners := map[string]int{}
	for _, svc := range rel.release.Services {
		for _, key := range plan.ports[svc.Name] {
			owners[key]++
		}
	}
	for _, svc := range rel.release.Services {
		item := Item{Kind: "service", Target: svc.Name, Action: ActionCreate}
		if contains(o.Services, svc.Name) && !allowed["service:"+svc.Name] {
			item.Conflict = ConflictServiceExists
		}
		v.Services = append(v.Services, item)
		for _, key := range plan.ports[svc.Name] {
			port := Item{Kind: "port", Target: svc.Name + ":" + key, Action: ActionBind}
			if used[key] && !allowed["port:"+key] {
				port.Conflict = ConflictPortInUse
				extra = append(extra, port)
			}
			if owners[key] > 1 {
				port.Conflict = ConflictPortDuplicate
			}
			v.Ports = append(v.Ports, port)
		}
	}
	hosts := map[string]int{}
	for _, host := range plan.volumes {
		hosts[host]++
	}
	for _, vol := range rel.release.Volumes {
		host := plan.volumes[vol.Name]
		item := Item{Kind: "volume", Target: host, Action: ActionCreate}
		switch {
		case contains(o.Volumes, host) && !allowed["volume:"+host]:
			item.Conflict = ConflictExistingVolume
			v.Data = append(v.Data, DataEffect{Volume: host, Effect: EffectUntouched})
			extra = append(extra, item)
		case vol.Data:
			v.Data = append(v.Data, DataEffect{Volume: host, Effect: EffectNew})
		}
		if hosts[host] > 1 {
			item.Conflict = ConflictVolumeDuplicate
		}
		v.Volumes = append(v.Volumes, item)
	}
	sort.Slice(v.Data, func(a, b int) bool { return v.Data[a].Volume < v.Data[b].Volume })
	for _, p := range rel.release.permissions() {
		if !contains(o.Granted, p) {
			v.Permissions = append(v.Permissions, p)
		}
	}
	for _, list := range [][]Item{v.Services, v.Ports, v.Volumes} {
		for _, item := range list {
			if item.Conflict != "" && item.Conflict != ConflictPortInUse && item.Conflict != ConflictExistingVolume {
				v.Conflicts = append(v.Conflicts, item)
			}
		}
	}
	v.Conflicts = append(v.Conflicts, extra...)
	if s.config.Version != "" && s.config.Version != rel.release.Version && len(s.config.Ports)+len(s.config.Volumes) > 0 {
		v.Conflicts = append(v.Conflicts, Item{Kind: "config", Target: s.config.Version, Conflict: ConflictConfigMismatch})
	}
	sort.Slice(v.Conflicts, func(a, b int) bool {
		x, y := v.Conflicts[a], v.Conflicts[b]
		return x.Conflict < y.Conflict || x.Conflict == y.Conflict && x.Target < y.Target
	})
	v.Hash = r.hash(s, rel, v)
	return v, nil
}

func (r *Registry) hash(s *site, rel *release, v PreviewView) string {
	var e encoder
	for _, value := range []string{previewTag, s.view.ID, s.view.Tenant, s.view.Probe, rel.release.Version, rel.digest, s.observation.digest()} {
		e.text(value)
	}
	e.number(int64(s.revision))
	for _, list := range [][]Item{v.Services, v.Ports, v.Volumes, v.Conflicts} {
		e.count(len(list))
		for _, item := range list {
			e.text(item.Kind)
			e.text(item.Target)
			e.text(item.Action)
			e.text(item.Conflict)
		}
	}
	e.count(len(v.Data))
	for _, d := range v.Data {
		e.text(d.Volume)
		e.text(d.Effect)
	}
	e.count(len(v.Permissions))
	for _, p := range v.Permissions {
		e.text(p)
	}
	return digestOf(e.out.Bytes())
}

func (r *Registry) Preview(id, version string) (PreviewView, error) {
	now := r.enter()
	defer r.leave()
	if _, ok := semver(version); !ok || !validID(id) {
		return PreviewView{}, ErrInvalid
	}
	s, rel := r.sites[id], r.releases[version]
	if s == nil || rel == nil {
		return PreviewView{}, ErrNotFound
	}
	return r.preview(s, rel, now, nil)
}

func (r *Registry) Site(id string) (Site, error) {
	r.enter()
	defer r.leave()
	if !validID(id) {
		return Site{}, ErrInvalid
	}
	s := r.sites[id]
	if s == nil {
		return Site{}, ErrNotFound
	}
	return s.view, nil
}

func (r *Registry) Administrator(id string) (string, error) {
	r.enter()
	defer r.leave()
	if !validID(id) {
		return "", ErrInvalid
	}
	s := r.sites[id]
	if s == nil || s.admin == "" {
		return "", ErrNotFound
	}
	return s.admin, nil
}
