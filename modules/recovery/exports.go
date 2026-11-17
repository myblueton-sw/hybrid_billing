package recovery

import "time"

func ManifestDigest(manifest []Inventory) string { return manifestDigest(manifest) }

func LostDigest(lost []uint64) string { return lostDigest(lost) }

func (r *Registry) Check(c Change) error {
	now := r.enter()
	defer r.leave()
	c, name, ok := clean(c)
	if !ok {
		return ErrInvalid
	}
	if name != registerScope && r.installations[name] == nil {
		return ErrNotFound
	}
	return r.check(c, now)
}

func (r *Registry) Status(id string) (string, uint64, time.Time, error) {
	r.enter()
	defer r.leave()
	s, err := r.located(id)
	if err != nil {
		return "", 0, time.Time{}, err
	}
	return s.state, s.generation, s.stoppedAt, nil
}
