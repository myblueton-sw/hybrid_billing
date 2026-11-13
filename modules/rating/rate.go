package rating

import (
	"errors"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/allocation"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type entry struct {
	line     Line
	cost     decimal.Decimal
	quantity decimal.Decimal
	costed   bool
	counted  bool
}

func bound(err error) error {
	if errors.Is(err, decimal.ErrResultBound) || errors.Is(err, decimal.ErrIntermediateBound) || errors.Is(err, allocation.ErrIntermediateBound) {
		return ErrLimit
	}
	return ErrSelfCheck
}

func (c *Compiled) amount(count *big.Int) (decimal.Decimal, error) {
	value, err := decimal.New(c.profile, new(big.Int).Set(count), c.scale)
	if err != nil {
		return decimal.Decimal{}, bound(err)
	}
	return value, nil
}

func (c *Compiled) share(total decimal.Decimal, ids []string, weights []decimal.Decimal) ([]decimal.Decimal, error) {
	out := make([]decimal.Decimal, len(ids))
	zero, err := c.amount(bigZero)
	if err != nil {
		return nil, err
	}
	weighted := false
	for index, weight := range weights {
		out[index] = zero
		weighted = weighted || weight.Sign() != 0
	}
	if !weighted {
		if total.Sign() != 0 {
			return nil, ErrSelfCheck
		}
		return out, nil
	}
	targets := make([]allocation.Target, len(ids))
	for index, id := range ids {
		targets[index] = allocation.Target{ID: id, Weight: weights[index]}
	}
	quantum, err := c.amount(bigOne)
	if err != nil {
		return nil, err
	}
	result, err := allocation.LargestRemainderV1(allocation.Request{Profile: c.profile, Limits: allocation.Limits{MaxTargets: c.limits.Lines, MaxIntermediateDigits: c.limits.Intermediate},
		Encoding: allocation.EncodingASCIIBytewiseV1, Total: total, Quantum: quantum, Mode: decimal.HalfEven, Targets: targets})
	if err != nil {
		return nil, bound(err)
	}
	if len(result.Shares) != len(ids) || result.RoundingComponent.Sign() != 0 {
		return nil, ErrSelfCheck
	}
	for index, part := range result.Shares {
		if part.ID != ids[index] {
			return nil, ErrSelfCheck
		}
		out[index] = part.Amount
	}
	return out, nil
}

type work struct {
	c       *Compiled
	held    *rule
	steps   []Step
	charges []Charge
	sum     decimal.Decimal
	passed  *big.Int
}

func (w *work) step(name, basis, parameter, outcome string) {
	w.steps = append(w.steps, Step{Name: name, Basis: basis, Parameter: parameter, Outcome: outcome})
}

func (w *work) charge(kind, line string, amount decimal.Decimal, always bool) error {
	if !always && amount.Sign() == 0 {
		return nil
	}
	sum, err := decimal.Add(w.c.profile, w.sum, amount)
	if err != nil {
		return bound(err)
	}
	w.sum = sum
	w.charges = append(w.charges, Charge{Rule: w.held.spec.ID, Kind: kind, Line: line, Amount: amount.String()})
	return nil
}

