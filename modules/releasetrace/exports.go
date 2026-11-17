package releasetrace

func ValidText(text string) bool { return validText(text) }

func (r *Registry) Check(c Change, proposer, approver string) error {
	r.enter()
	defer r.leave()
	c, err := clean(c)
	if err != nil {
		return err
	}
	return r.check(c, proposer, approver)
}
