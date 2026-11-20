package licensemode

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrStorage    = errors.New("license state storage unavailable; outcome may be unknown")
	ErrCheckpoint = errors.New("invalid license checkpoint")
	ErrOwned      = errors.New("license state already owned")
)

type Durable struct {
	mu           sync.Mutex
	registry     *Registry
	store        *durableFile
	installation string
	product      string
	failed       bool
	closed       bool
}

func CreateDurable(dir, installation, product string, actor Actor, config Config, clock func() time.Time) (*Durable, error) {
	if !validID(installation) || !validID(product) {
		return nil, ErrInvalid
	}
	r, err := NewRegistry(config, clock)
	if err != nil {
		return nil, err
	}
	if err = r.Install(installation, product, actor); err != nil {
		return nil, err
	}
	store, err := openDurableFile(dir, true)
	if err != nil {
		return nil, err
	}
	d := &Durable{registry: r, store: store, installation: installation, product: product}
	if err = d.save(); err != nil {
		_ = store.close()
		return nil, err
	}
	return d, nil
}

func OpenDurable(dir, installation, product string, config Config, clock func() time.Time) (*Durable, error) {
	if !validID(installation) || !validID(product) {
		return nil, ErrInvalid
	}
	r, err := NewRegistry(config, clock)
	if err != nil {
		return nil, err
	}
	store, err := openDurableFile(dir, false)
	if err != nil {
		return nil, err
	}
	data, err := store.read()
	if err == nil {
		err = restoreCheckpoint(r, data, installation, product)
	}
	if err != nil {
		_ = store.close()
		return nil, err
	}
	return &Durable{registry: r, store: store, installation: installation, product: product}, nil
}

func (d *Durable) save() error {
	data, err := makeCheckpoint(d.registry, d.installation, d.product)
	if err == nil {
		err = d.store.write(data)
	}
	if err != nil {
		d.failed = true
		return ErrStorage
	}
	return nil
}

func durableCall[T any](d *Durable, call func(*Registry) (T, error)) (T, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var zero T
	if d.failed || d.closed {
		return zero, ErrStorage
	}
	value, domainErr := call(d.registry)
	if err := d.save(); err != nil {
		return zero, err
	}
	return value, domainErr
}

func (d *Durable) SelectMode(mode string, actor Actor) error {
	_, err := durableCall(d, func(r *Registry) (struct{}, error) { return struct{}{}, r.SelectMode(d.installation, mode, actor) })
	return err
}
func (d *Durable) Activate(s Signed, acceptance Acceptance, actor Actor) (Activation, error) {
	return durableCall(d, func(r *Registry) (Activation, error) { return r.Activate(d.installation, s, acceptance, actor) })
}
func (d *Durable) Register(s Signed, actor Actor) (string, error) {
	return durableCall(d, func(r *Registry) (string, error) { return r.Register(d.installation, s, actor) })
}
func (d *Durable) Status() (StatusView, error) {
	return durableCall(d, func(r *Registry) (StatusView, error) { return r.Status(d.installation) })
}
func (d *Durable) Propose(c Change, actor Actor) (ProposalView, error) {
	return durableCall(d, func(r *Registry) (ProposalView, error) {
		if r.counter == ^uint64(0) {
			return ProposalView{}, ErrLimit
		}
		return r.Propose(c, actor)
	})
}
func (d *Durable) Approve(number uint64, actor Actor) (ProposalView, error) {
	return durableCall(d, func(r *Registry) (ProposalView, error) { return r.Approve(number, actor) })
}
func (d *Durable) Withdraw(number uint64, actor Actor) (ProposalView, error) {
	return durableCall(d, func(r *Registry) (ProposalView, error) { return r.Withdraw(number, actor) })
}
func (d *Durable) Proposal(number uint64) (ProposalView, error) {
	return durableCall(d, func(r *Registry) (ProposalView, error) { return r.Proposal(number) })
}
func (d *Durable) PendingProposers() ([]string, error) {
	return durableCall(d, func(r *Registry) ([]string, error) {
		r.enter()
		defer r.leave()
		out := make([]string, 0, len(r.pending))
		for _, number := range r.pending {
			out = append(out, r.proposals[number].view.Proposer)
		}
		return out, nil
	})
}
func (d *Durable) Key(id string) (KeyView, error) {
	return durableCall(d, func(r *Registry) (KeyView, error) { return r.Key(id) })
}
func (d *Durable) History() ([]ModeChange, []Submission, error) {
	type result struct {
		modes       []ModeChange
		submissions []Submission
	}
	v, err := durableCall(d, func(r *Registry) (result, error) { m, s, e := r.History(d.installation); return result{m, s}, e })
	return v.modes, v.submissions, err
}
func (d *Durable) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	return d.store.close()
}
