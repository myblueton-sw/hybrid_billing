package server

import (
	"slices"

	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func (a *serverGrantAuthority) financialCheck(workspace, actor string) (string, error) {
	facts, err := a.s.governor(workspace, actor)
	if err != nil {
		return "", err
	}
	admins, err := a.s.approvals.Admins(workspace)
	if err != nil {
		return "", err
	}
	if !slices.Contains(admins, actor) {
		return "", inputauthority.ErrDenied
	}
	slices.Sort(admins)
	return inputAuthorityFingerprint(struct {
		Governor string
		Admins   []string
	}{facts, admins}), nil
}

func (a *serverGrantAuthority) financialCurrent(spec inputauthority.Spec) (string, error) {
	if spec.Kind != financialpermission.DirectHuman || spec.CeilingID != "" || spec.Capability != financialpermission.Approve || spec.Workspace != a.s.bootWorkspace {
		return "", inputauthority.ErrInvalid
	}
	facts, err := a.s.accessFacts(a.s.caller)
	if err != nil {
		return "", err
	}
	member, ok := facts.members[spec.Recipient]
	user := facts.users[spec.Recipient]
	if !ok || !member.Effective || member.UserState != identity.Active || user.State != identity.Active || user.Kind != identity.Human {
		return "", inputauthority.ErrDenied
	}
	return inputAuthorityFingerprint(struct {
		Person     string
		Generation uint64
	}{facts.person(spec.Recipient), a.s.admin.Generation()}), nil
}

func governedGrantTerms(action gate.Action, kind string, fields []string) bool {
	return kind == financialpermission.ResourceKind || pricingGrantTerms(action, kind, fields)
}
