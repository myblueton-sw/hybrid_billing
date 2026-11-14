package cases

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

func (p printer) hex() string { return hex.EncodeToString(p.sum.Sum(nil)) }

func stamp(at time.Time) string { return at.UTC().Format(time.RFC3339Nano) }

func findingDigest(workspace string, f Finding, author string) string {
	p := newPrinter(findingLabel)
	t := f.Target
	p.fields(workspace, f.ID, f.Kind, t.Customer, t.Contract, t.Document, t.Line, t.Resource, f.PeriodStart, f.PeriodEnd, f.Meaning, f.Amount, f.Currency, f.Evidence, f.Severity, f.Owner, f.Source, f.Detector, author)
	return p.hex()
}

func stepDigest(workspace, finding string, s Step) string {
	p := newPrinter(stepLabel)
	p.fields(workspace, finding, s.ID, s.Kind, s.Reason, s.Evidence, s.Owner, s.Remedy.Kind, s.Remedy.Reference, s.Actor, s.From, s.To, stamp(s.At))
	return p.hex()
}

func caseDigest(workspace string, c Case, actor Actor) string {
	p := newPrinter(caseLabel)
	p.fields(workspace, c.ID, c.Kind, c.Customer, c.Topic, c.Amount, c.Currency, c.Body, actor.ID, actor.Side, strconv.Itoa(len(c.Subjects)))
	for _, subject := range c.Subjects {
		p.fields(subject.Document, subject.Line)
	}
	return p.hex()
}

func moveDigest(workspace, customer, id, kind, step, actor string, at time.Time, values ...string) string {
	p := newPrinter(moveLabel)
	p.fields(workspace, customer, id, kind, step, actor, stamp(at))
	p.fields(values...)
	return p.hex()
}

func entryDigest(workspace, customer, id string, e EntryView) string {
	p := newPrinter(entryLabel)
	p.fields(workspace, customer, id, e.ID, e.Kind, e.Body, e.Actor, e.Side, stamp(e.At))
	return p.hex()
}
