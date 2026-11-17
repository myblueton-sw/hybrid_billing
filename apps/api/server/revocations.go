package server

import (
	"sort"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type itemRow struct {
	Item   string   `json:"item"`
	State  string   `json:"state"`
	Reason string   `json:"reason"`
	Refs   []string `json:"refs"`
}

type receiptRow struct {
	Sequence  uint64 `json:"sequence"`
	Watermark string `json:"watermark"`
	Reason    string `json:"reason"`
}

type transitionRow struct {
	State string `json:"state"`
	At    string `json:"at"`
}

type revocationRow struct {
	ID             string          `json:"id"`
	Origin         string          `json:"origin"`
	RecordedAt     string          `json:"recorded_at"`
	TargetKind     string          `json:"target_kind"`
	Target         string          `json:"target"`
	TargetVersion  uint64          `json:"target_version"`
	TargetPresent  bool            `json:"target_present"`
	Replay         string          `json:"replay"`
	State          string          `json:"state"`
	History        []transitionRow `json:"history"`
	EndedDelegs    []string        `json:"ended_delegations"`
	DenyGeneration uint64          `json:"deny_generation"`
	JobState       string          `json:"job_state"`
	JobReason      string          `json:"job_reason"`
	RequestedBy    string          `json:"requested_by"`
	ApprovedBy     string          `json:"approved_by"`
	Request        string          `json:"request"`
	ReceivedAt     string          `json:"received_at"`
	DenyAt         string          `json:"deny_at"`
	Receipt        receiptRow      `json:"receipt"`
	Verified       bool            `json:"verified"`
	Cleanup        []itemRow       `json:"cleanup"`
	CompletedAt    string          `json:"completed_at"`
}

func chainItem(ended []string) accesschange.Item {
	if len(ended) == 0 {
		return accesschange.Item{Item: "C9", State: accesschange.ItemNotApplicable, Reason: "no_dependent_delegation"}
	}
	return accesschange.Item{Item: "C9", State: accesschange.ItemDone, Reason: "chain_rules", Refs: append([]string(nil), ended...)}
}

func cleanupItems(d accesschange.Denied) []accesschange.Item {
	return []accesschange.Item{
		{Item: "C1", State: accesschange.ItemDone, Reason: "deny_generation"},
		{Item: "C2", State: accesschange.ItemNotApplicable, Reason: "no_permission_in_sessions"},
		{Item: "C3", State: accesschange.ItemNotApplicable, Reason: "human_grants_only"},
		{Item: "C4", State: accesschange.ItemNotApplicable, Reason: "no_store"},
		{Item: "C5", State: accesschange.ItemOutside, Reason: "cannot_recall"},
		{Item: "C6", State: accesschange.ItemOutside, Reason: "not_cancelled"},
		{Item: "C7", State: accesschange.ItemOutside, Reason: "no_job_runner"},
		{Item: "C8", State: accesschange.ItemNotApplicable, Reason: "nothing_keyed_to_grants"},
		chainItem(d.Ended),
	}
}

func (s *Server) effectiveDelegations() map[string]bool {
	out := map[string]bool{}
	for _, d := range s.admin.Delegations() {
		if d.Effective && !d.Closed {
			out[d.ID] = true
		}
	}
	return out
}

func (s *Server) endedSince(before map[string]bool, self string) []string {
	after := s.effectiveDelegations()
	out := []string{}
	for id := range before {
		if !after[id] && id != self {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Server) effects(deny func() error, self string) accesschange.Effects {
	return accesschange.Effects{
		Deny: func() (accesschange.Denied, error) {
			before := s.effectiveDelegations()
			if err := deny(); err != nil {
				return accesschange.Denied{}, mapAccess(err)
			}
			return accesschange.Denied{Generation: s.admin.Generation(), Ended: s.endedSince(before, self)}, nil
		},
		Ledger:  ledgerAdapter{s},
		Cleanup: cleanupItems,
		Planned: cleanupItems(accesschange.Denied{}),
	}
}

func (s *Server) grantEffects(g gate.GrantView) accesschange.Effects {
	return s.effects(func() error { return s.admin.RevokeGrant(g.ID, g.Version) }, "")
}

func (s *Server) delegationEffects(d gate.DelegationView) accesschange.Effects {
	return s.effects(func() error { return s.admin.RevokeDelegation(d.ID, d.Version) }, d.ID)
}

type delegationCommand struct{ s *Server }

func (h delegationCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	if _, ok := s.delegationView(target); ok && workspace == s.bootWorkspace {
		return commands.Target{Revision: s.Revision()}, commands.Found
	}
	return commands.Target{}, commands.NotFound
}

func (h delegationCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
	}
	if err := s.humanActor(ctx.User); err != nil {
		return commands.Outcome{}, err
	}
	d, _ := s.delegationView(ctx.Target)
	v, err := s.changes.Revoke(accesschange.Revocation{Workspace: ctx.Workspace, Target: accesschange.Target{Kind: accesschange.TargetDelegation, ID: d.ID, Version: d.Version},
		RequestedBy: ctx.User}, s.delegationEffects(d))
	if err != nil {
		return commands.Outcome{}, err
	}
	return commands.Outcome{Accepted: true, JobID: v.ID, Fields: map[string]string{"revocation": v.ID}}, nil
}

func (s *Server) revocationRows() []revocationRow {
	out := []revocationRow{}
	for _, v := range s.changes.Revocations(s.bootWorkspace) {
		job, reason := accesschange.JobState(v.State)
		row := revocationRow{ID: v.ID, Origin: v.Origin, RecordedAt: stamp(v.RecordedAt), TargetKind: v.Target.Kind, Target: v.Target.ID, TargetVersion: v.Target.Version,
			TargetPresent: v.TargetPresent, Replay: v.Replay, State: v.State, History: []transitionRow{}, EndedDelegs: append([]string{}, v.Ended...),
			DenyGeneration: v.DenyGeneration, JobState: job, JobReason: reason, RequestedBy: v.RequestedBy, ApprovedBy: v.ApprovedBy, Request: v.Request,
			DenyAt: stamp(v.DenyAt), Receipt: receiptRow{Sequence: v.Receipt.Sequence, Watermark: v.Receipt.Watermark, Reason: v.Reason},
			Verified: v.Reason == "" && v.Receipt.Sequence > 0, Cleanup: []itemRow{}, CompletedAt: stampOrEmpty(v.CompletedAt)}
		for _, h := range v.History {
			row.History = append(row.History, transitionRow{State: h.State, At: stamp(h.At)})
		}
		row.ReceivedAt = row.History[0].At
		for _, it := range v.Cleanup {
			row.Cleanup = append(row.Cleanup, itemRow{Item: it.Item, State: it.State, Reason: it.Reason, Refs: append([]string{}, it.Refs...)})
		}
		out = append(out, row)
	}
	return out
}
