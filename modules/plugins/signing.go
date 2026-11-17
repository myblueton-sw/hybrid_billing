package plugins

func (m Manifest) SigningMessage() []byte { return m.encode() }

func KeyChallenge(publisher, version string) []byte { return challenge(publisher, version) }

func (r *Registry) Recheck(number uint64) error {
	now := r.enter()
	defer r.leave()
	p := r.proposals[number]
	switch {
	case p == nil:
		return ErrNotFound
	case p.view.State != ProposalPending:
		return refuse(ReasonNotPending)
	}
	return r.check(p.view.Change, now)
}
