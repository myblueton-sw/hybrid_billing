package formula

import (
	"sort"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type computed struct {
	exact  fraction
	seen   fraction
	value  string
	reason string
}

type run struct {
	compiled *Compiled
	values   []*fraction
	results  []computed
}

func (r *run) within(value fraction) (fraction, string, error) {
	if value.digits() > r.compiled.limits.Digits {
		return fraction{}, "", ErrLimit
	}
	return value, "", nil
}

func (r *run) eval(s *step) (fraction, string, error) {
	switch s.kind {
	case NodeConstant:
		return s.value, "", nil
	case NodeVariable:
		if r.values[s.variable] == nil {
			return fraction{}, ReasonNotKnown, nil
		}
		return *r.values[s.variable], "", nil
	case NodeReference:
		result := r.results[s.formula]
		return result.seen, result.reason, nil
	}
	left, reason, err := r.eval(s.left)
	if err != nil || reason != "" {
		return fraction{}, reason, err
	}
	right, reason, err := r.eval(s.right)
	if err != nil || reason != "" {
		return fraction{}, reason, err
	}
	switch s.kind {
	case NodeAdd:
		return r.within(left.plus(right))
	case NodeSub:
		return r.within(left.minus(right))
	case NodeMul:
		return r.within(left.times(right))
	case NodeDiv:
		if right.sign() == 0 {
			return fraction{}, ReasonDivideByZero, nil
		}
		return r.within(left.over(right))
	case NodeMin:
		if right.less(left) {
			return r.within(right)
		}
		return r.within(left)
	case NodeMax:
		if left.less(right) {
			return r.within(right)
		}
		return r.within(left)
	}
	if right.sign() < 0 || !right.less(whole(1)) {
		return fraction{}, ReasonMarginRange, nil
	}
	return r.within(left.over(whole(1).minus(right)))
}

func (r *run) outcome(index int) Outcome {
	u, result := r.compiled.units[index], r.results[index]
	out := Outcome{Formula: u.id, Type: u.output, Reason: result.reason}
	if result.reason == "" {
		out.Value, out.Exact = result.value, result.exact.text()
	}
	return out
}

func formed(profile decimal.Profile, inputs []Input) bool {
	given := map[string]bool{}
	for _, input := range inputs {
		if !validID(input.Name) || given[input.Name] || len(input.Value) > maxNumberBytes {
			return false
		}
		given[input.Name] = true
		if input.Value == "" {
			continue
		}
		if _, err := decimal.Parse(profile, input.Value); err != nil {
			return false
		}
	}
	return true
}

func (c *Compiled) Evaluate(target string, inputs []Input) (Result, error) {
	if len(inputs) > c.limits.Variables {
		return Result{}, ErrLimit
	}
	if !formed(c.profile, inputs) {
		return Result{}, ErrInvalid
	}
	values := make([]*fraction, len(c.variables))
	for _, input := range inputs {
		slot, declared := c.variables[input.Name]
		if !declared {
			return Result{}, ErrInvalid
		}
		if input.Value != "" {
			parsed, _ := decimal.Parse(c.profile, input.Value)
			value := of(parsed)
			values[slot] = &value
		}
	}
	if !validID(target) {
		return Result{}, ErrInvalid
	}
	chosen, ok := c.targets[target]
	if !ok {
		return Result{}, ErrNotFound
	}
	needed := make([]bool, len(c.units))
	needed[chosen] = true
	stack := []int{chosen}
	for len(stack) > 0 {
		index := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, ref := range c.units[index].refs {
			if !needed[ref] {
				needed[ref] = true
				stack = append(stack, ref)
			}
		}
	}
	r := &run{compiled: c, values: values, results: make([]computed, len(c.units))}
	for _, index := range c.order {
		if !needed[index] {
			continue
		}
		u := c.units[index]
		exact, reason, err := r.eval(u.root)
		if err != nil {
			return Result{}, err
		}
		result := computed{reason: reason}
		if reason == "" {
			units := exact.quanta(u.scale, u.mode)
			rounded, err := decimal.New(c.profile, units, u.scale)
			if err != nil {
				return Result{}, ErrLimit
			}
			result.exact, result.seen, result.value = exact, exact, rounded.String()
			if u.output.Kind == KindMoney {
				result.seen = lowest(units, power(u.scale))
			}
		}
		r.results[index] = result
	}
	out := Result{Semantics: Semantics, ProfileID: c.profile.ID(), ProfileRevision: c.profile.Revision(), Target: target, Outcome: r.outcome(chosen), Reached: []Outcome{}}
	for index := range c.units {
		if needed[index] && index != chosen {
			out.Reached = append(out.Reached, r.outcome(index))
		}
	}
	sort.Slice(out.Reached, func(a, b int) bool { return out.Reached[a].Formula < out.Reached[b].Formula })
	return out, nil
}
