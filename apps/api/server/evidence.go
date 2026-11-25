package server

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/perfgate"
	"github.com/myblueton-sw/hybrid_billing/modules/releasetrace"
)

const (
	evidenceShown        = 50
	evidenceStatesShown  = 10
	evidenceReplacements = 20
	evidenceProblems     = 100
	evidencePeriods      = 12
	evidenceOriginSeed   = "sample_seed"
	evidenceOriginEmpty  = "empty"
	evidenceCheckText    = "Release evidence check"
	noReplacement        = "no_replacement"
	outcomePass          = "pass"
	outcomeFail          = "fail"
)

type evidenceList[T any] struct {
	Total int  `json:"total"`
	More  bool `json:"more"`
	Rows  []T  `json:"rows"`
}

func listOf[T any](total, limit int, row func(index int) (T, error)) (evidenceList[T], error) {
	out := evidenceList[T]{Total: total, Rows: []T{}}
	for index := 0; index < total && index < limit; index++ {
		r, err := row(index)
		if err != nil {
			return evidenceList[T]{}, err
		}
		out.Rows = append(out.Rows, r)
	}
	out.More = total > len(out.Rows)
	return out, nil
}

func capped[T any](all []T, limit int) evidenceList[T] {
	out, _ := listOf(len(all), limit, func(index int) (T, error) { return all[index], nil })
	return out
}

type evidencePerson struct {
	ID        string  `json:"id"`
	Login     *string `json:"login"`
	Directory bool    `json:"directory"`
}

type evidenceCandidate struct {
	Version string `json:"version"`
	Build   string `json:"build"`
	Release string `json:"release"`
}

type evidenceCount struct {
	Count int `json:"count"`
	Of    int `json:"of"`
}

type evidenceColumns struct {
	Design         evidenceCount  `json:"design"`
	Implementation evidenceCount  `json:"implementation"`
	Test           *evidenceCount `json:"test"`
	CustomerProof  *evidenceCount `json:"customer_proof"`
	Failing        *int           `json:"failing"`
}

type evidenceRequirementRef struct {
	Label  string  `json:"label"`
	Target *string `json:"target"`
	State  *string `json:"state"`
}

type evidenceReconciliation struct {
	Period      string `json:"period"`
	Matched     bool   `json:"matched"`
	Differences int    `json:"differences"`
}

type evidenceProblem struct {
	Problem string `json:"problem"`
	Subject string `json:"subject"`
}

type evidencePathRow struct {
	ID                  string                               `json:"id"`
	Provider            string                               `json:"provider"`
	Contract            string                               `json:"contract"`
	Role                string                               `json:"role"`
	Version             string                               `json:"version"`
	Issuance            string                               `json:"issuance"`
	Template            *string                              `json:"template"`
	State               string                               `json:"state"`
	Build               *string                              `json:"build"`
	Requirements        evidenceList[evidenceRequirementRef] `json:"requirements"`
	Columns             evidenceColumns                      `json:"columns"`
	Categories          map[string]bool                      `json:"categories"`
	Reconciliations     evidenceList[evidenceReconciliation] `json:"reconciliations"`
	LastObserved        *string                              `json:"last_observed"`
	Blockers            []evidenceProblem                    `json:"blockers"`
	Activation          string                               `json:"activation"`
	SupportDecision     string                               `json:"support_decision"`
	CustomerProofIntake string                               `json:"customer_proof_intake"`
}

type evidenceMatrix struct {
	Origin string                        `json:"origin"`
	Paths  evidenceList[evidencePathRow] `json:"paths"`
}

type evidenceTarget struct {
	Target string `json:"target"`
	Label  string `json:"label"`
}

type evidenceTestRow struct {
	ID               string  `json:"id"`
	Category         *string `json:"category"`
	Counted          bool    `json:"counted"`
	LatestOutcome    *string `json:"latest_outcome"`
	Passes           int     `json:"passes"`
	Failures         int     `json:"failures"`
	NotRunReason     *string `json:"not_run_reason"`
	NotCountedReason *string `json:"not_counted_reason"`
}

