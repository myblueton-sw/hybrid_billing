package issuance

import (
	"crypto/sha256"
	"encoding/hex"
	"html"
	"strings"
)

func deepSnapshot(v Snapshot) Snapshot {
	v = copySnapshot(v)
	if v.Draft.Detail != nil {
		detail := *v.Draft.Detail
		v.Draft.Detail = &detail
	}
	return v
}

func (r *Registry) begin(workspace, id string, actor Actor) (*document, Snapshot, int, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return nil, Snapshot{}, 0, ErrInvalid
	}
	if err := actor.notAssistant(); err != nil {
		return nil, Snapshot{}, 0, err
	}
	s := r.space(workspace)
	if s == nil || s.documents[id] == nil {
		return nil, Snapshot{}, 0, ErrNotFound
	}
	doc := s.documents[id]
	if doc.view.Snapshot.External {
		return nil, Snapshot{}, 0, refuse(ReasonNotSelfIssued)
	}
	if doc.running > 0 && now.Sub(doc.view.Attempts[doc.running-1].Started) < r.limits.Lease {
		return nil, Snapshot{}, 0, refuse(ReasonRenderInProgress)
	}
	if len(doc.view.Attempts) >= r.limits.Attempts {
		return nil, Snapshot{}, 0, ErrLimit
	}
	if doc.running > 0 {
		running := &doc.view.Attempts[doc.running-1]
		running.Outcome, running.Finished = ReasonAttemptSuperseded, now
	}
	doc.token++
	doc.view.Attempts = append(doc.view.Attempts, Attempt{Number: len(doc.view.Attempts) + 1, Actor: actor.ID, Started: now})
	doc.running = len(doc.view.Attempts)
	if doc.view.File == FileNone {
		doc.view.File = FileRendering
	}
	return doc, deepSnapshot(doc.view.Snapshot), doc.token, nil
}

func (r *Registry) call(v Snapshot) (out []byte, failed bool) {
	defer func() {
		if recover() != nil {
			out, failed = nil, true
		}
	}()
	out, err := r.config.Renderer.Render(v)
	return out, err != nil
}

func digestOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (r *Registry) finish(doc *document, token int, out []byte, failed bool) (DocumentView, error) {
	out = append([]byte(nil), out...)
	now := r.enter()
	defer r.leave()
	if doc.token != token {
		return DocumentView{}, refuse(ReasonAttemptSuperseded)
	}
	attempt := &doc.view.Attempts[doc.running-1]
	doc.running = 0
	attempt.Finished = now
	reason := ""
	if failed {
		reason = ReasonRenderFailed
	} else {
		reason = inspect(out)
	}
	digest := ""
	if reason == "" {
		digest = digestOf(out)
		if doc.view.Digest != "" && digest != doc.view.Digest {
			reason = ReasonDigestMismatch
		}
	}
	if reason != "" {
		attempt.Outcome, attempt.Digest = reason, digest
		if doc.view.File == FileRendering {
			doc.view.File = FileNone
		}
		return DocumentView{}, refuse(reason)
	}
	attempt.Digest = digest
	if doc.view.Digest == "" {
		attempt.Outcome = OutcomeAccepted
		doc.file = out
		doc.view.File, doc.view.Digest, doc.view.Size = FileRendered, digest, len(out)
	} else {
		attempt.Outcome = OutcomeVerified
	}
	return viewDocument(doc), nil
}

func (r *Registry) Render(workspace, id string, actor Actor) (DocumentView, error) {
	doc, snap, token, err := r.begin(workspace, id, actor)
	if err != nil {
		return DocumentView{}, err
	}
	out, failed := r.call(snap)
	return r.finish(doc, token, out, failed)
}

func (r *Registry) rendered(workspace, id string, actor Actor) (*document, error) {
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return nil, ErrInvalid
	}
	if err := actor.notAssistant(); err != nil {
		return nil, err
	}
	s := r.space(workspace)
	if s == nil || s.documents[id] == nil {
		return nil, ErrNotFound
	}
	return s.documents[id], nil
}

func (r *Registry) Download(workspace, id string, actor Actor) ([]byte, error) {
	now := r.enter()
	defer r.leave()
	doc, err := r.rendered(workspace, id, actor)
	if err != nil {
		return nil, err
	}
	switch {
	case doc.view.Digest == "":
		return nil, refuse(ReasonNotRendered)
	case digestOf(doc.file) != doc.view.Digest:
		return nil, refuse(ReasonDigestMismatch)
	case len(doc.view.Downloads) >= r.limits.Downloads:
		return nil, ErrLimit
	}
	doc.view.Downloads = append(doc.view.Downloads, Download{Actor: actor.ID, Kind: actor.Kind, At: now})
	return append([]byte(nil), doc.file...), nil
}

func cell(b *strings.Builder, tag, text string) {
	b.WriteString("<" + tag + "><bdi>" + html.EscapeString(text) + "</bdi></" + tag + ">")
}

func (r *Registry) Preview(workspace, id string, actor Actor) (string, error) {
	r.enter()
	defer r.leave()
	doc, err := r.rendered(workspace, id, actor)
	if err != nil {
		return "", err
	}
	v := doc.view.Snapshot
	var b strings.Builder
	b.WriteString("<article><p>Viewing copy. The issued record is the snapshot ")
	b.WriteString(v.Digest)
	b.WriteString(".</p>")
	if !v.Statutory {
		b.WriteString("<p>Not a statutory document.</p>")
	}
	due := v.Due.EffectiveDue
	if v.Due.Status == DueNotApplicable {
		due = "Not applicable"
	}
	b.WriteString("<dl>")
	for _, pair := range [][2]string{{"Number", v.Number}, {"Customer", v.Draft.Customer}, {"Contract", v.Draft.Contract}, {"Currency", v.Draft.Currency},
		{"Issue date", v.Due.IssueDate}, {"Due date", due}, {"Total", v.Draft.Total}} {
		cell(&b, "dt", pair[0])
		cell(&b, "dd", pair[1])
	}
	b.WriteString("</dl><table><tr><th>Item</th><th>Amount</th></tr>")
	for _, line := range v.Draft.Lines {
		b.WriteString("<tr>")
		cell(&b, "td", line.Text)
		cell(&b, "td", line.Amount)
		b.WriteString("</tr>")
	}
	b.WriteString("</table>")
	if v.Draft.Detail != nil {
		b.WriteString("<p>Full detail is attached separately.</p>")
	}
	b.WriteString("</article>")
	return b.String(), nil
}
