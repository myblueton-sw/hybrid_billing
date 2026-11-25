package protected

func (r *Registry) ChannelsOf(workspace, subject string) ([]ChannelView, error) {
	if !validID(workspace) || !validID(subject) {
		return nil, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil {
		return nil, ErrNotFound
	}
	out := make([]ChannelView, 0, len(s.holders[subject]))
	for _, channel := range s.holders[subject] {
		out = append(out, *channel)
	}
	return out, nil
}
