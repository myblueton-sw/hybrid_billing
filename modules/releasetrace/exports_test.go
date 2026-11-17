package releasetrace

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
)

func snapshot(r *Registry) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out strings.Builder
	fmt.Fprint(&out, r.counter, r.seq, r.pending, r.done, r.identities)
	keys := []Key{}
	for k := range r.requirements {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool { return keyLess(keys[a], keys[b]) })
	for _, k := range keys {
		q := r.requirements[k]
		fmt.Fprint(&out, "|", k, q.view, q.by, q.retired, q.replacement, q.replaces, q.reason, q.impls, q.seq)
		for build, p := range r.proofs[k] {
			fmt.Fprint(&out, build, p.view, p.by)
		}
	}
	builds := []string{}
	for id := range r.builds {
		builds = append(builds, id)
	}
	sort.Strings(builds)
	for _, id := range builds {
		fmt.Fprint(&out, "#", id, *r.builds[id])
	}
	harnesses := []string{}
	for id := range r.harnesses {
		harnesses = append(harnesses, id)
	}
	sort.Strings(harnesses)
	for _, id := range harnesses {
		fmt.Fprint(&out, "&", id, *r.harnesses[id])
	}
	results := []string{}
	for key, res := range r.results {
		results = append(results, fmt.Sprint(key, res.latest, res.harness.id, res.at, res.passes, res.failures))
	}
	sort.Strings(results)
	out.WriteString(strings.Join(results, ","))
	paths := []string{}
	for id := range r.paths {
		paths = append(paths, id)
	}
	sort.Strings(paths)
	for _, id := range paths {
		p := r.paths[id]
		fmt.Fprint(&out, "~", id, p.view, p.active, p.build, p.proof, p.recon, p.reconBy)
	}
	numbers := []uint64{}
	for n := range r.proposals {
		numbers = append(numbers, n)
	}
	sort.Slice(numbers, func(a, b int) bool { return numbers[a] < numbers[b] })
	for _, n := range numbers {
		p := r.proposals[n]
		fmt.Fprint(&out, "@", n, p.view, p.code)
	}
	return out.String()
}

func refused(reason string) error { return Refusal{Reason: reason} }

func (f world) agree(name string, c Change, proposer Actor, want error) {
	f.t.Helper()
	before := snapshot(f.r)
	got := f.r.Check(c, proposer.ID, "")
	if after := snapshot(f.r); after != before {
		f.t.Errorf("%s: check changed state", name)
	}
	p, proposed := f.r.Propose(c, proposer)
	if fmt.Sprint(got) != fmt.Sprint(proposed) || fmt.Sprint(got) != fmt.Sprint(want) {
		f.t.Errorf("%s: check %v propose %v want %v", name, got, proposed, want)
	}
	if proposed == nil {
		f.must(second(f.r.Withdraw(p.Number, proposer)))
	}
}

func (f world) agreeApprove(name string, c Change, proposer, approver Actor, want error) {
	f.t.Helper()
	p, err := f.r.Propose(c, proposer)
	f.must(err)
	before := snapshot(f.r)
	got := f.r.Check(p.Change, p.Proposer, approver.ID)
	if after := snapshot(f.r); after != before {
		f.t.Errorf("%s: check changed state", name)
	}
	_, approved := f.r.Approve(p.Number, approver)
	if fmt.Sprint(got) != fmt.Sprint(approved) || fmt.Sprint(got) != fmt.Sprint(want) {
		f.t.Errorf("%s: check %v approve %v want %v", name, got, approved, want)
	}
	if approved != nil {
		f.must(second(f.r.Withdraw(p.Number, proposer)))
	}
}

func retireOf(id, replacement string) Change {
	x := Retire{Key: k(id), Reason: "scope"}
	if replacement != "" {
		x.Replacement = k(replacement)
	}
	return Change{Kind: ChangeRetire, Retire: x}
}

func proofOf(id, build string) Change {
	return Change{Kind: ChangeProof, Proof: Proof{Key: k(id), Build: build, Environment: "env", Customer: "cust", Period: "2026-10", Evidence: sum("proof")}}
}

