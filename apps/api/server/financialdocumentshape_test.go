package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/rating"
)

func TestDocumentCandidateShapeWithoutPacketDefaults(t *testing.T) {
	b, original, subject := documentFixtureValues(t)
	charges, sources, err := documentCandidateShape(original, subject, b.Connection)
	if err != nil || len(charges) != 2 || len(sources) != 2 {
		t.Fatal(err, charges, sources)
	}
	for name, change := range map[string]func(*chargeCandidate){
		"source duplicate": func(c *chargeCandidate) {
			c.Envelope.Input.Normalized.Rows[1].Key = c.Envelope.Input.Normalized.Rows[0].Key
		},
		"source missing":   func(c *chargeCandidate) { c.Envelope.Input.Normalized.Rows = c.Envelope.Input.Normalized.Rows[:1] },
		"zero":             func(c *chargeCandidate) { c.Envelope.Result.Charges[0].Amount = "0" },
		"negative":         func(c *chargeCandidate) { c.Envelope.Result.Charges[0].Amount = "-1" },
		"precision":        func(c *chargeCandidate) { c.Envelope.Result.Charges[0].Amount = "10.251" },
		"aggregate":        func(c *chargeCandidate) { c.Envelope.Result.Charges[0].Kind = rating.FlatCharge },
		"duplicate charge": func(c *chargeCandidate) { c.Envelope.Result.Charges[1] = c.Envelope.Result.Charges[0] },
		"period":           func(c *chargeCandidate) { c.Envelope.Input.From = "2026-10-02" },
		"month":            func(c *chargeCandidate) { c.Envelope.Input.Normalized.Binding.Period = "2026-11" },
		"total":            func(c *chargeCandidate) { c.Envelope.Result.Total = "16" },
		"currency":         func(c *chargeCandidate) { c.Envelope.Result.Currency = "KRW" },
	} {
		c := cloneDocumentValue(t, original)
		change(&c)
		if _, _, err := documentCandidateShape(c, subject, b.Connection); err == nil {
			t.Fatal("shape accepted", name)
		}
		if _, _, err := documentSubmission(b, c, subject, "shape-check"); err == nil {
			t.Fatal("submission accepted", name)
		}
	}
	b.Lines = nil
	if _, _, err := documentCandidateShape(original, subject, b.Connection); err != nil {
		t.Fatal("shape required packet inputs", err)
	}
}
