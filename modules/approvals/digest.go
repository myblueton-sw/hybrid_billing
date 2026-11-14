package approvals

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"strconv"
	"time"
)

type printer struct{ sum hash.Hash }

func newPrinter(label string) printer {
	p := printer{sum: sha256.New()}
	p.field(label)
	return p
}

func (p printer) field(text string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(text)))
	p.sum.Write(size[:])
	p.sum.Write([]byte(text))
}

func (p printer) fields(texts ...string) {
	for _, text := range texts {
		p.field(text)
	}
}

func (p printer) digests(m map[string]string) {
	keys := sortedKeys(m)
	p.field(strconv.Itoa(len(keys)))
	for _, k := range keys {
		p.fields(k, m[k])
	}
}

func (p printer) hex() string { return hex.EncodeToString(p.sum.Sum(nil)) }

func stamp(at time.Time) string { return at.UTC().Format(time.RFC3339Nano) }

func subjectDigest(workspace string, s Subject, policy int) string {
	p := newPrinter(subjectLabel)
	p.fields(workspace, s.ID, s.Stage, s.Target, s.Author, s.Amount, s.Currency, strconv.Itoa(policy))
	p.digests(s.Digests)
	p.field(strconv.Itoa(len(s.Entries)))
	for _, e := range s.Entries {
		p.fields(e.Item, e.Scope, e.Amount, e.Digest)
	}
	return p.hex()
}

func decisionDigest(workspace, request string, d DecisionView) string {
	p := newPrinter(decisionLabel)
	p.fields(workspace, request, d.Decision.Verdict, d.Decision.Reason, d.Actor, stamp(d.At), strconv.Itoa(len(d.Decision.Exclude)))
	for _, x := range d.Decision.Exclude {
		p.fields(x.Item, x.Reason)
	}
	return p.hex()
}

func executionDigest(workspace string, e Execution, actor string, at time.Time) string {
	p := newPrinter(executionLabel)
	p.fields(workspace, e.ID, e.Request, e.Stage, e.Target, e.Item, e.ItemDigest, e.Amount, e.Currency, actor, stamp(at))
	p.digests(e.Digests)
	return p.hex()
}

func proposalDigest(workspace string, x Proposal, actor string) string {
	p := newPrinter(proposalLabel)
	a := x.Authority
	p.fields(workspace, x.ID, x.Kind, x.Admin, a.ID, a.Approver, a.Stage, a.Currency, a.Limit, a.From, a.To)
	p.fields(x.Policy.Stage, strconv.Itoa(x.Policy.Revision), strconv.Itoa(x.Policy.Approvals), x.Policy.Validity.String(), strconv.Itoa(len(x.Policy.Required)))
	p.fields(x.Policy.Required...)
	p.field(actor)
	return p.hex()
}
