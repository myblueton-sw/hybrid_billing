package overview

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type entry struct {
	fact     Fact
	seq      int
	received time.Time
	actor    string
}

type subject struct {
	key     Subject
	sources map[string]bool
	names   []string
	entries []*entry
}

type space struct {
	facts     map[string]*entry
	revisions map[[2]string]*entry
	bySource  map[string][]*entry
	subjects  map[Subject]*subject
	order     []*entry
	saved     map[[2]string]*SavedView
}

type Registry struct {
	mu         sync.Mutex
	config     Config
	limits     Limits
	clock      func() time.Time
	now        time.Time
	aead       cipher.AEAD
	workspaces map[string]*space
}

func NewRegistry(config Config, limits Limits, clock func() time.Time) (*Registry, error) {
	if config.Scope == nil || config.Random == nil || clock == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	key := make([]byte, 32)
	if _, err := io.ReadFull(config.Random, key); err != nil {
		return nil, ErrInvalid
	}
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)
	return &Registry{config: config, limits: limits, clock: clock, aead: aead, workspaces: map[string]*space{}}, nil
}

func (l Limits) valid() bool {
	if l.Stale < minStale || l.Stale > maxStale {
		return false
	}
	pairs := [][2]int{{l.Workspaces, maxWorkspaces}, {l.Subjects, maxSubjects}, {l.Facts, maxFacts}, {l.Filters, maxFilters}, {l.Page, maxPage}}
	for _, p := range pairs {
		if p[0] < 1 || p[0] > p[1] {
			return false
		}
	}
	return true
}

func (r *Registry) enter() time.Time {
	r.mu.Lock()
	now := r.clock().UTC().Round(0)
	if now.After(r.now) {
		r.now = now
	}
	return r.now
}

func (r *Registry) leave() { r.mu.Unlock() }

func refuse(reason string) error { return Refusal{Reason: reason} }

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func optionalID(id string) bool { return id == "" || validID(id) }

func (a Actor) valid() bool {
	return validID(a.ID) && (a.Kind == KindHuman || a.Kind == KindAssistant || a.Kind == KindSystem)
}

func (a Actor) reader() error {
	if a.Kind == KindAssistant {
		return refuse(ReasonAINotAllowed)
	}
	return nil
}

func validDate(text string) bool {
	parsed, err := time.Parse(time.DateOnly, text)
	return err == nil && parsed.Format(time.DateOnly) == text
}

func validPeriod(text string) bool {
	parsed, err := time.Parse("2006-01", text)
	return err == nil && parsed.Format("2006-01") == text
}

func validCurrency(code string) bool {
	if len(code) != 3 {
		return false
	}
	for index := 0; index < 3; index++ {
		if code[index] < 'A' || code[index] > 'Z' {
			return false
		}
	}
	return true
}

func canonical(text string) bool {
	text = strings.TrimPrefix(text, "-")
	whole, fraction, dotted := strings.Cut(text, ".")
	if whole == "" || len(whole)+len(fraction) > maxDigits || len(fraction) > maxScale || dotted && (fraction == "" || fraction[len(fraction)-1] == '0') || len(whole) > 1 && whole[0] == '0' {
		return false
	}
	for _, part := range []string{whole, fraction} {
		for index := 0; index < len(part); index++ {
			if part[index] < '0' || part[index] > '9' {
				return false
			}
		}
	}
	return true
}

func signedCanonical(text string) bool {
	return canonical(text) && text != "-0" && !(strings.HasPrefix(text, "-") && strings.Trim(text, "-0.") == "")
}

var stages = map[string]bool{StageCollected: true, StageRated: true, StageReviewed: true, StageApproved: true, StageIssued: true, StageDelivered: true, StageSettled: true, StageBlocked: true}

var meanings = map[string]bool{MeaningCost: true, MeaningSales: true, MeaningAllocation: true, MeaningOutstanding: true}

func subjectForm(s Subject) bool {
	return validID(s.Customer) && validID(s.Contract) && validID(s.Entity) && validPeriod(s.Period) && validCurrency(s.Currency)
}

func factForm(f Fact) bool {
	if !validID(f.ID) || !subjectForm(f.Subject) || !validID(f.Source.ID) || f.Source.Revision < 1 || !stages[f.Stage] ||
		f.Due != "" && !validDate(f.Due) || !optionalID(f.Owner) || !optionalID(f.Next) || f.Occurred.IsZero() || len(f.Evidence) > maxEvidence {
		return false
	}
	for meaning, amount := range f.Amounts {
		if !meanings[meaning] || !signedCanonical(amount) {
			return false
		}
	}
	for _, ref := range f.Evidence {
		if !validID(ref) {
			return false
		}
	}
	return true
}

