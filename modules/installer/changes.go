package installer

import (
	"crypto/ed25519"
	"net/netip"
	"time"
)

var translated = []netip.Prefix{netip.MustParsePrefix("::/96"), netip.MustParsePrefix("64:ff9b::/96"), netip.MustParsePrefix("64:ff9b:1::/48")}

const (
	maxOverrides = 64
	maxRanges    = 8
	minIPv4Bits  = 16
	minIPv6Bits  = 48
	maxHostBytes = 253
	maxLabel     = 63
)

func validHost(host string) bool {
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.Zone() == "" && !addr.Is4In6() && addr.String() == host
	}
	if host == "" || len(host) > maxHostBytes {
		return false
	}
	label, letter := 0, false
	for index := 0; index < len(host); index++ {
		c := host[index]
		switch {
		case c == '.':
			if label == 0 || host[index-1] == '-' {
				return false
			}
			label, letter = 0, false
		case c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-':
			if c == '-' && label == 0 || label == maxLabel {
				return false
			}
			letter = letter || c >= 'a' && c <= 'z'
			label++
		default:
			return false
		}
	}
	return letter && host[len(host)-1] != '-'
}

func validRange(text string) (netip.Prefix, bool) {
	p, err := netip.ParsePrefix(text)
	if err != nil || p.Masked().String() != text || p.Addr().Is4In6() {
		return netip.Prefix{}, false
	}
	if p.Addr().Is4() {
		return p, p.Bits() >= minIPv4Bits
	}
	for _, special := range translated {
		if p.Overlaps(special) {
			return netip.Prefix{}, false
		}
	}
	return p, p.Bits() >= minIPv6Bits
}

func (d Destination) valid() bool {
	if !validID(d.Site) || !validID(d.ID) || !validHost(d.Host) || d.Port < 1 || d.Port > 65535 || len(d.Ranges) == 0 || len(d.Ranges) > maxRanges {
		return false
	}
	addr, literal := netip.ParseAddr(d.Host)
	inside := false
	for index, text := range d.Ranges {
		p, ok := validRange(text)
		if !ok || index > 0 && d.Ranges[index-1] >= text {
			return false
		}
		inside = inside || literal == nil && p.Contains(addr)
	}
	return literal != nil || inside
}

func (c SiteConfig) valid() bool {
	if _, ok := semver(c.Version); !ok || !validID(c.Site) || len(c.Ports) > maxOverrides || len(c.Volumes) > maxOverrides {
		return false
	}
	for index, o := range c.Ports {
		if !validID(o.Service) || !o.Port.valid() || o.Number < 1 || o.Number > 65535 {
			return false
		}
		if index > 0 {
			before := c.Ports[index-1]
			if !(before.Service < o.Service || before.Service == o.Service && portLess(before.Port, o.Port)) {
				return false
			}
		}
	}
	for index, o := range c.Volumes {
		if !validID(o.Volume) || !validID(o.Host) || index > 0 && c.Volumes[index-1].Volume >= o.Volume {
			return false
		}
	}
	return true
}

func clean(c Change) (Change, string, bool) {
	out := Change{Kind: c.Kind}
	switch c.Kind {
	case ChangeKey:
		k := c.Key
		k.NotBefore, k.NotAfter = k.NotBefore.UTC().Round(0), k.NotAfter.UTC().Round(0)
		k.Public, k.Proof = append([]byte(nil), k.Public...), append([]byte(nil), k.Proof...)
		out.Key = k
		return out, registerScope, validID(k.Version) && !k.NotBefore.IsZero() && k.NotBefore.Before(k.NotAfter) && len(k.Proof) == ed25519.SignatureSize
	case ChangeRelease:
		out.Release, out.Signature = copyRelease(c.Release), append([]byte(nil), c.Signature...)
		return out, registerScope, c.Release.valid() && len(c.Signature) == ed25519.SignatureSize
	case ChangeSite:
		out.Site = c.Site
		return out, registerScope, validID(c.Site.ID) && validID(c.Site.Tenant) && c.Site.Platform.valid() && validID(c.Site.Probe)
	case ChangeProbe:
		out.Site = Site{ID: c.Site.ID, Probe: c.Site.Probe}
		return out, c.Site.ID, validID(c.Site.ID) && validID(c.Site.Probe)
	case ChangeConfig:
		out.Config = c.Config
		out.Config.Ports = append([]PortOverride(nil), c.Config.Ports...)
		out.Config.Volumes = append([]VolumeOverride(nil), c.Config.Volumes...)
		return out, c.Config.Site, c.Config.valid()
	case ChangeDestination:
		out.Destination = c.Destination
		out.Destination.Ranges = append([]string(nil), c.Destination.Ranges...)
		return out, c.Destination.Site, c.Destination.valid()
	case ChangePlan, ChangeUpgrade, ChangeRollback:
		a := c.Acceptance
		out.Acceptance = a
		_, ok := semver(a.Version)
		ok = ok && validID(a.Site) && validHex(a.Hash)
		switch c.Kind {
		case ChangePlan:
			return out, a.Site, ok && validID(a.Admin) && a.Run == ""
		case ChangeUpgrade:
			return out, a.Site, ok && a.Admin == "" && a.Run == ""
		}
		return out, a.Site, ok && a.Admin == "" && validID(a.Run)
	}
	return out, registerScope, false
}

