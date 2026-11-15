package disclosure

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"html"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type reference struct {
	recipient string
	value     string
}

func digest(texts ...string) string {
	sum := sha256.New()
	for _, text := range texts {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(text)))
		sum.Write(size[:])
		sum.Write([]byte(text))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func canonicalRow(row Row) string {
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	texts := []string{}
	for _, k := range keys {
		texts = append(texts, k, row[k])
	}
	return digest(texts...)
}

func encodeRow(values []string) string { return digest(values...) }

func (r *Registry) current(s *space, policyID, recipient string) (*PolicyView, *GrantView, error) {
	g := s.grants[recipient]
	if g == nil || g.active == nil && !g.revoked {
		return nil, nil, ErrNotFound
	}
	if g.revoked {
		return nil, nil, refuse(ReasonRecipientRevoked)
	}
	if g.active.Recipient.Policy != policyID {
		return nil, nil, refuse(ReasonPolicyNotGranted)
	}
	pol := s.policies[policyID]
	if pol == nil || pol.active == nil {
		return nil, nil, refuse(ReasonNoActivePolicy)
	}
	if pol.active.State == stateSuspended {
		return nil, nil, refuse(ReasonPolicySuspended)
	}
	return pol.active, g.active, nil
}

func (r *Registry) Project(workspace string, req Request, actor Actor) (Projection, error) {
	now := r.enter()
	defer r.leave()
	if len(req.Rows) > r.limits.Rows {
		return Projection{}, ErrLimit
	}
	if !validID(workspace) || !validID(req.Policy) || !validID(req.Recipient) || !actor.valid() {
		return Projection{}, ErrInvalid
	}
	for _, row := range req.Rows {
		if !validID(row[CustomerField]) {
			return Projection{}, ErrInvalid
		}
		for k, v := range row {
			if !validID(k) || !validValue(v) {
				return Projection{}, ErrInvalid
			}
		}
	}
	s, err := r.space(workspace, false)
	if err != nil {
		return Projection{}, err
	}
	if err := actor.notAssistant(); err != nil {
		return Projection{}, err
	}
	if err := r.staff(actor); err != nil {
		return Projection{}, err
	}
	pv, gv, err := r.current(s, req.Policy, req.Recipient)
	if err != nil {
		return Projection{}, err
	}
	key, err := r.config.Keys(workspace)
	if err != nil || len(key) == 0 {
		return Projection{}, refuse(ReasonKeyUnavailable)
	}
	fields := append([]string(nil), pv.Policy.Roles[gv.Recipient.Role]...)
	if len(fields) == 0 {
		return Projection{}, refuse(ReasonNoFields)
	}
	scope := map[string]bool{}
	for _, c := range gv.Recipient.Customers {
		scope[c] = true
	}
	kept := map[string]bool{}
	for _, f := range fields {
		kept[f] = true
	}
	dropped := map[string]bool{}
	var rows [][]string
	var inputs []string
	refs := map[string]reference{}
	droppedRows := 0
	for _, row := range req.Rows {
		inputs = append(inputs, canonicalRow(row))
		for k := range row {
			if !kept[k] {
				dropped[k] = true
			}
		}
		if !scope[row[CustomerField]] {
			droppedRows++
			continue
		}
		values := make([]string, len(fields))
		for i, f := range fields {
			v := row[f]
			if s.fields[f].Field.Locator && v != "" {
				mac := hmac.New(sha256.New, key)
				mac.Write([]byte(digest(req.Recipient, v)))
				ref := "ref:" + hex.EncodeToString(mac.Sum(nil))[:24]
				prior, ok := refs[ref]
				if !ok {
					prior, ok = s.refsFor(ref)
				}
				if ok && prior != (reference{req.Recipient, v}) {
					return Projection{}, refuse(ReasonReferenceCollision)
				}
				refs[ref] = reference{req.Recipient, v}
				v = ref
			}
			values[i] = v
		}
		rows = append(rows, values)
	}
	sort.Strings(inputs)
	input := digest(append([]string{inputLabel}, inputs...)...)
	sort.Slice(rows, func(i, j int) bool { return encodeRow(rows[i]) < encodeRow(rows[j]) })
	texts := []string{projectionLabel, workspace, req.Recipient, strconv.Itoa(gv.Revision), req.Policy, strconv.Itoa(pv.Revision), input, strconv.Itoa(len(fields))}
	texts = append(texts, fields...)
	for _, row := range rows {
		texts = append(texts, encodeRow(row))
	}
	d := digest(texts...)
	out := Projection{Digest: d, Fields: fields, Rows: rows}
	if s.projections[d] == nil {
		if len(s.projections) >= r.limits.Projections {
			return Projection{}, ErrLimit
		}
		var names []string
		for k := range dropped {
			names = append(names, k)
		}
		sort.Strings(names)
		s.projections[d] = &Audit{Projection: copyProjection(out), Workspace: workspace, Recipient: req.Recipient, Grant: gv.Revision, Policy: req.Policy, Revision: pv.Revision,
			Input: input, DroppedRows: droppedRows, DroppedFields: names, Actor: actor.ID, At: now}
		for ref, value := range refs {
			s.refs[ref] = value.recipient + "\x00" + value.value
		}
	}
	return copyProjection(out), nil
}