func copyFact(f Fact) Fact {
	amounts := make(map[string]string, len(f.Amounts))
	for k, v := range f.Amounts {
		amounts[k] = v
	}
	f.Amounts = amounts
	f.Evidence = append([]string(nil), f.Evidence...)
	f.Occurred = f.Occurred.UTC().Round(0)
	return f
}

func sameFact(a, b Fact) bool {
	if a.ID != b.ID || a.Subject != b.Subject || a.Source != b.Source || a.Stage != b.Stage || a.Due != b.Due || a.Owner != b.Owner || a.Next != b.Next ||
		!a.Occurred.Equal(b.Occurred) || len(a.Amounts) != len(b.Amounts) || len(a.Evidence) != len(b.Evidence) {
		return false
	}
	for k, v := range a.Amounts {
		if b.Amounts[k] != v {
			return false
		}
	}
	for index := range a.Evidence {
		if a.Evidence[index] != b.Evidence[index] {
			return false
		}
	}
	return true
}

func (r *Registry) open(workspace string) (*space, error) {
	if s := r.workspaces[workspace]; s != nil {
		return s, nil
	}
	if len(r.workspaces) >= r.limits.Workspaces {
		return nil, ErrLimit
	}
	s := &space{facts: map[string]*entry{}, revisions: map[[2]string]*entry{}, bySource: map[string][]*entry{}, subjects: map[Subject]*subject{},
		saved: map[[2]string]*SavedView{}}
	r.workspaces[workspace] = s
	return s, nil
}

func (r *Registry) Ingest(workspace string, f Fact, actor Actor) (Entry, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !factForm(f) {
		return Entry{}, ErrInvalid
	}
	switch actor.Kind {
	case KindAssistant:
		return Entry{}, refuse(ReasonAINotAllowed)
	case KindHuman:
		return Entry{}, refuse(ReasonNotHuman)
	}
	f = copyFact(f)
	if s := r.workspaces[workspace]; s != nil {
		if found := s.facts[f.ID]; found != nil {
			if sameFact(found.fact, f) {
				return view(found, false), nil
			}
			return Entry{}, ErrConflict
		}
		if s.revisions[[2]string{f.Source.ID, strconv.Itoa(f.Source.Revision)}] != nil {
			return Entry{}, refuse(ReasonRevisionConflict)
		}
		if sub := s.subjects[f.Subject]; sub != nil && s.becomesCurrent(f) {
			for _, other := range s.included(sub, len(s.order)) {
				for meaning := range f.Amounts {
					if _, ok := other.fact.Amounts[meaning]; ok && other.fact.Source.ID != f.Source.ID {
						return Entry{}, refuse(ReasonMeaningOwned)
					}
				}
			}
		}
	}
	if f.Occurred.After(now) {
		return Entry{}, refuse(ReasonFutureOccurrence)
	}
	s, err := r.open(workspace)
	if err != nil {
		return Entry{}, err
	}
	if len(s.order) >= r.limits.Facts || s.subjects[f.Subject] == nil && len(s.subjects) >= r.limits.Subjects {
		return Entry{}, ErrLimit
	}
	e := &entry{fact: f, seq: len(s.order) + 1, received: now, actor: actor.ID}
	s.order = append(s.order, e)
	s.facts[f.ID] = e
	s.revisions[[2]string{f.Source.ID, strconv.Itoa(f.Source.Revision)}] = e
	s.bySource[f.Source.ID] = append(s.bySource[f.Source.ID], e)
	sub := s.subjects[f.Subject]
	if sub == nil {
		sub = &subject{key: f.Subject, sources: map[string]bool{}}
		s.subjects[f.Subject] = sub
	}
	sub.sources[f.Source.ID] = true
	sub.entries = append(sub.entries, e)
	return view(e, false), nil
}

func view(e *entry, superseded bool) Entry {
	return Entry{Fact: copyFact(e.fact), Received: e.received, Actor: e.actor, Superseded: superseded}
}

func (r *Registry) seal(label string, parts ...string) (string, error) {
	nonce := make([]byte, r.aead.NonceSize())
	if _, err := io.ReadFull(r.config.Random, nonce); err != nil {
		return "", refuse(ReasonRandom)
	}
	sealed := r.aead.Seal(nonce, nonce, []byte(strings.Join(parts, "\x00")), []byte(label))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (r *Registry) unseal(label, token string) ([]string, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) < r.aead.NonceSize() {
		return nil, false
	}
	plain, err := r.aead.Open(nil, raw[:r.aead.NonceSize()], raw[r.aead.NonceSize():], []byte(label))
	if err != nil {
		return nil, false
	}
	return strings.Split(string(plain), "\x00"), true
}

func digest(parts ...string) string {
	var b strings.Builder
	for _, part := range parts {
		b.WriteString(strconv.Itoa(len(part)))
		b.WriteByte(':')
		b.WriteString(part)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func sorted(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
