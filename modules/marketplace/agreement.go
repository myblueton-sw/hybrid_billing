package marketplace

import (
	"sort"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type kindTotals map[string]exact.Fraction

func (k kindTotals) add(kind string, value exact.Fraction) {
	if current, ok := k[kind]; ok {
		k[kind] = current.Plus(value)
		return
	}
	k[kind] = value
}

func (k kindTotals) sums() (KindSums, bool) {
	out := KindSums{}
	for kind, target := range map[string]*string{KindSoftware: &out.Software, KindInfrastructure: &out.Infrastructure, KindOther: &out.Other} {
		value, ok := k[kind]
		if !ok {
			value = exact.Whole(0)
		}
		text, formatted := format(value)
		if !formatted {
			return KindSums{}, false
		}
		*target = text
	}
	return out, true
}

func (r *Registry) AgreementCharges(workspace, provider, agreement, month string) (AgreementView, error) {
	if !validID(workspace) || !validID(provider) || !validID(agreement) || !validMonth(month) {
		return AgreementView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.space(workspace, false)
	if s == nil {
		return AgreementView{}, ErrNotFound
	}
	members := []*stored{}
	for _, st := range s.transactions {
		if st.t.Provider == provider && st.t.Agreement == agreement && month >= st.t.Term.Start && month <= st.t.Term.End {
			members = append(members, st)
		}
	}
	if len(members) == 0 {
		return AgreementView{}, ErrNotFound
	}
	sort.Slice(members, func(a, b int) bool { return members[a].t.ID < members[b].t.ID })
	out := AgreementView{Provider: provider, Agreement: agreement, Month: month, Entries: []AgreementEntry{}, Groups: []GroupSums{}, Complete: true}
	groups := map[[2]string]kindTotals{}
	for _, st := range members {
		entry := AgreementEntry{Transaction: st.t.ID, Role: st.t.Role, Currency: st.t.Term.Currency}
		s := st.statements[month]
		if s == nil {
			out.Complete = false
			out.Entries = append(out.Entries, entry)
			continue
		}
		current := currentOf(s)
		entry.State, entry.Revision, entry.Digest = current.State, current.Revision, current.Digest
		if current.State != CurrentSealed {
			out.Complete = false
			out.Entries = append(out.Entries, entry)
			continue
		}
		key := [2]string{st.t.Role, st.t.Term.Currency}
		if groups[key] == nil {
			groups[key] = kindTotals{}
		}
		all := groups[key]
		own := kindTotals{}
		for _, line := range s.highest().facts.Lines {
			value := r.sum(line.Charged)
			own.add(line.Kind, value)
			all.add(line.Kind, value)
		}
		sums, ok := own.sums()
		if !ok {
			return AgreementView{}, ErrSelfCheck
		}
		entry.Sums = sums
		out.Entries = append(out.Entries, entry)
	}
	keys := make([][2]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(a, b int) bool {
		return keys[a][0] < keys[b][0] || keys[a][0] == keys[b][0] && keys[a][1] < keys[b][1]
	})
	for _, key := range keys {
		sums, ok := groups[key].sums()
		if !ok {
			return AgreementView{}, ErrSelfCheck
		}
		out.Groups = append(out.Groups, GroupSums{Role: key[0], Currency: key[1], Sums: sums})
	}
	return out, nil
}
