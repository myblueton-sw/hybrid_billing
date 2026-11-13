package normalize

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type target struct {
	stated   Target
	print    string
	index    map[string]int
	currency int
	start    int
	end      int
}

type revision struct {
	key      MappingKey
	number   uint64
	target   *target
	entries  []Entry
	print    string
	sources  map[string]bool
	complete bool
	partial  []string
	skipped  []string
	gaps     []string
	noCost   bool
	period   bool
	at       time.Time
}

func (v *revision) possible() bool { return len(v.gaps) == 0 && !v.noCost }

type workspace struct {
	runs map[string]*run
	rows int
}

type Registry struct {
	mu         sync.Mutex
	built      bool
	limits     Limits
	profile    decimal.Profile
	clock      func() time.Time
	last       time.Time
	targets    map[string]*target
	mappings   map[MappingKey][]*revision
	workspaces map[string]*workspace
}

func NewRegistry(limits Limits, profile decimal.Profile, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || !profile.Valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{built: true, limits: limits, profile: profile, clock: clock, targets: map[string]*target{}, mappings: map[MappingKey][]*revision{}, workspaces: map[string]*workspace{}}, nil
}

func (r *Registry) enter() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	if !r.built {
		return false
	}
	if read := r.clock(); read.After(r.last) {
		r.last = read
	}
	return true
}

func (r *Registry) leave() {
	if r != nil {
		r.mu.Unlock()
	}
}

func join(fields ...string) string {
	var out strings.Builder
	for _, field := range fields {
		out.WriteString(strconv.Itoa(len(field)))
		out.WriteByte(':')
		out.WriteString(field)
	}
	return out.String()
}

func (r *Registry) DeclareTarget(stated Target) error {
	defer r.leave()
	if !r.enter() || !validID(stated.ID) || len(stated.Fields) == 0 {
		return ErrInvalid
	}
	if len(stated.Fields) > r.limits.Fields {
		return ErrLimit
	}
	made := &target{index: map[string]int{}, currency: -1, start: -1, end: -1}
	prints := []string{stated.ID, stated.CostCurrency, stated.PeriodStart, stated.PeriodEnd}
	costs := false
	for position, field := range stated.Fields {
		if _, twice := made.index[field.Name]; twice || !validID(field.Name) || !field.Kind.known() {
			return ErrInvalid
		}
		if (field.Kind == Cost) != field.Cost.target() || (field.Kind != Cost && field.Cost != "") {
			return ErrInvalid
		}
		costs = costs || field.Kind == Cost
		made.index[field.Name] = position
		prints = append(prints, field.Name, string(field.Kind), string(field.Cost), strconv.FormatBool(field.Format), strconv.FormatBool(field.Billing))
	}
	named := func(name string, kind Kind) int {
		position, known := made.index[name]
		if !known || stated.Fields[position].Kind != kind {
			return -1
		}
		return position
	}
	if costs || stated.CostCurrency != "" {
		if made.currency = named(stated.CostCurrency, Currency); made.currency < 0 || !costs {
			return ErrInvalid
		}
	}
	if stated.PeriodStart != "" || stated.PeriodEnd != "" {
		made.start, made.end = named(stated.PeriodStart, Time), named(stated.PeriodEnd, Time)
		if made.start < 0 || made.end < 0 || made.start == made.end {
			return ErrInvalid
		}
	}
	made.print = join(prints...)
	if held := r.targets[stated.ID]; held != nil {
		if held.print != made.print {
			return ErrConflict
		}
		return nil
	}
	if len(r.targets) >= r.limits.Targets {
		return ErrLimit
	}
	made.stated = stated
	made.stated.Fields = append([]Field(nil), stated.Fields...)
	r.targets[stated.ID] = made
	return nil
}

