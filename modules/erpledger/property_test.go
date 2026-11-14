package erpledger

import (
	"math/big"
	"math/rand"
	"strconv"
	"testing"
)

type mSub struct {
	key, conn, external string
	sources             []string
	state               string
	revision            int
	diverged            bool
	sent                bool
}

type mLedger struct {
	subs    []*mSub
	anchors map[string]bool
	refs    map[string]*mSub
}

func (m *mLedger) live(s *mSub) bool {
	if s.diverged {
		return true
	}
	switch s.state {
	case StateRejected, StateAbsent, StateSuperseded, StateWithdrawn:
		return false
	}
	return true
}

func (m *mLedger) find(key string) *mSub {
	for _, s := range m.subs {
		if s.key == key {
			return s
		}
	}
	return nil
}

func TestPropertyLedgerModel(t *testing.T) {
	reached := map[string]int{}
	for seed := int64(1); seed <= 300; seed++ {
		for k, v := range runLedgerModel(t, seed) {
			reached[k] += v
		}
	}
	for _, state := range []string{StatePrepared, StateSent, StateDraft, StatePosted, StateRejected, StateAbsent, StateUnknown, StateWithdrawn, StateSuperseded, "diverged", "stale", "orphan", "resolved", "linked"} {
		if reached[state] < 20 {
			t.Fatalf("%s reached %d times", state, reached[state])
		}
	}
}