func activateOf(path, build string) Change {
	return Change{Kind: ChangeActivate, Activate: Activate{Path: path, Build: build, Proof: sum("proof")}}
}

func reconcileOf(path, period string) Change {
	return Change{Kind: ChangeReconcile, Reconcile: Reconcile{Path: path, Period: period, Evidence: sum(path + period), Matched: true}}
}

func TestValidTextAgreesWithRetire(t *testing.T) {
	f := standard(t)
	cases := []string{"", "a", strings.Repeat("a", maxTextBytes), strings.Repeat("a", maxTextBytes+1), strings.Repeat("a", maxTextBytes-1) + "é",
		strings.Repeat("a", maxTextBytes-2) + "é", "a\x1f", "a\x7f", "a\u0085", "a\u009f", "a ", "\xff", "tab\t", "line\n", " ", "한국어", "العربية"}
	for _, text := range cases {
		p, err := f.r.Propose(Change{Kind: ChangeRetire, Retire: Retire{Key: k("R1"), Replacement: k("R1x"), Reason: text}}, ann)
		if ValidText(text) != (err != ErrInvalid) {
			t.Errorf("%q: valid %v, propose %v", text, ValidText(text), err)
		}
		if err == nil {
			f.must(second(f.r.Withdraw(p.Number, ann)))
		}
	}
}

