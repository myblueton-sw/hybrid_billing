package marketplace

import (
	"math/big"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type Charge struct {
	made bool
	view ChargeView
}

func (c Charge) View() ChargeView {
	view := c.view
	view.Lines = append([]ChargeLine{}, c.view.Lines...)
	view.Recognitions = append([]Recognition{}, c.view.Recognitions...)
	return view
}

func format(f exact.Fraction) (string, bool) {
	text := f.Text()
	num, den := text, "1"
	if index := strings.IndexByte(text, '/'); index >= 0 {
		num, den = text[:index], text[index+1:]
	}
	value, _ := new(big.Int).SetString(num, 10)
	divisor, _ := new(big.Int).SetString(den, 10)
	rest, twos, fives := new(big.Int).Set(divisor), 0, 0
	two, five, zero := big.NewInt(2), big.NewInt(5), new(big.Int)
	for new(big.Int).Rem(rest, two).Cmp(zero) == 0 {
		rest.Quo(rest, two)
		twos++
	}
	for new(big.Int).Rem(rest, five).Cmp(zero) == 0 {
		rest.Quo(rest, five)
		fives++
	}
	if rest.Cmp(big.NewInt(1)) != 0 {
		return "", false
	}
	scale := twos
	if fives > scale {
		scale = fives
	}
	units := new(big.Int).Mul(value, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
	units.Quo(units, divisor)
	negative := units.Sign() < 0
	digits := new(big.Int).Abs(units).String()
	for len(digits) <= scale {
		digits = "0" + digits
	}
	out := digits
	if scale > 0 {
		out = digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
	}
	if negative {
		out = "-" + out
	}
	return out, true
}

func of(profile decimal.Profile, text string) exact.Fraction {
	value, _ := decimal.Parse(profile, text)
	made, _ := exact.Of(value)
	return made
}

func sourceDiscounted(t Transaction, line Line) bool {
	private := t.Type == TypePrivate || t.Type == TypeChannelPrivate
	return line.Discount.State == Applied || private && line.Kind == KindSoftware
}

func discountable(t Transaction, policy Policy, line Line) bool {
	if !policy.scope[line.Kind] {
		return false
	}
	if sourceDiscounted(t, line) {
		return policy.terms.SourceDiscounted == Included
	}
	return line.Discount.State == NotApplied
}

func unknownInScope(t Transaction, policy Policy, facts Facts) bool {
	for _, line := range facts.Lines {
		if policy.rate.Sign() > 0 && policy.scope[line.Kind] && !sourceDiscounted(t, line) && line.Discount.State == Unknown {
			return true
		}
	}
	return false
}

func ComputeCharge(s Statement, policy Policy) (Charge, error) {
	if !s.made || !policy.made || policy.workspace != s.workspace || policy.terms.Transaction != s.transaction.ID ||
		s.month < policy.terms.First || s.month > policy.terms.Last {
		return Charge{}, ErrInvalid
	}
	t := s.transaction
	view := ChargeView{Label: ChargeLabel, ProfileID: s.profile.ID(), ProfileRev: s.profile.Revision(), Transaction: t.ID, Month: s.month, Revision: s.revision,
		StatementHash: s.digest, PolicyID: policy.terms.ID, PolicyVersion: policy.terms.Version, PolicyDigest: policy.digest, PolicyFirst: policy.terms.First, PolicyLast: policy.terms.Last, Customer: policy.terms.Customer,
		Lines: []ChargeLine{}, Recognitions: []Recognition{}}
	switch {
	case t.Role == RoleSeller:
		view.Hold = HoldSellerTransaction
	case t.Collector != CollectorMSP:
		view.Hold = HoldCollectedElsewhere
	case unknownInScope(t, policy, s.facts):
		view.Hold = HoldSourceUnknown
	}
	if view.Hold != "" {
		view.Digest = chargeDigest(view)
		return Charge{made: true, view: view}, nil
	}
	view.Recognitions = append(view.Recognitions, s.facts.Recognitions...)
	total, cost := exact.Whole(0), exact.Whole(0)
	for _, line := range s.facts.Lines {
		charged := of(s.profile, line.Charged)
		discount := exact.Whole(0)
		if discountable(t, policy, line) {
			units, ok := charged.Times(policy.rate).Quanta(policy.scale, decimal.TowardZero.String())
			if !ok {
				return Charge{}, ErrSelfCheck
			}
			discount, _ = exact.Units(units, policy.scale)
		}
		amount := charged.Minus(discount)
		if discount.Sign() < 0 || amount.Sign() < 0 {
			return Charge{}, ErrSelfCheck
		}
		total, cost = total.Plus(amount), cost.Plus(charged)
		discountText, okD := format(discount)
		amountText, okA := format(amount)
		if !okD || !okA {
			return Charge{}, ErrSelfCheck
		}
		view.Lines = append(view.Lines, ChargeLine{Source: line.Source, Kind: line.Kind, Charged: line.Charged, Discount: discountText, Amount: amountText})
	}
	if total.Digits() > s.limits.Digits || cost.Digits() > s.limits.Digits {
		return Charge{}, ErrLimit
	}
	margin := total.Minus(cost)
	totalText, okT := format(total)
	costText, okC := format(cost)
	marginText, okM := format(margin)
	if !okT || !okC || !okM || cost.Less(total) {
		return Charge{}, ErrSelfCheck
	}
	view.Total, view.ChargedTotal, view.Margin = totalText, costText, marginText
	view.Digest = chargeDigest(view)
	return Charge{made: true, view: view}, nil
}

func ViewForCustomer(c Charge) (CustomerView, error) {
	if !c.made {
		return CustomerView{}, ErrInvalid
	}
	if c.view.Hold != "" {
		return CustomerView{}, ErrRejected
	}
	out := CustomerView{Transaction: c.view.Transaction, Month: c.view.Month, Customer: c.view.Customer, Lines: []CustomerLine{}, Total: c.view.Total, Digest: c.view.Digest}
	for _, line := range c.view.Lines {
		out.Lines = append(out.Lines, CustomerLine{Source: line.Source, Kind: line.Kind, Amount: line.Amount})
	}
	return out, nil
}