func (w *work) spread(kind string, total decimal.Decimal, members []*entry, weights []decimal.Decimal) error {
	ids := make([]string, len(members))
	for index, member := range members {
		ids[index] = member.line.ID
	}
	parts, err := w.c.share(total, ids, weights)
	if err != nil {
		return err
	}
	for index, part := range parts {
		if err := w.charge(kind, ids[index], part, true); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiled) base(held *rule, members []*entry, w *work) (*big.Int, error) {
	mode := c.policy.Mode
	if held.spec.Type == Flat {
		amount, err := c.amount(held.amount)
		if err != nil {
			return nil, err
		}
		w.step(StepBaseAmount, held.spec.Amount, "", amount.String())
		return new(big.Int).Set(held.amount), w.charge(FlatCharge, "", amount, false)
	}
	if held.spec.Type.costBased() {
		above, below := whole(0), whole(0)
		var rated, passed []*entry
		var ratedWeights, passedWeights []decimal.Decimal
		for _, member := range members {
			if member.cost.Sign() < 0 {
				size, err := decimal.Neg(c.profile, member.cost)
				if err != nil {
					return nil, bound(err)
				}
				below = below.plus(of(member.cost))
				passed, passedWeights = append(passed, member), append(passedWeights, size)
				continue
			}
			above = above.plus(of(member.cost))
			rated, ratedWeights = append(rated, member), append(ratedWeights, member.cost)
		}
		w.step(StepCostsAbove, "", "", above.text())
		w.step(StepCostsBelow, "", "", below.text())
		value, parameter := above, ""
		switch held.spec.Type {
		case Markup:
			value, parameter = above.times(held.factor), held.spec.Rate
		case TargetMargin:
			value, parameter = above.over(held.factor), held.spec.Margin
		}
		w.step(StepBase, above.text(), parameter, value.text())
		count := value.quanta(c.scale, mode)
		amount, err := c.amount(count)
		if err != nil {
			return nil, err
		}
		w.step(StepBaseAmount, value.text(), c.policy.Quantum, amount.String())
		if err := w.spread(Rated, amount, rated, ratedWeights); err != nil {
			return nil, err
		}
		w.passed = below.quanta(c.scale, mode)
		credit, err := c.amount(w.passed)
		if err != nil {
			return nil, err
		}
		w.step(StepPassed, "", "", below.text())
		w.step(StepPassedAmount, below.text(), c.policy.Quantum, credit.String())
		return count, w.spread(PassedAtCost, credit, passed, passedWeights)
	}
	quantity := whole(0)
	weights := make([]decimal.Decimal, len(members))
	for index, member := range members {
		quantity = quantity.plus(of(member.quantity))
		weights[index] = member.quantity
	}
	over := quantity.minus(held.included)
	if over.sign() < 0 {
		over = whole(0)
	}
	w.step(StepQuantity, "", "", quantity.text())
	w.step(StepIncluded, "", "", held.included.text())
	w.step(StepAboveIncluded, quantity.text(), held.included.text(), over.text())
	value := whole(0)
	cost := func(part fraction, at price) fraction {
		return part.times(at.amount).over(whole(at.per))
	}
	switch held.spec.Type {
	case UnitPrice:
		value = cost(over, held.price)
	case Graduated:
		lower := whole(0)
		for _, each := range held.tiers {
			top := over
			if each.bounded && each.upTo.less(over) {
				top = each.upTo
			}
			part := top.minus(lower)
			if part.sign() <= 0 {
				break
			}
			priced := cost(part, each.price)
			w.step(StepTier, part.text(), each.price.text+"/"+strconv.FormatInt(each.price.per, 10), priced.text())
			value = value.plus(priced)
			lower = top
		}
	case Volume:
		for _, each := range held.tiers {
			if over.sign() == 0 {
				break
			}
			if each.bounded && each.upTo.less(over) {
				continue
			}
			value = cost(over, each.price)
			w.step(StepTier, over.text(), each.price.text+"/"+strconv.FormatInt(each.price.per, 10), value.text())
			break
		}
	}
	w.step(StepBase, over.text(), "", value.text())
	count := value.quanta(c.scale, mode)
	amount, err := c.amount(count)
	if err != nil {
		return nil, err
	}
	w.step(StepBaseAmount, value.text(), c.policy.Quantum, amount.String())
	return count, w.spread(Rated, amount, members, weights)
}

func (c *Compiled) settle(held *rule, members []*entry) (*work, error) {
	zero, err := c.amount(bigZero)
	if err != nil {
		return nil, err
	}
	w := &work{c: c, held: held, sum: zero, passed: bigZero}
	left, err := c.base(held, members, w)
	if err != nil {
		return nil, err
	}
	for _, each := range held.discounts {
		before, err := c.amount(left)
		if err != nil {
			return nil, err
		}
		size := each.fixed
		if each.percent {
			size = round(new(big.Int).Mul(left, each.rate.num), each.rate.den, c.policy.Mode)
		}
		if size.Cmp(left) > 0 {
			size = left
		}
		taken, err := c.amount(new(big.Int).Neg(size))
		if err != nil {
			return nil, err
		}
		left = new(big.Int).Sub(left, size)
		w.step(StepDiscount, before.String(), each.text, taken.String())
		if err := w.charge(DiscountCharge, "", taken, false); err != nil {
			return nil, err
		}
	}
	for _, limit := range []struct {
		name, kind, text string
		at               *big.Int
		below            int
	}{{StepMinimum, MinimumTopUp, held.spec.Minimum, held.minimum, -1}, {StepMaximum, MaximumCap, held.spec.Maximum, held.maximum, 1}} {
		if limit.at == nil {
			continue
		}
		before, err := c.amount(left)
		if err != nil {
			return nil, err
		}
		change := new(big.Int)
		if left.Cmp(limit.at) == limit.below {
			change.Sub(limit.at, left)
			left = limit.at
		}
		moved, err := c.amount(change)
		if err != nil {
			return nil, err
		}
		w.step(limit.name, before.String(), limit.text, moved.String())
		if err := w.charge(limit.kind, "", moved, false); err != nil {
			return nil, err
		}
	}
	whole, err := c.amount(new(big.Int).Add(left, w.passed))
	if err != nil {
		return nil, err
	}
	if !whole.Equal(w.sum) {
		return nil, ErrSelfCheck
	}
	w.step(StepTotal, "", "", w.sum.String())
	return w, nil
}

func (c *Compiled) read(line Line) (*entry, bool) {
	made := &entry{line: line}
	if !validID(line.ID) || !line.Class.known() || line.Currency != c.policy.Currency {
		return nil, false
	}
	for _, key := range []string{line.SKU, line.Service, line.Group} {
		if key != "" && !validID(key) {
			return nil, false
		}
	}
	var err error
	if line.Cost != "" {
		if made.cost, err = decimal.Parse(c.profile, line.Cost); err != nil {
			return nil, false
		}
		made.costed = true
	}
	if line.Quantity != "" {
		if made.quantity, err = decimal.Parse(c.profile, line.Quantity); err != nil || strings.HasPrefix(line.Quantity, "-") {
			return nil, false
		}
		made.counted = true
	}
	return made, true
}

func (c *Compiled) Rate(lines []Line) (Result, error) {
	if c == nil || !c.profile.Valid() {
		return Result{}, ErrInvalid
	}
	if len(lines) > c.limits.Lines {
		return Result{}, ErrLimit
	}
	entries := make([]*entry, 0, len(lines))
	seen := map[string]bool{}
	for _, line := range lines {
		made, formed := c.read(line)
		if !formed || seen[line.ID] {
			return Result{}, ErrInvalid
		}
		seen[line.ID] = true
		entries = append(entries, made)
	}
	sort.Slice(entries, func(a, b int) bool { return entries[a].line.ID < entries[b].line.ID })
	out := Result{PolicyID: c.policy.ID, Version: c.policy.Version, Currency: c.policy.Currency, CostBasis: c.policy.CostBasis, Path: c.policy.Path, Quantum: c.policy.Quantum, Mode: c.policy.Mode,
		ProfileID: c.profile.ID(), ProfileRevision: c.profile.Revision(), Algorithm: Algorithm, Lines: len(lines), NoRuleHeld: true}
	reasons := map[string]int{}
	exclude := func(line, reason string) {
		out.Excluded = append(out.Excluded, Excluded{Line: line, Reason: reason})
		reasons[reason]++
	}
	members := map[*rule][]*entry{}
	for _, made := range entries {
		switch held := c.match(made.line); {
		case !c.eligible[made.line.Class]:
			exclude(made.line.ID, ClassNotEligible)
		case held == nil:
			exclude(made.line.ID, NoRule)
		default:
			members[held] = append(members[held], made)
		}
	}
	total, err := c.amount(bigZero)
	if err != nil {
		return Result{}, err
	}
	for _, held := range c.rules {
		group := members[held]
		if len(group) == 0 {
			continue
		}
		unknown := 0
		for _, made := range group {
			if (held.spec.Type.costBased() && !made.costed) || (held.spec.Type.quantityBased() && !made.counted) {
				unknown++
			}
		}
		if unknown > 0 {
			out.NoRuleHeld = false
			for _, made := range group {
				switch {
				case held.spec.Type.costBased() && !made.costed:
					exclude(made.line.ID, CostNotKnown)
				case held.spec.Type.quantityBased() && !made.counted:
					exclude(made.line.ID, QuantityNotKnown)
				default:
					exclude(made.line.ID, RuleHeld)
				}
			}
			out.Traces = append(out.Traces, Trace{Rule: held.spec.ID, Steps: []Step{{Name: StepHeld, Outcome: strconv.Itoa(unknown)}}, Total: "0"})
			continue
		}
		done, err := c.settle(held, group)
		if err != nil {
			return Result{}, err
		}
		if held.spec.Type == Flat {
			for _, made := range group {
				out.Covered = append(out.Covered, Covered{Line: made.line.ID, Rule: held.spec.ID})
			}
		}
		out.Charges = append(out.Charges, done.charges...)
		out.Traces = append(out.Traces, Trace{Rule: held.spec.ID, Steps: done.steps, Total: done.sum.String()})
		if total, err = decimal.Add(c.profile, total, done.sum); err != nil {
			return Result{}, bound(err)
		}
	}
	sort.Slice(out.Excluded, func(a, b int) bool { return out.Excluded[a].Line < out.Excluded[b].Line })
	sort.Slice(out.Covered, func(a, b int) bool { return out.Covered[a].Line < out.Covered[b].Line })
	for _, reason := range []string{ClassNotEligible, NoRule, CostNotKnown, QuantityNotKnown, RuleHeld} {
		if reasons[reason] > 0 {
			out.Exclusions = append(out.Exclusions, Count{Reason: reason, Lines: reasons[reason]})
		}
	}
	out.Total = total.String()
	return out, nil
}
