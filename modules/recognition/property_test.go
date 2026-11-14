package recognition

import (
	"math/big"
	"math/rand"
	"strconv"
	"testing"
)

type mBatch struct {
	id, month, kind string
	provisional     bool
	net, tax        *big.Rat
	state           string
	target          *mBatch
	reversal        *mBatch
	results         []string
	carrier         *mAdj
}

type mAdj struct {
	id, month string
	net, tax  *big.Rat
	state     string
	carrier   *mBatch
}

func (b *mBatch) posted() bool {
	for _, r := range b.results {
		if r == ResultPosted {
			return true
		}
	}
	return false
}

func (b *mBatch) waiting() bool {
	for _, r := range b.results {
		if r == ResultSent || r == ResultAccepted || r == ResultUnknown {
			return true
		}
	}
	return false
}

func (b *mBatch) ended() bool { return b.state == BatchVoid || b.state == BatchAbandoned }

func (b *mBatch) holds() bool {
	return b.carrier != nil && (b.carrier.state == AdjustmentPending || b.carrier.state == AdjustmentApproved)
}

func (b *mBatch) cancelled() bool { return b.reversal != nil && b.reversal.posted() }

type mModel struct {
	batches []*mBatch
	adjs    []*mAdj
}

func (m *mModel) live(month string) *mBatch {
	for _, b := range m.batches {
		if b.month == month && b.kind == BatchInitial && !b.ended() && !b.cancelled() {
			return b
		}
	}
	return nil
}

func (m *mModel) canReverse(t *mBatch, month string) bool {
	if t == nil || t.month != month || t.kind == BatchReversal || !t.posted() || t.reversal != nil || t.holds() {
		return false
	}
	if t.kind == BatchInitial {
		for _, b := range m.batches {
			if b.month == month && b.kind == BatchAmendment && !b.ended() && !b.cancelled() {
				return false
			}
		}
	}
	return true
}

func rat(text string) *big.Rat {
	r, _ := new(big.Rat).SetString(text)
	return r
}

func ratText(r *big.Rat) string { return r.FloatString(0) }

type modelResult struct {
	net, tax, termNet, termTax *big.Rat
	state                      string
}

func (m *mModel) reconcile(months []string, billedNet, billedTax *big.Rat) modelResult {
	out := modelResult{net: new(big.Rat), tax: new(big.Rat), termNet: new(big.Rat), termTax: new(big.Rat)}
	collecting, unposted, pending, waiting, mixed := false, false, false, false, false
	for _, month := range months {
		live := m.live(month)
		if live == nil || live.state != BatchApproved {
			collecting = true
		}
		for _, b := range m.batches {
			if b.month != month {
				continue
			}
			if b.state == BatchOpen {
				collecting = true
			}
			if b.state == BatchApproved && b.posted() {
				out.net.Add(out.net, b.net)
				out.tax.Add(out.tax, b.tax)
				if b.provisional && b.kind != BatchReversal && !b.cancelled() {
					mixed = true
				}
			}
			if b.state == BatchApproved && !b.posted() && !b.holds() {
				unposted = true
			}
		}
		for _, a := range m.adjs {
			if a.month != month {
				continue
			}
			if a.state == AdjustmentPending {
				pending = true
			}
			if a.state == AdjustmentApproved && (a.carrier == nil || !a.carrier.posted()) {
				out.termNet.Add(out.termNet, a.net)
				out.termTax.Add(out.termTax, a.tax)
				waiting = true
			}
		}
	}
	baseNet := new(big.Rat).Add(out.net, out.termNet)
	baseTax := new(big.Rat).Add(out.tax, out.termTax)
	differs := baseNet.Cmp(billedNet) != 0 || baseTax.Cmp(billedTax) != 0
	switch {
	case collecting:
		out.state = StateCollecting
	case unposted:
		out.state = StatePostingUnconfirmed
	case mixed:
		out.state = StateMixedBasis
	case pending:
		out.state = StateAdjustmentPending
	case differs:
		out.state = StateAmountDifference
	case waiting:
		out.state = StateAdjustmentPosting
	default:
		out.state = StateMatched
	}
	return out
}

