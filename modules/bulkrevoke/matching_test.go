package bulkrevoke

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/rand/v2"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

var terms = []FilterTerm{{Key: "q", Value: "ann"}, {Key: "role", Value: "member"}}

func (f fixture) match(owner string, users ...string) View {
	f.t.Helper()
	v, err := f.r.AddMatching("w1", owner, terms, users, start, "r7")
	if err != nil {
		f.t.Fatal(err)
	}
	return v
}

func TestAddMatching(t *testing.T) {
	f := setup(t, defaults)
	bad := map[string]func() error{
		"workspace": func() error { return second(f.r.AddMatching("", "ann", terms, []string{"u1"}, start, "r7")) },
		"owner":     func() error { return second(f.r.AddMatching("w1", "a b", terms, []string{"u1"}, start, "r7")) },
		"key": func() error {
			return second(f.r.AddMatching("w1", "ann", []FilterTerm{{Key: "a b", Value: "x"}}, []string{"u1"}, start, "r7"))
		},
		"empty value": func() error {
			return second(f.r.AddMatching("w1", "ann", []FilterTerm{{Key: "q"}}, []string{"u1"}, start, "r7"))
		},
		"long value": func() error {
			return second(f.r.AddMatching("w1", "ann", []FilterTerm{{Key: "q", Value: strings.Repeat("x", 65)}}, []string{"u1"}, start, "r7"))
		},
		"invalid text": func() error {
			return second(f.r.AddMatching("w1", "ann", []FilterTerm{{Key: "q", Value: "\xff"}}, []string{"u1"}, start, "r7"))
		},
		"repeated key": func() error {
			return second(f.r.AddMatching("w1", "ann", []FilterTerm{{Key: "q", Value: "a"}, {Key: "q", Value: "b"}}, []string{"u1"}, start, "r7"))
		},
		"unordered keys": func() error {
			return second(f.r.AddMatching("w1", "ann", []FilterTerm{{Key: "role", Value: "a"}, {Key: "q", Value: "b"}}, []string{"u1"}, start, "r7"))
		},
		"no time":     func() error { return second(f.r.AddMatching("w1", "ann", terms, []string{"u1"}, time.Time{}, "r7")) },
		"no revision": func() error { return second(f.r.AddMatching("w1", "ann", terms, []string{"u1"}, start, "")) },
		"long revision": func() error {
			return second(f.r.AddMatching("w1", "ann", terms, []string{"u1"}, start, strings.Repeat("r", 129)))
		},
		"bad user":        func() error { return second(f.r.AddMatching("w1", "ann", terms, []string{"u 1"}, start, "r7")) },
		"repeated person": func() error { return second(f.r.AddMatching("w1", "ann", terms, []string{"u1", "u1"}, start, "r7")) },
	}
	for name, call := range bad {
		f.want(name, call(), ErrInvalid)
	}
	f.want("nobody", second(f.r.AddMatching("w1", "ann", terms, nil, start, "r7")), ErrEmpty)
	edge, err := f.r.AddMatching("w1", "edge", []FilterTerm{{Key: "q", Value: strings.Repeat("x", 64)}}, []string{"u1"}, start, strings.Repeat("r", 128))
	if err != nil || edge.Filter[0].Value != strings.Repeat("x", 64) || len(edge.FixedRevision) != 128 {
		t.Fatalf("a 64-byte value and a 128-byte revision are accepted: %v", err)
	}
	f.want("discard the edge draft", second(f.r.Discard("w1", "edge", edge.ID)), nil)
	f.want("too many", second(f.r.AddMatching("w1", "ann", terms, []string{"u1", "u2", "u3", "u4", "u5", "u6", "u7", "u8", "u9"}, start, "r7")), ErrTooMany)
	if _, err := f.r.AddMatching("w1", "ann", nil, []string{"u1", "u2", "u3", "u4", "u5", "u6", "u7", "u8"}, start.Add(time.Second), "r7"); err != nil {
		t.Fatalf("exactly the limit with no terms: %v", err)
	}
	f.want("a second open draft", second(f.r.AddMatching("w1", "ann", terms, []string{"u1"}, start, "r7")), ErrOpenDraft)
	f.add("bob", "", "u1")
	v := f.match("cat", "u3", "u2")
	if v.Selection != Filter || v.Matched != 2 || v.Count != 2 || v.FixedRevision != "r7" || !v.FixedAt.Equal(start) || !reflect.DeepEqual(v.Filter, terms) ||
		v.State != Draft || v.ID != DraftID(v.Number) || !v.ExpiresAt.Equal(start.Add(30*time.Minute)) {
		t.Fatalf("filter draft %+v", v)
	}
	f.want("the pending limit", second(f.r.AddMatching("w1", "dan", terms, []string{"u1"}, start, "r7")), ErrPending)
	f.want("adding to a filter draft", second(f.r.Add("w1", "cat", v.ID, []string{"u9"})), ErrState)
	f.want("adding to someone else's draft", second(f.r.Add("w1", "ann", v.ID, []string{"u9"})), ErrNotFound)
	if users, _ := f.r.DraftUsers("w1", "cat", v.ID); !reflect.DeepEqual(users, []string{"u2", "u3"}) {
		t.Errorf("draft users %v", users)
	}
	got, err := f.r.DraftView("w1", "cat", v.ID)
	if err != nil || got.Selection != Filter || !reflect.DeepEqual(got.Filter, terms) {
		t.Errorf("draft view %+v %v", got, err)
	}
	got.Filter[0].Value = "changed"
	if again, _ := f.r.DraftView("w1", "cat", v.ID); again.Filter[0].Value != "ann" {
		t.Error("the view shares its filter")
	}
	f.want("another owner's view", second(f.r.DraftView("w1", "ann", v.ID)), ErrNotFound)
	f.want("an unknown draft", second(f.r.DraftView("w1", "cat", "bulk-draft-99")), ErrNotFound)
	*f.now = start.Add(time.Hour)
	f.want("an expired draft", second(f.r.DraftView("w1", "cat", v.ID)), ErrNotFound)
}

