package support

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxItems      = 256
	maxFields     = 128
	maxValueBytes = 256
	maxInputBytes = 4 << 20
	maxOutput     = 4 << 20
	maxUploads    = 16
	maxDuration   = 366 * 24 * time.Hour
	bundleTag     = "hybrid-billing/support-bundle/v1"
	timeLayout    = "2006-01-02T15:04:05Z"
)

var pseudonymClasses = []string{ClassAddress, ClassEmail, ClassHost, ClassIdentifier, ClassUser}

func normal(text string) string {
	text = strings.TrimFunc(text, unicode.IsSpace)
	var b strings.Builder
	for index := 0; index < len(text); index++ {
		c := text[index]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b.WriteByte(c)
	}
	return b.String()
}

func formOK(items []Item) bool {
	if len(items) == 0 || len(items) > maxItems {
		return false
	}
	total := 0
	for _, item := range items {
		if len(item.Fields) > maxFields || len(item.Source) > maxValueBytes || !utf8.ValidString(item.Source) {
			return false
		}
		for _, f := range item.Fields {
			for _, text := range []string{f.Key, f.Text} {
				if len(text) > maxValueBytes || !utf8.ValidString(text) {
					return false
				}
				total += len(text)
			}
		}
	}
	return total <= maxInputBytes
}

type builder struct {
	seen  map[string]map[string]string
	drops map[[2]string]int
}

func (b *builder) pseudonym(class, value string) string {
	if b.seen[class] == nil {
		b.seen[class] = map[string]string{}
	}
	key := normal(value)
	if name, ok := b.seen[class][key]; ok {
		return name
	}
	name := class + "-" + strconv.Itoa(len(b.seen[class])+1)
	b.seen[class][key] = name
	return name
}

func zeroExcept(f Field, keep string) bool {
	return (keep == "int" || f.Int == 0) && (keep == "text" || f.Text == "") && (keep == "time" || f.Time.IsZero()) && (keep == "duration" || f.Duration == 0)
}

func (b *builder) render(spec FieldSpec, f Field) (string, string) {
	switch spec.Class {
	case ClassEnum:
		if !zeroExcept(f, "text") {
			return "", DropWrongShape
		}
		index := sort.SearchStrings(spec.Values, f.Text)
		if index == len(spec.Values) || spec.Values[index] != f.Text {
			return "", DropOutOfRange
		}
		return spec.Values[index], ""
	case ClassInteger:
		if !zeroExcept(f, "int") {
			return "", DropWrongShape
		}
		return strconv.FormatInt(f.Int, 10), ""
	case ClassDuration:
		if !zeroExcept(f, "duration") {
			return "", DropWrongShape
		}
		if f.Duration < 0 || f.Duration > maxDuration {
			return "", DropOutOfRange
		}
		return strconv.FormatInt(f.Duration.Milliseconds(), 10), ""
	case ClassTimestamp:
		if !zeroExcept(f, "time") || f.Time.IsZero() {
			return "", DropWrongShape
		}
		if year := f.Time.UTC().Year(); year < 1970 || year > 9999 {
			return "", DropOutOfRange
		}
		return f.Time.UTC().Format(timeLayout), ""
	case ClassVersion:
		if !zeroExcept(f, "text") {
			return "", DropWrongShape
		}
		if _, ok := semver(f.Text); !ok {
			return "", DropOutOfRange
		}
		return f.Text, ""
	case ClassIdentifier:
		if f.Text == "" {
			if !zeroExcept(f, "int") {
				return "", DropWrongShape
			}
			return b.pseudonym(spec.Class, strconv.FormatInt(f.Int, 10)), ""
		}
	}
	if !zeroExcept(f, "text") || f.Text == "" {
		return "", DropWrongShape
	}
	return b.pseudonym(spec.Class, f.Text), ""
}

