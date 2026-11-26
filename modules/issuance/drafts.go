package issuance

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func amount(text string, scale int) (*big.Int, bool) {
	whole, fraction, dotted := strings.Cut(text, ".")
	if whole == "" || len(whole)+len(fraction) > maxDigits || dotted && (fraction == "" || len(fraction) > scale || fraction[len(fraction)-1] == '0') {
		return nil, false
	}
	if len(whole) > 1 && whole[0] == '0' {
		return nil, false
	}
	for _, part := range []string{whole, fraction} {
		for index := 0; index < len(part); index++ {
			if part[index] < '0' || part[index] > '9' {
				return nil, false
			}
		}
	}
	value, _ := new(big.Int).SetString(whole+fraction+strings.Repeat("0", scale-len(fraction)), 10)
	return value, value.Sign() > 0
}

func validText(text string) bool {
	if text == "" || len(text) > maxTextBytes || !utf8.ValidString(text) {
		return false
	}
	for _, c := range text {
		if unicode.IsControl(c) || c >= '\u202a' && c <= '\u202e' || c >= '\u2066' && c <= '\u2069' {
			return false
		}
	}
	return true
}

func validCurrency(code string) bool {
	if len(code) != 3 {
		return false
	}
	for index := 0; index < 3; index++ {
		if code[index] < 'A' || code[index] > 'Z' {
			return false
		}
	}
	return true
}

func detailForm(d *Detail, scale int) bool {
	if d == nil {
		return true
	}
	_, ok := amount(d.Total, scale)
	return ok && validID(d.Artifact) && validDigest(d.Digest) && d.Rows >= 1 && d.Rows <= maxDetailRows
}

func validDigest(text string) bool {
	if len(text) != 64 {
		return false
	}
	_, err := hex.DecodeString(text)
	return err == nil && strings.ToLower(text) == text
}

func draftForm(d Draft) bool {
	if !validID(d.ID) || !subject(d.Kind) || !subject(d.Customer) || !subject(d.Entity) || !validID(d.Contract) || !validCurrency(d.Currency) ||
		d.Scale < 0 || d.Scale > maxScale || !languages[d.Language] || len(d.Lines) == 0 || len(d.Lines) > maxLines ||
		d.Original != "" && !validID(d.Original) || !detailForm(d.Detail, d.Scale) {
		return false
	}
	if _, ok := amount(d.Total, d.Scale); !ok {
		return false
	}
	for _, line := range d.Lines {
		if _, ok := amount(line.Amount, d.Scale); !ok || !validText(line.Text) {
			return false
		}
	}
	return true
}

func sums(d Draft) (*big.Int, *big.Int) {
	total, plain := new(big.Int), new(big.Int)
	for _, line := range d.Lines {
		value, _ := amount(line.Amount, d.Scale)
		total.Add(total, value)
		if !line.Tax {
			plain.Add(plain, value)
		}
	}
	return total, plain
}

func copyDraft(d Draft) Draft {
	d.Lines = append([]Line(nil), d.Lines...)
	if d.Detail != nil {
		detail := *d.Detail
		d.Detail = &detail
	}
	return d
}

type encoder struct {
	b strings.Builder
}

func (e *encoder) text(value string) *encoder {
	e.b.WriteString(strconv.Itoa(len(value)))
	e.b.WriteByte(':')
	e.b.WriteString(value)
	return e
}

func (e *encoder) number(value int64) *encoder { return e.text(strconv.FormatInt(value, 10)) }

func (e *encoder) flag(value bool) *encoder { return e.text(strconv.FormatBool(value)) }

func (e *encoder) digest() string {
	sum := sha256.Sum256([]byte(e.b.String()))
	return hex.EncodeToString(sum[:])
}

func encodeDraft(e *encoder, d Draft) {
	e.text(d.ID).text(d.Kind).text(d.Customer).text(d.Entity).text(d.Contract).text(d.Currency).number(int64(d.Scale)).text(d.Language).text(d.Total).text(d.Original)
	e.number(int64(len(d.Lines)))
	for _, line := range d.Lines {
		e.text(line.Text).text(line.Amount).flag(line.Tax)
	}
	e.flag(d.Detail != nil)
	if d.Detail != nil {
		e.text(d.Detail.Artifact).text(d.Detail.Digest).number(d.Detail.Rows).text(d.Detail.Total)
	}
}

func draftDigest(workspace string, d Draft) string {
	e := (&encoder{}).text(draftLabel).text(workspace)
	encodeDraft(e, d)
	return e.digest()
}

