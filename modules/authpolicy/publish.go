package authpolicy

import "time"

const epochNumberBits = 32

func EpochNumber(epoch uint64) uint64 { return epoch & (1<<epochNumberBits - 1) }

func EpochGeneration(epoch uint64) uint64 { return epoch >> epochNumberBits }

func (r *Registry) publishable(p Publication) error {
	c, err := clean(Change{Kind: ChangeRevision, Revision: p.Revision})
	if err != nil || EpochNumber(p.Epoch) == 0 || p.Number == 0 {
		return ErrInvalid
	}
	for _, actor := range []string{p.Proposer, p.Validator, p.Approver, p.Activator} {
		if actor != "" && !validID(actor) {
			return ErrInvalid
		}
	}
	w := r.workspaces[c.Revision.Workspace]
	switch {
	case w == nil && len(r.workspaces) >= maxWorkspaces:
		return ErrLimit
	case w != nil && (p.Epoch <= w.revisions[len(w.revisions)-1].Epoch || p.Number <= w.number):
		return refuse(ReasonOutOfOrder)
	}
	return nil
}

func (r *Registry) CanPublish(p Publication) error {
	r.enter()
	defer r.leave()
	return r.publishable(p)
}

func (r *Registry) Publish(p Publication) (RevisionView, error) {
	now := r.enter()
	defer r.leave()
	if err := r.publishable(p); err != nil {
		return RevisionView{}, err
	}
	space := p.Revision.Workspace
	w := r.workspaces[space]
	if w == nil {
		w = &workspace{}
		r.workspaces[space] = w
	}
	w.number = p.Number
	view := RevisionView{Revision: copyChange(Change{Revision: p.Revision}).Revision, Number: p.Number, Epoch: p.Epoch, Proposer: p.Proposer, Validator: p.Validator,
		Approver: p.Approver, Activator: p.Activator, At: now}
	w.revisions = append(w.revisions, view)
	if len(w.revisions) > maxKeptRevisions {
		w.revisions = w.revisions[1:]
	}
	r.bump(space)
	return copyRevision(view), nil
}

func (r *Registry) Blocked(space, user string) bool {
	r.enter()
	defer r.leave()
	return r.blocked[[2]string{space, user}] > 0
}

func (r *Registry) Judge(p Principal, a Authentication, candidate Revision, now time.Time) error {
	r.enter()
	defer r.leave()
	switch p.Kind {
	case KindAssistant:
		return refuse(ReasonAINotAllowed)
	case KindSystem:
		return refuse(ReasonNotHuman)
	case KindHuman:
	default:
		return ErrInvalid
	}
	if _, err := clean(Change{Kind: ChangeRevision, Revision: candidate}); err != nil || formOf(p, a, "") != nil || candidate.Workspace != p.Workspace {
		return ErrInvalid
	}
	return r.judge(p, a, "", candidate, now)
}

func Reachable(p Principal, candidate Revision) bool {
	return contains(candidate.Methods, MethodPassword) && candidate.MFA != MFAEveryone && !(candidate.MFA == MFAAdministrators && p.Role == RoleAdministrator)
}
