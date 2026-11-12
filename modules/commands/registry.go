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
}

type Registry struct {
	mu          sync.Mutex
	mutations   sync.Mutex
	built       bool
	gate        *gate.Gate
	limits      Limits
	clock       func() time.Time
	secret      [32]byte
	definitions map[string]Definition
	operations  map[operationKey]*operation
	journal     []Record
	reserved    int
}

func New(g *gate.Gate, limits Limits) (*Registry, error) {
	if g.Clock() == nil || !limits.valid() {
		return nil, ErrInvalid
	}
	r := &Registry{built: true, gate: g, limits: limits, clock: g.Clock(), definitions: map[string]Definition{}, operations: map[operationKey]*operation{}}
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
		permission.Action == gate.ApproveSupportSession || !r.gate.Describes(permission.Action, permission.Kind, permission.Fields) {
		return false
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
		entries = append(entries, Entry{ID: definition.ID, Permission: definition.Permission, Approval: definition.Approval, Mutating: definition.Mutating,
			Rationale: definition.Rationale, Fields: definition.Fields, Supported: definition.Handler != nil})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return entries
}

func (r *Registry) reserve() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.journal)+r.reserved >= r.limits.Journal {
		return false
	}
	r.reserved++
	return true
}

func (r *Registry) instant() (at time.Time) {
	defer func() {
		if recover() != nil {
			at = time.Time{}
		}
	}()
	return r.clock()
}

func (r *Registry) record(entry Record) {
	entry.At = r.instant()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reserved--
	entry.Sequence = uint64(len(r.journal)) + 1
	r.journal = append(r.journal, entry)
}

func (r *Registry) Journal(after uint64, limit int) []Record {
	if r == nil || !r.built || limit < 1 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if after >= uint64(len(r.journal)) {
		return []Record{}
	}
	rest := r.journal[after:]
	return append([]Record{}, rest[:min(limit, len(rest))]...)
}
