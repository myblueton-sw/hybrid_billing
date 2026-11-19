package commands

import (
	"crypto/rand"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type operationKey struct {
	installation string
	client       string
	command      string
	key          string
}

type operation struct {
	workspace string
	user      string
	digest    [32]byte
	result    Result
	created   time.Time
	pending   bool
}

type ordered struct {
	key operationKey
	op  *operation
}

type Registry struct {
	mu          sync.Mutex
	built       bool
	gate        *gate.Gate
	limits      Limits
	clock       func() time.Time
	secret      [32]byte
	definitions map[string]Definition
	operations  map[operationKey]*operation
	order       []ordered
	first       int
	stored      held
	journal     []Record
	kept        []time.Time
	head        int
	sequence    uint64
	lastKept    time.Time
	records     held
	reserved    int
	lastCreated time.Time
	locks       map[lockKey]*lockEntry
	buckets     map[holder]*bucket
	cleaned     int
	throttled   map[string]map[string]*Throttle
	aggregated  map[unrecordedKey]*Unrecorded
}

func New(g *gate.Gate, limits Limits) (*Registry, error) {
	if g.Clock() == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	r := &Registry{built: true, gate: g, limits: limits, clock: g.Clock(), definitions: map[string]Definition{}, operations: map[operationKey]*operation{},
		stored: newHeld(), records: newHeld(), buckets: map[holder]*bucket{}, cleaned: 1, throttled: map[string]map[string]*Throttle{}, aggregated: map[unrecordedKey]*Unrecorded{}, locks: map[lockKey]*lockEntry{}}
	if _, err := rand.Read(r.secret[:]); err != nil {
		return nil, ErrUnavailable
	}
	return r, nil
}

func knownType(kind FieldType) bool {
	switch kind {
	case ID, Decimal, Integer, Boolean, Text, Timestamp, Enum:
		return true
	}
	return false
}

func (r *Registry) validDefinition(definition Definition) bool {
	permission := definition.Permission
	if !validID(definition.ID) || !validID(permission.Kind) || permission.Action == "" || len(definition.Fields) > r.limits.Fields ||
		definition.Approval != (permission.Action == gate.Approve) || (definition.Approval && !definition.Mutating) ||
		definition.Author && (!definition.Mutating || definition.Approval || !gate.InstallationAction(permission.Action)) ||
		permission.Action == gate.ApproveSupportSession || !r.gate.Describes(permission.Action, permission.Kind, permission.Fields) {
		return false
	}
	refusals := map[Code]bool{}
	for _, code := range definition.Refusals {
		if !declarable[code] || refusals[code] || code == SelfApprovalDenied && definition.Approval {
			return false
		}
		refusals[code] = true
	}
	names := map[string]bool{}
	for _, field := range definition.Fields {
		if !validID(field.Name) || names[field.Name] || !knownType(field.Type) || (field.Type == Enum) != (len(field.Values) > 0) {
			return false
		}
		names[field.Name] = true
		values := map[string]bool{}
		for _, value := range field.Values {
			if value == "" || len(value) > r.limits.Value || values[value] {
				return false
			}
			values[value] = true
		}
	}
	return true
}

func copyDefinition(definition Definition) Definition {
	definition.Permission.Fields = append([]string(nil), definition.Permission.Fields...)
	fields := make([]Field, len(definition.Fields))
	for index, field := range definition.Fields {
		field.Values = append([]string(nil), field.Values...)
		fields[index] = field
	}
	definition.Fields = fields
	definition.Refusals = append([]Code(nil), definition.Refusals...)
	return definition
}

func (r *Registry) Register(definition Definition) error {
	if r == nil || !r.built {
		return ErrUnavailable
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	definition = copyDefinition(definition)
	if handler := reflect.ValueOf(definition.Handler); definition.Handler != nil && handler.Kind() == reflect.Pointer && handler.IsNil() {
		definition.Handler = nil
	}
	if !r.validDefinition(definition) {
		return ErrInvalid
	}
	if _, registered := r.definitions[definition.ID]; registered {
		return ErrConflict
	}
	r.definitions[definition.ID] = definition
	return nil
}

func (r *Registry) Catalogue() []Entry {
	if r == nil || !r.built {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	entries := make([]Entry, 0, len(r.definitions))
	for _, stored := range r.definitions {
		definition := copyDefinition(stored)
		entries = append(entries, Entry{ID: definition.ID, Permission: definition.Permission, Approval: definition.Approval, Author: definition.Author, Mutating: definition.Mutating,
			Rationale: definition.Rationale, Fields: definition.Fields, Refusals: definition.Refusals, Supported: definition.Handler != nil})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return entries
}

func (r *Registry) instant() (at time.Time) {
	defer func() {
		if recover() != nil {
			at = time.Time{}
		}
	}()
	return r.clock()
}