func (s *space) refsFor(ref string) (reference, bool) {
	stored, ok := s.refs[ref]
	if !ok {
		return reference{}, false
	}
	recipient, value, _ := strings.Cut(stored, "\x00")
	return reference{recipient, value}, true
}

func copyProjection(p Projection) Projection {
	p.Fields = append([]string(nil), p.Fields...)
	rows := make([][]string, len(p.Rows))
	for i, row := range p.Rows {
		rows[i] = append([]string(nil), row...)
	}
	p.Rows = rows
	return p
}

func copyAudit(a Audit) Audit {
	a.Projection = copyProjection(a.Projection)
	a.DroppedFields = append([]string(nil), a.DroppedFields...)
	return a
}

func (r *Registry) staffSpace(workspace string, actor Actor) (*space, error) {
	s, err := r.space(workspace, false)
	if err != nil {
		return nil, err
	}
	if err := actor.human(); err != nil {
		return nil, err
	}
	if err := r.staff(actor); err != nil {
		return nil, err
	}
	return s, nil
}

func (r *Registry) ProjectionOf(workspace, digest string, actor Actor) (Audit, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(digest) || !actor.valid() {
		return Audit{}, ErrInvalid
	}
	s, err := r.staffSpace(workspace, actor)
	if err != nil {
		return Audit{}, err
	}
	a := s.projections[digest]
	if a == nil {
		return Audit{}, ErrNotFound
	}
	return copyAudit(*a), nil
}

func (r *Registry) Resolve(workspace, ref string, actor Actor) (string, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !strings.HasPrefix(ref, "ref:") || !actor.valid() {
		return "", ErrInvalid
	}
	s, err := r.staffSpace(workspace, actor)
	if err != nil {
		return "", err
	}
	found, ok := s.refsFor(ref)
	if !ok {
		return "", ErrNotFound
	}
	return found.value, nil
}

func risky(text string) bool {
	if canonical(text) {
		return false
	}
	trimmed := strings.TrimLeft(text, " ")
	if trimmed == "" {
		return false
	}
	switch []rune(trimmed)[0] {
	case '=', '+', '-', '@', '\t', '\r', '\'', '＝', '＋', '－', '＠':
		return true
	}
	return false
}

func canonical(text string) bool {
	body := strings.TrimPrefix(text, "-")
	whole, fraction, dotted := strings.Cut(body, ".")
	if whole == "" || len(whole) > 18 || dotted && (fraction == "" || len(fraction) > 8) || len(whole) > 1 && whole[0] == '0' {
		return false
	}
	for _, c := range whole + fraction {
		if c < '0' || c > '9' {
			return false
		}
	}
	return !(strings.HasPrefix(text, "-") && strings.Trim(whole+fraction, "0") == "")
}

func neutral(text string) string {
	if risky(text) {
		return "'" + text
	}
	return text
}

func render(p Projection) []byte {
	var out bytes.Buffer
	out.WriteString("\ufeff")
	w := csv.NewWriter(&out)
	header := make([]string, len(p.Fields))
	for i, f := range p.Fields {
		header[i] = neutral(f)
	}
	w.Write(header)
	for _, row := range p.Rows {
		values := make([]string, len(row))
		for i, v := range row {
			values[i] = neutral(v)
		}
		w.Write(values)
	}
	w.Flush()
	return out.Bytes()
}

