package releasetrace

import (
	"strconv"
	"time"
)

func oneOf(value string, set ...string) bool {
	for _, item := range set {
		if item == value {
			return true
		}
	}
	return false
}

func hexOf(text string, sizes ...int) bool {
	ok := false
	for _, size := range sizes {
		ok = ok || len(text) == size
	}
	if !ok {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func digestOK(text string) bool { return hexOf(text, 64) }

func commitOK(text string) bool { return hexOf(text, 40, 64) }

func month(period string) (int, bool) {
	if len(period) != 7 || period[4] != '-' || !digits(period[:4], 4) || period[6] < '0' || period[6] > '9' {
		return 0, false
	}
	year, _ := strconv.Atoi(period[:4])
	m := (int(period[5])-'0')*10 + int(period[6]) - '0'
	if m < 1 || m > 12 {
		return 0, false
	}
	return year*12 + m - 1, true
}

func keyLess(a, b Key) bool {
	return a.Namespace < b.Namespace || a.Namespace == b.Namespace && a.ID < b.ID
}

func (k Key) valid() bool { return validID(k.Namespace) && validID(k.ID) }

func ascendingIDs(list []string, low, high int) bool {
	if len(list) < low || len(list) > high {
		return false
	}
	for index, id := range list {
		if !validID(id) || index > 0 && list[index-1] >= id {
			return false
		}
	}
	return true
}

func ascendingKeys(list []Key, low, high int) bool {
	if len(list) < low || len(list) > high {
		return false
	}
	for index, k := range list {
		if !k.valid() || index > 0 && !keyLess(list[index-1], k) {
			return false
		}
	}
	return true
}

func (q Requirement) valid() bool {
	if !validID(q.Namespace) || !validID(q.ID) || !validID(q.Source) || !validID(q.Owner) || !ascendingIDs(q.Interfaces, 0, maxInterfaces) || !digestOK(q.Design) ||
		len(q.Tests) == 0 || len(q.Tests) > maxTests {
		return false
	}
	for index, t := range q.Tests {
		if !validID(t.ID) || t.Category != "" && !oneOf(t.Category, Categories...) || index > 0 && q.Tests[index-1].ID >= t.ID {
			return false
		}
	}
	return true
}

func (p Path) valid() bool {
	return validID(p.ID) && validID(p.Provider) && validID(p.Contract) && validID(p.Role) && validID(p.Version) && oneOf(p.Issuance, IssuanceAPI, IssuanceFile, IssuanceXLSX) &&
		(p.Template == "" || validID(p.Template)) && ascendingKeys(p.Requirements, 1, maxPathReqs)
}

func clean(c Change) (Change, error) {
	out := copyChange(c)
	ok := false
	switch c.Kind {
	case ChangeRequirement:
		out = Change{Kind: c.Kind, Requirement: out.Requirement}
		ok = c.Requirement.valid()
	case ChangeRetire:
		out = Change{Kind: c.Kind, Retire: c.Retire}
		x := c.Retire
		ok = x.Key.valid() && (x.Replacement == Key{} || x.Replacement.valid() && x.Replacement != x.Key) && validText(x.Reason)
	case ChangeImplemented:
		out = Change{Kind: c.Kind, Implemented: c.Implemented}
		ok = c.Implemented.Key.valid() && commitOK(c.Implemented.Commit)
	case ChangeBuild:
		out = Change{Kind: c.Kind, Build: c.Build}
		ok = semver(c.Build.Build) && commitOK(c.Build.Commit)
	case ChangeHarness:
		out = Change{Kind: c.Kind, Harness: c.Harness}
		ok = validID(c.Harness.ID) && validID(c.Harness.Identity)
	case ChangeProof:
		out = Change{Kind: c.Kind, Proof: c.Proof}
		x := c.Proof
		_, period := month(x.Period)
		ok = x.Key.valid() && semver(x.Build) && validID(x.Environment) && validID(x.Customer) && period && digestOK(x.Evidence)
	case ChangePath:
		out = Change{Kind: c.Kind, Path: out.Path}
		ok = c.Path.valid()
	case ChangeReconcile:
		out = Change{Kind: c.Kind, Reconcile: c.Reconcile}
		x := c.Reconcile
		_, period := month(x.Period)
		ok = validID(x.Path) && period && digestOK(x.Evidence) && x.Differences >= 0 && x.Matched == (x.Differences == 0)
	case ChangeActivate:
		out = Change{Kind: c.Kind, Activate: c.Activate}
		ok = validID(c.Activate.Path) && semver(c.Activate.Build) && digestOK(c.Activate.Proof)
	}
	if !ok {
		return out, ErrInvalid
	}
	return out, nil
}

func copyChange(c Change) Change {
	c.Requirement.Interfaces = append([]string(nil), c.Requirement.Interfaces...)
	c.Requirement.Tests = append([]TestRef(nil), c.Requirement.Tests...)
	c.Path.Requirements = append([]Key(nil), c.Path.Requirements...)
	return c
}

func copyProposal(p ProposalView) ProposalView {
	p.Change = copyChange(p.Change)
	return p
}

func boolText(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func encodeChange(c Change) []byte {
	var e encoder
	q, x, p := c.Requirement, c.Retire, c.Path
	for _, value := range []string{c.Kind, q.Namespace, q.ID, q.Source, q.Owner, q.Design, boolText(q.Common), boolText(q.Blocking), x.Key.Namespace, x.Key.ID,
		x.Replacement.Namespace, x.Replacement.ID, x.Reason, c.Implemented.Key.Namespace, c.Implemented.Key.ID, c.Implemented.Commit, c.Build.Build, c.Build.Commit,
		c.Harness.ID, c.Harness.Identity, c.Proof.Key.Namespace, c.Proof.Key.ID, c.Proof.Build, c.Proof.Environment, c.Proof.Customer, c.Proof.Period,
		c.Proof.Evidence, p.ID, p.Provider, p.Contract, p.Role, p.Version, p.Issuance, p.Template, c.Reconcile.Path, c.Reconcile.Period, c.Reconcile.Evidence,
		boolText(c.Reconcile.Matched), strconv.Itoa(c.Reconcile.Differences), c.Activate.Path, c.Activate.Build, c.Activate.Proof,
		strconv.Itoa(len(q.Interfaces)), strconv.Itoa(len(q.Tests)), strconv.Itoa(len(p.Requirements))} {
		e.text(value)
	}
	for _, v := range q.Interfaces {
		e.text(v)
	}
	for _, t := range q.Tests {
		e.text(t.ID)
		e.text(t.Category)
	}
	for _, k := range p.Requirements {
		e.text(k.Namespace)
		e.text(k.ID)
	}
	return e.out.Bytes()
}

func (r *Registry) check(c Change, proposer, approver string) error {
	switch c.Kind {
	case ChangeRequirement:
		q := c.Requirement
		switch {
		case r.requirements[Key{q.Namespace, q.ID}] != nil:
			return ErrConflict
		case len(r.requirements) >= maxRequirements:
			return ErrLimit
		}
	case ChangeRetire:
		return r.checkRetire(c.Retire)
	case ChangeImplemented:
		q := r.requirements[c.Implemented.Key]
		switch {
		case q == nil:
			return ErrNotFound
		case q.retired:
			return refuse(ReasonRetired)
		}
	case ChangeBuild:
		switch {
		case r.builds[c.Build.Build] != nil:
			return ErrConflict
		case len(r.builds) >= maxBuilds:
			return ErrLimit
		}
	case ChangeHarness:
		switch {
		case r.harnesses[c.Harness.ID] != nil:
			return ErrConflict
		case r.identities[c.Harness.Identity] != "":
			return refuse(ReasonIdentityTaken)
		case len(r.harnesses) >= maxHarnesses:
			return ErrLimit
		}
	case ChangeProof:
		x := c.Proof
		q := r.requirements[x.Key]
		switch {
		case q == nil:
			return ErrNotFound
		case r.builds[x.Build] == nil:
			return refuse(ReasonUnknownBuild)
		case r.proofs[x.Key][x.Build] != nil:
			return ErrConflict
		case !r.tested(q, x.Build):
			return refuse(ReasonNotTested)
		}
	case ChangePath:
		p := c.Path
		for _, k := range p.Requirements {
			if r.requirements[k] == nil {
				return ErrNotFound
			}
		}
		switch {
		case r.paths[p.ID] != nil:
			return ErrConflict
		case len(r.paths) >= maxPaths:
			return ErrLimit
		}
	case ChangeReconcile:
		return r.checkReconcile(c.Reconcile)
	case ChangeActivate:
		return r.checkActivate(c.Activate, proposer, approver)
	}
	return nil
}

func (r *Registry) checkRetire(x Retire) error {
	q := r.requirements[x.Key]
	if q == nil {
		return ErrNotFound
	}
	var rep *requirement
	if x.Replacement != (Key{}) {
		if rep = r.requirements[x.Replacement]; rep == nil {
			return ErrNotFound
		}
	}
	switch {
	case q.retired:
		return refuse(ReasonRetired)
	case rep == nil && q.view.Common:
		return refuse(ReasonCommonNeedsReplacement)
	case rep == nil:
		return nil
	case rep.retired || rep.replaces || rep.seq < q.seq:
		return refuse(ReasonWeakerReplacement)
	case q.view.Common && !rep.view.Common || q.view.Blocking && !rep.view.Blocking:
		return refuse(ReasonWeakerReplacement)
	}
	return nil
}

func (r *Registry) checkReconcile(x Reconcile) error {
	p := r.paths[x.Path]
	if p == nil {
		return ErrNotFound
	}
	for _, old := range p.recon {
		if old.Period == x.Period {
			return ErrConflict
		}
	}
	m, _ := month(x.Period)
	if now := r.now.Year()*12 + int(r.now.Month()) - 1; m >= now {
		return refuse(ReasonFuturePeriod)
	}
	return nil
}

func (r *Registry) checkActivate(x Activate, proposer, approver string) error {
	p := r.paths[x.Path]
	if p == nil {
		return ErrNotFound
	}
	if r.builds[x.Build] == nil {
		return refuse(ReasonUnknownBuild)
	}
	if !r.reconciled(p) {
		return refuse(ReasonReconciliationMissing)
	}
	for _, cat := range Categories {
		if !r.category(p, cat, x.Build) {
			return refuse(ReasonCategoryMissing)
		}
	}
	for _, k := range p.view.Requirements {
		if !r.tested(r.requirements[k], x.Build) {
			return refuse(ReasonRequirementNotTested)
		}
	}
	for _, person := range r.pathPeople(p, x.Build) {
		if person == proposer || approver != "" && person == approver {
			return refuse(ReasonSelfVerification)
		}
	}
	return nil
}

func (r *Registry) apply(c Change, by people, now time.Time) {
	c = copyChange(c)
	r.seq++
	switch c.Kind {
	case ChangeRequirement:
		q := c.Requirement
		k := Key{q.Namespace, q.ID}
		r.requirements[k] = &requirement{view: q, by: by, seq: r.seq}
		for _, t := range q.Tests {
			tk := [2]string{q.Namespace, t.ID}
			r.tests[tk] = append(r.tests[tk], k)
		}
	case ChangeRetire:
		q := r.requirements[c.Retire.Key]
		q.retired, q.replacement, q.reason = true, c.Retire.Replacement, c.Retire.Reason
		if rep := r.requirements[c.Retire.Replacement]; rep != nil {
			rep.replaces = true
		}
	case ChangeImplemented:
		q := r.requirements[c.Implemented.Key]
		q.impls = append(q.impls, impl{commit: c.Implemented.Commit, seq: r.seq, by: by})
		if len(q.impls) > maxImpls {
			q.impls = q.impls[1:]
		}
	case ChangeBuild:
		r.builds[c.Build.Build] = &build{commit: c.Build.Commit, seq: r.seq, by: by}
	case ChangeHarness:
		r.harnesses[c.Harness.ID] = &harness{id: c.Harness.ID, identity: c.Harness.Identity, by: by}
		r.identities[c.Harness.Identity] = c.Harness.ID
	case ChangeProof:
		x := c.Proof
		if r.proofs[x.Key] == nil {
			r.proofs[x.Key] = map[string]*proof{}
		}
		r.proofs[x.Key][x.Build] = &proof{view: x, by: by}
	case ChangePath:
		r.paths[c.Path.ID] = &path{view: c.Path}
	case ChangeReconcile:
		p := r.paths[c.Reconcile.Path]
		p.recon = append(p.recon, c.Reconcile)
		p.reconBy = append(p.reconBy, by)
		for i := len(p.recon) - 1; i > 0; i-- {
			a, _ := month(p.recon[i-1].Period)
			b, _ := month(p.recon[i].Period)
			if a < b {
				break
			}
			p.recon[i-1], p.recon[i] = p.recon[i], p.recon[i-1]
			p.reconBy[i-1], p.reconBy[i] = p.reconBy[i], p.reconBy[i-1]
		}
		if len(p.recon) > maxPeriods {
			p.recon, p.reconBy = p.recon[1:], p.reconBy[1:]
		}
	case ChangeActivate:
		p := r.paths[c.Activate.Path]
		p.active, p.build, p.proof = true, c.Activate.Build, c.Activate.Proof
	}
	_ = now
}
