package commitment

import (
	"sort"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

func salePolicyDigest(s SalePolicy) string {
	p := newPrinter(salePolicyLbl)
	fee := "false"
	if s.FeeOnUnused {
		fee = "true"
	}
	p.fields(s.Customer, s.ID, s.Version, s.Method, s.Fee, fee, s.Share, s.Mode)
	return p.hex()
}

func saleDigest(s Sale) string {
	p := newPrinter(SaleLbl)
	p.fields(s.ProfileID, s.ProfileRevision, s.AllocationDigest)
	p.number(len(s.Charges))
	for _, c := range s.Charges {
		p.fields(c.Customer, c.PolicyID, c.PolicyVersion, c.PolicyDigest, c.Hold, c.Amount, c.Exact)
	}
	priced := "false"
	if s.AllCustomersPriced {
		priced = "true"
	}
	below := "false"
	if s.BelowCost {
		below = "true"
	}
	p.fields(s.Total, s.CostCompared, s.Recovered, priced, s.MSPResult, below)
	return p.hex()
}

func rate(text string, upTo bool) (exact.Fraction, string, bool) {
	if len(text) > maxNumberBytes || text == "" {
		return exact.Fraction{}, "", false
	}
	value, ok := parseRate(text)
	if !ok || value.Sign() < 0 || (upTo && exact.Whole(1).Less(value)) || (!upTo && !value.Less(exact.Whole(1))) {
		return exact.Fraction{}, "", false
	}
	shown := text
	if strings.Contains(shown, ".") {
		shown = strings.TrimRight(strings.TrimRight(shown, "0"), ".")
	}
	return value, shown, true
}

func parseRate(text string) (exact.Fraction, bool) {
	units, scale, seen := int64(0), 0, false
	dot := false
	for index := 0; index < len(text); index++ {
		c := text[index]
		switch {
		case c == '.' && !dot && index > 0 && index < len(text)-1:
			dot = true
		case c >= '0' && c <= '9':
			if units > 1000000000000000 {
				return exact.Fraction{}, false
			}
			units = units*10 + int64(c-'0')
			seen = true
			if dot {
				scale++
			}
		default:
			return exact.Fraction{}, false
		}
	}
	if !seen || scale > 8 || (len(text) > 1 && text[0] == '0' && text[1] != '.') {
		return exact.Fraction{}, false
	}
	divisor := int64(1)
	for step := 0; step < scale; step++ {
		divisor *= 10
	}
	value, _ := exact.Whole(units).Over(exact.Whole(divisor))
	return value, true
}

func Sell(a Allocation, policies []SalePolicy) (Sale, error) {
	if !a.made || a.view.Hold != "" {
		return Sale{}, ErrInvalid
	}
	if len(policies) > a.limits.Customers {
		return Sale{}, ErrLimit
	}
	byCustomer := map[string]SalePolicy{}
	for _, s := range policies {
		if !validID(s.Customer) || !validID(s.ID) || !validID(s.Version) || !exact.ValidMode(s.Mode) || byCustomer[s.Customer].ID != "" {
			return Sale{}, ErrInvalid
		}
		switch s.Method {
		case MethodCostPlus:
			if _, shown, ok := rate(s.Fee, false); !ok || s.Share != "" {
				return Sale{}, ErrInvalid
			} else {
				s.Fee = shown
			}
		case MethodSavingsShare:
			if _, shown, ok := rate(s.Share, true); !ok || s.Fee != "" || s.FeeOnUnused {
				return Sale{}, ErrInvalid
			} else {
				s.Share = shown
			}
		default:
			return Sale{}, ErrInvalid
		}
		if a.customers[s.Customer] == nil {
			return Sale{}, ErrInvalid
		}
		byCustomer[s.Customer] = s
	}
	names := make([]string, 0, len(a.customers))
	for name := range a.customers {
		names = append(names, name)
	}
	sort.Strings(names)
	out := Sale{Label: SaleLbl, ProfileID: a.view.ProfileID, ProfileRevision: a.view.ProfileRevision, AllocationDigest: a.view.Digest, Charges: []Charge{}, AllCustomersPriced: true}
	total := exact.Whole(0)
	for _, name := range names {
		part := a.customers[name]
		s, ok := byCustomer[name]
		charge := Charge{Customer: name}
		if !ok {
			charge.Hold, out.AllCustomersPriced = HoldNoSalePolicy, false
			out.Charges = append(out.Charges, charge)
			continue
		}
		charge.PolicyID, charge.PolicyVersion, charge.PolicyDigest = s.ID, s.Version, salePolicyDigest(s)
		used, unused := part.usedRecurring.Plus(part.usedAmortized), part.unusedRecurring.Plus(part.unusedAmortized)
		if a.recovered {
			used, unused = part.usedRecurring, part.unusedRecurring
		}
		var value exact.Fraction
		switch s.Method {
		case MethodCostPlus:
			fee, _, _ := rate(s.Fee, false)
			factor := exact.Whole(1).Plus(fee)
			value = used.Times(factor)
			if s.FeeOnUnused {
				value = value.Plus(unused.Times(factor))
			} else {
				value = value.Plus(unused)
			}
		default:
			if a.recovered {
				charge.Hold = HoldRecoveredWithSavingsShare
			} else if part.negative > 0 {
				charge.Hold = HoldNegativeSavings
			}
			if charge.Hold != "" {
				out.AllCustomersPriced = false
				out.Charges = append(out.Charges, charge)
				continue
			}
			share, _, _ := rate(s.Share, true)
			savings := part.list.Minus(part.usedRecurring).Minus(part.usedAmortized)
			value = part.list.Minus(savings.Times(share)).Plus(unused)
		}
		if value.Digits() > a.limits.Digits {
			return Sale{}, ErrLimit
		}
		units, _ := value.Quanta(a.scale, s.Mode)
		rounded, _ := exact.Units(units, a.scale)
		charge.Amount, _ = format(rounded)
		charge.Exact, _ = format(value)
		total = total.Plus(rounded)
		out.Charges = append(out.Charges, charge)
	}
	compared := a.cost
	recovered := exact.Whole(0)
	if a.recovered {
		compared, recovered = a.cost.Minus(a.amortized), a.amortized
	}
	out.Total, _ = format(total)
	out.CostCompared, _ = format(compared)
	out.Recovered, _ = format(recovered)
	if out.AllCustomersPriced {
		result := total.Minus(compared)
		out.MSPResult, _ = format(result)
		out.BelowCost = result.Sign() < 0
	}
	out.Digest = saleDigest(out)
	return out, nil
}
