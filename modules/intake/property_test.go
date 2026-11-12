package intake

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func errName(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, ErrInvalid):
		return "invalid"
	case errors.Is(err, ErrNotFound):
		return "notfound"
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrRejected):
		return "rejected"
	case errors.Is(err, ErrLimit):
		return "limit"
	}
	return "unknown"
}

type plan struct {
	spec   Spec
	chunks [][]Record
}

func propertyLimits() Limits {
	return Limits{MaxSize: 1000, MaxRatio: 10, Chunks: 3, ChunkRecords: 3, Artifacts: 8, Records: 14, Manifests: 3, Page: 100, Listed: 1000}
}

func randomPlan(t *testing.T, random *rand.Rand) plan {
	amounts := []string{"1", "2.5", "-1", "0", "1.0", "0.0001"}
	currencies := []string{"USD", "EUR"}
	keys := []string{"k1", "k2", "k3", "k4", "k5", "k6"}
	random.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	var chunks [][]Record
	used := 0
	for count := 1 + random.Intn(2); count > 0; count-- {
		var chunk []Record
		for size := 1 + random.Intn(3); size > 0 && used < len(keys); size-- {
			key := keys[used]
			used++
			if random.Intn(25) == 0 {
				key = keys[0]
			}
			amount, currency := amounts[random.Intn(len(amounts))], currencies[random.Intn(len(currencies))]
			digest := digestOf(fmt.Sprint(mScaled(amount)), currency)[:16]
			if random.Intn(12) == 0 {
				digest = digestOf("variant")[:16]
			}
			chunk = append(chunk, Record{Key: key, Digest: digest, Amount: amount, Currency: currency})
		}
		chunks = append(chunks, chunk)
	}
	workspace := "w1"
	if random.Intn(6) == 0 {
		workspace = "w2"
	}
	document := fmt.Sprintf("d%d", 1+random.Intn(4))
	kind := EventKeys
	if (document == "d3") != (random.Intn(14) == 0) {
		kind = LocatorKeys
	}
	period := "p1"
	if (document == "d4") != (random.Intn(14) == 0) {
		period = "p2"
	}
	id := fmt.Sprintf("a%d", 1+random.Intn(9))
	revision := uint64(1 + random.Intn(3))
	var s Spec
	switch roll := random.Intn(100); {
	case roll < 55:
		s = original(t, id, document, revision, chunks...)
	case roll < 65:
		s, chunks = base(id, document, revision), nil
	case roll < 80:
		s, chunks = scan(id, document, revision), nil
	default:
		s = derived(t, id, document, revision, fmt.Sprintf("a%d", 1+random.Intn(9)), chunks...)
	}
	s.Subject = Subject{WorkspaceID: workspace, Provider: "aws", Scope: "s1", Period: period}
	s.Kind = kind
	s.Bytes = digestOf("file", fmt.Sprint(random.Intn(4)))
	s.Location = "store:1"
	if random.Intn(8) == 0 {
		switch random.Intn(16) {
		case 0:
			s.Rows++
		case 1:
			s.Chunks++
		case 2:
			s.Sums = map[string]string{"USD": "1"}
		case 3:
			s.Size = 0
		case 4:
			s.Size, s.Expanded = 1001, 1001
		case 5:
			s.Expanded = s.Size*10 + 1
		case 6:
			s.Scan = Failed
		case 7:
			s.Bodies = NotRun
		case 8:
			s.Syntax = "maybe"
		case 9:
			s.Sums = map[string]string{"USD": "1.00001"}
		case 10:
			s.Chunks = 4
			s.Rows = 4 + random.Intn(9)
		case 11:
			s.Rows = s.Chunks*3 + 1
		case 12:
			s.ID = "a b"
		case 13:
			s.Totals = ""
		case 14:
			s.Sums = nil
		case 15:
			s.Holds = Rows
		}
	}
	return plan{s, chunks}
}