type evidenceRequirementRow struct {
	Target              string                         `json:"target"`
	Label               string                         `json:"label"`
	Namespace           string                         `json:"namespace"`
	ID                  string                         `json:"id"`
	Source              string                         `json:"source"`
	Owner               string                         `json:"owner"`
	Interfaces          []string                       `json:"interfaces"`
	Common              bool                           `json:"common"`
	Blocking            bool                           `json:"blocking"`
	Design              string                         `json:"design"`
	Commit              *string                        `json:"commit"`
	Retired             bool                           `json:"retired"`
	State               *string                        `json:"state"`
	Replacement         *string                        `json:"replacement"`
	States              map[string]string              `json:"states"`
	BuildsMore          bool                           `json:"builds_more"`
	Tests               *evidenceList[evidenceTestRow] `json:"tests"`
	Retire              *uint64                        `json:"retire"`
	Replacements        evidenceList[evidenceTarget]   `json:"replacements"`
	ReplacementRequired bool                           `json:"replacement_required"`
	CanRetire           bool                           `json:"can_retire"`
	RetireBlocked       string                         `json:"retire_blocked"`
}

type evidenceGate struct {
	Ready           bool                          `json:"ready"`
	Problems        evidenceList[evidenceProblem] `json:"problems"`
	RetiredBlocking []string                      `json:"retired_blocking"`
	Templates       []string                      `json:"templates"`
	Release         string                        `json:"release"`
}

type evidenceRequirements struct {
	Origin       string                               `json:"origin"`
	Requirements evidenceList[evidenceRequirementRow] `json:"requirements"`
	Next         string                               `json:"next"`
	Gate         *evidenceGate                        `json:"gate"`
}

type evidenceBuildRow struct {
	Build  string `json:"build"`
	Commit string `json:"commit"`
	Order  int    `json:"order"`
}

type evidenceHarnessRow struct {
	Module     string `json:"module"`
	ID         string `json:"id"`
	Revocation string `json:"revocation"`
}

type evidenceResultRow struct {
	Number           uint64  `json:"number"`
	Namespace        string  `json:"namespace"`
	Test             string  `json:"test"`
	Build            string  `json:"build"`
	Counted          bool    `json:"counted"`
	LatestOutcome    string  `json:"latest_outcome"`
	Evidence         string  `json:"evidence"`
	Harness          string  `json:"harness"`
	At               string  `json:"at"`
	Passes           int     `json:"passes"`
	Failures         int     `json:"failures"`
	NotCountedReason *string `json:"not_counted_reason"`
}

type evidenceArtifacts struct {
	Origin      string                           `json:"origin"`
	Builds      evidenceList[evidenceBuildRow]   `json:"builds"`
	Harnesses   evidenceList[evidenceHarnessRow] `json:"harnesses"`
	Results     evidenceList[evidenceResultRow]  `json:"results"`
	ResultsNext string                           `json:"results_next"`
	Files       string                           `json:"files"`
	Logs        string                           `json:"logs"`
	Environment string                           `json:"environment"`
	Content     string                           `json:"content"`
}

type evidenceProposalRow struct {
	Module           string          `json:"module"`
	Number           uint64          `json:"number"`
	Target           string          `json:"target"`
	Kind             string          `json:"kind"`
	State            string          `json:"state"`
	Created          string          `json:"created_at"`
	Proposer         *evidencePerson `json:"proposer"`
	Approver         *evidencePerson `json:"approver"`
	Rationale        string          `json:"rationale"`
	ClosedBy         *evidencePerson `json:"closed_by"`
	ClosingRationale string          `json:"closing_rationale"`
	Summary          map[string]any  `json:"summary"`
	CanApprove       bool            `json:"can_approve"`
	ApproveBlocked   string          `json:"approve_blocked"`
	CanWithdraw      bool            `json:"can_withdraw"`
	WithdrawBlocked  string          `json:"withdraw_blocked"`
}

type evidenceDecisions struct {
	Origin           string                            `json:"origin"`
	Proposals        evidenceList[evidenceProposalRow] `json:"proposals"`
	ProposalHistory  keptFacts                         `json:"proposal_history"`
	SupportDecisions string                            `json:"support_decisions"`
	ActivationReview string                            `json:"activation_review"`
	Ledger           string                            `json:"ledger"`
}