func TestPropertyReconciliationModel(t *testing.T) {
	seen := map[string]int{}
	for seed := int64(1); seed <= 400; seed++ {
		for state, n := range runRecognitionModel(t, seed) {
			seen[state] += n
		}
	}
	for _, state := range []string{StateCollecting, StatePostingUnconfirmed, StateMixedBasis, StateAdjustmentPending, StateAmountDifference, StateAdjustmentPosting, StateMatched} {
		if seen[state] < 50 {
			t.Fatalf("state %s seen %d times", state, seen[state])
		}
	}
}

func runRecognitionModel(t *testing.T, seed int64) map[string]int {
	rng := rand.New(rand.NewSource(seed))
	clock := &testClock{now: start}
	r := must(NewRegistry(testNumbers(t), Limits{Workspaces: 1, Profiles: 1, Batches: 1000, Lines: 4, Adjustments: 1000}, clock.read))
	must(r.Define("w1", testProfile(), human("controller")))
	m := &mModel{}
	months := []string{"2026-01", "2026-02", "2026-03"}
	amount := func() string { return strconv.Itoa(rng.Intn(21) - 5) }
	pickBatch := func() *mBatch {
		if len(m.batches) == 0 {
			return nil
		}
		return m.batches[rng.Intn(len(m.batches))]
	}
	pickAdj := func() *mAdj {
		if len(m.adjs) == 0 {
			return nil
		}
		return m.adjs[rng.Intn(len(m.adjs))]
	}
	counts := map[string]int{}
	keys := 0
	for step := 0; step < 120; step++ {
		label := "seed " + strconv.FormatInt(seed, 10) + " step " + strconv.Itoa(step)
		month := months[rng.Intn(3)]
		id := "b" + strconv.Itoa(step)
		check := func(op string, err error, want bool) {
			if (err == nil) != want {
				t.Fatalf("%s %s: got %v, model accepts %v", label, op, err, want)
			}
		}
		switch op := rng.Intn(13); op {
		case 0, 1:
			net, tax := amount(), amount()
			b := initial(id, month, net, tax)
			b.Provisional = rng.Intn(4) == 0
			_, err := r.Open("w1", b, human("accountant"))
			check("initial", err, m.live(month) == nil)
			if err == nil {
				m.batches = append(m.batches, &mBatch{id: id, month: month, kind: BatchInitial, provisional: b.Provisional, net: rat(net), tax: rat(tax), state: BatchOpen})
			}
		case 2:
			net, tax := amount(), amount()
			_, err := r.Open("w1", amendment(id, month, "2026-04", net, tax), human("accountant"))
			live := m.live(month)
			want := live != nil && live.state == BatchApproved && live.posted() && live.reversal == nil && (net != "0" || tax != "0")
			check("amendment", err, want)
			if err == nil {
				m.batches = append(m.batches, &mBatch{id: id, month: month, kind: BatchAmendment, net: rat(net), tax: rat(tax), state: BatchOpen})
			}
		case 3:
			target := pickBatch()
			if amend := m.amendments(); len(amend) > 0 && rng.Intn(2) == 0 {
				target = amend[rng.Intn(len(amend))]
				month = target.month
			}
			if target == nil {
				continue
			}
			rev := reversal(id, month, "2026-04", target.id)
			rev.Provisional = target.provisional
			_, err := r.Open("w1", rev, human("accountant"))
			live := m.live(month)
			check("reversal", err, live != nil && live.state == BatchApproved && m.canReverse(target, month))
			if err == nil {
				b := &mBatch{id: id, month: month, kind: BatchReversal, provisional: target.provisional, net: new(big.Rat).Neg(target.net), tax: new(big.Rat).Neg(target.tax), state: BatchOpen, target: target}
				target.reversal = b
				m.batches = append(m.batches, b)
			}
		case 4:
			b := pickBatch()
			if b == nil || b.state != BatchOpen {
				continue
			}
			v := must(r.BatchOf("w1", b.id))
			_, err := r.Approve("w1", b.id, v.Digest, human("controller"))
			check("approve", err, true)
			b.state = BatchApproved
		case 5:
			b := pickBatch()
			if b == nil || b.state == BatchVoid {
				continue
			}
			_, err := r.Void("w1", b.id, human("clerk"))
			check("void", err, b.state == BatchOpen && !b.holds())
			if err == nil {
				b.state = BatchVoid
				if b.target != nil {
					b.target.reversal = nil
				}
			}
		case 6:
			b := pickBatch()
			if b == nil || b.state == BatchAbandoned {
				continue
			}
			_, err := r.Abandon("w1", b.id, human("clerk"))
			check("abandon", err, b.state == BatchApproved && !b.posted() && !b.waiting() && !b.holds())
			if err == nil {
				b.state = BatchAbandoned
				if b.target != nil {
					b.target.reversal = nil
				}
			}
		case 7, 8:
			b := pickBatch()
			for _, x := range m.batches {
				if x.state == BatchApproved && !x.posted() && rng.Intn(3) == 0 {
					b = x
				}
			}
			if b == nil {
				continue
			}
			keys++
			key := "k" + strconv.Itoa(keys)
			_, err := r.Submit("w1", b.id, key, human("accountant"))
			check("submit", err, b.state == BatchApproved && !b.posted() && !b.waiting())
			if err != nil {
				continue
			}
			b.results = append(b.results, ResultSent)
			at := len(b.results) - 1
			for hop := 0; hop < 1+rng.Intn(3); hop++ {
				next := []string{ResultAccepted, ResultPosted, ResultFailed, ResultUnknown, ResultPosted}[rng.Intn(5)]
				res := Result{State: next}
				switch next {
				case ResultPosted:
					res.External = "je-" + key
				case ResultFailed:
					res.Reason = "erp_error"
				}
				_, err := r.Record("w1", key, res, system("erp"))
				current := b.results[at]
				allowed := reports[current][next] || current == next
				check("record "+current+"->"+next, err, allowed)
				if err == nil {
					b.results[at] = next
				}
			}
		case 9:
			a := &mAdj{id: "a" + strconv.Itoa(step), month: month, state: AdjustmentPending}
			beforeNet, afterNet, afterTax := amount(), amount(), amount()
			var carrier *mBatch
			if amend := m.amendments(); len(amend) > 0 && rng.Intn(2) == 0 {
				like := amend[rng.Intn(len(amend))]
				for _, x := range amend {
					if x.reversal != nil && rng.Intn(2) == 0 {
						like = x
					}
				}
				month, a.month = like.month, like.month
				beforeNet, afterNet, afterTax = "0", ratText(like.net), ratText(like.tax)
				if rng.Intn(2) == 0 {
					carrier = like
				}
			} else if rng.Intn(2) == 0 {
				carrier = pickBatch()
			}
			a.net, a.tax = new(big.Rat).Sub(rat(afterNet), rat(beforeNet)), rat(afterTax)
			adj := Adjustment{ID: a.id, Profile: "p1", Month: month, Cause: "other", Before: Amounts{beforeNet, "0"}, After: Amounts{afterNet, afterTax}}
			if carrier != nil {
				adj.Carrier = carrier.id
			}
			_, err := r.Propose("w1", adj, human("accountant"))
			want := a.net.Sign() != 0 || a.tax.Sign() != 0
			if carrier != nil {
				want = want && carrierFits(carrier, a)
			}
			check("propose", err, want)
			if err == nil {
				if carrier != nil {
					a.carrier, carrier.carrier = carrier, a
				}
				m.adjs = append(m.adjs, a)
			}
		case 10:
			a := pickAdj()
			for _, x := range m.adjs {
				if x.state == AdjustmentPending && rng.Intn(2) == 0 {
					a = x
				}
			}
			if a == nil || a.state != AdjustmentPending {
				continue
			}
			v := must(r.AdjustmentOf("w1", a.id))
			verdict := VerdictApprove
			if rng.Intn(5) == 0 {
				verdict = VerdictReject
			}
			_, err := r.Decide("w1", a.id, Decision{Verdict: verdict, Digest: v.Digest}, human("controller"))
			check("decide", err, true)
			a.state = map[string]string{VerdictApprove: AdjustmentApproved, VerdictReject: AdjustmentRejected}[verdict]
		case 11:
			a := pickAdj()
			if a == nil || a.state == AdjustmentWithdrawn {
				continue
			}
			_, err := r.Withdraw("w1", a.id, human("clerk"))
			check("withdraw", err, (a.state == AdjustmentPending || a.state == AdjustmentApproved) && (a.carrier == nil || !a.carrier.posted()))
			if err == nil {
				a.state = AdjustmentWithdrawn
			}
		case 12:
			a, b := pickAdj(), pickBatch()
			if a == nil || b == nil || a.carrier != nil {
				continue
			}
			for _, like := range m.amendments() {
				if like.month == a.month && like.net.Cmp(a.net) == 0 && like.tax.Cmp(a.tax) == 0 && rng.Intn(2) == 0 {
					b = like
				}
			}
			_, err := r.Link("w1", a.id, b.id, human("accountant"))
			check("link", err, (a.state == AdjustmentPending || a.state == AdjustmentApproved) && carrierFits(b, a))
			if err == nil {
				a.carrier, b.carrier = b, a
			}
		}
		for _, q := range [][]string{months, months[:1], months[1:]} {
			probe := m.reconcile(q, new(big.Rat), new(big.Rat))
			billedNet := new(big.Rat).Add(probe.net, probe.termNet)
			billedTax := new(big.Rat).Add(probe.tax, probe.termTax)
			if rng.Intn(4) == 0 {
				billedNet.Add(billedNet, big.NewRat(1, 1))
			}
			want := m.reconcile(q, billedNet, billedTax)
			got, err := r.Reconcile("w1", "p1", quarter("Q", ratText(billedNet), ratText(billedTax), q...))
			if err != nil {
				t.Fatalf("%s reconcile: %v", label, err)
			}
			if got.State != want.state || got.Recognised.Net != ratText(want.net) || got.Recognised.Tax != ratText(want.tax) || got.Term.Net != ratText(want.termNet) || got.Term.Tax != ratText(want.termTax) {
				t.Fatalf("%s reconcile %v: got %s %+v %+v, want %s %s/%s %s/%s", label, q, got.State, got.Recognised, got.Term, want.state, ratText(want.net), ratText(want.tax), ratText(want.termNet), ratText(want.termTax))
			}
			counts[got.State]++
			once := m.counted(q)
			if got.Base.Net != ratText(once[0]) || got.Base.Tax != ratText(once[1]) {
				t.Fatalf("%s counted once %v: base %+v, want %s/%s", label, q, got.Base, ratText(once[0]), ratText(once[1]))
			}
		}
	}
	return counts
}

