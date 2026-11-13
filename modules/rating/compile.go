package rating

import (
	"math/big"
	"sort"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type price struct {
	amount fraction
	per    int64
	text   string
}

type tier struct {
	upTo    fraction
	bounded bool
	price   price
}

type discount struct {
	percent bool
	rate    fraction
	fixed   *big.Int
	text    string
}

type rule struct {
	spec      Rule
	factor    fraction
	price     price
	included  fraction
	tiers     []tier
	amount    *big.Int
	discounts []discount
	minimum   *big.Int
	maximum   *big.Int
}

type Compiled struct {
	profile   decimal.Profile
	limits    Limits
	policy    Policy
	scale     int
	eligible  map[Class]bool
	rules     []*rule
	bySKU     map[string]*rule
	byService map[string]*rule
	byGroup   map[string]*rule
	fallback  *rule
}

type parser struct {
	profile decimal.Profile
	scale   int
}

func (p parser) number(text string) (decimal.Decimal, bool) {
	if strings.HasPrefix(text, "-") {
		return decimal.Decimal{}, false
	}
	value, err := decimal.Parse(p.profile, text)
	return value, err == nil
}

func (p parser) quanta(text string) (*big.Int, bool) {
	value, formed := p.number(text)
	if !formed || value.Scale() > p.scale {
		return nil, false
	}
	return of(value).quanta(p.scale, "toward_zero"), true
}

func (p parser) price(spec Price) (price, bool) {
	amount, formed := p.number(spec.Amount)
	if !formed || (spec.Per != 1 && spec.Per != 1000 && spec.Per != 1000000) {
		return price{}, false
	}
	return price{amount: of(amount), per: int64(spec.Per), text: spec.Amount}, true
}

func (p parser) rule(spec Rule) (*rule, bool) {
	made := &rule{spec: spec, included: whole(0)}
	made.spec.Tiers, made.spec.Discounts = nil, nil
	keyed := spec.Level == SKU || spec.Level == Service || spec.Level == Group
	if !validID(spec.ID) || (!keyed && spec.Level != Default) || keyed != (spec.Key != "") || (keyed && !validID(spec.Key)) {
		return nil, false
	}
	quantity := spec.Type.quantityBased()
	tiered := spec.Type == Graduated || spec.Type == Volume
	if (spec.Rate != "") != (spec.Type == Markup) || (spec.Margin != "") != (spec.Type == TargetMargin) || (spec.Amount != "") != (spec.Type == Flat) ||
		(spec.Price != Price{}) != (spec.Type == UnitPrice) || (len(spec.Tiers) > 0) != tiered || (spec.Included != "" && !quantity) {
		return nil, false
	}
	switch spec.Type {
	case PassThrough:
	case Markup:
		rate, formed := p.number(spec.Rate)
		if !formed {
			return nil, false
		}
		made.factor = whole(1).plus(of(rate))
	case TargetMargin:
		margin, formed := p.number(spec.Margin)
		if !formed || !of(margin).less(whole(1)) {
			return nil, false
		}
		made.factor = whole(1).minus(of(margin))
	case UnitPrice:
		parsed, formed := p.price(spec.Price)
		if !formed {
			return nil, false
		}
		made.price = parsed
	case Graduated, Volume:
		last := whole(0)
		for index, each := range spec.Tiers {
			parsed, formed := p.price(each.Price)
			if !formed {
				return nil, false
			}
			entry := tier{price: parsed}
			if index < len(spec.Tiers)-1 {
				bound, formed := p.number(each.UpTo)
				if !formed || !last.less(of(bound)) {
					return nil, false
				}
				entry.upTo, entry.bounded = of(bound), true
				last = entry.upTo
			} else if each.UpTo != "" {
				return nil, false
			}
			made.tiers = append(made.tiers, entry)
		}
	case Flat:
		amount, formed := p.quanta(spec.Amount)
		if !formed || spec.Level == Default {
			return nil, false
		}
		made.amount = amount
	default:
		return nil, false
	}
	if spec.Included != "" {
		included, formed := p.number(spec.Included)
		if !formed {
			return nil, false
		}
		made.included = of(included)
	}
	for _, each := range spec.Discounts {
		entry := discount{}
		switch {
		case each.Percent != "" && each.Fixed == "":
			rate, formed := p.number(each.Percent)
			if !formed || !of(rate).less(whole(1)) {
				return nil, false
			}
			entry.percent, entry.rate, entry.text = true, of(rate), each.Percent
		case each.Fixed != "" && each.Percent == "":
			fixed, formed := p.quanta(each.Fixed)
			if !formed {
				return nil, false
			}
			entry.fixed, entry.text = fixed, each.Fixed
		default:
			return nil, false
		}
		made.discounts = append(made.discounts, entry)
	}
	var formed bool
	if spec.Minimum != "" {
		if made.minimum, formed = p.quanta(spec.Minimum); !formed {
			return nil, false
		}
	}
	if spec.Maximum != "" {
		if made.maximum, formed = p.quanta(spec.Maximum); !formed {
			return nil, false
		}
	}
	if made.minimum != nil && made.maximum != nil && made.minimum.Cmp(made.maximum) > 0 {
		return nil, false
	}
	return made, true
}

func Compile(profile decimal.Profile, limits Limits, policy Policy) (*Compiled, error) {
	if !limits.valid() {
		return nil, ErrInvalid
	}
	if !validID(policy.ID) || !validID(policy.Version) || !validCurrency(policy.Currency) || !validQuantum(policy.Quantum) || len(policy.Rules) == 0 {
		return nil, ErrInvalid
	}
	if policy.CostBasis != Billed && policy.CostBasis != Effective && policy.CostBasis != List && policy.CostBasis != Contracted {
		return nil, ErrInvalid
	}
	quantum, err := decimal.Parse(profile, policy.Quantum)
	if err != nil || !modes[policy.Mode] {
		return nil, ErrInvalid
	}
	made := &Compiled{profile: profile, limits: limits, policy: policy, scale: quantum.Scale(), eligible: map[Class]bool{}, bySKU: map[string]*rule{}, byService: map[string]*rule{}, byGroup: map[string]*rule{}}
	made.policy.Eligible, made.policy.Rules = nil, nil
	switch policy.Path {
	case Direct:
		if len(policy.Eligible) != 0 {
			return nil, ErrInvalid
		}
		made.eligible[ServiceFee] = true
	case Resale:
		for _, class := range policy.Eligible {
			if !class.known() || made.eligible[class] {
				return nil, ErrInvalid
			}
			made.eligible[class] = true
		}
	default:
		return nil, ErrInvalid
	}
	if len(policy.Rules) > limits.Rules {
		return nil, ErrLimit
	}
	reader := parser{profile: profile, scale: made.scale}
	ids := map[string]bool{}
	for _, spec := range policy.Rules {
		if len(spec.Tiers) > limits.Tiers || len(spec.Discounts) > limits.Discounts {
			return nil, ErrLimit
		}
		entry, formed := reader.rule(spec)
		if !formed || ids[spec.ID] {
			return nil, ErrInvalid
		}
		ids[spec.ID] = true
		index := map[Level]map[string]*rule{SKU: made.bySKU, Service: made.byService, Group: made.byGroup}[spec.Level]
		switch {
		case spec.Level == Default && made.fallback != nil, index != nil && index[spec.Key] != nil:
			return nil, ErrInvalid
		case spec.Level == Default:
			made.fallback = entry
		default:
			index[spec.Key] = entry
		}
		made.rules = append(made.rules, entry)
	}
	sort.Slice(made.rules, func(a, b int) bool { return made.rules[a].spec.ID < made.rules[b].spec.ID })
	return made, nil
}

func (c *Compiled) match(line Line) *rule {
	if found := c.bySKU[line.SKU]; line.SKU != "" && found != nil {
		return found
	}
	if found := c.byService[line.Service]; line.Service != "" && found != nil {
		return found
	}
	if found := c.byGroup[line.Group]; line.Group != "" && found != nil {
		return found
	}
	return c.fallback
}
