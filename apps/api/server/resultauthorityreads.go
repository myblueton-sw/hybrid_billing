package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

const resultAuthorityProjectionLimit = 1000

type resultAuthorityRead struct {
	s      *Server
	kind   string
	domain grantAuthorityDomain
}
type resultAuthorityRow struct {
	ID     string
	Fields map[string]string
}

func (h resultAuthorityRead) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || !validRequestID(target) {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}
func (a *serverGrantAuthority) resultPolicy() map[string]any {
	return map[string]any{"capabilities": a.domain.readCapabilities(), "proposal_lifetime_seconds": strconv.FormatInt(int64(a.limits.Lifetime/time.Second), 10), "max_grant_validity_seconds": strconv.FormatInt(int64(a.limits.MaxGrantValidity/time.Second), 10)}
}
func (a *serverGrantAuthority) resultRecords(workspace string) ([]inputauthority.Record, error) {
	records, err := a.registry.Records(workspace)
	if err != nil {
		return nil, inputAuthorityError(err)
	}
	if len(records) > resultAuthorityProjectionLimit || len(a.bindings) > resultAuthorityProjectionLimit || len(a.ceilings) > resultAuthorityProjectionLimit {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	if len(records) != len(a.bindings) {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	for _, r := range records {
		key := a.domain.grantPrefix() + r.Digest
		if !validPlanHash(r.Digest) || !a.readShape(r.Proposal.Spec) || a.bindings[key] != r.Proposal.Spec {
			return nil, commands.Fail(commands.EvidenceStale)
		}
	}
	applied := map[string]inputauthority.Spec{}
	for _, r := range records {
		if r.State == inputauthority.Applied && r.Proposal.Spec.Kind == inputauthority.Console {
			if r.Decision == nil || r.Decision.GrantID != a.domain.grantPrefix()+r.Digest {
				return nil, commands.Fail(commands.EvidenceStale)
			}
			applied[r.Decision.GrantID] = r.Proposal.Spec
		}
	}
	if len(applied) != len(a.ceilings) {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	for key, spec := range applied {
		if a.ceilings[key] != spec {
			return nil, commands.Fail(commands.EvidenceStale)
		}
	}
	return records, nil
}
func (h resultAuthorityRead) Apply(ctx commands.Context) (commands.Outcome, error) {
	a := h.s.grantAuthority(h.domain)
	if a == nil {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	if err := a.caller(ctx.Workspace, ctx.User); err != nil {
		return commands.Outcome{}, inputAuthorityError(err)
	}
	if _, err := a.Check(ctx.Workspace, ctx.User); err != nil {
		return commands.Outcome{}, inputAuthorityError(err)
	}
	if ctx.Target != ctx.Workspace {
		return commands.Outcome{}, commands.Fail(commands.EvidenceStale)
	}
	if h.s.cfg.Commands.Fields < 19 || h.s.cfg.Commands.Value < 1024 {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	if a.limits.Lifetime <= 0 || a.limits.MaxGrantValidity <= 0 || a.limits.Lifetime%time.Second != 0 || a.limits.MaxGrantValidity%time.Second != 0 {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	headers := map[string]string{"mode": "synthetic", "workspace": ctx.Workspace, "actor": ctx.User, "current_eligibility": "unchecked"}
	var rows []resultAuthorityRow
	var err error
	if h.kind == "history" {
		var records []inputauthority.Record
		records, err = a.resultRecords(ctx.Workspace)
		for _, r := range records {
			actor := ""
			if r.Decision != nil {
				actor = r.Decision.Actor
			}
			p := r.Proposal
			rows = append(rows, resultAuthorityRow{p.ID, map[string]string{"proposal": p.ID, "digest": r.Digest, "kind": p.Spec.Kind, "capability": p.Spec.Capability, "recipient": p.Spec.Recipient, "state": r.State, "proposer": p.Proposer, "proposed_at": r.ProposedAt.Format(time.RFC3339Nano), "expires_at": r.ExpiresAt.Format(time.RFC3339Nano), "decision_actor": actor}})
		}
	} else {
		capability, collection := ctx.Fields["capability"], ctx.Fields["collection"]
		if _, e := a.domain.terms(ctx.Workspace, capability); e != nil || !a.domain.readCapability(capability) {
			return commands.Outcome{}, commands.Fail(commands.InvalidRequest, "capability")
		}
		headers["capability"], headers["collection"] = capability, collection
		policy, _ := json.Marshal(a.resultPolicy())
		headers["policy"] = string(policy)
		switch collection {
		case "recipients":
			rows, err = a.resultRecipients(ctx.User)
		case "ceilings":
			rows, err = a.resultCeilings(ctx.Workspace, capability)
		default:
			return commands.Outcome{}, commands.Fail(commands.InvalidRequest, "collection")
		}
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	snapshot := inputAuthorityFingerprint(struct {
		Domain, Command string
		Headers         map[string]string
		Policy          map[string]any
		Rows            []resultAuthorityRow
	}{a.domain.readVersion(), h.kind, headers, a.resultPolicy(), rows})
	after, expected := ctx.Fields["after"], ctx.Fields["snapshot"]
	start := 0
	if after != "" || expected != "" {
		if !validRequestID(after) || !validPlanHash(expected) || expected != snapshot {
			return commands.Outcome{}, commands.Fail(commands.EvidenceStale)
		}
		index := sort.Search(len(rows), func(i int) bool { return rows[i].ID >= after })
		if index == len(rows) || rows[index].ID != after {
			return commands.Outcome{}, commands.Fail(commands.EvidenceStale)
		}
		start = index + 1
	}
	headers["snapshot"], headers["next"] = snapshot, ""
	end := min(start+10, len(rows))
	if end < len(rows) {
		headers["next"] = rows[end-1].ID
	}
	for i, row := range rows[start:end] {
		raw, e := json.Marshal(row.Fields)
		if e != nil {
			return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
		}
		headers[fmt.Sprintf("row_%02d", i)] = string(raw)
	}
	if len(headers) > h.s.cfg.Commands.Fields {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	for _, value := range headers {
		if len(value) > h.s.cfg.Commands.Value {
			return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
		}
	}
	return commands.Outcome{Fields: headers}, nil
}
func (s *Server) resultAuthorityReadDefinitions() []commands.Definition {
	out := []commands.Definition{}
	for _, domain := range []grantAuthorityDomain{resultGrantAuthority, protectedGrantAuthority, termGrantAuthority, draftGrantAuthority} {
		out = append(out, s.grantAuthorityReadDefinitions(domain)...)
	}
	return out
}
func (s *Server) grantAuthorityReadDefinitions(domain grantAuthorityDomain) []commands.Definition {
	out := []commands.Definition{}
	for _, kind := range []string{"history", "context"} {
		fields := []commands.Field{{Name: "after", Type: commands.ID}, {Name: "snapshot", Type: commands.Text}}
		if kind == "context" {
			fields = append(fields, commands.Field{Name: "collection", Type: commands.ID, Required: true}, commands.Field{Name: "capability", Type: commands.ID, Required: true})
		}
		out = append(out, commands.Definition{ID: domain.readNamespace() + "." + kind, Permission: commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}, Fields: fields, Refusals: []commands.Code{commands.EvidenceStale}, Handler: resultAuthorityRead{s: s, kind: kind, domain: domain}})
	}
	return out
}
