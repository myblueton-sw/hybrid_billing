package publication

import (
	"container/heap"
	"math/big"
	"sort"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var ten = big.NewInt(10)

func validCurrency(text string) bool {
	if len(text) != 3 {
		return false
	}
	for index := 0; index < 3; index++ {
		if text[index] < 'A' || text[index] > 'Z' {
			return false
		}
	}
	return true
}

func (p *partial) add(coefficient *big.Int, scale int) {
	if scale > p.scale {
		p.coefficient.Mul(p.coefficient, new(big.Int).Exp(ten, big.NewInt(int64(scale-p.scale)), nil))
		p.scale = scale
	}
	p.coefficient.Add(p.coefficient, new(big.Int).Mul(coefficient, new(big.Int).Exp(ten, big.NewInt(int64(p.scale-scale)), nil)))
}

func chunkDigest(index int, rows []Row) string {
	var e encoder
	e.text("hybrid-billing/publication-chunk/v1")
	e.count(index)
	e.count(len(rows))
	for _, row := range rows {
		for _, value := range []string{row.Key, row.Group, row.Currency, row.Amount} {
			e.text(value)
		}
	}
	return digest(e.out.Bytes())
}

func versionDigest(chunks []chunk) string {
	var e encoder
	e.text("hybrid-billing/publication-version/v1")
	e.count(len(chunks))
	for _, c := range chunks {
		e.text(c.digest)
	}
	return digest(e.out.Bytes())
}

func (r *Registry) parse(rows []Row) (map[TotalKey]*partial, bool) {
	partials := map[TotalKey]*partial{}
	for index, row := range rows {
		if !validID(row.Key) || !validID(row.Group) || !validCurrency(row.Currency) || index > 0 && rows[index-1].Key >= row.Key {
			return nil, false
		}
		amount, err := decimal.Parse(r.profile, row.Amount)
		if err != nil || amount.String() != row.Amount {
			return nil, false
		}
		key := TotalKey{Group: row.Group, Currency: row.Currency}
		p := partials[key]
		if p == nil {
			p = &partial{coefficient: new(big.Int)}
			partials[key] = p
		}
		p.add(amount.Coefficient(), amount.Scale())
	}
	return partials, true
}

func (r *Registry) AddChunk(c Chunk) (View, error) {
	now := r.enter()
	defer r.leave()
	if c.Index < 0 || len(c.Rows) == 0 || len(c.Rows) > r.limits.ChunkRows {
		return View{}, ErrInvalid
	}
	partials, ok := r.parse(c.Rows)
	if !ok {
		return View{}, ErrInvalid
	}
	_, v, err := r.version(c.Tenant, c.Dataset, c.Version, now)
	if err != nil {
		return View{}, err
	}
	if v.view.State != StateStaging {
		return View{}, refuse(ReasonSealed)
	}
	sum := chunkDigest(c.Index, c.Rows)
	if c.Index < len(v.chunks) {
		if v.chunks[c.Index].digest != sum {
			return View{}, ErrConflict
		}
		out := v.view
		out.Repeat = true
		return out, nil
	}
	switch {
	case c.Index > len(v.chunks):
		return View{}, refuse(ReasonGap)
	case v.view.ReceivedRows+len(c.Rows) > v.view.Rows:
		return View{}, ErrLimit
	}
	v.chunks = append(v.chunks, chunk{digest: sum, rows: append([]Row(nil), c.Rows...), partials: partials})
	v.view.NextIndex = len(v.chunks)
	v.view.ReceivedRows += len(c.Rows)
	return v.view, nil
}

type cursor struct {
	chunk, row int
}

type merger struct {
	chunks []chunk
	heads  []cursor
}

func (m *merger) Len() int { return len(m.heads) }

func (m *merger) Less(a, b int) bool {
	x, y := m.heads[a], m.heads[b]
	return m.chunks[x.chunk].rows[x.row].Key < m.chunks[y.chunk].rows[y.row].Key
}

func (m *merger) Swap(a, b int) { m.heads[a], m.heads[b] = m.heads[b], m.heads[a] }

func (m *merger) Push(x any) { m.heads = append(m.heads, x.(cursor)) }

func (m *merger) Pop() any {
	last := m.heads[len(m.heads)-1]
	m.heads = m.heads[:len(m.heads)-1]
	return last
}

func merge(chunks []chunk, total int) ([]Row, bool) {
	m := &merger{chunks: chunks}
	for index := range chunks {
		heap.Push(m, cursor{chunk: index})
	}
	out := make([]Row, 0, total)
	for m.Len() > 0 {
		head := m.heads[0]
		row := chunks[head.chunk].rows[head.row]
		if len(out) > 0 && out[len(out)-1].Key == row.Key {
			return nil, false
		}
		out = append(out, row)
		if head.row+1 < len(chunks[head.chunk].rows) {
			m.heads[0].row++
			heap.Fix(m, 0)
		} else {
			heap.Pop(m)
		}
	}
	return out, true
}

func (r *Registry) Seal(tenant, name, id string) (View, error) {
	now := r.enter()
	defer r.leave()
	d, v, err := r.version(tenant, name, id, now)
	if err != nil {
		return View{}, err
	}
	switch {
	case v.view.State != StateStaging && v == d.unpublished:
		return v.view, nil
	case v.view.State != StateStaging:
		return View{}, refuse(ReasonPublished)
	case v.view.ReceivedRows != v.view.Rows:
		return View{}, refuse(ReasonCountMismatch)
	case versionDigest(v.chunks) != v.view.Digest:
		return View{}, refuse(ReasonDigestMismatch)
	}
	rows, ok := merge(v.chunks, v.view.Rows)
	if !ok {
		return View{}, refuse(ReasonDuplicateKey)
	}
	sums := map[TotalKey]*partial{}
	for _, c := range v.chunks {
		for key, p := range c.partials {
			s := sums[key]
			if s == nil {
				if len(sums) >= r.limits.Groups {
					return View{}, ErrLimit
				}
				s = &partial{coefficient: new(big.Int)}
				sums[key] = s
			}
			s.add(p.coefficient, p.scale)
		}
	}
	totals := make([]Total, 0, len(sums))
	for key, s := range sums {
		amount, err := decimal.New(r.profile, s.coefficient, s.scale)
		if err != nil {
			return View{}, refuse(ReasonOverflow)
		}
		totals = append(totals, Total{Key: key, Amount: amount.String()})
	}
	sort.Slice(totals, func(a, b int) bool { return keyLess(totals[a].Key, totals[b].Key) })
	r.unstage(tenant, v.view.Rows)
	v.rows, v.totals, v.chunks = rows, totals, nil
	v.view.State, v.view.Sealed = StateSealed, now
	return v.view, nil
}

func keyLess(a, b TotalKey) bool {
	return a.Group < b.Group || a.Group == b.Group && a.Currency < b.Currency
}