func gapNames(gaps []Gap) string {
	names := make([]string, len(gaps))
	for index, gap := range gaps {
		names[index] = gap.What
		if gap.Currency != "" {
			names[index] += ":" + gap.Currency
		}
	}
	return strings.Join(names, ",")
}

func blockerNames(blockers []Blocker) string {
	names := make([]string, len(blockers))
	for index, blocker := range blockers {
		if blocker.What == KeyConflict {
			names[index] = blocker.What + ":" + blocker.Key
		} else {
			names[index] = blocker.What + ":" + blocker.Document
		}
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

type attempt struct {
	what     string
	spec     Spec
	ws, id   string
	number   int
	records  []Record
	digest   string
	verifier string
	abandon  AbandonReason
	retire   RetireReason
	at       Subject
	expected uint64
}

func manifestLines(t *testing.T, r *Registry, at Subject, number uint64) []string {
	t.Helper()
	var lines []string
	for {
		page, err := r.Records(at, number, len(lines), r.limits.Page)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			sort.Strings(lines)
			return lines
		}
		for _, record := range page {
			lines = append(lines, fmt.Sprintf("%s/%s/%s=%s%s#%s@%s", record.Kind, record.Document, record.Key, record.Amount, record.Currency, record.Digest, record.Artifact))
		}
	}
}

func historyPlans(t *testing.T, random *rand.Rand) []plan {
	var plans []plan
	for len(plans) < 14 {
		made := randomPlan(t, random)
		if len(plans) > 0 {
			earlier := plans[random.Intn(len(plans))]
			switch {
			case made.spec.Role == Derived && random.Intn(10) < 8:
				for _, candidate := range plans {
					if candidate.spec.Role == Original && candidate.spec.Holds == AwaitingExtraction {
						made.spec.Subject, made.spec.Document, made.spec.Kind, made.spec.Revision, made.spec.Original = candidate.spec.Subject, candidate.spec.Document, candidate.spec.Kind, candidate.spec.Revision, candidate.spec.ID
					}
				}
			case random.Intn(5) == 0:
				id := made.spec.ID
				made = plan{earlier.spec, earlier.chunks}
				made.spec.ID = id
			}
		}
		plans = append(plans, made)
	}
	return plans
}

func TestRandomHistoriesAgainstAReferenceModel(t *testing.T) {
	random := rand.New(rand.NewSource(18600))
	limits := propertyLimits()
	outcomes := map[string]int{}
	for history := 0; history < 700; history++ {
		clock := &testClock{at: start}
		r, err := NewRegistry(limits, testProfile(t), clock.read)
		if err != nil {
			t.Fatal(err)
		}
		m := newModel(limits)
		plans := historyPlans(t, random)
		published := map[Subject][]string{}
		step := 0
		fail := func(what, got, want string) {
			t.Helper()
			t.Fatalf("history %d step %d %s:\n got %s\nwant %s", history, step, what, got, want)
		}
		apply := func(a attempt) {
			var got, want string
			switch a.what {
			case "register":
				state, _, err := r.Register(a.spec)
				got = errName(err)
				if err == nil {
					got = "ok:" + string(state)
				}
				want = m.register(a.spec)
			case "stage":
				state, err := r.Stage(a.ws, a.id, a.number, a.records)
				got = errName(err)
				if err == nil {
					got = "ok:" + string(state)
				}
				want = m.stage(a.ws, a.id, a.number, a.records)
			case "seal":
				state, gaps, err := r.Seal(a.ws, a.id)
				got = errName(err)
				if err == nil {
					got = "ok:" + string(state)
				} else if len(gaps) > 0 {
					got += ":" + gapNames(gaps)
				}
				want = m.seal(a.ws, a.id)
			case "verify":
				got = errName(r.Verify(a.ws, a.id, a.digest, a.verifier, "e1"))
				want = m.verify(a.ws, a.id, a.digest, a.verifier, "e1")
			case "abandon":
				got = errName(r.Abandon(a.ws, a.id, a.abandon))
				want = m.abandon(a.ws, a.id, a.abandon)
			case "retire":
				got = errName(r.Retire(a.ws, a.id, a.retire))
				want = m.retire(a.ws, a.id, a.retire)
			case "publish":
				before := r.dump()
				out, err := r.Publish(a.at, a.expected)
				outcome, pic := m.publish(a.at, a.expected)
				want = outcome
				switch {
				case err == nil && out.Written:
					got = fmt.Sprintf("written:%d", out.Number)
				case err == nil:
					got = fmt.Sprintf("same:%d", out.Number)
				case err == ErrRejected && out.BlockerTotal > 0:
					got = "blocked:" + blockerNames(out.Blockers)
				default:
					got = errName(err)
				}
				if !strings.HasPrefix(got, "written") && r.dump() != before {
					fail(a.what, "a write", "no write")
				}
				if got == want && out.Written {
					view, err := r.Manifest(a.at, out.Number)
					if err != nil {
						t.Fatal(err)
					}
					lines := manifestLines(t, r, a.at, out.Number)
					keys := map[string]bool{}
					for _, line := range lines {
						full, rest, _ := strings.Cut(line, "=")
						if keys[full] {
							fail(a.what, "two rows of "+full, "one")
						}
						keys[full] = true
						if _, artifact, _ := strings.Cut(rest, "@"); !pic.live[artifact] {
							fail(a.what, "a row of "+artifact, "rows of live artifacts")
						}
					}
					if strings.Join(lines, " ") != strings.Join(pic.attribution, " ") {
						fail("manifest", strings.Join(lines, " "), strings.Join(pic.attribution, " "))
					}
					sums := map[string]string{}
					for currency, sum := range pic.sums {
						sums[currency] = mAmount(sum)
					}
					if !reflect.DeepEqual(view.Sums, sums) || view.Records != len(lines) || view.Staging != pic.staging ||
						fmt.Sprint(view.Behind.Items) != fmt.Sprint(pic.behind) || fmt.Sprint(view.SameBytes.Items) != fmt.Sprint(pic.same) ||
						view.Behind.Total != len(pic.behind) || view.SameBytes.Total != len(pic.same) {
						fail("manifest summary", fmt.Sprintf("%+v", view), fmt.Sprintf("%+v", pic))
					}
					published[a.at] = append(published[a.at], fmt.Sprintf("%+v %s", view, strings.Join(lines, " ")))
				}
			}
			if got != want {
				fail(a.what+" "+a.ws+"/"+a.id, got, want)
			}
			outcomes[a.what+" "+strings.SplitN(got, ":", 2)[0]]++
			if strings.HasPrefix(got, "blocked") {
				for _, name := range strings.Split(strings.TrimPrefix(got, "blocked:"), ",") {
					outcomes["blocked "+strings.SplitN(name, ":", 2)[0]]++
				}
			}
			if strings.HasPrefix(got, "ok:") {
				outcomes[a.what+" "+got]++
			}
		}
		for step = 0; step < 110; step++ {
			chosen := plans[random.Intn(len(plans))]
			if random.Intn(3) > 0 {
				chosen = plans[random.Intn(7)]
			}
			ws, id := chosen.spec.Subject.WorkspaceID, chosen.spec.ID
			at := chosen.spec.Subject
			if random.Intn(100) < 60 {
				view, err := r.Artifact(ws, id)
				staged, stagedErr := r.Staged(ws, id)
				roll := random.Intn(10)
				switch {
				case err != nil:
					apply(attempt{what: "register", spec: chosen.spec})
				case view.State == Staging && stagedErr == nil && !staged.Complete && len(chosen.chunks) > 0:
					number := 1 + random.Intn(len(chosen.chunks))
					apply(attempt{what: "stage", ws: ws, id: id, number: number, records: chosen.chunks[number-1]})
				case view.State == Staging && view.Spec.Role == Derived && view.Verification == nil && stagedErr == nil:
					apply(attempt{what: "verify", ws: ws, id: id, digest: staged.Digest, verifier: "v1"})
				case view.State == Staging:
					apply(attempt{what: "seal", ws: ws, id: id})
				case view.State == Sealed && roll == 0:
					apply(attempt{what: "retire", ws: ws, id: id, retire: Reissued})
				default:
					apply(attempt{what: "publish", at: view.Spec.Subject, expected: uint64(len(m.manifests[view.Spec.Subject]))})
				}
			} else {
				if random.Intn(30) == 0 {
					ws = "w2"
				}
				switch roll := random.Intn(100); {
				case roll < 22:
					apply(attempt{what: "register", spec: chosen.spec})
				case roll < 50:
					number := 1 + random.Intn(2)
					var records []Record
					if number <= len(chosen.chunks) {
						records = chosen.chunks[number-1]
					}
					switch random.Intn(14) {
					case 0:
						other := plans[random.Intn(len(plans))]
						if len(other.chunks) > 0 {
							records = other.chunks[0]
						}
					case 1:
						number = random.Intn(5)
					case 2:
						if len(records) > 0 {
							records = append([]Record(nil), records...)
							records[0].Amount = []string{"x", "1.00001", "3", "1.00"}[random.Intn(4)]
						}
					case 3:
						if len(records) > 1 {
							records = []Record{records[1], records[0]}
						}
					case 4:
						if len(records) > 0 {
							records = append([]Record(nil), records...)
							records[0].Currency = []string{"usd", "KRW"}[random.Intn(2)]
						}
					case 5:
						records = append(append([]Record(nil), records...), rec("k9", "1", "USD"), rec("k8", "1", "USD"))
					}
					apply(attempt{what: "stage", ws: ws, id: id, number: number, records: records})
				case roll < 65:
					apply(attempt{what: "seal", ws: ws, id: id})
				case roll < 75:
					digest := digestOf("wrong")
					if staged, err := r.Staged(ws, id); err == nil && random.Intn(4) > 0 {
						digest = staged.Digest
					}
					apply(attempt{what: "verify", ws: ws, id: id, digest: digest, verifier: []string{"v1", "v1", "v1", "v2", ""}[random.Intn(5)]})
				case roll < 79:
					apply(attempt{what: "abandon", ws: ws, id: id, abandon: []AbandonReason{TransferFailed, Superseded, Superseded, "other"}[random.Intn(4)]})
				case roll < 87:
					apply(attempt{what: "retire", ws: ws, id: id, retire: []RetireReason{Reissued, Reissued, WrongSource, "other"}[random.Intn(4)]})
				default:
					expected := uint64(len(m.manifests[at]))
					if random.Intn(6) == 0 {
						expected = uint64(random.Intn(4))
					}
					apply(attempt{what: "publish", at: at, expected: expected})
				}
			}
			if step%7 != 0 {
				continue
			}
			status, err := r.Status(at)
			pic := m.picture(at)
			if len(m.bySubject[at]) == 0 {
				if err != ErrNotFound {
					fail("status", errName(err), "notfound")
				}
				continue
			}
			gotStatus := fmt.Sprintf("%d %s %s %d %v %v", status.Latest, status.Differs, blockerNames(status.Blockers), status.Staging, status.Behind.Items, status.SameBytes.Items)
			wantStatus := fmt.Sprintf("%d %s %s %d %v %v", len(m.manifests[at]), m.differs(at, pic), strings.Join(pic.blockers, ","), pic.staging, pic.behind, pic.same)
			if err != nil || gotStatus != wantStatus {
				fail("status", gotStatus, wantStatus)
			}
			outcomes["status "+status.Differs]++
			for subject, held := range published {
				for index, line := range held {
					view, _ := r.Manifest(subject, uint64(index+1))
					if again := fmt.Sprintf("%+v %s", view, strings.Join(manifestLines(t, r, subject, uint64(index+1)), " ")); again != line {
						fail("published manifest", again, line)
					}
				}
			}
		}
		for key, a := range m.artifacts {
			ws, id, _ := strings.Cut(key, "|")
			view, err := r.Artifact(ws, id)
			if err != nil || view.State != a.state {
				t.Fatalf("history %d artifact %s: %v %v, model %v", history, key, view.State, err, a.state)
			}
		}
		for ws, count := range m.records {
			if r.workspaces[ws].records != count {
				t.Fatalf("history %d records of %s: %d, model %d", history, ws, r.workspaces[ws].records, count)
			}
		}
	}
	needed := []string{"register ok:staging", "register ok:rejected", "register ok:sealed", "register invalid", "register conflict", "register notfound", "register rejected", "register limit",
		"stage ok:staging", "stage ok:inconsistent", "stage ok:sealed", "stage invalid", "stage notfound", "stage rejected", "stage limit",
		"seal ok:sealed", "seal ok:retired", "seal rejected", "seal notfound", "verify ok", "verify conflict", "verify rejected", "verify invalid", "verify notfound",
		"abandon ok", "abandon conflict", "abandon rejected", "abandon invalid", "retire ok", "retire conflict", "retire rejected", "retire invalid",
		"publish written", "publish same", "publish conflict", "publish rejected", "publish limit", "publish notfound", "publish blocked",
		"blocked " + InConflict, "blocked " + KeyConflict, "blocked " + Unreadable, "status " + DiffersContent, "status " + DiffersAttribution, "status " + Blocked, "status " + Same}
	var missing []string
	for _, name := range needed {
		if outcomes[name] == 0 {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("the histories never reached %q: %v", missing, outcomes)
	}
}

func TestSealingOrderDoesNotChangeWhatCounts(t *testing.T) {
	random := rand.New(rand.NewSource(18601))
	limits := propertyLimits()
	limits.Artifacts, limits.Records = 100, 1000
	compared := 0
	for round := 0; round < 400; round++ {
		var plans []plan
		taken := map[string]bool{}
		for len(plans) < 8 {
			made := randomPlan(t, random)
			made.spec.Subject.WorkspaceID = "w1"
			if taken[made.spec.ID] || made.spec.Role == Derived || made.spec.Holds == AwaitingExtraction {
				continue
			}
			made.spec.ID = fmt.Sprintf("%s-%d", made.spec.ID, len(plans))
			taken[made.spec.ID] = true
			plans = append(plans, made)
		}
		var contents, blockers []string
		for run := 0; run < 2; run++ {
			clock := &testClock{at: start}
			r, _ := NewRegistry(limits, testProfile(t), clock.read)
			for _, made := range plans {
				r.Register(made.spec)
				for index, chunk := range made.chunks {
					r.Stage("w1", made.spec.ID, index+1, chunk)
				}
			}
			for _, index := range random.Perm(len(plans)) {
				r.Seal("w1", plans[index].spec.ID)
			}
			for _, period := range []string{"p1", "p2"} {
				at := Subject{WorkspaceID: "w1", Provider: "aws", Scope: "s1", Period: period}
				out, err := r.Publish(at, 0)
				blockers = append(blockers, fmt.Sprint(run, period, errName(err), out.BlockerTotal, blockerNames(out.Blockers)))
				if err == nil {
					view, _ := r.Manifest(at, out.Number)
					contents = append(contents, fmt.Sprint(run, period, view.Content, view.Records, view.Sums))
					compared++
				}
			}
		}
		half := len(blockers) / 2
		for index := 0; index < half; index++ {
			if blockers[index][1:] != blockers[half+index][1:] {
				t.Fatalf("round %d: %s / %s", round, blockers[index], blockers[half+index])
			}
		}
		for index := 0; index < len(contents)/2; index++ {
			if contents[index][1:] != contents[len(contents)/2+index][1:] {
				t.Fatalf("round %d: %s / %s", round, contents[index], contents[len(contents)/2+index])
			}
		}
	}
	if compared < 200 {
		t.Fatalf("only %d manifests compared", compared)
	}
}