type evidencePage struct {
	Revision     string               `json:"revision"`
	Now          string               `json:"now"`
	State        string               `json:"state"`
	Candidate    *evidenceCandidate   `json:"candidate"`
	Matrix       evidenceMatrix       `json:"matrix"`
	Requirements evidenceRequirements `json:"requirements"`
	Artifacts    evidenceArtifacts    `json:"artifacts"`
	Load         evidenceLoad         `json:"load"`
	Decisions    evidenceDecisions    `json:"decisions"`
}

type evidenceFacts struct {
	s         *Server
	reader    string
	eligible  bool
	origin    string
	candidate *releasetrace.Release
	build     string
	replacing map[releasetrace.Key]bool
	numbers   map[releasetrace.Key]int
}

func (s *Server) evidencePerson(id string) *evidencePerson {
	if id == "" {
		return nil
	}
	p := &evidencePerson{ID: id}
	if login, ok := s.dir.loginOf(id); ok {
		p.Login, p.Directory = &login, true
	}
	return p
}

func optional(text string) *string {
	if text == "" {
		return nil
	}
	return &text
}

func label(k releasetrace.Key) string { return k.Namespace + "/" + k.ID }

func requirementTarget(index int) string { return traceRequirementTarget + strconv.Itoa(index) }

func traceBlocker(err error) (string, error) {
	var refusal releasetrace.Refusal
	switch {
	case err == nil:
		return "", nil
	case errors.As(err, &refusal):
		return refusal.Reason, nil
	case errors.Is(err, releasetrace.ErrNotFound):
		return "not_found", nil
	}
	return "", err
}

func (f *evidenceFacts) check(c releasetrace.Change, proposer, approver string) (string, error) {
	s := f.s
	return traceBlocker(s.traceCall("check", func() error { return s.trace.Check(c, proposer, approver) }))
}

func tested(state string) bool {
	return state == releasetrace.StateTested || state == releasetrace.StateCustomerProven
}

func (f *evidenceFacts) targetOf(k releasetrace.Key) *string {
	if index := f.s.ev.requirementIndex(k); index > 0 {
		target := requirementTarget(index)
		return &target
	}
	return nil
}

func belongs(subject, path string, labels []string) bool {
	for _, owner := range append([]string{path}, labels...) {
		if subject == owner || strings.HasPrefix(subject, owner+">") {
			return true
		}
	}
	return false
}

func (f *evidenceFacts) lastObserved(views []releasetrace.RequirementView) (*string, error) {
	tests := map[[2]string]bool{}
	for _, q := range views {
		for _, t := range q.Requirement.Tests {
			tests[[2]string{q.Requirement.Namespace, t.ID}] = true
		}
	}
	var latest time.Time
	for _, k := range f.s.ev.results {
		if k.build != f.build || !tests[[2]string{k.namespace, k.test}] {
			continue
		}
		v, err := f.s.traceResults(k)
		if err != nil {
			return nil, err
		}
		if v.At.After(latest) {
			latest = v.At
		}
	}
	if latest.IsZero() {
		return nil, nil
	}
	return optional(stamp(latest)), nil
}