func filterList(f fixture, owner string, at time.Time, users ...string) View {
	f.t.Helper()
	d, err := f.r.AddMatching("w1", owner, terms, users, at, "r7")
	if err != nil {
		f.t.Fatal(err)
	}
	targets := []Target{}
	for _, u := range users {
		targets = append(targets, target(u))
	}
	v, reused, err := f.r.Prepare("w1", owner, d.ID, SuspendMembership, "r8", targets, nil, Impact{})
	if err != nil || reused {
		f.t.Fatalf("prepare %v %v", err, reused)
	}
	return v
}

func TestFilterListDigest(t *testing.T) {
	f := setup(t, Limits{MaxTargets: 5, MaxMatching: 5, Pending: 10, Kept: 5, Lifetime: time.Hour})
	first := filterList(f, "ann", start, "u1", "u2")
	explicit := Digest(SuspendMembership, Explicit, []Target{target("u1"), target("u2")}, nil)
	if first.Digest == explicit || first.Digest == Digest(SuspendMembership, Filter, []Target{target("u1"), target("u2")}, nil) {
		t.Fatal("the filter digest ignores the fixed facts")
	}
	if first.Digest != ListDigest(first, SuspendMembership, first.Targets, nil) || first.Filter == nil {
		t.Fatal("the digest is not the list digest")
	}
	second := filterList(f, "ann", start.Add(time.Second), "u1", "u2")
	if second.ID == first.ID || second.Digest == first.Digest {
		t.Fatal("two fixes of one filter collapsed")
	}
	base := first
	for name, change := range map[string]func(*View){
		"terms":    func(v *View) { v.Filter = []FilterTerm{{Key: "q", Value: "anne"}, {Key: "role", Value: "member"}} },
		"matched":  func(v *View) { v.Matched++ },
		"time":     func(v *View) { v.FixedAt = v.FixedAt.Add(time.Nanosecond) },
		"revision": func(v *View) { v.FixedRevision = "r9" },
	} {
		v := base
		change(&v)
		if ListDigest(v, SuspendMembership, base.Targets, nil) == base.Digest {
			t.Errorf("the digest ignores the %s", name)
		}
	}
	f.want("an explicit digest for a filter list", second2(f.r.Execute("w1", first.ID, explicit, "bob", 1, nil)), ErrDigest)
	if _, err := f.r.Execute("w1", first.ID, first.Digest, "bob", 1, nil); err != nil {
		t.Fatal(err)
	}
}