func runLedgerModel(t *testing.T, seed int64) map[string]int {
	rng := rand.New(rand.NewSource(seed))
	f := newFixture(t)
	r := must(NewRegistry(testNumbers(t), Limits{Workspaces: 1, Connections: 4, Mappings: 100, Anchors: 1000, Submissions: 1000, Lines: 4, Documents: 10, Rows: 4, Observations: 10000, Links: 1000, Surveys: 100}, f.clock.read))
	f.r = r
	f.connect(t, apiConnection("api"))
	f.connect(t, apiConnection("api2"))
	m := &mLedger{anchors: map[string]bool{}, refs: map[string]*mSub{}}
	sourceNames := []string{"s1", "s2", "s3", "s4", "s5"}
	counts := map[string]int{}
	obs, docs := 0, 0
	var docNames [][2]string
	pick := func() *mSub {
		if len(m.subs) == 0 {
			return nil
		}
		return m.subs[rng.Intn(len(m.subs))]
	}
	for step := 0; step < 80; step++ {
		label := "seed " + strconv.FormatInt(seed, 10) + " step " + strconv.Itoa(step)
		check := func(op string, err error, want string) {
			if got := kind(err); got != want {
				t.Fatalf("%s %s: got %s want %s", label, op, got, want)
			}
		}
		switch op := rng.Intn(13); op {
		case 0, 1:
			conn := []string{"api", "api2"}[rng.Intn(2)]
			key := "k" + strconv.Itoa(step)
			srcs := []string{sourceNames[rng.Intn(5)]}
			if second := sourceNames[rng.Intn(5)]; second != srcs[0] && rng.Intn(2) == 0 {
				srcs = append(srcs, second)
			}
			var lines []Line
			for i, s := range srcs {
				lines = append(lines, line("l"+strconv.Itoa(i), s, "1"))
			}
			_, err := f.r.Prepare("w1", invoice(key, conn, lines...), human("biller"))
			want := "ok"
			for _, s := range srcs {
				if m.anchors[s] {
					want = ReasonAlreadyReflected
				}
			}
			if want == "ok" {
				for _, s := range srcs {
					for _, other := range m.subs {
						if m.live(other) && contains(other.sources, s) {
							want = ReasonAlreadySubmitted
						}
					}
				}
			}
			check("prepare", err, want)
			if err == nil {
				m.subs = append(m.subs, &mSub{key: key, conn: conn, sources: srcs, state: StatePrepared})
			}
		case 2:
			s := pick()
			if s == nil {
				continue
			}
			_, err := f.r.Command("w1", s.key, CommandSubmit, human("biller"))
			want := "ok"
			if s.state != StatePrepared {
				want = ReasonWrongState
			} else if s.diverged {
				want = ReasonDiverged
			}
			check("submit", err, want)
			if err == nil {
				s.state, s.sent = StateSent, true
			}
		case 3, 4, 5:
			s := pick()
			if s == nil {
				continue
			}
			obs++
			state := []string{StateDraft, StatePosted, StateRejected, StateAbsent, StateUnknown}[rng.Intn(5)]
			source := []string{SourceWebhook, SourcePoll, SourceResultFile}[rng.Intn(3)]
			if state == StateAbsent && source == SourceWebhook {
				source = SourcePoll
			}
			revision := s.revision + 1
			if rng.Intn(4) == 0 && s.revision > 0 {
				revision = 1 + rng.Intn(s.revision)
			}
			external := ""
			if state == StateDraft || state == StatePosted {
				external = s.external
				if external == "" {
					external = "INV-" + s.key
				}
			}
			view, err := f.r.Observe("w1", Observation{ID: "o" + strconv.Itoa(obs), Connection: s.conn, Key: s.key, External: external, Revision: revision, State: state, Source: source}, system("connector"))
			if s.state == StatePrepared {
				check("observe prepared", err, ReasonWrongState)
				continue
			}
			check("observe", err, "ok")
			outcome := m.observe(s, state, external, revision)
			if view.Outcome != outcome {
				t.Fatalf("%s observe %s %s r%d on %s/r%d: got %s want %s", label, s.key, state, revision, s.state, s.revision, view.Outcome, outcome)
			}
			counts[outcome]++
		case 6:
			obs++
			v := must(f.r.Observe("w1", Observation{ID: "o" + strconv.Itoa(obs), Connection: "api", External: "GHOST-" + strconv.Itoa(obs), Revision: 1, State: StatePosted, Source: SourceWebhook}, system("connector")))
			if v.Outcome != "orphan" {
				t.Fatalf("%s orphan %+v", label, v)
			}
			counts["orphan"]++
		case 7:
			s := pick()
			if s == nil {
				continue
			}
			_, err := f.r.Withdraw("w1", s.key, human("clerk"))
			want := "ok"
			if s.state != StatePrepared {
				want = ReasonWrongState
				if s.state == StateWithdrawn {
					want = "ok"
				}
			}
			check("withdraw", err, want)
			if err == nil && s.state == StatePrepared {
				s.state, s.diverged = StateWithdrawn, false
			}
		case 8:
			src := sourceNames[rng.Intn(5)]
			id := "a" + strconv.Itoa(step)
			must(f.r.Anchor("w1", Anchor{ID: id, Connection: "api", Kind: AnchorSalesOrder, External: id, Customer: "acme", Sources: []string{src}}, human("admin")))
			m.anchors[src] = true
			for _, s := range m.subs {
				if m.live(s) && contains(s.sources, src) {
					s.diverged = true
				}
			}
		case 9:
			s := pick()
			if s == nil || !s.diverged {
				continue
			}
			_, err := f.r.Resolve("w1", s.key, Resolution{Outcome: ResolveSuperseded, Evidence: "e"}, human("lead"))
			check("supersede", err, "ok")
			s.state, s.diverged = StateSuperseded, false
		case 10:
			s := pick()
			if s == nil || !s.diverged {
				continue
			}
			outcome := []string{ResolveKeep, ResolveAccept}[rng.Intn(2)]
			if _, err := f.r.Resolve("w1", s.key, Resolution{Outcome: outcome, Evidence: "e"}, human("lead")); err == nil {
				v := must(f.r.SubmissionOf("w1", s.key))
				s.state, s.diverged, s.external = v.State, v.Diverged, v.External
				counts["resolved"]++
			}
		case 11:
			s := pick()
			if s == nil || s.external == "" {
				continue
			}
			docs++
			revision := 1
			if rng.Intn(3) == 0 && len(docNames) > 0 {
				name := docNames[rng.Intn(len(docNames))]
				prev := must(f.r.DocumentOf("w1", name[0], name[1]))
				revision = prev.Document.Revision + 1
				_, err := f.r.Document("w1", Document{Connection: name[0], External: name[1], Revision: revision, Rows: []Row{row("r1", strconv.Itoa(rng.Intn(5)), "0", "1"), row("r2", strconv.Itoa(-rng.Intn(3)), "0", "1")}}, system("connector"))
				check("document revision", err, "ok")
				continue
			}
			_, err := f.r.Document("w1", Document{Connection: s.conn, External: "D" + strconv.Itoa(docs), Revision: 1, Rows: []Row{row("r1", strconv.Itoa(rng.Intn(4)), "0", "1"), row("r2", "1", "0", "1")}}, system("connector"))
			check("document", err, "ok")
			docNames = append(docNames, [2]string{s.conn, "D" + strconv.Itoa(docs)})
		case 12:
			s := pick()
			if s == nil || len(docNames) == 0 {
				continue
			}
			name := docNames[rng.Intn(len(docNames))]
			l := Link{Connection: name[0], External: name[1], Row: []string{"r1", "r2"}[rng.Intn(2)], Key: s.key, Line: "l" + strconv.Itoa(rng.Intn(len(s.sources))), Net: strconv.Itoa(rng.Intn(3)), Tax: "0", Quantity: strconv.Itoa(rng.Intn(2))}
			if _, err := f.r.Link("w1", l, human("biller")); err == nil {
				counts["linked"]++
			}
		}
		allocationInvariants(t, f, label, docNames, m)
		for _, s := range m.subs {
			v := must(f.r.SubmissionOf("w1", s.key))
			if v.State != s.state || v.Revision != s.revision || v.Diverged != s.diverged {
				t.Fatalf("%s %s: got %s r%d d%v, want %s r%d d%v", label, s.key, v.State, v.Revision, v.Diverged, s.state, s.revision, s.diverged)
			}
			counts[s.state]++
			if s.diverged {
				counts["diverged"]++
			}
		}
		for _, src := range sourceNames {
			var holders []SubmissionView
			for _, s := range m.subs {
				v := must(f.r.SubmissionOf("w1", s.key))
				if contains(s.sources, src) && (v.Diverged || v.State != StateRejected && v.State != StateAbsent && v.State != StateSuperseded && v.State != StateWithdrawn) {
					holders = append(holders, v)
				}
			}
			if len(holders) > 1 {
				for _, h := range holders {
					if !h.Diverged {
						t.Fatalf("%s source %s held by %d live submissions, %s not diverged", label, src, len(holders), h.Submission.Key)
					}
				}
			}
			if m.anchors[src] {
				for _, h := range holders {
					if !h.Diverged {
						t.Fatalf("%s anchored source %s live in %s without divergence", label, src, h.Submission.Key)
					}
				}
			}
		}
	}
	return counts
}