func (f *evidenceFacts) pathRow(id string) (evidencePathRow, error) {
	s := f.s
	v, err := s.tracePath(id)
	if err != nil {
		return evidencePathRow{}, err
	}
	p := v.Path
	row := evidencePathRow{ID: p.ID, Provider: p.Provider, Contract: p.Contract, Role: p.Role, Version: p.Version, Issuance: p.Issuance, Template: optional(p.Template),
		State: v.State, Build: optional(v.Build), Activation: notConnected, SupportDecision: notConnected, CustomerProofIntake: notConnected}
	categories := map[string]bool{}
	for _, category := range releasetrace.Categories {
		categories[category] = false
	}
	views, labels := []releasetrace.RequirementView{}, []string{}
	of := len(p.Requirements)
	testCount, proofCount, failing := evidenceCount{Of: of}, evidenceCount{Of: of}, 0
	row.Columns = evidenceColumns{Design: evidenceCount{Of: of}, Implementation: evidenceCount{Of: of}}
	for _, k := range p.Requirements {
		q, err := s.traceRequirement(k, f.build)
		if err != nil {
			return evidencePathRow{}, err
		}
		views, labels = append(views, q), append(labels, label(k))
		row.Columns.Design.Count++
		if q.Commit != "" {
			row.Columns.Implementation.Count++
		}
		if tested(q.State) {
			testCount.Count++
		}
		if q.State == releasetrace.StateCustomerProven {
			proofCount.Count++
		}
		if q.State == releasetrace.StateFailing {
			failing++
		}
		for index, t := range q.Requirement.Tests {
			if x := q.Tests[index]; t.Category != "" && x.Counted && x.Passed {
				categories[t.Category] = true
			}
		}
	}
	row.Requirements, _ = listOf(len(views), evidenceShown, func(index int) (evidenceRequirementRef, error) {
		ref := evidenceRequirementRef{Label: labels[index], Target: f.targetOf(p.Requirements[index])}
		if f.candidate != nil {
			ref.State = optional(views[index].State)
		}
		return ref, nil
	})
	recon := v.Reconciliations
	row.Reconciliations, _ = listOf(len(recon), evidencePeriods, func(index int) (evidenceReconciliation, error) {
		x := recon[len(recon)-1-index]
		return evidenceReconciliation{Period: x.Period, Matched: x.Matched, Differences: x.Differences}, nil
	})
	if f.candidate == nil {
		return row, nil
	}
	row.Columns.Test, row.Columns.CustomerProof, row.Columns.Failing, row.Categories = &testCount, &proofCount, &failing, categories
	if row.LastObserved, err = f.lastObserved(views); err != nil {
		return evidencePathRow{}, err
	}
	g, err := s.traceGate(releasetrace.Release{Version: f.candidate.Version, Build: f.build, Requirements: p.Requirements, Paths: []string{id}})
	if err != nil {
		return evidencePathRow{}, err
	}
	row.Blockers = []evidenceProblem{}
	for _, x := range g.Problems {
		if belongs(x.Subject, id, labels) {
			row.Blockers = append(row.Blockers, evidenceProblem{Problem: x.Problem, Subject: x.Subject})
		}
	}
	return row, nil
}

func (f *evidenceFacts) matrix() (evidenceMatrix, error) {
	paths := f.s.ev.paths
	list, err := listOf(len(paths), evidenceShown, func(index int) (evidencePathRow, error) { return f.pathRow(paths[index]) })
	return evidenceMatrix{Origin: f.origin, Paths: list}, err
}

func (f *evidenceFacts) notCounted(k releasetrace.Key, q releasetrace.RequirementView, build string) *string {
	if q.Commit == "" {
		return optional(noImplementation)
	}
	return optional(f.s.ev.notCounted(k, build))
}

func (f *evidenceFacts) testRows(k releasetrace.Key, q releasetrace.RequirementView) *evidenceList[evidenceTestRow] {
	list, _ := listOf(len(q.Tests), evidenceShown, func(index int) (evidenceTestRow, error) {
		x := q.Tests[index]
		row := evidenceTestRow{ID: x.Test, Category: optional(q.Requirement.Tests[index].Category), Counted: x.Counted, Passes: x.Passes, Failures: x.Failures}
		if x.Passes+x.Failures > 0 && !x.Counted {
			row.NotCountedReason = f.notCounted(k, q, f.build)
		}
		switch {
		case x.Passes+x.Failures == 0:
			row.NotRunReason = optional(noResultRecorded)
		case x.Passed:
			row.LatestOutcome = optional(outcomePass)
		default:
			row.LatestOutcome = optional(outcomeFail)
		}
		return row, nil
	})
	return &list
}

func retireCheck(k, replacement releasetrace.Key) releasetrace.Change {
	return releasetrace.Change{Kind: releasetrace.ChangeRetire, Retire: releasetrace.Retire{Key: k, Replacement: replacement, Reason: evidenceCheckText}}
}

