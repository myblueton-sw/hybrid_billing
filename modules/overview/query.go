package overview

import (
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *space) current(source string, n int) *entry {
	var best *entry
	for _, e := range s.bySource[source] {
		if e.seq <= n && (best == nil || e.fact.Source.Revision > best.fact.Source.Revision) {
			best = e
		}
	}
	return best
}

func (s *space) becomesCurrent(f Fact) bool {
	cur := s.current(f.Source.ID, len(s.order))
	return cur == nil || f.Source.Revision > cur.fact.Source.Revision
}

func (s *space) included(sub *subject, n int) []*entry {
	var out []*entry
	names := sub.names
	if len(names) != len(sub.sources) {
		names = sorted(keys(sub.sources))
		sub.names = names
	}
	for _, source := range names {
		if cur := s.current(source, n); cur != nil && cur.fact.Subject == sub.key {
			out = append(out, cur)
		}
	}
	return out
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (r *Registry) state(s *space, sub *subject, n int, evaluated time.Time, cost bool) (Row, bool) {
	facts := s.included(sub, n)
	if len(facts) == 0 {
		return Row{Subject: sub.key, Status: StatusAbsent, Amounts: map[string]string{}}, false
	}
	lead := facts[0]
	blocked := false
	row := Row{Subject: sub.key, Amounts: map[string]string{}}
	for _, e := range facts {
		if e.fact.Occurred.After(lead.fact.Occurred) || e.fact.Occurred.Equal(lead.fact.Occurred) && e.seq > lead.seq {
			lead = e
		}
		if e.fact.Stage == StageBlocked {
			blocked = true
		}
		if e.received.After(row.Received) {
			row.Received = e.received
		}
		if e.fact.Occurred.After(row.Occurred) {
			row.Occurred = e.fact.Occurred
		}
		for meaning, amount := range e.fact.Amounts {
			if meaning != MeaningCost || cost {
				row.Amounts[meaning] = amount
			}
		}
	}
	row.Stage, row.Due, row.Owner, row.Next = lead.fact.Stage, lead.fact.Due, lead.fact.Owner, lead.fact.Next
	if blocked {
		row.Stage = StageBlocked
	}
	row.Status = StatusCurrent
	if row.Received.Before(evaluated.Add(-r.limits.Stale)) {
		row.Status = StatusStale
	}
	return row, true
}

func visible(v Visibility, s Subject) bool {
	if v.All {
		return true
	}
	for _, sel := range v.Selectors {
		if sel == (Selector{}) {
			continue
		}
		if (sel.Customer == "" || sel.Customer == s.Customer) && (sel.Entity == "" || sel.Entity == s.Entity) && (sel.Contract == "" || sel.Contract == s.Contract) &&
			(sel.From == "" || s.Period >= sel.From) && (sel.To == "" || s.Period <= sel.To) {
			return true
		}
	}
	return false
}

func visibilityDigest(v Visibility) string {
	parts := []string{strconv.FormatBool(v.All), strconv.FormatBool(v.Cost)}
	var selectors []string
	for _, s := range v.Selectors {
		selectors = append(selectors, digest(s.Customer, s.Entity, s.Contract, s.From, s.To))
	}
	return digest(append(parts, sorted(selectors)...)...)
}

func (r *Registry) visibility(workspace string, actor Actor) (Visibility, error) {
	v, err := r.config.Scope.Visible(workspace, actor.ID)
	if err != nil {
		return Visibility{}, refuse(ReasonScopeUnavailable)
	}
	return v, nil
}

var statuses = map[string]bool{"": true, StatusCurrent: true, StatusStale: true}

var sortKeys = map[string]bool{"": true, SortDue: true, SortCustomer: true, SortPeriod: true, SortReceived: true}

func filterForm(f Filter) bool {
	for _, list := range [][]string{f.Customers, f.Contracts, f.Entities, f.Owners} {
		for _, id := range list {
			if !validID(id) {
				return false
			}
		}
	}
	for _, p := range f.Periods {
		if !validPeriod(p) {
			return false
		}
	}
	for _, c := range f.Currencies {
		if !validCurrency(c) {
			return false
		}
	}
	for _, st := range f.Stages {
		if !stages[st] {
			return false
		}
	}
	return (f.DueBy == "" || validDate(f.DueBy)) && statuses[f.Status]
}

func empty(f Filter) bool {
	return len(f.Customers)+len(f.Contracts)+len(f.Entities)+len(f.Periods)+len(f.Currencies)+len(f.Stages)+len(f.Owners) == 0 && f.DueBy == "" && f.Status == ""
}

func filterDigest(f Filter, o Sort) string {
	var parts []string
	for _, list := range [][]string{f.Customers, f.Contracts, f.Entities, f.Periods, f.Currencies, f.Stages, f.Owners} {
		parts = append(parts, strings.Join(sorted(list), ","))
	}
	return digest(append(parts, f.DueBy, f.Status, o.Key, strconv.FormatBool(o.Descending))...)
}

func member(list []string, value string) bool {
	if len(list) == 0 {
		return true
	}
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func matches(f Filter, row Row) bool {
	s := row.Subject
	return member(f.Customers, s.Customer) && member(f.Contracts, s.Contract) && member(f.Entities, s.Entity) && member(f.Periods, s.Period) &&
		member(f.Currencies, s.Currency) && member(f.Stages, row.Stage) && member(f.Owners, row.Owner) &&
		(f.DueBy == "" || row.Due != "" && row.Due <= f.DueBy) && (f.Status == "" || f.Status == row.Status)
}

func subjectLess(a, b Subject) bool {
	for _, pair := range [][2]string{{a.Customer, b.Customer}, {a.Contract, b.Contract}, {a.Entity, b.Entity}, {a.Period, b.Period}, {a.Currency, b.Currency}} {
		if pair[0] != pair[1] {
			return pair[0] < pair[1]
		}
	}
	return false
}

func less(a, b *Row, o Sort) bool {
	c := 0
	switch o.Key {
	case SortDue:
		if (a.Due == "") != (b.Due == "") {
			return b.Due == ""
		}
		c = strings.Compare(a.Due, b.Due)
	case SortCustomer:
		c = strings.Compare(a.Subject.Customer, b.Subject.Customer)
	case SortPeriod:
		c = strings.Compare(a.Subject.Period, b.Subject.Period)
	case SortReceived:
		c = a.Received.Compare(b.Received)
	}
	if c != 0 {
		return c < 0 != o.Descending
	}
	return subjectLess(a.Subject, b.Subject)
}

func order(rows []*Row, o Sort) {
	sort.Slice(rows, func(i, j int) bool { return less(rows[i], rows[j], o) })
}

var meaningOrder = []string{MeaningCost, MeaningSales, MeaningAllocation, MeaningOutstanding}

func micro(text string) *big.Int {
	negative := strings.HasPrefix(text, "-")
	whole, fraction, _ := strings.Cut(strings.TrimPrefix(text, "-"), ".")
	value, _ := new(big.Int).SetString(whole+fraction+strings.Repeat("0", maxScale-len(fraction)), 10)
	if negative {
		value.Neg(value)
	}
	return value
}

func decimal(value *big.Int) string {
	sign := ""
	if value.Sign() < 0 {
		sign = "-"
	}
	digits := new(big.Int).Abs(value).String()
	for len(digits) <= maxScale {
		digits = "0" + digits
	}
	whole, fraction := digits[:len(digits)-maxScale], strings.TrimRight(digits[len(digits)-maxScale:], "0")
	if fraction == "" {
		return sign + whole
	}
	return sign + whole + "." + fraction
}

type bucket struct {
	current, stale        *big.Int
	rows, staleRows, none int
}

func totals(rows []*Row, cost bool) []Total {
	buckets := map[[2]string]*bucket{}
	seen := map[string]bool{}
	var currencies []string
	for _, row := range rows {
		c := row.Subject.Currency
		if !seen[c] {
			seen[c] = true
			currencies = append(currencies, c)
		}
		for _, meaning := range meaningOrder {
			if meaning == MeaningCost && !cost {
				continue
			}
			b := buckets[[2]string{c, meaning}]
			if b == nil {
				b = &bucket{}
				buckets[[2]string{c, meaning}] = b
			}
			b.rows++
			stale := row.Status == StatusStale
			if stale {
				b.staleRows++
			}
			amount, ok := row.Amounts[meaning]
			if !ok {
				b.none++
				continue
			}
			target := &b.current
			if stale {
				target = &b.stale
			}
			if *target == nil {
				*target = new(big.Int)
			}
			(*target).Add(*target, micro(amount))
		}
	}
	sort.Strings(currencies)
	var out []Total
	for _, c := range currencies {
		for _, meaning := range meaningOrder {
			b := buckets[[2]string{c, meaning}]
			if b == nil {
				continue
			}
			t := Total{Currency: c, Meaning: meaning, Rows: b.rows, StaleRows: b.staleRows, Unknown: b.none}
			if b.current != nil {
				t.Current = decimal(b.current)
			}
			if b.stale != nil {
				t.Stale = decimal(b.stale)
			}
			out = append(out, t)
		}
	}
	return out
}

func sortSaved(list []SavedView) {
	sort.Slice(list, func(i, j int) bool {
		if list[i].Owner != list[j].Owner {
			return list[i].Owner < list[j].Owner
		}
		return list[i].Saved.ID < list[j].Saved.ID
	})
}
