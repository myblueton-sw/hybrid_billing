package server

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

type trustItemRow struct {
	Item   string `json:"item"`
	Result string `json:"result"`
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

type validationRow struct {
	Result          string         `json:"result"`
	At              string         `json:"at"`
	EvidenceDigest  string         `json:"evidence_digest"`
	EvidenceCurrent bool           `json:"evidence_current"`
	Items           []trustItemRow `json:"items"`
}

type trustImpactRow struct {
	Members          int      `json:"members"`
	Sessions         int      `json:"sessions"`
	AffectedIDs      []string `json:"affected_ids"`
	WeakensMFA       bool     `json:"weakens_mfa"`
	SignsOutEveryone bool     `json:"signs_out_everyone"`
}

type activationRow struct {
	Epoch          epochRow `json:"epoch"`
	LedgerSequence uint64   `json:"ledger_sequence"`
	Watermark      string   `json:"watermark"`
	At             string   `json:"at"`
}

type jobRow struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}

type progressRow struct {
	Item      string `json:"item"`
	Result    string `json:"result"`
	Ended     int    `json:"ended"`
	Remaining int    `json:"remaining"`
	Reason    string `json:"reason"`
}

type invalidationRow struct {
	State string        `json:"state"`
	Items []progressRow `json:"items"`
}

type canRow struct {
	Revise   bool `json:"revise"`
	Validate bool `json:"validate"`
	Approve  bool `json:"approve"`
	Activate bool `json:"activate"`
	Reject   bool `json:"reject"`
	Cancel   bool `json:"cancel"`
}

type whyRow struct {
	Approve  string `json:"approve"`
	Activate string `json:"activate"`
}

type trustHistoryRow struct {
	State     string `json:"state"`
	By        string `json:"by"`
	At        string `json:"at"`
	Rationale string `json:"rationale"`
}

type changeRow struct {
	ID              string            `json:"id"`
	Number          uint64            `json:"number"`
	State           string            `json:"state"`
	ChangeRevision  uint64            `json:"change_revision"`
	ChangeDigest    string            `json:"change_digest"`
	OldEpoch        epochRow          `json:"old_epoch"`
	OldPolicy       uint64            `json:"old_policy"`
	OldDigest       string            `json:"old_digest"`
	NewDigest       string            `json:"new_digest"`
	Candidate       revisionRow       `json:"candidate"`
	Proposer        string            `json:"proposer"`
	Validator       string            `json:"validator"`
	Approver        string            `json:"approver"`
	Activator       string            `json:"activator"`
	SupersededBy    string            `json:"superseded_by"`
	SupersededEpoch *epochRow         `json:"superseded_epoch"`
	EarlierRun      bool              `json:"earlier_run"`
	RunStartedAt    string            `json:"run_started_at"`
	Validation      *validationRow    `json:"validation"`
	Impact          trustImpactRow    `json:"impact"`
	Activation      *activationRow    `json:"activation"`
	Job             *jobRow           `json:"job"`
	Invalidation    *invalidationRow  `json:"invalidation"`
	Can             canRow            `json:"can"`
	WhyNot          whyRow            `json:"why_not"`
	ActUntil        string            `json:"act_until"`
	History         []trustHistoryRow `json:"history"`
}

type rowContext struct {
	s                   *Server
	you                 string
	base                trustchange.Base
	current             authpolicy.Revision
	now                 time.Time
	verifier            identity.Verifier
	since               time.Time
	limit               time.Duration
	rights              rightsRow
	unavailable         bool
	activateUnavailable bool
}

func progressOf(p trustProgress, replayed bool) *invalidationRow {
	sessions := progressRow{Item: "console_sessions", Result: "done", Ended: p.ended, Remaining: p.remaining}
	state := "complete"
	switch {
	case replayed:
		sessions.Reason = "server_restarted"
	case p.remaining > 0:
		sessions.Result, state = "in_progress", "in_progress"
	}
	return &invalidationRow{State: state, Items: []progressRow{sessions,
		{Item: "password_confirmation", Result: "done", Reason: "held_in_session_record"},
		{Item: "secret_pickups", Result: "done", Reason: "dropped_with_sessions"},
		{Item: "admitted_reads", Result: "unknown", Reason: "admitted_before_activation"},
		{Item: "service_credentials", Result: "not_applicable", Reason: "no_credential_authentication"},
		{Item: "tokens_queues_caches", Result: "not_applicable", Reason: "none_exist"},
		{Item: "browser_copies", Result: "outside_scope", Reason: "cannot_recall"}}}
}