func (m *mLedger) observe(s *mSub, state, external string, revision int) string {
	if revision <= s.revision {
		return "stale"
	}
	if state == StateAbsent && (s.state == StateSent || s.state == StateDraft) {
		s.revision, s.state = revision, StateUnknown
		return "premature"
	}
	s.revision = revision
	if state == s.state {
		return "repeated"
	}
	switch s.state {
	case StatePosted, StateRejected, StateAbsent, StateSuperseded, StateWithdrawn:
		retiredBefore := s.state != StatePosted
		s.diverged = true
		if retiredBefore {
			for _, other := range m.subs {
				if other == s || !m.live(other) {
					continue
				}
				for _, src := range s.sources {
					if contains(other.sources, src) {
						other.diverged = true
					}
				}
			}
		}
		return "diverged"
	}
	s.state = state
	if external != "" && s.external == "" {
		s.external = external
	}
	return "applied"
}

func contains(list []string, item string) bool {
	for _, x := range list {
		if x == item {
			return true
		}
	}
	return false
}

func allocationInvariants(t *testing.T, f fixture, label string, docNames [][2]string, m *mLedger) {
	perLine := map[string]*big.Int{}
	for _, name := range docNames {
		d := must(f.r.DocumentOf("w1", name[0], name[1]))
		rows := map[string]*big.Int{}
		for _, r := range d.Document.Rows {
			v, _ := new(big.Int).SetString(r.Net, 10)
			rows[r.ID] = v
		}
		used := map[string]*big.Int{}
		for _, l := range d.Links {
			if l.Stale {
				continue
			}
			v, _ := new(big.Int).SetString(l.Link.Net, 10)
			if used[l.Link.Row] == nil {
				used[l.Link.Row] = new(big.Int)
			}
			used[l.Link.Row].Add(used[l.Link.Row], v)
			k := l.Link.Key + "|" + l.Link.Line
			if perLine[k] == nil {
				perLine[k] = new(big.Int)
			}
			perLine[k].Add(perLine[k], v)
		}
		for id, sum := range used {
			whole := rows[id]
			if whole == nil || sum.Sign() != 0 && (sum.Sign() != whole.Sign() || new(big.Int).Abs(sum).Cmp(new(big.Int).Abs(whole)) > 0) {
				t.Fatalf("%s row %s/%s over-allocated: %v of %v", label, name[1], id, sum, whole)
			}
		}
	}
	for k, sum := range perLine {
		if sum.Cmp(big.NewInt(1)) > 0 {
			t.Fatalf("%s line %s allocated %v of 1", label, k, sum)
		}
	}
	for _, s := range m.subs {
		rec := must(f.r.Reconcile("w1", s.key))
		for _, d := range rec.Documents {
			doc := must(f.r.DocumentOf("w1", d.Connection, d.External))
			want := new(big.Int)
			for _, l := range doc.Links {
				if l.Stale {
					continue
				}
				holder := m.find(l.Link.Key)
				if holder == nil || !m.live(holder) {
					continue
				}
				v, _ := new(big.Int).SetString(l.Link.Net, 10)
				want.Add(want, v)
			}
			if d.Attributed.Net != want.String() {
				t.Fatalf("%s attributed %s: got %s want %s", label, d.External, d.Attributed.Net, want)
			}
		}
	}
}