func (r *Registry) token(n int) (string, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(r.config.Random, b); err != nil {
		return "", ErrInvalid
	}
	return hex.EncodeToString(b), nil
}

func (r *Registry) Package(workspace, projection string, actor Actor) (PackageView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(projection) || !actor.valid() {
		return PackageView{}, ErrInvalid
	}
	s, err := r.space(workspace, false)
	if err != nil {
		return PackageView{}, err
	}
	a := s.projections[projection]
	if a == nil {
		return PackageView{}, ErrNotFound
	}
	if err := actor.notAssistant(); err != nil {
		return PackageView{}, err
	}
	if err := r.staff(actor); err != nil {
		return PackageView{}, err
	}
	if err := s.live(a.Policy, a.Revision, a.Recipient, a.Grant); err != nil {
		return PackageView{}, err
	}
	if len(s.packages) >= r.limits.Packages {
		return PackageView{}, ErrLimit
	}
	id, err := r.token(16)
	if err != nil {
		return PackageView{}, err
	}
	id = "pkg-" + id
	if s.packages[id] != nil {
		return PackageView{}, ErrConflict
	}
	body := render(a.Projection)
	sum := sha256.Sum256(body)
	view := &PackageView{ID: id, Projection: projection, Input: a.Input, Policy: a.Policy, Revision: a.Revision, Recipient: a.Recipient, Grant: a.Grant,
		File: id + ".csv", FileDigest: hex.EncodeToString(sum[:]), Bytes: body, Actor: actor.ID, At: now}
	s.packages[id] = view
	return copyPackage(*view), nil
}

func copyPackage(p PackageView) PackageView {
	p.Bytes = append([]byte(nil), p.Bytes...)
	return p
}

func (s *space) live(policyID string, revision int, recipient string, grantRevision int) error {
	pol := s.policies[policyID]
	v := pol.revisions[revision-1]
	if v.State == stateSuspended {
		return refuse(ReasonPolicySuspended)
	}
	if v.State != stateActive {
		return refuse(ReasonPolicySuperseded)
	}
	g := s.grants[recipient]
	if g.revoked {
		return refuse(ReasonRecipientRevoked)
	}
	if g.active == nil || g.active.Revision != grantRevision {
		return refuse(ReasonRecipientChanged)
	}
	return nil
}

func (r *Registry) Link(workspace, pkg string, lifetime time.Duration, actor Actor) (string, LinkView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(pkg) || lifetime < minLifetime || lifetime > maxLifetime || !actor.valid() {
		return "", LinkView{}, ErrInvalid
	}
	s, err := r.space(workspace, false)
	if err != nil {
		return "", LinkView{}, err
	}
	p := s.packages[pkg]
	if p == nil {
		return "", LinkView{}, ErrNotFound
	}
	if err := actor.notAssistant(); err != nil {
		return "", LinkView{}, err
	}
	if err := r.staff(actor); err != nil {
		return "", LinkView{}, err
	}
	if err := s.live(p.Policy, p.Revision, p.Recipient, p.Grant); err != nil {
		return "", LinkView{}, err
	}
	if s.linkCount >= r.limits.Links {
		return "", LinkView{}, ErrLimit
	}
	token, err := r.token(16)
	if err != nil {
		return "", LinkView{}, err
	}
	sum := sha256.Sum256([]byte(token))
	if _, taken := r.tokens[hex.EncodeToString(sum[:])]; taken {
		return "", LinkView{}, ErrConflict
	}
	view := &LinkView{Digest: hex.EncodeToString(sum[:]), Package: pkg, Expires: now.Add(lifetime), Issuer: actor.ID}
	s.links[view.Digest] = view
	s.linkCount++
	r.tokens[view.Digest] = linkRef{workspace: workspace, link: view}
	return token, *view, nil
}

func (r *Registry) RevokeLink(workspace, linkDigest string, actor Actor) (LinkView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(linkDigest) || !actor.valid() {
		return LinkView{}, ErrInvalid
	}
	s, err := r.staffSpace(workspace, actor)
	if err != nil {
		return LinkView{}, err
	}
	l := s.links[linkDigest]
	if l == nil {
		return LinkView{}, ErrNotFound
	}
	if l.RevokedBy != "" && l.RevokedBy != actor.ID {
		return LinkView{}, refuse(ReasonLinkRevoked)
	}
	l.RevokedBy = actor.ID
	return *l, nil
}