func sameDraft(a, b Draft) bool {
	return draftDigest("", a) == draftDigest("", b)
}

func viewDraft(d *DraftView) DraftView {
	v := *d
	v.Draft = copyDraft(v.Draft)
	v.References = append([]Reference(nil), v.References...)
	if v.Coverage != nil {
		c := *v.Coverage
		v.Coverage = &c
	}
	return v
}

func payable(k Kind) bool { return k.Class != ClassInternal && !k.Credit }

func (r *Registry) rules(workspace string, s *space, k *kind, d Draft) error {
	total, plain := sums(d)
	declared, _ := amount(d.Total, d.Scale)
	switch {
	case k.view.State != StateActive:
		return refuse(ReasonKindPending)
	case !contains(k.view.Kind.Languages, d.Language):
		return refuse(ReasonLanguage)
	case total.Cmp(declared) != 0:
		return refuse(ReasonTotalMismatch)
	}
	if d.Detail != nil {
		detail, _ := amount(d.Detail.Total, d.Scale)
		if detail.Cmp(plain) != 0 {
			return refuse(ReasonDetailMismatch)
		}
		facts, err := r.config.Details.Detail(workspace, d.Detail.Artifact)
		if err != nil || facts.Digest != d.Detail.Digest || facts.Rows != d.Detail.Rows || facts.Total != d.Detail.Total || facts.Currency != d.Currency || facts.Scale != d.Scale {
			return refuse(ReasonDetailUnverified)
		}
	}
	if k.view.Kind.Credit {
		return r.creditRoom(s, k, d)
	}
	return nil
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func (r *Registry) creditRoom(s *space, k *kind, d Draft) error {
	original := s.documents[d.Original]
	if original == nil {
		return refuse(ReasonOriginalNotIssued)
	}
	o := original.view.Snapshot
	ok := s.kinds[o.Draft.Kind].view.Kind
	if ok.Credit || ok.Class != k.view.Kind.Class || ok.Statutory != k.view.Kind.Statutory || o.Draft.Customer != d.Customer || o.Draft.Entity != d.Entity || o.Draft.Currency != d.Currency || o.Draft.Scale != d.Scale {
		return refuse(ReasonCreditMismatch)
	}
	limit, _ := amount(o.Draft.Total, o.Draft.Scale)
	used, _ := amount(d.Total, d.Scale)
	for _, number := range original.credits {
		credit := s.documents[number].view.Snapshot.Draft
		value, _ := amount(credit.Total, credit.Scale)
		used.Add(used, value)
	}
	if used.Cmp(limit) > 0 {
		return refuse(ReasonExceedsOriginal)
	}
	return nil
}

func (r *Registry) Submit(workspace string, d Draft, actor Actor) (DraftView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !actor.valid() || !draftForm(d) {
		return DraftView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return DraftView{}, err
	}
	s, k, err := r.lookupKind(workspace, d.Kind)
	if err != nil {
		return DraftView{}, err
	}
	if k.view.Kind.Credit != (d.Original != "") {
		return DraftView{}, ErrInvalid
	}
	existing := s.drafts[d.ID]
	if existing != nil {
		if actor.is(existing.current.Submitter) && sameDraft(existing.current.Draft, d) {
			return viewDraft(existing.current), nil
		}
		if existing.current.Draft.Kind != d.Kind {
			return DraftView{}, ErrConflict
		}
		if existing.document != nil {
			return DraftView{}, refuse(ReasonAlreadyIssued)
		}
	}
	if err := r.rules(workspace, s, k, d); err != nil {
		return DraftView{}, err
	}
	if existing == nil {
		if len(s.drafts) >= r.limits.Drafts {
			return DraftView{}, ErrLimit
		}
		existing = &draft{}
		s.drafts[d.ID] = existing
	}
	d = copyDraft(d)
	v := &DraftView{Draft: d, Revision: len(existing.revisions) + 1, Digest: draftDigest(workspace, d), Submitter: actor.ID, At: now}
	if existing.reserved != nil {
		v.Number = existing.reserved.Number
	}
	existing.revisions = append(existing.revisions, v)
	existing.current = v
	return viewDraft(v), nil
}

func (r *Registry) lookupDraft(workspace, id string) (*space, *draft, error) {
	s := r.space(workspace)
	if s == nil || s.drafts[id] == nil {
		return nil, nil, ErrNotFound
	}
	return s, s.drafts[id], nil
}

func (r *Registry) Approve(workspace, id, digest string, actor Actor) (DraftView, error) {
	return r.approve(workspace, id, digest, actor, nil)
}

func (r *Registry) approve(workspace, id, digest string, actor Actor, reviewed *reviewedApproval) (DraftView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validDigest(digest) || !actor.valid() {
		return DraftView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return DraftView{}, err
	}
	s, d, err := r.lookupDraft(workspace, id)
	if err != nil {
		return DraftView{}, err
	}
	v := d.current
	if d.document != nil {
		return DraftView{}, refuse(ReasonAlreadyIssued)
	}
	if v.Approver != "" && v.Digest == digest && !actor.is(v.Approver) {
		return DraftView{}, ErrConflict
	}
	switch {
	case v.Digest != digest:
		return DraftView{}, refuse(ReasonDigestChanged)
	case actor.ID == v.Submitter:
		return DraftView{}, refuse(ReasonSelfApproval)
	}
	k := s.kinds[v.Draft.Kind]
	refs, coverage, _, err := r.bind(workspace, k.view.Kind, v.Draft, now)
	if err != nil {
		return DraftView{}, err
	}
	if reviewed != nil {
		if err := reviewed.matches(refs, coverage); err != nil {
			return DraftView{}, err
		}
		if v.Approver != "" {
			if err := reviewed.matches(v.References, v.Coverage); err != nil {
				return DraftView{}, err
			}
			return viewDraft(v), nil
		}
	}
	if v.Approver == "" || !sameReferences(v.References, refs) || !sameCoverage(v.Coverage, coverage) {
		v.Approver, v.References, v.Coverage, v.At = actor.ID, refs, coverage, now
	}
	return viewDraft(v), nil
}

func (r *Registry) DraftOf(workspace, id string) (DraftView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) {
		return DraftView{}, ErrInvalid
	}
	_, d, err := r.lookupDraft(workspace, id)
	if err != nil {
		return DraftView{}, err
	}
	return viewDraft(d.current), nil
}