func (r *Registry) build(items []Item) ([]Kept, Summary) {
	b := &builder{seen: map[string]map[string]string{}, drops: map[[2]string]int{}}
	kept := []Kept{}
	for _, item := range items {
		specs, known := r.schemas[item.Source]
		if !known {
			b.drops[[2]string{"", DropUnknownSource}]++
			continue
		}
		keys := map[string]bool{}
		duplicate := false
		for _, f := range item.Fields {
			duplicate = duplicate || keys[f.Key]
			keys[f.Key] = true
		}
		if duplicate {
			b.drops[[2]string{item.Source, DropDuplicateKey}]++
			continue
		}
		out := Kept{Source: item.Source, Fields: []Rendered{}}
		for _, f := range item.Fields {
			spec, declared := specs[f.Key]
			if !declared {
				b.drops[[2]string{item.Source, DropUndeclared}]++
				continue
			}
			value, reason := b.render(spec, f)
			if reason != "" {
				b.drops[[2]string{item.Source, reason}]++
				continue
			}
			out.Fields = append(out.Fields, Rendered{Key: spec.Key, Class: spec.Class, Value: value})
		}
		if len(out.Fields) == 0 {
			b.drops[[2]string{item.Source, DropNoFields}]++
			continue
		}
		sort.Slice(out.Fields, func(x, y int) bool { return out.Fields[x].Key < out.Fields[y].Key })
		out.Number = len(kept) + 1
		kept = append(kept, out)
	}
	summary := Summary{Kept: len(kept), Pseudonyms: map[string]int{}, Drops: []Drop{}}
	for _, class := range pseudonymClasses {
		summary.Pseudonyms[class] = len(b.seen[class])
	}
	for key, count := range b.drops {
		summary.Drops = append(summary.Drops, Drop{Source: key[0], Reason: key[1], Count: count})
	}
	sort.Slice(summary.Drops, func(x, y int) bool {
		a, c := summary.Drops[x], summary.Drops[y]
		return a.Source < c.Source || a.Source == c.Source && a.Reason < c.Reason
	})
	return kept, summary
}

func encode(handle, id string, items []Kept, s Summary) []byte {
	var e encoder
	e.text(bundleTag)
	e.text(handle)
	e.text(id)
	e.count(len(items))
	for _, item := range items {
		e.count(item.Number)
		e.text(item.Source)
		e.count(len(item.Fields))
		for _, f := range item.Fields {
			e.text(f.Key)
			e.text(f.Class)
			e.text(f.Value)
		}
	}
	e.count(s.Kept)
	for _, class := range pseudonymClasses {
		e.text(class)
		e.count(s.Pseudonyms[class])
	}
	e.count(len(s.Drops))
	for _, d := range s.Drops {
		e.text(d.Source)
		e.text(d.Reason)
		e.count(d.Count)
	}
	return e.out.Bytes()
}

func (r *Registry) site(id string) (*site, error) {
	if !validID(id) {
		return nil, ErrInvalid
	}
	s := r.sites[id]
	if s == nil {
		return nil, ErrNotFound
	}
	return s, nil
}

func (r *Registry) expire(s *site, now time.Time) {
	for id, b := range s.bundles {
		if !now.Before(b.view.Expiry) {
			r.remove(s, id)
		}
	}
}

func (r *Registry) Create(id string, items []Item, actor Actor) (BundleView, error) {
	now := r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return BundleView{}, err
	}
	if !formOK(items) {
		return BundleView{}, ErrInvalid
	}
	s, err := r.site(id)
	if err != nil {
		return BundleView{}, err
	}
	r.expire(s, now)
	kept, summary := r.build(items)
	if len(kept) == 0 {
		return BundleView{}, refuse(ReasonEmpty)
	}
	name := "bundle-" + strconv.FormatUint(s.counter+1, 10)
	data := encode(s.handle, name, kept, summary)
	size := int64(len(data))
	if len(data) > maxOutput || len(s.bundles) >= r.limits.Bundles || s.bytes+size > r.limits.SiteBytes {
		return BundleView{}, ErrLimit
	}
	if r.bytes+size > r.limits.Bytes {
		for _, other := range r.sites {
			r.expire(other, now)
		}
		if r.bytes+size > r.limits.Bytes {
			return BundleView{}, ErrLimit
		}
	}
	s.counter++
	sum := sha256.Sum256(data)
	b := &bundle{view: BundleView{ID: name, Site: id, State: StateDraft, Items: kept, Summary: summary, Digest: hex.EncodeToString(sum[:]), Bytes: data,
		Expiry: now.Add(r.limits.BundleLife)}, sources: map[string]bool{}, previewed: map[string]bool{}}
	for _, item := range kept {
		b.sources[item.Source] = true
	}
	s.bundles[name] = b
	s.bytes += size
	r.bytes += size
	return view(b), nil
}

