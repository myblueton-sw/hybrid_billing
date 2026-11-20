package identity

import "math"

type ServiceQueryVersion struct {
	Subjects    uint64
	Credentials uint64
}

type MemberScan struct {
	Rows    []Person
	After   string
	Scanned int
	More    bool
}

func (r *Registry) querySubjectChanged() {
	if r.querySubjects < math.MaxUint64 {
		r.querySubjects++
	}
}

func (r *Registry) subjectChanged() {
	r.generation++
	r.querySubjectChanged()
}

func (r *Registry) credentialChanged(service string) {
	found := r.users[service]
	if found != nil && r.queryCredentials[found.workspace] < math.MaxUint64 {
		r.queryCredentials[found.workspace]++
	}
}

func (h *Workspace) ServiceQueryBasis(actor Verifier, workspace string) (ServiceQueryVersion, error) {
	r, ok := h.open()
	if !ok {
		return ServiceQueryVersion{}, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) {
		return ServiceQueryVersion{}, ErrInvalid
	}
	if _, err := r.administrator(actor, workspace, r.clock()); err != nil {
		return ServiceQueryVersion{}, err
	}
	version := ServiceQueryVersion{Subjects: r.querySubjects, Credentials: r.queryCredentials[workspace]}
	if version.Subjects == math.MaxUint64 || version.Credentials == math.MaxUint64 {
		return ServiceQueryVersion{}, ErrUnavailable
	}
	return version, nil
}

func (h *Workspace) MemberPositions(actor Verifier, workspace, after string, limit int) (MemberScan, error) {
	r, ok := h.open()
	if !ok {
		return MemberScan{}, ErrUnavailable
	}
	defer r.mu.Unlock()
	if !validID(workspace) || !r.pageable(after, limit) {
		return MemberScan{}, ErrInvalid
	}
	now := r.clock()
	if _, err := r.administrator(actor, workspace, now); err != nil {
		return MemberScan{}, err
	}
	ids := r.memberIndex[workspace]
	start := beyond(ids, after)
	end := start + min(limit, len(ids)-start)
	out := MemberScan{Rows: []Person{}, After: after, Scanned: end - start, More: end < len(ids)}
	for _, id := range ids[start:end] {
		out.After = id
		member := r.member(workspace, id)
		if member == nil {
			continue
		}
		value := r.users[id]
		if value == nil {
			return MemberScan{}, ErrUnavailable
		}
		info := r.memberInfo(id, member, now)
		out.Rows = append(out.Rows, Person{User: userInfo(id, value), Member: &info})
	}
	return out, nil
}