func encodeChange(c Change) []byte {
	var e encoder
	e.text(c.Kind)
	e.text(c.Key.Version)
	e.text(string(c.Key.Public))
	for _, at := range []time.Time{c.Key.NotBefore, c.Key.NotAfter} {
		e.number(at.Unix())
		e.number(int64(at.Nanosecond()))
	}
	e.text(string(c.Key.Proof))
	e.text(string(c.Release.encode()))
	e.text(string(c.Signature))
	for _, value := range []string{c.Site.ID, c.Site.Tenant, c.Site.Platform.OS, c.Site.Platform.Arch, c.Site.Probe, c.Config.Site, c.Config.Version} {
		e.text(value)
	}
	e.count(len(c.Config.Ports))
	for _, o := range c.Config.Ports {
		e.text(o.Service)
		e.count(o.Port.Number)
		e.text(o.Port.Protocol)
		e.count(o.Number)
	}
	e.count(len(c.Config.Volumes))
	for _, o := range c.Config.Volumes {
		e.text(o.Volume)
		e.text(o.Host)
	}
	d := c.Destination
	for _, value := range []string{d.Site, d.ID, d.Host} {
		e.text(value)
	}
	e.count(d.Port)
	e.count(len(d.Ranges))
	for _, value := range d.Ranges {
		e.text(value)
	}
	a := c.Acceptance
	for _, value := range []string{a.Site, a.Version, a.Hash, a.Admin, a.Run} {
		e.text(value)
	}
	return e.out.Bytes()
}

func copyChange(c Change) Change {
	c.Key.Public, c.Key.Proof = append([]byte(nil), c.Key.Public...), append([]byte(nil), c.Key.Proof...)
	c.Release, c.Signature = copyRelease(c.Release), append([]byte(nil), c.Signature...)
	c.Config.Ports = append([]PortOverride(nil), c.Config.Ports...)
	c.Config.Volumes = append([]VolumeOverride(nil), c.Config.Volumes...)
	c.Destination.Ranges = append([]string(nil), c.Destination.Ranges...)
	return c
}

func copyProposal(p ProposalView) ProposalView {
	p.Change = copyChange(p.Change)
	return p
}

func (r *Registry) check(c Change, now time.Time) error {
	switch c.Kind {
	case ChangeKey:
		return r.checkKey(c.Key)
	case ChangeRelease:
		return r.checkRelease(c.Release, c.Signature, now)
	case ChangeSite:
		switch {
		case r.sites[c.Site.ID] != nil:
			return ErrConflict
		case r.probes[c.Site.Probe] != "":
			return refuse(ReasonProbeTaken)
		case len(r.sites) >= r.limits.Sites:
			return ErrLimit
		}
		return nil
	case ChangeProbe:
		switch {
		case r.probes[c.Site.Probe] != "":
			return refuse(ReasonProbeTaken)
		case r.sites[c.Site.ID].probes >= probesPerSite:
			return ErrLimit
		}
		return nil
	case ChangeConfig:
		return r.checkConfig(c.Config)
	case ChangeDestination:
		return r.checkDestination(c.Destination)
	}
	return r.checkPlan(c.Kind, c.Acceptance, now)
}