func second2(_ View, err error) error { return err }

func oldDigest(effect Effect, selection string, targets []Target, exclusions []Exclusion) string {
	sum := sha256.New()
	field := func(texts ...string) {
		for _, text := range texts {
			var size [8]byte
			binary.BigEndian.PutUint64(size[:], uint64(len(text)))
			sum.Write(size[:])
			sum.Write([]byte(text))
		}
	}
	field("bulk_revocation_v1", string(effect), selection, strconv.Itoa(len(targets)))
	for _, t := range sortedTargets(targets) {
		until := ""
		if !t.Until.IsZero() {
			until = t.Until.UTC().Format(time.RFC3339Nano)
		}
		field(t.User, t.Login, t.Role, until, strconv.FormatUint(t.Version, 10), strconv.FormatUint(t.Epoch, 10), t.UserState, t.MembershipState, strconv.FormatBool(t.Administrator))
	}
	field(strconv.Itoa(len(exclusions)))
	for _, x := range sortedExclusions(exclusions) {
		field(x.User, x.Login, x.Reason)
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func TestExplicitDigestIsUnchanged(t *testing.T) {
	pinned := Digest(SuspendMembership, Explicit, []Target{target("u2"), target("u1")}, []Exclusion{{User: "u3", Reason: ExcludedSelf}})
	if pinned != "d2166a50866e70d4ad4101f850d7aac4820469a782dcad61335c287815f0dedd" {
		t.Fatal("the pinned explicit digest changed")
	}
	r := rand.New(rand.NewPCG(323, 1))
	for round := 0; round < 300; round++ {
		targets, exclusions := []Target{}, []Exclusion{}
		for i := 0; i < r.IntN(6); i++ {
			x := target("u" + strconv.Itoa(r.IntN(50)))
			x.Version, x.Administrator = uint64(r.IntN(9)), r.IntN(2) == 0
			if r.IntN(2) == 0 {
				x.Until = start.Add(time.Duration(r.IntN(1000)) * time.Minute)
			}
			targets = append(targets, x)
		}
		for i := 0; i < r.IntN(4); i++ {
			exclusions = append(exclusions, Exclusion{User: "x" + strconv.Itoa(i), Login: strconv.Itoa(r.IntN(9)), Reason: ExcludedNotFound})
		}
		effect := []Effect{SuspendMembership, SuspendUser}[r.IntN(2)]
		if Digest(effect, Explicit, targets, exclusions) != oldDigest(effect, Explicit, targets, exclusions) {
			t.Fatalf("round %d: the explicit digest differs from the previous algorithm", round)
		}
		v := View{Selection: Explicit}
		if ListDigest(v, effect, targets, exclusions) != oldDigest(effect, Explicit, targets, exclusions) {
			t.Fatalf("round %d: an explicit list digest differs", round)
		}
	}
}

func TestCanonicalFilterIsInjective(t *testing.T) {
	r := rand.New(rand.NewPCG(323, 2))
	alphabet := []string{"a", ":", "1", "2", "0", " ", "q", "é"}
	keys := []string{"holder", "invitation", "membership", "period", "q", "role", "signin", "status"}
	seen := map[string][]FilterTerm{}
	for round := 0; round < 20000; round++ {
		list := []FilterTerm{}
		for _, key := range keys {
			if r.IntN(3) != 0 {
				continue
			}
			var b strings.Builder
			for i := 0; i <= r.IntN(4); i++ {
				b.WriteString(alphabet[r.IntN(len(alphabet))])
			}
			list = append(list, FilterTerm{Key: key, Value: b.String()})
		}
		code := CanonicalFilter(list)
		if before, ok := seen[code]; ok && !reflect.DeepEqual(before, list) && !(len(before) == 0 && len(list) == 0) {
			t.Fatalf("%v and %v both encode as %q", before, list, code)
		}
		seen[code] = list
	}
	if CanonicalFilter([]FilterTerm{{Key: "q", Value: "1:a"}}) != "1:q3:1:a" {
		t.Errorf("encoding %q", CanonicalFilter([]FilterTerm{{Key: "q", Value: "1:a"}}))
	}
}

func TestListSummariesEqualTheFullViewsWithoutLists(t *testing.T) {
	f := setup(t, Limits{MaxTargets: 5, MaxMatching: 5, Pending: 10, Kept: 5, Lifetime: time.Hour})
	f.add("ann", "", "u1")
	v := filterList(f, "bob", start, "u1", "u2")
	executed, _ := f.r.Execute("w1", v.ID, v.Digest, "cat", 1, nil)
	for _, row := range f.r.List("w1", "ann") {
		full, _ := f.r.Get("w1", "ann", row.ID)
		full.Items, full.Targets, full.Exclusions = nil, nil, nil
		if !reflect.DeepEqual(row, full) {
			t.Errorf("summary %+v\nfull %+v", row, full)
		}
	}
	rows := f.r.List("w1", "ann")
	rows[1].Filter[0].Value = "changed"
	if again, _ := f.r.Get("w1", "ann", executed.ID); again.Filter[0].Value != "ann" {
		t.Error("a summary shares its filter")
	}
}

func TestSpanCopiesOnlyItsItems(t *testing.T) {
	f := setup(t, Limits{MaxTargets: 5, MaxMatching: 5, Pending: 10, Kept: 5, Lifetime: time.Hour})
	v := filterList(f, "ann", start, "u1", "u2", "u3")
	if _, err := f.r.Span("w1", v.ID, 0, 1); !errors.Is(err, ErrNotFound) {
		t.Error("a span of a prepared list")
	}
	executed, _ := f.r.Execute("w1", v.ID, v.Digest, "bob", 1, nil)
	executed.Items[0].Cleanup = []Cleanup{{Item: "x"}}
	span, err := f.r.Span("w1", v.ID, 1, 9)
	if err != nil || len(span.Items) != 2 || span.Items[0].Index != 1 || span.Items[1].Index != 2 || span.Targets != nil || span.ID != v.ID || !span.DenyAt.Equal(executed.DenyAt) {
		t.Fatalf("span %+v %v", span, err)
	}
	if span, _ := f.r.Span("w1", v.ID, -3, 0); len(span.Items) != 1 || span.Items[0].Index != 0 {
		t.Errorf("span from before the start %+v", span.Items)
	}
	if span, _ := f.r.Span("w1", v.ID, 5, 9); len(span.Items) != 0 {
		t.Errorf("span past the end %+v", span.Items)
	}
	_ = f.r.Header("w1", v.ID, Receipt{Sequence: 1})
	_ = f.r.Record("w1", v.ID, []int{0}, []Receipt{{Sequence: 2}})
	_ = f.r.Cleaning("w1", v.ID, 0)
	_ = f.r.Cleaned("w1", v.ID, 0, []Cleanup{{Item: "B2", State: "done"}})
	span, _ = f.r.Span("w1", v.ID, 0, 0)
	span.Items[0].Cleanup[0].Item = "changed"
	if again, _ := f.r.Span("w1", v.ID, 0, 0); again.Items[0].Cleanup[0].Item != "B2" {
		t.Error("the span shares its cleanup records")
	}
}
