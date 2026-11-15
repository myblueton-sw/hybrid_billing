package installer

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
)

const (
	maxPlatforms     = 16
	maxArtifacts     = 32
	maxServices      = 32
	maxServicePorts  = 8
	maxServiceVolume = 8
	maxVolumes       = 32
	releaseTag       = "hybrid-billing/release/v1"
	challengeTag     = "hybrid-billing/release-key/v1"
)

func ascending(values []string) bool {
	for index, value := range values {
		if !validID(value) || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func portLess(a, b Port) bool {
	return a.Number < b.Number || a.Number == b.Number && a.Protocol < b.Protocol
}

func (p Port) valid() bool {
	return p.Number >= 1 && p.Number <= 65535 && (p.Protocol == ProtocolTCP || p.Protocol == ProtocolUDP)
}

func (p Port) key() string { return strconv.Itoa(p.Number) + "/" + p.Protocol }

func validPorts(ports []Port, limit int) bool {
	if len(ports) > limit {
		return false
	}
	for index, p := range ports {
		if !p.valid() || index > 0 && !portLess(ports[index-1], p) {
			return false
		}
	}
	return true
}

func (p Platform) valid() bool { return validID(p.OS) && validID(p.Arch) }

func (rel Release) valid() bool {
	if _, ok := semver(rel.Version); !ok || !validID(rel.Key) {
		return false
	}
	if len(rel.Platforms) == 0 || len(rel.Platforms) > maxPlatforms || len(rel.Artifacts) > maxArtifacts ||
		len(rel.Services) == 0 || len(rel.Services) > maxServices || len(rel.Volumes) > maxVolumes {
		return false
	}
	for index, p := range rel.Platforms {
		if !p.valid() || index > 0 && !(rel.Platforms[index-1].OS < p.OS || rel.Platforms[index-1].OS == p.OS && rel.Platforms[index-1].Arch < p.Arch) {
			return false
		}
	}
	artifacts := map[string]bool{}
	for index, a := range rel.Artifacts {
		if !validID(a.Name) || index > 0 && rel.Artifacts[index-1].Name >= a.Name || !validHex(a.Digest) ||
			a.Kind != ArtifactContainer && a.Kind != ArtifactArchive && a.Kind != ArtifactPackage {
			return false
		}
		artifacts[a.Name] = true
	}
	volumes := map[string]bool{}
	for index, v := range rel.Volumes {
		if !validID(v.Name) || index > 0 && rel.Volumes[index-1].Name >= v.Name {
			return false
		}
		volumes[v.Name] = false
	}
	for index, s := range rel.Services {
		if !validID(s.Name) || index > 0 && rel.Services[index-1].Name >= s.Name || !artifacts[s.Artifact] || !validPorts(s.Ports, maxServicePorts) ||
			len(s.Volumes) > maxServiceVolume || !ascending(s.Volumes) {
			return false
		}
		for _, v := range s.Volumes {
			if _, ok := volumes[v]; !ok {
				return false
			}
			volumes[v] = true
		}
	}
	for _, used := range volumes {
		if !used {
			return false
		}
	}
	return true
}

func (rel Release) permissions() []string {
	out := []string{PermissionRunService, PermissionWriteConfig}
	for _, s := range rel.Services {
		if len(s.Ports) > 0 {
			out = append(out, PermissionListenPort)
			break
		}
	}
	if len(rel.Volumes) > 0 {
		out = append(out, PermissionCreateVolume)
	}
	sort.Strings(out)
	return out
}

func (rel Release) encode() []byte {
	var e encoder
	e.text(releaseTag)
	e.text(rel.Version)
	e.text(rel.Key)
	e.count(len(rel.Platforms))
	for _, p := range rel.Platforms {
		e.text(p.OS)
		e.text(p.Arch)
	}
	e.count(len(rel.Artifacts))
	for _, a := range rel.Artifacts {
		e.text(a.Name)
		e.text(a.Kind)
		e.text(a.Digest)
	}
	e.count(len(rel.Services))
	for _, s := range rel.Services {
		e.text(s.Name)
		e.text(s.Artifact)
		e.count(len(s.Ports))
		for _, p := range s.Ports {
			e.count(p.Number)
			e.text(p.Protocol)
		}
		e.count(len(s.Volumes))
		for _, v := range s.Volumes {
			e.text(v)
		}
	}
	e.count(len(rel.Volumes))
	for _, v := range rel.Volumes {
		e.text(v.Name)
		if v.Data {
			e.text("data")
		} else {
			e.text("")
		}
	}
	return e.out.Bytes()
}

func challenge(version string) []byte {
	var e encoder
	e.text(challengeTag)
	e.text(version)
	return e.out.Bytes()
}

func copyRelease(rel Release) Release {
	rel.Platforms = append([]Platform(nil), rel.Platforms...)
	rel.Artifacts = append([]Artifact(nil), rel.Artifacts...)
	rel.Volumes = append([]Volume(nil), rel.Volumes...)
	services := make([]Service, len(rel.Services))
	for index, s := range rel.Services {
		s.Ports = append([]Port(nil), s.Ports...)
		s.Volumes = append([]string(nil), s.Volumes...)
		services[index] = s
	}
	rel.Services = services
	return rel
}

func digestOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (r *Registry) revoked(rel *release) bool {
	return rel.revocation != nil || r.keys[rel.release.Key].compromise != nil
}

func (r *Registry) RetireKey(version string, actor Actor) error {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	if !validID(version) {
		return ErrInvalid
	}
	k := r.keys[version]
	if k == nil {
		return ErrNotFound
	}
	k.retired = true
	return nil
}

func (r *Registry) slot(t Target) (**Mark, error) {
	switch t.Kind {
	case MarkRelease:
		if _, ok := semver(t.Version); !ok {
			return nil, ErrInvalid
		}
		rel := r.releases[t.Version]
		if rel == nil {
			return nil, ErrNotFound
		}
		return &rel.revocation, nil
	case MarkCompromise:
		if !validID(t.Key) {
			return nil, ErrInvalid
		}
		k := r.keys[t.Key]
		if k == nil {
			return nil, ErrNotFound
		}
		return &k.compromise, nil
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
			r.keys[t.Key].retired = true
		}
		r.sweep()
	}
	return **slot, nil
}

func (r *Registry) sweep() {
	for _, s := range r.sites {
		for _, id := range s.order {
			run := s.runs[id]
			if active(run.view.State) && r.revoked(r.releases[run.view.Version]) {
				r.invalidate(s, run, ReasonReleaseRevoked)
			}
		}
	}
	all := make([]*scope, 0, len(r.scopes))
	for _, s := range r.scopes {
		all = append(all, s)
	}
	r.retire(all, func(a Acceptance) bool { return r.revoked(r.releases[a.Version]) })
}

func (r *Registry) RevokeRelease(version, reason string, actor Actor) (Mark, error) {
	return r.mark(Target{Kind: MarkRelease, Version: version}, reason, actor)
}

func (r *Registry) CompromiseKey(version, reason string, actor Actor) (Mark, error) {
	return r.mark(Target{Kind: MarkCompromise, Key: version}, reason, actor)
}

func (r *Registry) marked(t Target, actor Actor) (*Mark, **Mark, error) {
	if err := actor.human(); err != nil {
		return nil, nil, err
	}
	slot, err := r.slot(t)
	if err != nil {
		return nil, nil, err
	}
	if *slot == nil {
		return nil, nil, ErrNotFound
	}
	return *slot, slot, nil
}

func (r *Registry) ConfirmMark(t Target, actor Actor) (Mark, error) {
	r.enter()
	defer r.leave()
	m, _, err := r.marked(t, actor)
	if err != nil {
		return Mark{}, err
	}
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
	m, slot, err := r.marked(t, actor)
	if err != nil {
		return Mark{}, err
	}
	switch {
	case m.By != actor.ID:
		return Mark{}, refuse(ReasonNotProposer)
	case m.ConfirmedBy != "":
		return Mark{}, refuse(ReasonNotPending)
	}
	*slot = nil
	return *m, nil
}

func (r *Registry) Release(version string) (Release, *Mark, error) {
	r.enter()
	defer r.leave()
	if _, ok := semver(version); !ok {
		return Release{}, nil, ErrInvalid
	}
	rel := r.releases[version]
	if rel == nil {
		return Release{}, nil, ErrNotFound
	}
	var m *Mark
	if rel.revocation != nil {
		copied := *rel.revocation
		m = &copied
	}
	return copyRelease(rel.release), m, nil
}

func verify(public, message, signature []byte) bool {
	return ed25519.Verify(public, message, signature)
}