func (c rowContext) impact(v trustchange.View, f trustFacts) trustImpactRow {
	ids := f.affected
	if len(ids) > maxAffectedShown {
		ids = ids[:maxAffectedShown]
	}
	return trustImpactRow{Members: f.members, Sessions: f.sessions, AffectedIDs: append([]string{}, ids...), WeakensMFA: mfaRank(v.Candidate.MFA) < mfaRank(c.current.MFA),
		SignsOutEveryone: true}
}

func (c rowContext) whyApprove(v trustchange.View, current bool) string {
	switch {
	case c.unavailable:
		return "trust_unavailable"
	case !c.rights.Approve:
		return "no_right"
	case v.Proposer == c.you:
		return "self"
	case v.State != trustchange.StateValidated:
		return "not_validated"
	case !current:
		return "evidence_stale"
	}
	return ""
}

func (c rowContext) whyActivate(v trustchange.View, current bool) string {
	switch {
	case c.activateUnavailable:
		return "trust_unavailable"
	case !c.rights.Activate:
		return "no_right"
	case v.Proposer == c.you:
		return "self"
	case v.Approver == c.you:
		return "approver"
	case v.State != trustchange.StateApproved:
		return "needs_approval"
	case !current:
		return "evidence_stale"
	}
	return ""
}

func (c rowContext) row(v trustchange.View) changeRow {
	s := c.s
	row := changeRow{ID: v.ID, Number: v.Number, State: v.State, ChangeRevision: v.Revision, ChangeDigest: v.ChangeDigest, OldEpoch: epochOf(v.OldEpoch),
		OldPolicy: v.OldPolicy, OldDigest: v.OldDigest, NewDigest: v.NewDigest, Candidate: revisionOf(revisionFor(v.Workspace, v.Candidate)), Proposer: v.Proposer,
		Validator: v.Validator, Approver: v.Approver, Activator: v.Activator, SupersededBy: v.SupersededBy, EarlierRun: v.EarlierRun,
		RunStartedAt: stampOrEmpty(v.RunStartedAt), History: []trustHistoryRow{}, Impact: trustImpactRow{AffectedIDs: []string{}, SignsOutEveryone: true}}
	for _, e := range v.History {
		row.History = append(row.History, trustHistoryRow{State: e.State, By: e.By, At: stamp(e.At), Rationale: e.Rationale})
	}
	if v.SupersededBy != "" {
		e := epochOf(v.SupersededEpoch)
		row.SupersededEpoch = &e
	}
	candidate := revisionFor(v.Workspace, v.Candidate)
	row.ActUntil = stamp(c.since.Add(min(c.limit, window(candidate))))
	current := false
	open := trustchange.Open(v.State)
	if open {
		if f, err := s.trustFacts(c.verifier, v.Workspace, &candidate, c.now); err == nil {
			row.Impact = c.impact(v, f)
			current = v.Validation != nil && trustchange.EvidenceDigest(v.ChangeDigest, f.digestFacts(c.base)) == v.Validation.Digest
		}
	}
	if x := v.Validation; x != nil {
		row.Validation = &validationRow{Result: x.Result, At: stampOrEmpty(x.At), EvidenceDigest: x.Digest, EvidenceCurrent: current, Items: []trustItemRow{}}
		for _, item := range x.Items {
			row.Validation.Items = append(row.Validation.Items, trustItemRow{Item: item.Item, Result: item.Result, Reason: item.Reason, Detail: item.Detail})
		}
	}
	if a := v.Activation; a != nil {
		row.Activation = &activationRow{Epoch: epochOf(a.Epoch), LedgerSequence: a.Sequence, Watermark: a.Watermark, At: stamp(a.At)}
		row.Invalidation = progressOf(s.progress[v.ID], v.EarlierRun)
	}
	if v.State == trustchange.StateUnknown {
		row.Job = &jobRow{ID: trustJobPrefix + v.ID, State: "waiting", Reason: v.Unknown}
	}
	row.WhyNot = whyRow{Approve: c.whyApprove(v, current), Activate: c.whyActivate(v, current)}
	act := open && !c.unavailable
	row.Can = canRow{Revise: act && c.rights.Propose && v.Proposer == c.you, Validate: act && c.rights.Validate, Approve: row.WhyNot.Approve == "",
		Activate: row.WhyNot.Activate == "", Reject: act && c.rights.Approve && v.Proposer != c.you, Cancel: act && c.rights.Propose && v.Proposer == c.you}
	return row
}