func (r *Registry) Access(token string, actor Actor) (Delivery, error) {
	now := r.enter()
	defer r.leave()
	if token == "" || len(token) > 64 || !actor.valid() {
		return Delivery{}, ErrInvalid
	}
	sum := sha256.Sum256([]byte(token))
	found, ok := r.tokens[hex.EncodeToString(sum[:])]
	if !ok {
		return Delivery{}, ErrNotFound
	}
	if err := actor.notAssistant(); err != nil {
		return Delivery{}, err
	}
	l := found.link
	s := r.workspaces[found.workspace]
	p := s.packages[l.Package]
	if actor.ID != p.Recipient {
		return Delivery{}, refuse(ReasonRecipientMismatch)
	}
	if l.RevokedBy != "" {
		return Delivery{}, refuse(ReasonLinkRevoked)
	}
	if !now.Before(l.Expires) {
		return Delivery{}, refuse(ReasonLinkExpired)
	}
	if err := s.live(p.Policy, p.Revision, p.Recipient, p.Grant); err != nil {
		return Delivery{}, err
	}
	return Delivery{ID: p.ID, File: p.File, FileDigest: p.FileDigest, Bytes: append([]byte(nil), p.Bytes...)}, nil
}

func safe(text string) string {
	var b strings.Builder
	for _, c := range text {
		if unicode.IsControl(c) || unicode.Is(unicode.Bidi_Control, c) || c == '‎' || c == '‏' || c == '؜' {
			b.WriteRune('�')
			continue
		}
		b.WriteRune(c)
	}
	return html.EscapeString(b.String())
}

func (r *Registry) Preview(workspace, projection string, page, size int, internal bool, actor Actor) (string, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(projection) || page < 1 || size < 1 || size > maxPage || !actor.valid() {
		return "", ErrInvalid
	}
	s, err := r.space(workspace, false)
	if err != nil {
		return "", err
	}
	a := s.projections[projection]
	if a == nil {
		return "", ErrNotFound
	}
	if err := actor.notAssistant(); err != nil {
		return "", err
	}
	if internal {
		if err := actor.human(); err != nil {
			return "", err
		}
		if err := r.staff(actor); err != nil {
			return "", err
		}
	} else if actor.ID != a.Recipient {
		if r.recipients[actor.ID] {
			return "", refuse(ReasonRecipientMismatch)
		}
		if err := actor.human(); err != nil {
			return "", err
		}
	} else if err := s.live(a.Policy, a.Revision, a.Recipient, a.Grant); err != nil {
		return "", err
	}
	p := a.Projection
	var b strings.Builder
	b.WriteString("<section><p>rows: " + strconv.Itoa(len(p.Rows)) + "</p><p>fields: " + safe(strings.Join(p.Fields, ", ")) + "</p>")
	if internal {
		b.WriteString("<p>dropped rows: " + strconv.Itoa(a.DroppedRows) + "</p><p>dropped fields: " + safe(strings.Join(a.DroppedFields, ", ")) + "</p>")
	}
	b.WriteString("<table><tr>")
	for _, f := range p.Fields {
		b.WriteString("<th>" + safe(f) + "</th>")
	}
	b.WriteString("</tr>")
	first := (page - 1) * size
	for i := first; i < len(p.Rows) && i < first+size; i++ {
		b.WriteString("<tr>")
		for _, v := range p.Rows[i] {
			b.WriteString("<td>" + safe(v) + "</td>")
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</table></section>")
	return b.String(), nil
}

func (r *Registry) Validity(workspace, pkg string) (Validity, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(pkg) {
		return Validity{}, ErrInvalid
	}
	s, err := r.space(workspace, false)
	if err != nil {
		return Validity{}, err
	}
	p := s.packages[pkg]
	if p == nil {
		return Validity{}, ErrNotFound
	}
	v := Validity{Package: p.ID, Recipient: p.Recipient, FileDigest: p.FileDigest, Policy: p.Policy, Revision: p.Revision, Grant: p.Grant, Live: true}
	if err := s.live(p.Policy, p.Revision, p.Recipient, p.Grant); err != nil {
		refusal, _ := err.(Refusal)
		v.Live, v.Reason = false, refusal.Reason
	}
	return v, nil
}

func (r *Registry) Recipient(id string) bool {
	r.enter()
	defer r.leave()
	return r.recipients[id]
}
