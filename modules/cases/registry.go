package cases

import (
	"sync"
	"time"
)

type finding struct {
	f        Finding
	state    string
	owner    string
	author   string
	digest   string
	written  time.Time
	steps    []Step
	reopener string
	counted  int
}

type caseStep struct {
	kind  string
	step  CaseStep
	actor string
}

type billingCase struct {
	c           Case
	state       string
	openedBy    string
	openedSide  string
	opened      time.Time
	owner       string
	originalDue time.Time
	due         time.Time
	dues        []DueChange
	owners      []OwnerChange
	entries     []EntryView
	public      map[string]int
	notes       map[string]int
	messages    int
	written     int
	steps       map[[2]string]caseStep
	counted     int
	outcome     string
	remedy      Remedy
	closedBy    string
	closed      time.Time
	closings    []ClosingView
	awaiting    bool
	digest      string
}

type space struct {
	findings  map[string]*finding
	steps     map[[2]string]Step
	cases     map[[2]string]*billingCase
	customers map[string]int
}

type Registry struct {
	mu         sync.Mutex
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*space
}

func NewRegistry(limits Limits, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
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

func (r *Registry) space(workspace string) *space {
	s := r.workspaces[workspace]
	if s == nil {
		s = &space{findings: map[string]*finding{}, steps: map[[2]string]Step{}, cases: map[[2]string]*billingCase{}, customers: map[string]int{}}
		r.workspaces[workspace] = s
	}
	return s
}
