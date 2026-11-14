package disclosure

import (
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestPropertyProjectionModel(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	customers := []string{"acme", "beta", "gamma"}
	for trial := 0; trial < 200; trial++ {
		f := newFixture(t)
		scope := []string{customers[rng.Intn(3)]}
		if rng.Intn(2) == 0 {
			scope = append(scope, customers[(rng.Intn(2)+1+indexOf(customers, scope[0]))%3])
		}
		id := "r" + strconv.Itoa(trial)
		f.grant(t, Recipient{ID: id, Role: RolePartner, Customers: scope, Policy: "std"})
		var input []Row
		for i := 0; i < 1+rng.Intn(9); i++ {
			row := Row{"customer": customers[rng.Intn(3)], "service": "svc" + strconv.Itoa(rng.Intn(4)), "amount": strconv.Itoa(rng.Intn(1000)), "cost": "c" + strconv.Itoa(rng.Intn(9))}
			if rng.Intn(3) == 0 {
				row["extra"+strconv.Itoa(rng.Intn(3))] = "e"
			}
			input = append(input, row)
		}
		p := must(f.r.Project("w1", Request{Policy: "std", Recipient: id, Rows: input}, system("x")))
		var want []string
		in := map[string]bool{}
		for _, c := range scope {
			in[c] = true
		}
		for _, row := range input {
			if in[row["customer"]] {
				want = append(want, row["customer"]+"|"+row["service"])
			}
		}
		sort.Strings(want)
		var got []string
		for _, row := range p.Rows {
			got = append(got, strings.Join(row, "|"))
		}
		sort.Strings(got)
		if strings.Join(p.Fields, ",") != "customer,service" || strings.Join(got, ";") != strings.Join(want, ";") {
			t.Fatalf("trial %d: got %v want %v", trial, got, want)
		}
		shuffled := append([]Row(nil), input...)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		if q := must(f.r.Project("w1", Request{Policy: "std", Recipient: id, Rows: shuffled}, system("y"))); q.Digest != p.Digest {
			t.Fatalf("trial %d: order changed digest", trial)
		}
	}
}

func indexOf(list []string, v string) int {
	for i, x := range list {
		if x == v {
			return i
		}
	}
	return -1
}