func TestCheckMatchesPropose(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("N1", false, TestRef{ID: "n1"})})
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("C2", true, TestRef{ID: "c2"})})
	b := req("B1", false, TestRef{ID: "b1"})
	b.Blocking = true
	f.approve(Change{Kind: ChangeRequirement, Requirement: b})
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("N2", false, TestRef{ID: "n2"})})
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("C3", true, TestRef{ID: "c3"})})
	f.agree("malformed", Change{Kind: "other"}, ann, ErrInvalid)
	f.agree("malformed retire", Change{Kind: ChangeRetire, Retire: Retire{Key: k("N1")}}, ann, ErrInvalid)
	f.agree("self replacement", Change{Kind: ChangeRetire, Retire: Retire{Key: k("N1"), Replacement: k("N1"), Reason: "x"}}, ann, ErrInvalid)
	f.agree("requirement", Change{Kind: ChangeRequirement, Requirement: req("R9", false, TestRef{ID: "r9"})}, ann, nil)
	f.agree("requirement taken", Change{Kind: ChangeRequirement, Requirement: req("R1", false, TestRef{ID: "r1"})}, ann, ErrConflict)
	f.agree("retire unknown", retireOf("Z", ""), ann, ErrNotFound)
	f.agree("retire unknown replacement", retireOf("N1", "Z"), ann, ErrNotFound)
	f.agree("retire", retireOf("N1", ""), ann, nil)
	f.agree("retire common", retireOf("C2", ""), ann, refused(ReasonCommonNeedsReplacement))
	f.agree("retire weaker common", retireOf("C2", "N2"), ann, refused(ReasonWeakerReplacement))
	f.agree("retire weaker blocking", retireOf("B1", "N2"), ann, refused(ReasonWeakerReplacement))
	f.agree("retire older", retireOf("N2", "N1"), ann, refused(ReasonWeakerReplacement))
	f.agree("retire replaced", retireOf("C2", "C3"), ann, nil)
	f.approve(retireOf("C2", "C3"))
	f.agree("retire retired", retireOf("C2", ""), ann, refused(ReasonRetired))
	f.agree("retire used replacement", retireOf("R1", "C3"), ann, refused(ReasonWeakerReplacement))
	f.agree("implemented", Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("N1"), Commit: sum("i")[:40]}}, ann, nil)
	f.agree("implemented unknown", Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("Z"), Commit: sum("i")[:40]}}, ann, ErrNotFound)
	f.agree("implemented retired", Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("C2"), Commit: sum("i")[:40]}}, ann, refused(ReasonRetired))
	f.agree("build", Change{Kind: ChangeBuild, Build: Build{Build: "2.0.0", Commit: sum("b2")[:40]}}, ann, nil)
	f.agree("build taken", Change{Kind: ChangeBuild, Build: Build{Build: "1.0.0", Commit: sum("b2")[:40]}}, ann, ErrConflict)
	f.agree("harness", Change{Kind: ChangeHarness, Harness: Harness{ID: "h2", Identity: "r2"}}, ann, nil)
	f.agree("harness taken", Change{Kind: ChangeHarness, Harness: Harness{ID: "h1", Identity: "r2"}}, ann, ErrConflict)
	f.agree("identity taken", Change{Kind: ChangeHarness, Harness: Harness{ID: "h2", Identity: runner.ID}}, ann, refused(ReasonIdentityTaken))
	f.agree("proof", proofOf("R1", "1.0.0"), ann, nil)
	f.agree("proof unknown", proofOf("Z", "1.0.0"), ann, ErrNotFound)
	f.agree("proof unknown build", proofOf("R1", "9.9.9"), ann, refused(ReasonUnknownBuild))
	f.agree("proof untested", proofOf("N2", "1.0.0"), ann, refused(ReasonNotTested))
	f.approve(proofOf("R1", "1.0.0"))
	f.agree("proof repeat", proofOf("R1", "1.0.0"), ann, ErrConflict)
	path := Path{ID: "p2", Provider: "aws", Contract: "msp", Role: "payer", Version: "v1", Issuance: IssuanceAPI, Requirements: []Key{k("R1")}}
	f.agree("path", Change{Kind: ChangePath, Path: path}, ann, nil)
	path.Requirements = []Key{k("Z")}
	f.agree("path unknown requirement", Change{Kind: ChangePath, Path: path}, ann, ErrNotFound)
	path.ID, path.Requirements = "p1", []Key{k("R1")}
	f.agree("path taken", Change{Kind: ChangePath, Path: path}, ann, ErrConflict)
	f.agree("reconcile", reconcileOf("p1", "2026-09"), ann, nil)
	f.agree("reconcile unknown", reconcileOf("nope", "2026-09"), ann, ErrNotFound)
	f.agree("reconcile future", reconcileOf("p1", "2026-11"), ann, refused(ReasonFuturePeriod))
	f.reconcile("p1", "2026-09", true)
	f.agree("reconcile repeat", reconcileOf("p1", "2026-09"), ann, ErrConflict)
	f.agree("activate unknown", activateOf("nope", "1.0.0"), gus, ErrNotFound)
	f.agree("activate unknown build", activateOf("p1", "9.9.9"), gus, refused(ReasonUnknownBuild))
	f.agree("activate unreconciled", activateOf("p1", "1.0.0"), gus, refused(ReasonReconciliationMissing))
	f.reconcile("p1", "2026-10", true)
	f.agree("activate binder", activateOf("p1", "1.0.0"), cat, refused(ReasonSelfVerification))
	f.agree("activate", activateOf("p1", "1.0.0"), gus, nil)
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("R3", false, TestRef{ID: "u1", Category: CategoryAmount})})
	f.approve(Change{Kind: ChangePath, Path: Path{ID: "p3", Provider: "aws", Contract: "msp", Role: "payer", Version: "v1", Issuance: IssuanceAPI, Requirements: []Key{k("R3")}}})
	f.reconcile("p3", "2026-09", true)
	f.reconcile("p3", "2026-10", true)
	f.agree("activate category", activateOf("p3", "1.0.0"), gus, refused(ReasonCategoryMissing))
	f.approve(Change{Kind: ChangePath, Path: Path{ID: "p4", Provider: "aws", Contract: "msp", Role: "payer", Version: "v1", Issuance: IssuanceAPI, Requirements: []Key{k("R1"), k("R3")}}})
	f.reconcile("p4", "2026-09", true)
	f.reconcile("p4", "2026-10", true)
	f.agree("activate untested", activateOf("p4", "1.0.0"), gus, refused(ReasonRequirementNotTested))
	if f.r.Check(activateOf("p1", "1.0.0"), "", "") != nil || f.r.Check(retireOf("N1", ""), "", "") != nil {
		t.Error("check without actors")
	}
}

