package publication

import (
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var start = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)

type world struct {
	t   testing.TB
	r   *Registry
	now *time.Time
	mu  *sync.Mutex
}

func profile(t testing.TB) decimal.Profile {
	t.Helper()
	p, err := decimal.NewProfile(decimal.ProfileSpec{ID: "publication-test", Revision: "1", InputIntegerDigits: 6, InputScale: 4, ResultIntegerDigits: 8, ResultScale: 4, IntermediateDigits: 40})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func setupWith(t testing.TB, l Limits) world {
	t.Helper()
	now := start
	mu := &sync.Mutex{}
	r, err := NewRegistry(Config{Profile: profile(t), Limits: l}, func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	})
	if err != nil {
		t.Fatal(err)
	}
	return world{t: t, r: r, now: &now, mu: mu}
}

func setup(t testing.TB) world { return setupWith(t, Limits{}) }

func (f world) advance(d time.Duration) {
	f.mu.Lock()
	*f.now = f.now.Add(d)
	f.mu.Unlock()
}

func (f world) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func row(key, group, currency, amount string) Row {
	return Row{Key: key, Group: group, Currency: currency, Amount: amount}
}

func rows(n int, group string) []Row {
	out := []Row{}
	for index := 0; index < n; index++ {
		out = append(out, row("k"+pad(index), group, "KRW", strconv.Itoa(index+1)))
	}
	return out
}

func pad(n int) string {
	s := strconv.Itoa(n)
	for len(s) < 6 {
		s = "0" + s
	}
	return s
}

func split(list []Row, size int) [][]Row {
	out := [][]Row{}
	for len(list) > 0 {
		n := min(size, len(list))
		out = append(out, list[:n])
		list = list[n:]
	}
	return out
}

func digestOf(parts [][]Row) string {
	chunks := []chunk{}
	for index, part := range parts {
		chunks = append(chunks, chunk{digest: chunkDigest(index, part)})
	}
	return versionDigest(chunks)
}

func (f world) load(tenant, dataset, id string, parts [][]Row) View {
	f.t.Helper()
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	f.must(second(f.r.Begin(Spec{Tenant: tenant, Dataset: dataset, Version: id, Rows: total, Digest: digestOf(parts)})))
	for index, p := range parts {
		f.must(second(f.r.AddChunk(Chunk{Tenant: tenant, Dataset: dataset, Version: id, Index: index, Rows: p})))
	}
	v, err := f.r.Seal(tenant, dataset, id)
	f.must(err)
	return v
}

func (f world) publish(tenant, dataset, id string, parts [][]Row) View {
	f.t.Helper()
	f.load(tenant, dataset, id, parts)
	v, err := f.r.Publish(tenant, dataset, id)
	f.must(err)
	return v
}

func expect(t testing.TB, name string, err, want error) {
	t.Helper()
	if !errors.Is(err, want) || want == nil && err != nil {
		t.Errorf("%s: got %v, want %v", name, err, want)
	}
}

func reason(t testing.TB, name string, err error, want string) {
	t.Helper()
	var refusal Refusal
	if !errors.As(err, &refusal) || refusal.Reason != want {
		t.Errorf("%s: got %v, want %s", name, err, want)
	}
}

func second[T any](_ T, err error) error { return err }

func (f world) audit() {
	f.t.Helper()
	staged, held, open := map[string]int{}, map[string]int{}, 0
	for _, d := range f.r.datasets {
		live := append([]*version{d.current, d.unpublished}, d.retained...)
		for _, v := range live {
			if v == nil {
				continue
			}
			held[d.tenant] += v.view.Rows
			if v.view.State == StateStaging {
				staged[d.tenant] += v.view.Rows
			}
		}
		if d.unpublished != nil {
			open++
		}
		for _, v := range live {
			if v != nil && v.view.State != StateStaging && len(v.rows) != v.view.Rows {
				f.t.Fatalf("version %s holds %d rows, not %d", v.view.Version, len(v.rows), v.view.Rows)
			}
		}
	}
	sum := func(m map[string]int) int {
		total := 0
		for _, n := range m {
			total += n
		}
		return total
	}
	if f.r.staged != sum(staged) || f.r.held != sum(held) || f.r.open != open {
		f.t.Fatalf("counters staged %d/%d held %d/%d open %d/%d", f.r.staged, sum(staged), f.r.held, sum(held), f.r.open, open)
	}
	for tenant := range f.r.tenantHeld {
		if f.r.tenantHeld[tenant] != held[tenant] || f.r.tenantStaged[tenant] != staged[tenant] || f.r.tenantHeld[tenant] < 0 || f.r.tenantStaged[tenant] < 0 {
			f.t.Fatalf("tenant %s counters", tenant)
		}
	}
}