func (f *evidenceFacts) replacements(index int) (evidenceList[evidenceTarget], error) {
	stored := f.s.ev.requirements
	if f.replacing == nil {
		f.replacing = f.s.ev.pendingReplacements()
	}
	if f.numbers == nil {
		f.numbers = make(map[releasetrace.Key]int, len(stored))
		for m, k := range stored {
			f.numbers[k] = m + 1
		}
	}
	var accepted []releasetrace.Key
	err := f.s.traceCall("replacements", func() (err error) { accepted, err = f.s.trace.Replacements(stored[index]); return })
	if err != nil && !errors.Is(err, releasetrace.ErrNotFound) {
		return evidenceList[evidenceTarget]{}, err
	}
	numbers := []int{}
	for _, other := range accepted {
		if n := f.numbers[other]; n > 0 && !f.replacing[other] {
			numbers = append(numbers, n)
		}
	}
	sort.Ints(numbers)
	return listOf(len(numbers), evidenceReplacements, func(at int) (evidenceTarget, error) {
		return evidenceTarget{Target: requirementTarget(numbers[at]), Label: label(stored[numbers[at]-1])}, nil
	})
}

func (f *evidenceFacts) retireBlocked(row *evidenceRequirementRow, q releasetrace.RequirementView, k releasetrace.Key) (string, error) {
	switch {
	case q.Retired:
		return releasetrace.ReasonRetired, nil
	case row.Retire != nil:
		return alreadyProposed, nil
	case !f.eligible:
		return noEligibleApprover, nil
	}
	reason, err := f.check(retireCheck(k, releasetrace.Key{}), f.reader, "")
	if err != nil || reason != releasetrace.ReasonCommonNeedsReplacement {
		return reason, err
	}
	row.ReplacementRequired = true
	if row.Replacements.Total == 0 {
		return noReplacement, nil
	}
	return "", nil
}

func (f *evidenceFacts) requirementRow(index int) (evidenceRequirementRow, error) {
	s := f.s
	k := s.ev.requirements[index]
	q, err := s.traceRequirement(k, f.build)
	if err != nil {
		return evidenceRequirementRow{}, err
	}
	r := q.Requirement
	row := evidenceRequirementRow{Target: requirementTarget(index + 1), Label: label(k), Namespace: r.Namespace, ID: r.ID, Source: r.Source, Owner: r.Owner,
		Interfaces: append([]string{}, r.Interfaces...), Common: r.Common, Blocking: r.Blocking, Design: r.Design, Commit: optional(q.Commit), Retired: q.Retired,
		States: map[string]string{}}
	if q.Replacement != (releasetrace.Key{}) {
		row.Replacement = optional(label(q.Replacement))
	}
	if f.candidate != nil {
		row.State = optional(q.State)
		builds := s.ev.builds
		for n := len(builds) - 1; n >= 0 && len(row.States) < evidenceStatesShown; n-- {
			v, err := s.traceRequirement(k, builds[n].id)
			if err != nil {
				return evidenceRequirementRow{}, err
			}
			row.States[builds[n].id] = v.State
		}
		row.BuildsMore = len(builds) > len(row.States)
		row.Tests = f.testRows(k, q)
	}
	if e := s.ev.pendingRetire(k); e != nil {
		number := e.number
		row.Retire = &number
	}
	if row.Replacements, err = f.replacements(index); err != nil {
		return evidenceRequirementRow{}, err
	}
	if row.RetireBlocked, err = f.retireBlocked(&row, q, k); err != nil {
		return evidenceRequirementRow{}, err
	}
	row.CanRetire = row.RetireBlocked == ""
	return row, nil
}

func (f *evidenceFacts) requirements() (evidenceRequirements, error) {
	s := f.s
	section := evidenceRequirements{Origin: f.origin}
	var err error
	if section.Requirements, err = listOf(len(s.ev.requirements), evidenceShown, f.requirementRow); err != nil {
		return section, err
	}
	if section.Requirements.More {
		section.Next = strconv.Itoa(len(section.Requirements.Rows))
	}
	if f.candidate == nil {
		return section, nil
	}
	g, err := s.traceGate(*f.candidate)
	if err != nil {
		return evidenceRequirements{}, err
	}
	gate := &evidenceGate{Ready: g.Ready, RetiredBlocking: []string{}, Templates: append([]string{}, g.Templates...), Release: notConnected}
	gate.Problems, _ = listOf(len(g.Problems), evidenceProblems, func(index int) (evidenceProblem, error) {
		return evidenceProblem{Problem: g.Problems[index].Problem, Subject: g.Problems[index].Subject}, nil
	})
	for _, k := range g.RetiredBlocking {
		gate.RetiredBlocking = append(gate.RetiredBlocking, label(k))
	}
	section.Gate = gate
	return section, nil
}