func carrierFits(b *mBatch, a *mAdj) bool {
	return b.kind == BatchAmendment && !b.ended() && b.reversal == nil && b.month == a.month && b.net.Cmp(a.net) == 0 && b.tax.Cmp(a.tax) == 0 && !b.holds()
}

func (m *mModel) counted(months []string) [2]*big.Rat {
	in := map[string]bool{}
	for _, month := range months {
		in[month] = true
	}
	out := [2]*big.Rat{new(big.Rat), new(big.Rat)}
	for _, a := range m.adjs {
		if in[a.month] && a.state == AdjustmentApproved {
			out[0].Add(out[0], a.net)
			out[1].Add(out[1], a.tax)
		}
	}
	for _, b := range m.batches {
		replaced := func(x *mBatch) bool { return x != nil && x.carrier != nil && x.carrier.state == AdjustmentApproved }
		if !in[b.month] || b.state != BatchApproved || !b.posted() || replaced(b) || replaced(b.target) {
			continue
		}
		out[0].Add(out[0], b.net)
		out[1].Add(out[1], b.tax)
	}
	return out
}

func (m *mModel) amendments() []*mBatch {
	var out []*mBatch
	for _, b := range m.batches {
		if b.kind == BatchAmendment {
			out = append(out, b)
		}
	}
	return out
}
