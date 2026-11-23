package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"time"
)

func (a *serverGrantAuthority) resultRecipients(actor string) ([]resultAuthorityRow, error) {
	members, err := a.s.roster(a.s.caller)
	if err != nil {
		return nil, inputAuthorityError(err)
	}
	if len(members) > resultAuthorityProjectionLimit {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	rows := []resultAuthorityRow{}
	count := 0
	for after := ""; ; {
		users, err := a.s.users(after)
		if err != nil {
			return nil, inputAuthorityError(err)
		}
		count += len(users)
		if count > resultAuthorityProjectionLimit {
			return nil, commands.Fail(commands.PrerequisiteFailed)
		}
		for _, user := range users {
			member, ok := members[user.ID]
			if user.ID == actor || !ok || user.Kind != identity.Human || user.State != identity.Active || !member.Effective || member.UserState != identity.Active {
				continue
			}
			login, _ := a.s.dir.loginOf(user.ID)
			rows = append(rows, resultAuthorityRow{user.ID, map[string]string{"user": user.ID, "login": login}})
		}
		if len(users) < 100 {
			break
		}
		after = users[len(users)-1].ID
	}
	return rows, nil
}
func (a *serverGrantAuthority) resultCeilings(workspace, capability string) ([]resultAuthorityRow, error) {
	records, err := a.resultRecords(workspace)
	if err != nil {
		return nil, err
	}
	owned := map[string]inputauthority.Record{}
	for _, r := range records {
		if r.State == inputauthority.Applied && r.Proposal.Spec.Kind == inputauthority.Console && r.Decision != nil {
			owned[r.Decision.GrantID] = r
		}
	}
	rows := []resultAuthorityRow{}
	for key, spec := range a.ceilings {
		record, ok := owned[key]
		if !ok || key != a.domain.grantPrefix()+record.Digest || record.Proposal.Spec != spec || !a.readShape(spec) || spec.Kind != inputauthority.Console {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		effect, err := a.Lookup(key)
		if err != nil {
			return nil, inputAuthorityError(err)
		}
		if !effect.Found || effect.Spec != spec {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		if effect.Status != inputauthority.ActiveGrant || a.s.clock().Before(spec.NotBefore) || spec.Capability != capability {
			continue
		}
		rows = append(rows, resultAuthorityRow{key, map[string]string{"ceiling": key, "capability": spec.Capability, "not_before": spec.NotBefore.Format(time.RFC3339Nano), "not_after": spec.NotAfter.Format(time.RFC3339Nano)}})
	}
	return rows, nil
}