func (r *Registry) Reserve(workspace, id string, actor Actor) (NumberView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return NumberView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return NumberView{}, err
	}
	s, d, err := r.lookupDraft(workspace, id)
	if err != nil {
		return NumberView{}, err
	}
	if d.reserved != nil {
		if actor.is(d.reserved.Actor) && d.reserved.State == NumberReserved {
			return *d.reserved, nil
		}
		if d.reserved.State == NumberIssued {
			return NumberView{}, refuse(ReasonAlreadyIssued)
		}
		return NumberView{}, refuse(ReasonAlreadyReserved)
	}
	if d.document != nil {
		return NumberView{}, refuse(ReasonAlreadyIssued)
	}
	v, err := r.seriesFor(s, d.current.Draft.Kind)
	if err != nil {
		return NumberView{}, err
	}
	if s.kinds[d.current.Draft.Kind].view.Kind.Sequential {
		return NumberView{}, refuse(ReasonReservationsOff)
	}
	n, err := r.take(s, v, id, actor, now)
	if err != nil {
		return NumberView{}, err
	}
	d.reserved = n
	d.current.Number = n.Number
	return *n, nil
}

func (r *Registry) seriesFor(s *space, kindID string) (*series, error) {
	if s.kinds[kindID].view.Kind.Class == ClassERP {
		return nil, refuse(ReasonERPNumbered)
	}
	v := s.byKind[kindID]
	switch {
	case v == nil:
		return nil, refuse(ReasonNoSeries)
	case v.view.State != StateActive:
		return nil, refuse(ReasonSeriesPending)
	}
	return v, nil
}

var cancelReasons = map[string]bool{CancelAbandoned: true, CancelOperator: true, CancelOther: true}

func (r *Registry) Cancel(workspace, id, reason string, actor Actor) (NumberView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !cancelReasons[reason] || !actor.valid() {
		return NumberView{}, ErrInvalid
	}
	if err := actor.human(); err != nil {
		return NumberView{}, err
	}
	_, d, err := r.lookupDraft(workspace, id)
	if err != nil {
		return NumberView{}, err
	}
	n := d.reserved
	if n == nil {
		if last := len(d.cancelled) - 1; last >= 0 && actor.is(d.cancelled[last].Actor) && reason == d.cancelled[last].Reason {
			return *d.cancelled[last], nil
		}
		return NumberView{}, refuse(ReasonNotReserved)
	}
	if n.State == NumberIssued {
		return NumberView{}, refuse(ReasonAlreadyIssued)
	}
	n.State, n.Actor, n.At, n.Reason = NumberCancelled, actor.ID, now, reason
	d.reserved = nil
	d.current.Number = ""
	d.cancelled = append(d.cancelled, n)
	return *n, nil
}
