package components

import "github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"

func (r *Registry) SetPricing(workspace string, p Pricing, actor string) (PricingView, error) {
	if !validID(workspace) || !validID(actor) || !validID(p.Contract) || p.Scale < 0 || p.Scale > maxScale || !exact.ValidMode(p.Mode) {
		return PricingView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	if s := r.workspaces[workspace]; s != nil && s.pricing[p.Contract] != nil {
		old := s.pricing[p.Contract]
		if old.Pricing != p || old.Author != actor {
			return PricingView{}, ErrConflict
		}
		return *old, nil
	}
	s, err := r.open(workspace)
	if err != nil {
		return PricingView{}, err
	}
	v := &PricingView{Pricing: p, Author: actor, Written: now, Digest: pricingDigest(workspace, p, actor)}
	s.pricing[p.Contract] = v
	return *v, nil
}

func copyComponent(c Component) Component {
	c.Includes = append([]Inclusion(nil), c.Includes...)
	return c
}

func sameComponent(a, b Component) bool {
	if a.ID != b.ID || a.Contract != b.Contract || a.Currency != b.Currency || a.Kind != b.Kind || a.From != b.From || a.To != b.To || a.Fee != b.Fee ||
		a.Meter != b.Meter || a.Rate != b.Rate || len(a.Includes) != len(b.Includes) {
		return false
	}
	for index := range a.Includes {
		if a.Includes[index] != b.Includes[index] {
			return false
		}
	}
	return true
}

func viewComponent(c *component) ComponentView {
	v := c.view
	v.Component = copyComponent(c.view.Component)
	return v
}

func (r *Registry) componentForm(c Component) (*component, bool) {
	if !validID(c.ID) || !validID(c.Contract) || !validCurrency(c.Currency) || !validDate(c.From) || c.To != "" && (!validDate(c.To) || c.To < c.From) {
		return nil, false
	}
	made := &component{included: map[string]exact.Fraction{}, all: map[string]bool{}}
	switch {
	case c.Kind == KindUsage:
		rate, printed, ok := r.number(c.Rate, false)
		if !ok || !validID(c.Meter) || c.Fee != "" || len(c.Includes) != 0 {
			return nil, false
		}
		c.Rate, made.rate = printed, rate
	case fixedKinds[c.Kind]:
		fee, printed, ok := r.number(c.Fee, false)
		if !ok || c.Meter != "" || c.Rate != "" {
			return nil, false
		}
		c.Fee, made.fee = printed, fee
		for index, inc := range c.Includes {
			if !validID(inc.Meter) || made.all[inc.Meter] || made.included[inc.Meter].Valid() {
				return nil, false
			}
			if inc.Quantity == All {
				made.all[inc.Meter] = true
				continue
			}
			quantity, text, ok := r.number(inc.Quantity, true)
			if !ok {
				return nil, false
			}
			c.Includes[index].Quantity = text
			made.included[inc.Meter] = quantity
		}
	default:
		return nil, false
	}
	made.view.Component = c
	return made, true
}

func (r *Registry) Define(workspace string, c Component, actor string) (ComponentView, error) {
	if !validID(workspace) || !validID(actor) {
		return ComponentView{}, ErrInvalid
	}
	c = copyComponent(c)
	made, ok := r.componentForm(c)
	if !ok {
		return ComponentView{}, ErrInvalid
	}
	c = made.view.Component
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s != nil && s.components[c.ID] != nil {
		old := s.components[c.ID]
		if !sameComponent(old.view.Component, c) || old.view.Author != actor {
			return ComponentView{}, ErrConflict
		}
		return viewComponent(old), nil
	}
	if s == nil || s.pricing[c.Contract] == nil {
		return ComponentView{}, refuse(ReasonNoPricing)
	}
	p := s.pricing[c.Contract].Pricing
	if fixedKinds[c.Kind] && !fits(made.fee, p.Scale) {
		return ComponentView{}, ErrInvalid
	}
	if len(s.contracts[c.Contract]) >= r.limits.Components || len(c.Includes) > r.limits.Inclusions {
		return ComponentView{}, ErrLimit
	}
	made.view = ComponentView{Component: c, State: StatePending, Author: actor, Written: now, Digest: componentDigest(workspace, c, actor)}
	s.components[c.ID] = made
	s.contracts[c.Contract] = append(s.contracts[c.Contract], made)
	return viewComponent(made), nil
}

func (c *component) includes(meter string) bool {
	return c.all[meter] || c.included[meter].Valid()
}

func (r *Registry) ApproveComponent(workspace, id, actor string) (ComponentView, error) {
	if !validID(workspace) || !validID(id) || !validID(actor) {
		return ComponentView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.components[id] == nil {
		return ComponentView{}, ErrNotFound
	}
	c := s.components[id]
	if c.view.State == StateApproved && c.view.Approver == actor {
		return viewComponent(c), nil
	}
	if c.view.State != StatePending || c.view.Author == actor {
		return ComponentView{}, ErrRejected
	}
	me := c.view.Component
	for _, other := range s.contracts[me.Contract] {
		o := other.view.Component
		if other == c || other.view.State != StateApproved || !overlap(me.From, me.To, o.From, o.To) {
			continue
		}
		if me.Kind == KindUsage && o.Kind == KindUsage && me.Meter == o.Meter {
			return ComponentView{}, refuse(ReasonMeterOverlap)
		}
		for _, inc := range me.Includes {
			if other.includes(inc.Meter) {
				return ComponentView{}, refuse(ReasonInclusionOverlap)
			}
		}
	}
	c.view.State, c.view.Approver, c.view.ApprovedAt = StateApproved, actor, now
	return viewComponent(c), nil
}

func (r *Registry) ComponentOf(workspace, id string) (ComponentView, error) {
	if !validID(workspace) || !validID(id) {
		return ComponentView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.components[id] == nil {
		return ComponentView{}, ErrNotFound
	}
	return viewComponent(s.components[id]), nil
}
