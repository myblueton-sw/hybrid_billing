package sandbox

import (
	"net/netip"
	"strconv"
	"strings"
)

const (
	maxHostBytes = 253
	maxLabel     = 63
	maxKeyBytes  = 64
)

var (
	sensitive = []string{"pass", "pwd", "secret", "token", "key", "credential", "private", "auth", "cookie", "session", "signature", "sig", "pin", "otp",
		"cert", "dsn", "connection", "card", "pan", "iban", "account", "customer", "phone", "ssn", "tax", "invoice",
		"url", "uri", "host", "endpoint", "server", "vault", "bucket", "path", "addr", "domain", "port", "ip", "target", "proxy", "smtp", "remote", "callback"}
	reserved = []string{".test", ".invalid", ".example"}
	kinds    = []string{DestinationERP, DestinationMailbox, DestinationWebhook, DestinationPayment, DestinationProvider, DestinationNotification}
)

func oneOf(value string, set ...string) bool {
	for _, item := range set {
		if item == value {
			return true
		}
	}
	return false
}

func lowerName(text string, first func(byte) bool, limit int) bool {
	if text == "" || len(text) > limit || !first(text[0]) {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

func letter(c byte) bool { return c >= 'a' && c <= 'z' }

func validEnvID(id string) bool {
	return lowerName(id, func(byte) bool { return true }, maxEnvID) && !strings.Contains(id, "_")
}

func validDestID(id string) bool {
	if id == "" || len(id) > maxKeyBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func validStore(name string) bool { return lowerName(name, letter, maxKeyBytes-1) }

func validKey(key string) bool { return lowerName(key, letter, maxKeyBytes) }

func validEnum(value string) bool {
	if value == "" || len(value) > maxKeyBytes || !letter(value[0]) {
		return false
	}
	for index := 0; index < len(value); index++ {
		c := value[index]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func validPath(path string) bool {
	if len(path) < 2 || len(path) > maxPathBytes || path[0] != '/' {
		return false
	}
	segments := strings.Split(path[1:], "/")
	if len(segments) > maxSegments {
		return false
	}
	for _, segment := range segments {
		if segment == "" || len(segment) > maxSegmentBytes {
			return false
		}
		for index := 0; index < len(segment); index++ {
			c := segment[index]
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
				return false
			}
		}
	}
	return true
}

func within(path, root string) bool { return path == root || strings.HasPrefix(path, root+"/") }

func validPrefix(prefix string) bool {
	body, ok := strings.CutSuffix(prefix, "-")
	if !ok || len(body) < 2 || len(body) > 16 {
		return false
	}
	for index := 0; index < len(body); index++ {
		c := body[index]
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func validHost(host string) bool {
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.Zone() == "" && !addr.Is4In6() && addr.String() == host
	}
	if host == "" || len(host) > maxHostBytes {
		return false
	}
	label, letters := 0, false
	for index := 0; index < len(host); index++ {
		c := host[index]
		switch {
		case c == '.':
			if label == 0 || host[index-1] == '-' {
				return false
			}
			label, letters = 0, false
		case c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-':
			if c == '-' && label == 0 || label == maxLabel {
				return false
			}
			letters = letters || c >= 'a' && c <= 'z'
			label++
		default:
			return false
		}
	}
	return letters && host[len(host)-1] != '-'
}

func isReserved(host string) bool {
	for _, suffix := range reserved {
		if strings.HasSuffix(host, suffix) && len(host) > len(suffix) {
			return true
		}
	}
	return false
}

func sensitiveKey(key string) bool {
	for _, fragment := range sensitive {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}

func (e Environment) valid() bool {
	return validEnvID(e.ID) && oneOf(e.Mode, ModeDemo, ModeProduction) && validStore(e.Database) && validStore(e.KeyRing) && validPath(e.EvidencePath) &&
		validPrefix(e.NumberPrefix)
}

func (d Destination) valid() bool {
	return validDestID(d.ID) && validEnvID(d.Environment) && oneOf(d.Kind, kinds...) && validHost(d.Host) && d.Port >= 1 && d.Port <= 65535
}

func (f FieldSpec) valid() bool {
	if !validKey(f.Key) {
		return false
	}
	switch f.Kind {
	case FieldEnum:
		if len(f.Values) == 0 || len(f.Values) > maxValues {
			return false
		}
		for index, value := range f.Values {
			if !validEnum(value) || index > 0 && f.Values[index-1] >= value {
				return false
			}
		}
		return true
	case FieldDestination:
		return len(f.Values) == 1 && oneOf(f.Values[0], kinds...)
	case FieldInteger, FieldFlag:
		return len(f.Values) == 0
	}
	return false
}

func (s Schema) valid() bool {
	if !validID(s.ID) || len(s.Fields) == 0 || len(s.Fields) > maxFields {
		return false
	}
	for index, f := range s.Fields {
		if !f.valid() || index > 0 && s.Fields[index-1].Key >= f.Key {
			return false
		}
	}
	return true
}

func clean(c Change) (Change, bool) {
	out := Change{Kind: c.Kind}
	switch c.Kind {
	case ChangeEnvironment:
		out.Environment = c.Environment
		return out, c.Environment.valid()
	case ChangeDestination:
		out.Destination = c.Destination
		return out, c.Destination.valid()
	case ChangeSchema:
		out.Schema = copySchema(c.Schema)
		return out, out.Schema.valid()
	}
	return out, false
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

func copyChange(c Change) Change {
	c.Schema = copySchema(c.Schema)
	return c
}

func copyProposal(p ProposalView) ProposalView {
	p.Change = copyChange(p.Change)
	return p
}

func encodeChange(c Change) []byte {
	var e encoder
	v, d := c.Environment, c.Destination
	for _, value := range []string{c.Kind, v.ID, v.Mode, v.Database, v.KeyRing, v.EvidencePath, v.NumberPrefix, d.ID, d.Environment, d.Kind, d.Host, c.Schema.ID} {
		e.text(value)
	}
	e.count(d.Port)
	e.count(len(c.Schema.Fields))
	for _, f := range c.Schema.Fields {
		e.text(f.Key)
		e.text(f.Kind)
		e.count(len(f.Values))
		for _, value := range f.Values {
			e.text(value)
		}
	}
	return e.out.Bytes()
}

func endpoint(host string, port int) string { return host + "|" + strconv.Itoa(port) }

func (r *Registry) check(c Change) error {
	switch c.Kind {
	case ChangeEnvironment:
		return r.checkEnvironment(c.Environment)
	case ChangeDestination:
		return r.checkDestination(c.Destination)
	}
	if r.schemas[c.Schema.ID] != nil {
		return ErrConflict
	}
	for _, f := range c.Schema.Fields {
		if sensitiveKey(f.Key) {
			return refuse(ReasonSensitiveKey)
		}
	}
	if len(r.schemas) >= maxSchemas {
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkEnvironment(e Environment) error {
	if r.environments[e.ID] != nil {
		return ErrConflict
	}
	demo := e.Mode == ModeDemo
	switch {
	case demo != strings.HasPrefix(e.NumberPrefix, MarkingDemo):
		return refuse(ReasonDemoPrefix)
	case demo && r.production == 0:
		return refuse(ReasonNoProduction)
	}
	for _, other := range r.environments {
		o := other.view
		if o.Database == e.Database || o.KeyRing == e.KeyRing || within(o.EvidencePath, e.EvidencePath) || within(e.EvidencePath, o.EvidencePath) ||
			o.NumberPrefix == e.NumberPrefix {
			return refuse(ReasonNamespaceShared)
		}
	}
	if len(r.environments) >= maxEnvironments {
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkDestination(d Destination) error {
	env := r.environments[d.Environment]
	switch {
	case env == nil:
		return ErrNotFound
	case r.destinations[d.ID] != nil || r.endpoints[endpoint(d.Host, d.Port)] != "":
		return ErrConflict
	case env.view.Mode != ModeDemo:
		return refuse(ReasonNotDemo)
	case !isReserved(d.Host):
		return refuse(ReasonNotReserved)
	case len(env.destinations) >= maxDestinations:
		return ErrLimit
	}
	return nil
}

func (r *Registry) apply(c Change) {
	c = copyChange(c)
	switch c.Kind {
	case ChangeEnvironment:
		r.environments[c.Environment.ID] = &environment{view: c.Environment, resets: map[uint64]*reset{}, keys: map[string]uint64{}}
		if c.Environment.Mode == ModeProduction {
			r.production++
		}
	case ChangeDestination:
		d := c.Destination
		r.destinations[d.ID] = &destination{view: d, enabled: true}
		r.endpoints[endpoint(d.Host, d.Port)] = d.ID
		env := r.environments[d.Environment]
		env.destinations = append(env.destinations, d.ID)
	case ChangeSchema:
		s := c.Schema
		r.schemas[s.ID] = &s
	}
}