func TestCheckMatchesLimits(t *testing.T) {
	f := standard(t)
	for len(f.r.requirements) < maxRequirements {
		f.r.requirements[Key{"fill", itoa(len(f.r.requirements))}] = &requirement{}
	}
	f.agree("requirement limit", Change{Kind: ChangeRequirement, Requirement: req("R9", false, TestRef{ID: "r9"})}, ann, ErrLimit)
	for len(f.r.builds) < maxBuilds {
		f.r.builds["9.9."+itoa(len(f.r.builds))] = &build{}
	}
	f.agree("build limit", Change{Kind: ChangeBuild, Build: Build{Build: "2.0.0", Commit: sum("b2")[:40]}}, ann, ErrLimit)
	for len(f.r.harnesses) < maxHarnesses {
		f.r.harnesses["fill-"+itoa(len(f.r.harnesses))] = &harness{}
	}
	f.agree("harness limit", Change{Kind: ChangeHarness, Harness: Harness{ID: "h2", Identity: "r2"}}, ann, ErrLimit)
	for len(f.r.paths) < maxPaths {
		f.r.paths["fill-"+itoa(len(f.r.paths))] = &path{}
	}
	f.agree("path limit", Change{Kind: ChangePath, Path: Path{ID: "p2", Provider: "aws", Contract: "msp", Role: "payer", Version: "v1", Issuance: IssuanceAPI,
		Requirements: []Key{k("R1")}}}, ann, ErrLimit)
}

func TestCheckMatchesApprove(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("N1", false, TestRef{ID: "n1"})})
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("C2", true, TestRef{ID: "c2"})})
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("C3", true, TestRef{ID: "c3"})})
	f.agreeApprove("retire", retireOf("N1", ""), ann, bob, nil)
	replaced := f.r.Check(retireOf("C2", "C3"), ann.ID, bob.ID)
	if replaced != nil {
		t.Errorf("replacement %v", replaced)
	}
	pending, err := f.r.Propose(retireOf("C2", "C3"), ann)
	f.must(err)
	f.by(retireOf("R1", "C3"), cat, dan)
	before := snapshot(f.r)
	got := f.r.Check(pending.Change, pending.Proposer, bob.ID)
	if snapshot(f.r) != before {
		t.Error("check changed state")
	}
	_, approved := f.r.Approve(pending.Number, bob)
	if fmt.Sprint(got) != fmt.Sprint(approved) || fmt.Sprint(got) != fmt.Sprint(refused(ReasonWeakerReplacement)) {
		t.Errorf("replacement used meanwhile: check %v approve %v", got, approved)
	}
	f.reconcile("p1", "2026-09", true)
	f.reconcile("p1", "2026-10", true)
	f.agreeApprove("activate approver binds the harness", activateOf("p1", "1.0.0"), gus, dan, refused(ReasonSelfVerification))
	f.agreeApprove("activate approver reconciled", activateOf("p1", "1.0.0"), gus, fay, refused(ReasonSelfVerification))
	f.agreeApprove("activate", activateOf("p1", "1.0.0"), gus, hal, nil)
	f.agreeApprove("proof", proofOf("R1", "1.0.0"), ann, bob, nil)
	f.agreeApprove("build", Change{Kind: ChangeBuild, Build: Build{Build: "2.0.0", Commit: sum("b2")[:40]}}, ann, bob, nil)
}

func TestCheckIsConcurrent(t *testing.T) {
	f := standard(t)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				f.r.Check(retireOf("R1", ""), ann.ID, "")
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				p, err := f.r.Propose(Change{Kind: ChangeBuild, Build: Build{Build: "3.0." + itoa(i*100+j), Commit: sum("b")[:40]}}, ann)
				if err == nil {
					f.r.Withdraw(p.Number, ann)
				}
			}
		}()
	}
	wg.Wait()
}
