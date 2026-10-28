package customeraccess

import (
	"context"
	"reflect"
)

type Checkpoint struct {
	store        *Store
	action       Action
	installation string
	subject      string
	epoch        uint64
	selection    Selection
	stamp        authorityStamp
}

func (s *Service) Capture(ctx context.Context, request Selection) (*Checkpoint, error) {
	if ctx == nil || s == nil || s.store == nil || !s.validSelection(request) {
		return nil, ErrDenied
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request = copySelection(request)
	p, err := s.verify(ctx)
	if err != nil {
		return nil, ErrDenied
	}
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	stamp, err := s.store.authorize(p, request, s.action)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &Checkpoint{store: s.store, action: s.action, installation: p.Installation, subject: p.SubjectID, epoch: p.Epoch, selection: request, stamp: stamp}, nil
}

func (c *Checkpoint) matches(s *Service, p Principal, request Selection, stamp authorityStamp) bool {
	return c.store == s.store && c.action == s.action && c.installation == p.Installation && c.subject == p.SubjectID && c.epoch == p.Epoch && reflect.DeepEqual(c.selection, request) && c.stamp.equals(stamp)
}