func (f *evidenceFacts) artifacts() (evidenceArtifacts, error) {
	s := f.s
	section := evidenceArtifacts{Origin: f.origin, Files: notConnected, Logs: notConnected, Environment: notConnected, Content: notConnected}
	builds := s.ev.builds
	section.Builds, _ = listOf(len(builds), evidenceShown, func(index int) (evidenceBuildRow, error) {
		n := len(builds) - 1 - index
		return evidenceBuildRow{Build: builds[n].id, Commit: builds[n].commit, Order: n + 1}, nil
	})
	harnesses := s.ev.harnesses
	section.Harnesses, _ = listOf(len(harnesses), evidenceShown, func(index int) (evidenceHarnessRow, error) {
		return evidenceHarnessRow{Module: harnesses[index].module, ID: harnesses[index].id, Revocation: notConnected}, nil
	})
	section.Results = evidenceList[evidenceResultRow]{Total: len(s.ev.results), Rows: []evidenceResultRow{}}
	for n := len(s.ev.results) - 1; n >= 0 && len(section.Results.Rows) < evidenceShown; n-- {
		row, err := f.resultRow(s.ev.results[n])
		if err != nil {
			return evidenceArtifacts{}, err
		}
		section.Results.Rows = append(section.Results.Rows, row)
	}
	if shown := len(section.Results.Rows); shown < section.Results.Total {
		section.Results.More, section.ResultsNext = true, keptCursor(section.Results.Rows[shown-1].Number)
	}
	return section, nil
}

func traceSummary(c releasetrace.Change) map[string]any {
	switch c.Kind {
	case releasetrace.ChangeRequirement:
		return map[string]any{"label": label(releasetrace.Key{Namespace: c.Requirement.Namespace, ID: c.Requirement.ID})}
	case releasetrace.ChangeRetire:
		var replacement *string
		if c.Retire.Replacement != (releasetrace.Key{}) {
			replacement = optional(label(c.Retire.Replacement))
		}
		return map[string]any{"label": label(c.Retire.Key), "replacement": replacement, "reason": c.Retire.Reason}
	case releasetrace.ChangeImplemented:
		return map[string]any{"label": label(c.Implemented.Key), "commit": c.Implemented.Commit}
	case releasetrace.ChangeBuild:
		return map[string]any{"build": c.Build.Build, "commit": c.Build.Commit}
	case releasetrace.ChangeHarness:
		return map[string]any{"harness": c.Harness.ID}
	case releasetrace.ChangePath:
		return map[string]any{"path": c.Path.ID}
	case releasetrace.ChangeReconcile:
		x := c.Reconcile
		return map[string]any{"path": x.Path, "period": x.Period, "matched": x.Matched, "differences": x.Differences}
	}
	switch {
	case c.Activate.Path != "":
		return map[string]any{"path": c.Activate.Path, "build": c.Activate.Build}
	case c.Proof.Key != (releasetrace.Key{}):
		return map[string]any{"label": label(c.Proof.Key), "build": c.Proof.Build, "period": c.Proof.Period}
	}
	return map[string]any{}
}

func (f *evidenceFacts) blockers(e *evidenceEntry) (string, string, error) {
	if e.state != entryPending {
		blocked := releasetrace.ReasonNotPending
		if e.state == storeRetired {
			blocked = storeRetired
		}
		return blocked, blocked, nil
	}
	withdraw := ""
	if e.proposer != f.reader {
		withdraw = releasetrace.ReasonNotProposer
	}
	switch {
	case e.kind != releasetrace.ChangeRetire:
		return notConsoleKind, withdraw, nil
	case e.proposer == f.reader:
		return ownProposal, withdraw, nil
	case e.module == moduleTrace:
		approve, err := f.check(e.trace, e.proposer, f.reader)
		return approve, withdraw, err
	}
	_, retired, err := f.s.perfProfile(e.perf.Retire.Profile)
	switch {
	case err != nil:
		return "", "", err
	case retired:
		return perfgate.ReasonRetired, withdraw, nil
	}
	return "", withdraw, nil
}

func evidenceProposalTarget(module string, number uint64) string {
	prefix := traceProposalTarget
	if module == modulePerf {
		prefix = perfProposalTarget
	}
	return prefix + strconv.FormatUint(number, 10)
}