func view(b *bundle) BundleView {
	v := b.view
	v.Bytes = append([]byte(nil), v.Bytes...)
	v.Items = make([]Kept, len(b.view.Items))
	for index, item := range b.view.Items {
		item.Fields = append([]Rendered(nil), item.Fields...)
		v.Items[index] = item
	}
	v.Summary.Pseudonyms = map[string]int{}
	for class, count := range b.view.Summary.Pseudonyms {
		v.Summary.Pseudonyms[class] = count
	}
	v.Summary.Drops = append([]Drop(nil), b.view.Summary.Drops...)
	return v
}

func (r *Registry) located(id, name string, actor Actor) (*site, *bundle, error) {
	now := r.now
	if err := actor.human(); err != nil {
		return nil, nil, err
	}
	s, err := r.site(id)
	if err != nil {
		return nil, nil, err
	}
	if !validID(name) {
		return nil, nil, ErrInvalid
	}
	b := s.bundles[name]
	if b != nil && !now.Before(b.view.Expiry) {
		r.expire(s, now)
		return nil, nil, refuse(ReasonExpired)
	}
	r.expire(s, now)
	if b == nil {
		return nil, nil, ErrNotFound
	}
	return s, b, nil
}

func (r *Registry) Preview(id, name string, actor Actor) (BundleView, error) {
	r.enter()
	defer r.leave()
	_, b, err := r.located(id, name, actor)
	if err != nil {
		return BundleView{}, err
	}
	b.view.State = StatePreviewed
	b.previewed[actor.ID] = true
	return view(b), nil
}

func (r *Registry) ready(id, name, digest string, actor Actor) (*site, *bundle, error) {
	if !validHex(digest) {
		return nil, nil, ErrInvalid
	}
	s, b, err := r.located(id, name, actor)
	switch {
	case err != nil:
		return nil, nil, err
	case b.view.State != StatePreviewed || !b.previewed[actor.ID]:
		return nil, nil, refuse(ReasonNotPreviewed)
	case b.view.Digest != digest:
		return nil, nil, refuse(ReasonDigestMismatch)
	}
	return s, b, nil
}

func (r *Registry) Export(id, name, digest string, actor Actor) ([]byte, error) {
	r.enter()
	defer r.leave()
	_, b, err := r.ready(id, name, digest, actor)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), b.view.Bytes...), nil
}

func (r *Registry) Upload(id, name, digest, destination string, actor Actor) (UploadRecord, error) {
	now := r.enter()
	defer r.leave()
	if !validID(destination) {
		return UploadRecord{}, ErrInvalid
	}
	s, b, err := r.ready(id, name, digest, actor)
	if err != nil {
		return UploadRecord{}, err
	}
	d := r.destinations[destination]
	if s.destination != destination || d == nil || d.disabled {
		return UploadRecord{}, refuse(ReasonUploadDisabled)
	}
	for _, record := range b.uploads {
		if record.Destination == destination {
			return record, nil
		}
	}
	if len(b.uploads) >= maxUploads {
		return UploadRecord{}, ErrLimit
	}
	record := UploadRecord{Destination: destination, Digest: b.view.Digest, By: actor.ID, At: now}
	b.uploads = append(b.uploads, record)
	return record, nil
}

func (r *Registry) Discard(id, name string, actor Actor) error {
	r.enter()
	defer r.leave()
	s, _, err := r.located(id, name, actor)
	if err != nil {
		return err
	}
	r.remove(s, name)
	return nil
}
