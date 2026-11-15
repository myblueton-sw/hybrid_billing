package trust

import (
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type destination struct {
	view    DestinationView
	origin  string
	port    uint16
	literal netip.Addr
}

func printable(raw string) bool {
	if raw == "" || len(raw) > maxLocation {
		return false
	}
	for index := 0; index < len(raw); index++ {
		c := raw[index]
		if c <= ' ' || c >= 0x7f || c == '\\' {
			return false
		}
	}
	return true
}

func dnsHost(host string) (string, bool) {
	for index := 0; index < len(host); index++ {
		if host[index] >= 0x80 {
			return "", false
		}
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "" || len(host) > maxHostBytes {
		return "", false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if label == "" || len(label) > maxLabelBytes || label[0] == '-' || label[len(label)-1] == '-' {
			return "", false
		}
		for index := 0; index < len(label); index++ {
			c := label[index]
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return "", false
			}
		}
	}
	last := labels[len(labels)-1]
	if last[0] < 'a' || last[0] > 'z' {
		return "", false
	}
	return host, true
}

type canonical struct {
	url     string
	origin  string
	port    uint16
	literal netip.Addr
}

func canon(u *url.URL, query bool) (canonical, bool) {
	if u.Scheme != "https" || u.User != nil || u.Opaque != "" || u.Host == "" || strings.HasSuffix(u.Host, ":") || strings.Contains(u.Host, "%") || !query && (u.RawQuery != "" || u.ForceQuery || u.Fragment != "") {
		return canonical{}, false
	}
	port := uint64(defaultPort)
	if text := u.Port(); text != "" {
		parsed, err := strconv.ParseUint(text, 10, 16)
		if err != nil || parsed == 0 || text[0] == '0' {
			return canonical{}, false
		}
		port = parsed
	}
	hostname := u.Hostname()
	var host string
	var literal netip.Addr
	if strings.HasPrefix(u.Host, "[") {
		address, err := netip.ParseAddr(hostname)
		if err != nil || !address.Is6() || address.Is4In6() {
			return canonical{}, false
		}
		host, literal = "["+address.String()+"]", address
	} else if address, err := netip.ParseAddr(hostname); err == nil {
		host, literal = address.String(), address
	} else {
		name, ok := dnsHost(hostname)
		if !ok {
			return canonical{}, false
		}
		host = name
	}
	origin := host + ":" + strconv.FormatUint(port, 10)
	text := "https://" + origin + u.EscapedPath()
	if query && u.RawQuery != "" {
		text += "?" + u.RawQuery
	}
	return canonical{url: text, origin: origin, port: uint16(port), literal: literal}, true
}

func parse(raw string, query bool) (canonical, bool) {
	if !printable(raw) {
		return canonical{}, false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return canonical{}, false
	}
	return canon(u, query)
}

func copyDestination(d Destination) Destination {
	d.Private = append([]netip.Prefix(nil), d.Private...)
	sort.Slice(d.Private, func(i, j int) bool { return d.Private[i].String() < d.Private[j].String() })
	return d
}

func sameDestination(a, b Destination) bool {
	a, b = copyDestination(a), copyDestination(b)
	if a.URL != b.URL || len(a.Private) != len(b.Private) {
		return false
	}
	for index := range a.Private {
		if a.Private[index] != b.Private[index] {
			return false
		}
	}
	return true
}

func viewDestination(d *destination) DestinationView {
	v := d.view
	v.Destination = copyDestination(v.Destination)
	return v
}

func (r *Registry) ProposeDestination(workspace string, d Destination, actor Actor) (DestinationView, error) {
	r.enter()
	defer r.leave()
	if err := r.configure(workspace, actor, d.ID); err != nil || len(d.Private) > r.limits.Allowlist {
		return DestinationView{}, ErrInvalid
	}
	seenPrefix := map[netip.Prefix]bool{}
	for _, prefix := range d.Private {
		if !allowable(prefix) || seenPrefix[prefix] {
			return DestinationView{}, ErrInvalid
		}
		seenPrefix[prefix] = true
	}
	c, ok := parse(d.URL, false)
	if !ok {
		return DestinationView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return DestinationView{}, err
	}
	d.URL = c.url
	s := r.workspaces[workspace]
	if s != nil && s.destinations[d.ID] != nil {
		found := s.destinations[d.ID]
		switch {
		case found.view.State == StateRevoked:
			return DestinationView{}, refuse(ReasonRevoked)
		case found.view.State == StateProposed && found.view.Proposer == actor.ID && sameDestination(found.view.Destination, d):
			return viewDestination(found), nil
		}
		return DestinationView{}, ErrConflict
	}
	if c.literal.IsValid() {
		if reason := classify(c.literal, d.Private); reason != "" {
			return DestinationView{}, refuse(reason)
		}
	}
	if s != nil {
		for _, other := range s.destinations {
			if other.origin == c.origin && other.view.State != StateRevoked {
				return DestinationView{}, ErrConflict
			}
		}
	}
	s, err := r.open(workspace)
	if err != nil {
		return DestinationView{}, err
	}
	if len(s.destinations) >= r.limits.Destinations {
		return DestinationView{}, ErrLimit
	}
	made := &destination{view: DestinationView{Destination: copyDestination(d), State: StateProposed, Proposer: actor.ID}, origin: c.origin, port: c.port, literal: c.literal}
	s.destinations[d.ID] = made
	return viewDestination(made), nil
}

func (r *Registry) destinationOf(workspace, id string) (*space, *destination, error) {
	s := r.workspaces[workspace]
	if s == nil || s.destinations[id] == nil {
		return nil, nil, ErrNotFound
	}
	return s, s.destinations[id], nil
}

func (r *Registry) ApproveDestination(workspace, id string, actor Actor) (DestinationView, error) {
	r.enter()
	defer r.leave()
	if err := r.configure(workspace, actor, id); err != nil {
		return DestinationView{}, err
	}
	if err := actor.human(); err != nil {
		return DestinationView{}, err
	}
	_, d, err := r.destinationOf(workspace, id)
	if err != nil {
		return DestinationView{}, err
	}
	switch {
	case d.view.State == StateActive && d.view.Approver == actor.ID:
		return viewDestination(d), nil
	case d.view.State != StateProposed:
		return DestinationView{}, refuse(ReasonNotPending)
	case d.view.Proposer == actor.ID:
		return DestinationView{}, refuse(ReasonSelfApproval)
	}
	d.view.State, d.view.Approver = StateActive, actor.ID
	return viewDestination(d), nil
}

func (r *Registry) RevokeDestination(workspace, id string, actor Actor) (DestinationView, error) {
	r.enter()
	defer r.leave()
	if err := r.configure(workspace, actor, id); err != nil {
		return DestinationView{}, err
	}
	if err := actor.human(); err != nil {
		return DestinationView{}, err
	}
	_, d, err := r.destinationOf(workspace, id)
	if err != nil {
		return DestinationView{}, err
	}
	d.view.State = StateRevoked
	return viewDestination(d), nil
}

func (r *Registry) DestinationOf(workspace, id string) (DestinationView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return DestinationView{}, ErrInvalid
	}
	_, d, err := r.destinationOf(workspace, id)
	if err != nil {
		return DestinationView{}, err
	}
	return viewDestination(d), nil
}