func (f *evidenceFacts) proposalRow(e *evidenceEntry) (evidenceProposalRow, error) {
	s := f.s
	row := evidenceProposalRow{Module: e.module, Number: e.number, Target: evidenceProposalTarget(e.module, e.number), Kind: e.kind, State: e.state, Created: stamp(e.created),
		Proposer: s.evidencePerson(e.proposer), Approver: s.evidencePerson(e.approver), Rationale: e.rationale, ClosedBy: s.evidencePerson(e.closedBy),
		ClosingRationale: e.closing}
	if e.module == moduleTrace {
		row.Summary = traceSummary(e.trace)
	} else {
		row.Summary = perfSummary(e.perf)
	}
	var err error
	if row.ApproveBlocked, row.WithdrawBlocked, err = f.blockers(e); err != nil {
		return evidenceProposalRow{}, err
	}
	row.CanApprove, row.CanWithdraw = row.ApproveBlocked == "", row.WithdrawBlocked == ""
	return row, nil
}

func (f *evidenceFacts) decisions() (evidenceDecisions, error) {
	entries := f.s.ev.entries
	listed, closed, next := listKept(entries, evidencePending, evidenceSequence, evidenceShown)
	list, err := listOf(len(listed), len(listed), func(index int) (evidenceProposalRow, error) { return f.proposalRow(listed[index]) })
	list.Total, list.More = len(entries), len(entries) > len(listed)
	return evidenceDecisions{Origin: f.origin, Proposals: list, ProposalHistory: keptFacts{Total: closed, Next: next, Kept: installKept, Dropped: f.s.ev.dropped},
		SupportDecisions: notConnected, ActivationReview: notConnected, Ledger: retentionLedger}, err
}

func (s *Server) evidenceFactsFor(verifier identity.Verifier, reader string, now time.Time) (*evidenceFacts, error) {
	approvers, err := s.retentionApprovers(verifier, now)
	if err != nil {
		return nil, err
	}
	f := &evidenceFacts{s: s, reader: reader, origin: evidenceOriginEmpty}
	for _, user := range approvers {
		f.eligible = f.eligible || user != reader
	}
	if s.ev.seeded {
		f.origin = evidenceOriginSeed
	}
	if c := s.ev.candidate; c != nil {
		f.candidate, f.build = c, c.Build
	}
	return f, nil
}

func (s *Server) evidencePage(verifier identity.Verifier, reader string, now time.Time) (evidencePage, error) {
	if s.faulty("release-evidence-read") {
		return evidencePage{}, releasetrace.ErrInvalid
	}
	f, err := s.evidenceFactsFor(verifier, reader, now)
	if err != nil {
		return evidencePage{}, err
	}
	page := evidencePage{Revision: s.Revision(), Now: stamp(now), State: stateConfigured}
	if c := f.candidate; c != nil {
		page.Candidate = &evidenceCandidate{Version: c.Version, Build: c.Build, Release: notConnected}
	}
	if page.Matrix, err = f.matrix(); err != nil {
		return evidencePage{}, err
	}
	if page.Requirements, err = f.requirements(); err != nil {
		return evidencePage{}, err
	}
	if page.Artifacts, err = f.artifacts(); err != nil {
		return evidencePage{}, err
	}
	if page.Load, err = f.load(); err != nil {
		return evidencePage{}, err
	}
	page.Decisions, err = f.decisions()
	return page, err
}

func (s *Server) readEvidence(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	now := s.clock()
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, installationEmpty{Revision: s.Revision(), Now: stamp(now), State: stateNotConfigured})
		return
	}
	page, err := s.evidencePage(found.verifier, answer.User(), now)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	writeJSON(w, http.StatusOK, request, page)
}

func (s *Server) resultNotCounted(k evidenceResultKey) (string, error) {
	for _, owner := range s.ev.testOwners(k) {
		view, err := s.traceRequirement(owner, k.build)
		if err != nil {
			return "", err
		}
		reason := noImplementation
		if view.Commit != "" {
			reason = s.ev.notCounted(owner, k.build)
		}
		if reason != notCountedByModule {
			return reason, nil
		}
	}
	return notCountedByModule, nil
}
