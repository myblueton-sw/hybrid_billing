package authpolicy

import "time"

func (a Authentication) valid() bool {
	if !oneOf(a.Method, Methods...) || a.At.IsZero() || !ascending(a.Factors, 0, len(Factors), func(f string) bool { return oneOf(f, Factors...) }) {
		return false
	}
	federated := a.Method == MethodOIDC || a.Method == MethodSAML
	switch {
	case a.Method == MethodPassword && a.Issuer != "":
		return false
	case a.Method != MethodPassword && !validID(a.Issuer):
		return false
	case !federated && contains(a.Factors, FactorIdP):
		return false
	}
	return true
}

func (a Authentication) second() bool {
	return contains(a.Factors, FactorOTP) || contains(a.Factors, FactorWebAuthn) || contains(a.Factors, FactorIdP)
}

func formOf(p Principal, a Authentication, action string) error {
	if !validID(p.User) || !validID(p.Workspace) || !oneOf(p.Role, RoleMember, RoleAdministrator) || !a.valid() || action != "" && !validID(action) {
		return ErrInvalid
	}
	return nil
}

func (r *Registry) judge(p Principal, a Authentication, action string, x Revision, now time.Time) error {
	age := now.Sub(a.At)
	switch {
	case r.blocked[[2]string{p.Workspace, p.User}] > 0:
		return refuse(ReasonBlocked)
	case !contains(x.Methods, a.Method):
		return refuse(ReasonMethodNotAllowed)
	case len(x.Issuers) > 0 && a.Method != MethodPassword && !contains(x.Issuers, a.Issuer):
		return refuse(ReasonIssuerNotAllowed)
	case a.At.After(now):
		return refuse(ReasonFutureAuthentication)
	case age > x.MaxAge:
		return refuse(ReasonStale)
	case !a.second() && (x.MFA == MFAEveryone || x.MFA == MFAAdministrators && p.Role == RoleAdministrator):
		return refuse(ReasonMFARequired)
	}
	for _, rule := range x.StepUp {
		if rule.Action != action {
			continue
		}
		switch {
		case age > rule.MaxAge:
			return refuse(ReasonStepUpRequired)
		case rule.MFA && !a.second():
			return refuse(ReasonStepUpMFARequired)
		}
	}
	return nil
}

func (r *Registry) Evaluate(p Principal, a Authentication, action string) (Decision, error) {
	now := r.enter()
	defer r.leave()
	switch p.Kind {
	case KindAssistant:
		return Decision{}, refuse(ReasonAINotAllowed)
	case KindSystem:
		return Decision{}, refuse(ReasonNotHuman)
	case KindHuman:
	default:
		return Decision{}, ErrInvalid
	}
	if err := formOf(p, a, action); err != nil {
		return Decision{}, err
	}
	rev := r.current(p.Workspace)
	if rev == nil {
		return Decision{}, refuse(ReasonNoPolicy)
	}
	if err := r.judge(p, a, action, rev.Revision, now); err != nil {
		return Decision{}, err
	}
	return Decision{Revision: rev.Number, Version: r.workspaces[p.Workspace].version}, nil
}

func (r *Registry) Current(space string, version uint64) bool {
	r.enter()
	defer r.leave()
	w := r.workspaces[space]
	return w != nil && w.version == version
}