func (r *Registry) approved(workspace, id string) (*space, *destination, error) {
	s, d, err := r.destinationOf(workspace, id)
	if err != nil {
		return nil, nil, err
	}
	if d.view.State != StateActive {
		return nil, nil, refuse(ReasonNotActive)
	}
	return s, d, nil
}

func (r *Registry) CheckAddress(workspace, id string, address netip.AddrPort) error {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !address.IsValid() {
		return ErrInvalid
	}
	_, d, err := r.approved(workspace, id)
	if err != nil {
		return err
	}
	if address.Port() != d.port {
		return refuse(ReasonPortMismatch)
	}
	if d.literal.IsValid() && address.Addr().Unmap() != d.literal {
		return refuse(ReasonAddressMismatch)
	}
	if reason := classify(address.Addr(), d.view.Destination.Private); reason != "" {
		return refuse(reason)
	}
	return nil
}

func (r *Registry) CheckRedirect(workspace, id, current string, hop int, location string) (Redirect, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || hop < 0 || len(location) > maxLocation {
		return Redirect{}, ErrInvalid
	}
	s, d, err := r.approved(workspace, id)
	if err != nil {
		return Redirect{}, err
	}
	at, ok := parse(current, true)
	if !ok || at.origin != d.origin {
		return Redirect{}, ErrInvalid
	}
	if hop >= r.limits.Redirects {
		return Redirect{}, refuse(ReasonRedirectLimit)
	}
	base, _ := url.Parse(at.url)
	if !printable(location) {
		return Redirect{}, refuse(ReasonUnapprovedRedirect)
	}
	next, err := url.Parse(location)
	if err != nil {
		return Redirect{}, refuse(ReasonUnapprovedRedirect)
	}
	target, ok := canon(base.ResolveReference(next), true)
	if !ok {
		return Redirect{}, refuse(ReasonUnapprovedRedirect)
	}
	for otherID, other := range s.destinations {
		if other.origin == target.origin && other.view.State == StateActive {
			return Redirect{Destination: otherID, URL: target.url, Hop: hop + 1}, nil
		}
	}
	return Redirect{}, refuse(ReasonUnapprovedRedirect)
}
