package support

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	maxSpecs     = 128
	maxValues    = 64
	maxKeyBytes  = 64
	maxEnumBytes = 64
)

var (
	keyPattern       = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	enumPattern      = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)
	pseudonymPattern = regexp.MustCompile(`^(host|address|email|user|identifier)-[0-9]+$`)
	sensitive        = []string{"pass", "pwd", "secret", "token", "key", "credential", "private", "auth", "cookie", "session", "signature", "sig", "pin", "otp",
		"cert", "dsn", "connection", "card", "pan", "iban", "account", "customer", "phone", "ssn", "tax", "invoice"}
	sources = map[string]bool{SourceLog: true, SourceConfig: true, SourceDiagnostic: true, SourceMetric: true, SourceVersion: true}
	classes = map[string]bool{ClassEnum: true, ClassInteger: true, ClassDuration: true, ClassTimestamp: true, ClassVersion: true, ClassHost: true,
		ClassAddress: true, ClassEmail: true, ClassUser: true, ClassIdentifier: true}
)

func sensitiveKey(key string) bool {
	for _, fragment := range sensitive {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}

func (s Schema) valid() bool {
	if !sources[s.Source] || len(s.Fields) == 0 || len(s.Fields) > maxSpecs {
		return false
	}
	for index, f := range s.Fields {
		if !keyPattern.MatchString(f.Key) || index > 0 && s.Fields[index-1].Key >= f.Key || !classes[f.Class] {
			return false
		}
		if f.Class != ClassEnum {
			if len(f.Values) > 0 {
				return false
			}
			continue
		}
		if len(f.Values) == 0 || len(f.Values) > maxValues {
			return false
		}
		for at, v := range f.Values {
			if !enumPattern.MatchString(v) || pseudonymPattern.MatchString(v) || at > 0 && f.Values[at-1] >= v {
				return false
			}
		}
	}
	return true
}

func copySchema(s Schema) Schema {
	fields := make([]FieldSpec, len(s.Fields))
	for index, f := range s.Fields {
		f.Values = append([]string(nil), f.Values...)
		fields[index] = f
	}
	s.Fields = fields
	return s
}

func clean(c Change) (Change, string, bool) {
	out := Change{Kind: c.Kind}
	switch c.Kind {
	case ChangeSite:
		out.Site = c.Site
		return out, registerScope, validID(c.Site.ID) && validID(c.Site.Tenant)
	case ChangeSchema:
		if !c.Schema.valid() {
			return out, registerScope, false
		}
		out.Schema = copySchema(c.Schema)
		return out, registerScope, true
	case ChangeDestination:
		out.Destination = c.Destination
		return out, registerScope, validID(c.Destination.ID) && validHost(c.Destination.Host) && c.Destination.Port >= 1 && c.Destination.Port <= 65535
	case ChangeUpload:
		out.Upload = c.Upload
		return out, c.Upload.Site, validID(c.Upload.Site) && validID(c.Upload.Destination)
	}
	return out, registerScope, false
}

func encodeChange(c Change) []byte {
	var e encoder
	for _, value := range []string{c.Kind, c.Site.ID, c.Site.Tenant, c.Schema.Source} {
		e.text(value)
	}
	e.count(len(c.Schema.Fields))
	for _, f := range c.Schema.Fields {
		e.text(f.Key)
		e.text(f.Class)
		e.count(len(f.Values))
		for _, v := range f.Values {
			e.text(v)
		}
	}
	for _, value := range []string{c.Destination.ID, c.Destination.Host, c.Upload.Site, c.Upload.Destination} {
		e.text(value)
	}
	e.count(c.Destination.Port)
	return e.out.Bytes()
}

func copyChange(c Change) Change {
	c.Schema = copySchema(c.Schema)
	return c
}

func copyProposal(p ProposalView) ProposalView {
	p.Change = copyChange(p.Change)
	return p
}

func (r *Registry) check(c Change, _ time.Time) error {
	switch c.Kind {
	case ChangeSite:
		switch {
		case r.sites[c.Site.ID] != nil:
			return ErrConflict
		case len(r.sites) >= r.limits.Sites:
			return ErrLimit
		}
	case ChangeSchema:
		for _, f := range c.Schema.Fields {
			if sensitiveKey(f.Key) {
				return refuse(ReasonSensitiveKey)
			}
		}
	case ChangeDestination:
		switch {
		case r.destinations[c.Destination.ID] != nil:
			return ErrConflict
		case len(r.destinations) >= r.limits.Destinations:
			return ErrLimit
		}
	case ChangeUpload:
		d := r.destinations[c.Upload.Destination]
		switch {
		case d == nil:
			return ErrNotFound
		case d.disabled:
			return refuse(ReasonDestinationDisabled)
		}
	}
	return nil
}

func (r *Registry) apply(c Change, now time.Time) {
	switch c.Kind {
	case ChangeSite:
		r.handles++
		r.sites[c.Site.ID] = &site{view: c.Site, handle: "site-" + strconv.FormatUint(r.handles, 10), bundles: map[string]*bundle{}}
	case ChangeSchema:
		specs := map[string]FieldSpec{}
		for _, f := range copySchema(c.Schema).Fields {
			specs[f.Key] = f
		}
		r.schemas[c.Schema.Source] = specs
		for _, s := range r.sites {
			for id, b := range s.bundles {
				if b.sources[c.Schema.Source] {
					r.remove(s, id)
				}
			}
		}
	case ChangeDestination:
		r.destinations[c.Destination.ID] = &destination{view: c.Destination}
	case ChangeUpload:
		s := r.sites[c.Upload.Site]
		s.destination = c.Upload.Destination
		reset(s)
	}
}

func reset(s *site) {
	for _, b := range s.bundles {
		b.view.State = StateDraft
		b.previewed = map[string]bool{}
	}
}

func (r *Registry) remove(s *site, id string) {
	size := int64(len(s.bundles[id].view.Bytes))
	s.bytes -= size
	r.bytes -= size
	delete(s.bundles, id)
}

func (r *Registry) DisableUpload(id string, actor Actor) error {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	if !validID(id) {
		return ErrInvalid
	}
	s := r.sites[id]
	if s == nil {
		return ErrNotFound
	}
	s.destination = ""
	reset(s)
	return nil
}

func (r *Registry) DisableDestination(id string, actor Actor) error {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	if !validID(id) {
		return ErrInvalid
	}
	d := r.destinations[id]
	if d == nil {
		return ErrNotFound
	}
	d.disabled = true
	for _, s := range r.sites {
		if s.destination == id {
			s.destination = ""
			reset(s)
		}
	}
	return nil
}
