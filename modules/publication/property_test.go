package publication

import (
	"math/big"
	"math/rand"
	"sort"
	"strconv"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestPublicationProperty(t *testing.T) {
	runs := 150
	if testing.Short() {
		runs = 30
	}
	reached := map[string]int{}
	for seed := 0; seed < runs; seed++ {
		runPublicationProperty(t, int64(seed), reached)
	}
	for _, name := range []string{"sealed", "overflow", "resend", "conflict", "gap"} {
		if reached[name] == 0 {
			t.Errorf("%s never reached", name)
		}
	}
	t.Logf("reached %v", reached)
}

func ratString(r *big.Rat) string {
	text := r.FloatString(4)
	for len(text) > 0 && text[len(text)-1] == '0' {
		text = text[:len(text)-1]
	}
	if len(text) > 0 && text[len(text)-1] == '.' {
		text = text[:len(text)-1]
	}
	if text == "-0" {
		text = "0"
	}
	return text
}

func runPublicationProperty(t *testing.T, seed int64, reached map[string]int) {
	rng := rand.New(rand.NewSource(seed))
	f := setup(t)
	n := 1 + rng.Intn(60)
	groups := []string{"a", "a-", "b", "a.b"}
	currencies := []string{"KRW", "USD"}
	big8 := new(big.Rat).SetInt64(100000000)
	all := []Row{}
	heavy := seed%3 == 0
	if heavy {
		n = 95 + rng.Intn(20)
		if seed%2 == 1 {
			n = 130 + rng.Intn(20)
		}
	}
	keys := rng.Perm(1000)[:n]
	for _, k := range keys {
		var coefficient int64
		switch {
		case heavy:
			coefficient = 9000000000 + rng.Int63n(999999999)
			if seed%2 == 1 && rng.Intn(4) == 0 {
				coefficient = -coefficient
			}
		case rng.Intn(3) == 0:
			coefficient = rng.Int63n(19999999999) - 9999999999
		default:
			coefficient = rng.Int63n(2000000) - 1000000
		}
		group := groups[rng.Intn(len(groups))]
		if heavy {
			group = "a"
		}
		amount, err := decimal.New(f.r.profile, big.NewInt(coefficient), 4)
		if err != nil {
			t.Fatal(err)
		}
		currency := currencies[rng.Intn(2)]
		if heavy {
			currency = "KRW"
		}
		all = append(all, row("k"+strconv.Itoa(k), group, currency, amount.String()))
	}
	shuffled := append([]Row(nil), all...)
	parts := [][]Row{}
	for len(shuffled) > 0 {
		size := min(1+rng.Intn(15), len(shuffled))
		part := append([]Row(nil), shuffled[:size]...)
		sort.Slice(part, func(a, b int) bool { return part[a].Key < part[b].Key })
		parts = append(parts, part)
		shuffled = shuffled[size:]
	}
	f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v1", Rows: n, Digest: digestOf(parts)})))
	sent := 0
	for sent < len(parts) {
		switch index := rng.Intn(sent + 2); {
		case index < sent:
			v, err := f.r.AddChunk(Chunk{Tenant: "t", Dataset: "d", Version: "v1", Index: index, Rows: parts[index]})
			if err != nil || !v.Repeat {
				t.Fatalf("seed %d resend %v", seed, err)
			}
			reached["resend"]++
			if other := (index + 1) % len(parts); other != index {
				expect(t, "conflict", second(f.r.AddChunk(Chunk{Tenant: "t", Dataset: "d", Version: "v1", Index: index, Rows: parts[other]})), ErrConflict)
				reached["conflict"]++
			}
		case index == sent:
			f.must(second(f.r.AddChunk(Chunk{Tenant: "t", Dataset: "d", Version: "v1", Index: index, Rows: parts[index]})))
			sent++
		default:
			if sent+1 < len(parts) {
				reason(t, "gap", second(f.r.AddChunk(Chunk{Tenant: "t", Dataset: "d", Version: "v1", Index: sent + 1, Rows: parts[sent+1]})), ReasonGap)
				reached["gap"]++
			}
		}
	}
	model := map[TotalKey]*big.Rat{}
	for _, r := range all {
		amount, ok := new(big.Rat).SetString(r.Amount)
		if !ok {
			t.Fatal(r.Amount)
		}
		key := TotalKey{Group: r.Group, Currency: r.Currency}
		if model[key] == nil {
			model[key] = new(big.Rat)
		}
		model[key].Add(model[key], amount)
	}
	overflow := false
	for _, sum := range model {
		if new(big.Rat).Abs(sum).Cmp(big8) >= 0 {
			overflow = true
		}
	}
	f.audit()
	_, err := f.r.Seal("t", "d", "v1")
	f.audit()
	if overflow {
		reason(t, "overflow", err, ReasonOverflow)
		reached["overflow"]++
		return
	}
	if err != nil {
		t.Fatalf("seed %d seal %v", seed, err)
	}
	reached["sealed"]++
	f.must(second(f.r.Publish("t", "d", "v1")))
	f.audit()
	sort.Slice(all, func(a, b int) bool { return all[a].Key < all[b].Key })
	got, after := []Row{}, ""
	for {
		page, err := f.r.Rows("t", "d", "v1", after, 1+rng.Intn(9))
		f.must(err)
		got = append(got, page.Rows...)
		if page.Next == "" {
			break
		}
		after = page.Next
	}
	if len(got) != len(all) {
		t.Fatalf("seed %d paged %d of %d", seed, len(got), len(all))
	}
	for index := range all {
		if got[index] != all[index] {
			t.Fatalf("seed %d row %d %+v %+v", seed, index, got[index], all[index])
		}
	}
	wantKeys := []TotalKey{}
	for key := range model {
		wantKeys = append(wantKeys, key)
	}
	sort.Slice(wantKeys, func(a, b int) bool {
		x, y := wantKeys[a], wantKeys[b]
		return x.Group < y.Group || x.Group == y.Group && x.Currency < y.Currency
	})
	totals := []Total{}
	var cursor *TotalKey
	for {
		page, err := f.r.Totals("t", "d", "v1", cursor, 1+rng.Intn(3))
		f.must(err)
		totals = append(totals, page.Totals...)
		if page.Next == nil {
			break
		}
		cursor = page.Next
	}
	if len(totals) != len(wantKeys) {
		t.Fatalf("seed %d totals %d of %d", seed, len(totals), len(wantKeys))
	}
	for index, key := range wantKeys {
		if totals[index].Key != key || totals[index].Amount != ratString(model[key]) {
			t.Fatalf("seed %d total %+v want %s %s", seed, totals[index], key, ratString(model[key]))
		}
	}
}