func (r *Registry) checkKey(k ReleaseKey) error {
	switch {
	case r.names[k.Version]:
		return ErrConflict
	case weak(k.Public):
		return refuse(ReasonWeakKey)
	case k.NotAfter.Sub(k.NotBefore) > r.limits.KeyLife:
		return refuse(ReasonKeyLife)
	case !verify(k.Public, challenge(k.Version), k.Proof):
		return refuse(ReasonKeyMismatch)
	}
	active := 0
	for _, other := range r.keys {
		if !other.retired {
			active++
		}
	}
	if len(r.names) >= r.limits.Keys*namesPerKey || active >= r.limits.Keys {
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkRelease(rel Release, signature []byte, now time.Time) error {
	k := r.keys[rel.Key]
	switch {
	case r.releases[rel.Version] != nil:
		return ErrConflict
	case k == nil:
		return ErrNotFound
	case k.retired || now.Before(k.notBefore) || now.After(k.notAfter):
		return refuse(ReasonKeyInactive)
	case !verify(k.public, rel.encode(), signature):
		return refuse(ReasonSignatureInvalid)
	case len(r.releases) >= r.limits.Releases:
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkConfig(c SiteConfig) error {
	rel := r.releases[c.Version]
	if rel == nil {
		return ErrNotFound
	}
	ports := map[string]bool{}
	for _, s := range rel.release.Services {
		for _, p := range s.Ports {
			ports[s.Name+"/"+p.key()] = true
		}
	}
	for _, o := range c.Ports {
		if !ports[o.Service+"/"+o.Port.key()] {
			return refuse(ReasonNotDeclared)
		}
	}
	for _, o := range c.Volumes {
		if !declaredVolume(rel.release, o.Volume) {
			return refuse(ReasonNotDeclared)
		}
	}
	return nil
}

func declaredVolume(rel Release, name string) bool {
	for _, v := range rel.Volumes {
		if v.Name == name {
			return true
		}
	}
	return false
}

func (r *Registry) checkDestination(d Destination) error {
	s := r.sites[d.Site]
	if s.destinations[d.ID].ID != "" {
		return ErrConflict
	}
	for _, other := range s.destinations {
		if other.Host == d.Host && other.Port == d.Port {
			return ErrConflict
		}
	}
	if len(s.destinations) >= r.limits.Destinations {
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkPlan(kind string, a Acceptance, now time.Time) error {
	if r.releases[a.Version] == nil {
		return ErrNotFound
	}
	s := r.sites[a.Site]
	if kind != ChangePlan && r.activeRun(s) {
		return refuse(ReasonRunOpen)
	}
	view, err := r.kindPreview(s, kind, a, now, nil)
	switch {
	case err != nil:
		return err
	case view.Hash != a.Hash:
		return refuse(ReasonPlanChanged)
	case len(view.Conflicts) > 0:
		return refuse(ReasonConflicts)
	}
	return nil
}

func (r *Registry) apply(c Change) {
	switch c.Kind {
	case ChangeKey:
		r.keys[c.Key.Version] = &key{version: c.Key.Version, public: append([]byte(nil), c.Key.Public...), notBefore: c.Key.NotBefore, notAfter: c.Key.NotAfter}
		r.names[c.Key.Version] = true
	case ChangeRelease:
		r.releases[c.Release.Version] = &release{release: copyRelease(c.Release), signature: append([]byte(nil), c.Signature...), digest: digestOf(c.Signature)}
	case ChangeSite:
		r.sites[c.Site.ID] = &site{view: c.Site, probes: 1, destinations: map[string]Destination{}, runs: map[string]*run{}, created: map[string][]string{},
			createdVols: map[string]*ownedVolume{}, removed: map[string]bool{}, rolledBack: map[string]bool{}}
		r.probes[c.Site.Probe] = c.Site.ID
	case ChangeProbe:
		s := r.sites[c.Site.ID]
		s.view.Probe = c.Site.Probe
		s.probes++
		r.probes[c.Site.Probe] = c.Site.ID
		r.changed(s, ReasonProbeChanged)
	case ChangeConfig:
		s := r.sites[c.Config.Site]
		s.config = copyChange(c).Config
		s.revision++
		r.changed(s, ReasonConfigChanged)
	case ChangeDestination:
		d := c.Destination
		d.Ranges = append([]string(nil), d.Ranges...)
		r.sites[d.Site].destinations[d.ID] = d
	}
}

func (r *Registry) changed(s *site, reason string) {
	for _, id := range s.order {
		if run := s.runs[id]; active(run.view.State) {
			r.invalidate(s, run, reason)
		}
	}
	r.forget(s)
	s.observation = nil
	r.retire([]*scope{r.scope(s.view.ID)}, func(Acceptance) bool { return true })
}

func (r *Registry) invalidate(s *site, run *run, reason string) {
	run.view.State, run.view.Reason, run.view.Note = RunInvalidated, reason, ""
	r.ended(s, run)
	r.forget(s)
}

func (r *Registry) forget(s *site) {
	if s.code != nil {
		delete(r.codes, *s.code)
		s.code = nil
	}
}