func (r *Registry) arrange(key MappingKey, aim *target, entries []Entry) (*revision, error) {
	if len(entries) != len(aim.stated.Fields) {
		return nil, ErrInvalid
	}
	made := &revision{key: key, target: aim, entries: make([]Entry, len(entries)), sources: map[string]bool{}}
	placed := make([]bool, len(entries))
	for _, entry := range entries {
		position, known := aim.index[entry.Field]
		if !known || placed[position] {
			return nil, ErrInvalid
		}
		placed[position] = true
		if entry.Missing != "" {
			if !entry.Missing.known() || entry.Source != "" || entry.SourceKind != "" || entry.SourceCost != "" || entry.Transform != "" || entry.Power != 0 {
				return nil, ErrInvalid
			}
		} else {
			if !validID(entry.Source) || !entry.SourceKind.known() || (entry.SourceKind == Cost) != (entry.SourceCost.target() || entry.SourceCost == Sale) || (entry.SourceKind != Cost && entry.SourceCost != "") {
				return nil, ErrInvalid
			}
			switch entry.Transform {
			case Copy, Negate:
				if entry.Power != 0 {
					return nil, ErrInvalid
				}
			case Scale:
				if entry.Power == 0 || entry.Power < -12 || entry.Power > 12 {
					return nil, ErrInvalid
				}
			default:
				return nil, ErrInvalid
			}
		}
		made.entries[position] = entry
	}
	declared := map[string]string{}
	mapped := 0
	prints := []string{}
	for position, entry := range made.entries {
		field := aim.stated.Fields[position]
		prints = append(prints, entry.Field, entry.Source, string(entry.SourceKind), string(entry.SourceCost), string(entry.Transform), strconv.Itoa(entry.Power), string(entry.Missing))
		if entry.Missing != "" {
			switch {
			case entry.Missing != NotApplicable:
				made.partial = append(made.partial, field.Name)
			case field.Format || field.Billing:
				return nil, ErrRejected
			default:
				made.skipped = append(made.skipped, field.Name)
			}
			if field.Billing {
				made.gaps = append(made.gaps, field.Name)
			}
			continue
		}
		if entry.SourceKind != field.Kind || entry.SourceCost != field.Cost {
			return nil, ErrRejected
		}
		if entry.Transform != Copy && !field.Kind.numeric() {
			return nil, ErrRejected
		}
		what := string(entry.SourceKind) + "/" + string(entry.SourceCost)
		if earlier, known := declared[entry.Source]; known && earlier != what {
			return nil, ErrRejected
		}
		declared[entry.Source] = what
		made.sources[entry.Source] = true
		mapped++
	}
	maps := func(position int) bool { return position >= 0 && made.entries[position].Missing == "" }
	made.noCost = true
	for position, field := range aim.stated.Fields {
		if field.Kind != Cost {
			continue
		}
		if field.Billing {
			made.noCost = false
		}
		if maps(position) && !maps(aim.currency) {
			return nil, ErrRejected
		}
	}
	if mapped == 0 || maps(aim.start) != maps(aim.end) {
		return nil, ErrRejected
	}
	made.period = maps(aim.start)
	made.complete = len(made.partial) == 0
	made.print = join(prints...)
	return made, nil
}

func (r *Registry) DeclareMapping(key MappingKey, expected uint64, entries []Entry) (uint64, error) {
	defer r.leave()
	if !r.enter() || !key.valid() {
		return 0, ErrInvalid
	}
	aim := r.targets[key.Target]
	if aim == nil {
		return 0, ErrNotFound
	}
	made, err := r.arrange(key, aim, entries)
	if err != nil {
		return 0, err
	}
	held, known := r.mappings[key]
	if len(held) > 0 && held[len(held)-1].print == made.print {
		return uint64(len(held)), nil
	}
	if expected != uint64(len(held)) {
		return 0, ErrConflict
	}
	if !known && len(r.mappings) >= r.limits.Mappings || len(held) >= r.limits.Revisions {
		return 0, ErrLimit
	}
	made.number, made.at = expected+1, r.last
	r.mappings[key] = append(held, made)
	return made.number, nil
}

func (v *revision) view() RevisionView {
	return RevisionView{Key: v.key, Number: v.number, Entries: append([]Entry(nil), v.entries...), Complete: v.complete, Partial: append([]string(nil), v.partial...),
		NotApplicable: append([]string(nil), v.skipped...), BillingPossible: v.possible(), BillingGaps: append([]string(nil), v.gaps...), NoRequiredCost: v.noCost, At: v.at}
}

func (r *Registry) Revision(key MappingKey, number uint64) (RevisionView, error) {
	defer r.leave()
	if !r.enter() || !key.valid() || number < 1 {
		return RevisionView{}, ErrInvalid
	}
	held := r.mappings[key]
	if number > uint64(len(held)) {
		return RevisionView{}, ErrNotFound
	}
	return held[number-1].view(), nil
}

func (r *Registry) Latest(key MappingKey) (RevisionView, error) {
	defer r.leave()
	if !r.enter() || !key.valid() {
		return RevisionView{}, ErrInvalid
	}
	held := r.mappings[key]
	if len(held) == 0 {
		return RevisionView{}, ErrNotFound
	}
	return held[len(held)-1].view(), nil
}
