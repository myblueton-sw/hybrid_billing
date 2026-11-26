package server

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/rating"
)

func documentFixtureValues(t *testing.T) (financialDocumentBasis, chargeCandidate, approvals.Subject) {
	t.Helper()
	w := explicitDocumentWorld(t)
	c, err := w.f.s.chargeProducer.candidates.financialCandidate(SampleWorkspace, w.basis.Candidate)
	w.f.must(err)
	v, _, err := w.f.s.financialChargeRecord(SampleWorkspace, w.basis.Request)
	w.f.must(err)
	return w.basis, c, v.Subject
}

func cloneDocumentValue[T any](t *testing.T, v T) T {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out T
	if err = json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestFinancialDocumentRequiresExplicitCompleteEvidence(t *testing.T) {
	basis, candidate, subject := documentFixtureValues(t)
	for _, mode := range []string{"quantity", "tax", "unit", "negative-tax", "zero-quantity", "precision", "net-total", "tax-total", "duplicate", "missing", "origin", "charge", "aggregate", "currency", "path", "profile", "overflow", "document"} {
		t.Run(mode, func(t *testing.T) {
			b, c, s := cloneDocumentValue(t, basis), cloneDocumentValue(t, candidate), cloneDocumentValue(t, subject)
			switch mode {
			case "quantity":
				b.Lines[0].Quantity = ""
			case "tax":
				b.Lines[0].Tax = ""
			case "unit":
				b.Lines[0].Unit = ""
			case "negative-tax":
				b.Lines[0].Tax = "-1"
			case "zero-quantity":
				b.Lines[0].Quantity = "0"
			case "precision":
				b.Lines[0].Tax = "0.001"
			case "net-total":
				b.ExpectedNet = "16"
			case "tax-total":
				b.ExpectedTax = "2"
			case "duplicate":
				b.Lines[1] = b.Lines[0]
			case "missing":
				b.Lines = b.Lines[:1]
			case "origin":
				c.Envelope.Input.Normalized.Rows[1] = c.Envelope.Input.Normalized.Rows[0]
			case "charge":
				c.Envelope.Result.Charges[1] = c.Envelope.Result.Charges[0]
			case "aggregate":
				c.Envelope.Result.Charges[0].Kind = rating.FlatCharge
				b.Lines[0].Kind = rating.FlatCharge
			case "currency":
				c.Envelope.Input.Lines[0].Currency = "KRW"
			case "path":
				b.Path = "inferred"
			case "profile":
				b.Profile = "unknown"
			case "overflow":
				b.Lines[0].Quantity = "999999999999999999999999999999"
			case "document":
				b.Document.PayerID = "changed"
			}
			if _, _, err := documentSubmission(b, c, s, "prepare"); err == nil {
				t.Fatal("invalid evidence accepted", mode)
			}
		})
	}
}

func centsText(n int64, canonical bool) string {
	s := fmt.Sprintf("%d.%02d", n/100, n%100)
	if canonical {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

func TestFinancialDocumentExactTotalsAndStableSourceProperties(t *testing.T) {
	basis, candidate, subject := documentFixtureValues(t)
	random := rand.New(rand.NewSource(340))
	for iteration := 0; iteration < 1000; iteration++ {
		b, c, s := cloneDocumentValue(t, basis), cloneDocumentValue(t, candidate), cloneDocumentValue(t, subject)
		net, tax, quantity := int64(0), int64(0), int64(0)
		for i := range b.Lines {
			n, x, q := random.Int63n(1000000)+1, random.Int63n(100000), random.Int63n(100)+1
			c.Envelope.Result.Charges[i].Amount = centsText(n, true)
			b.Lines[i].Tax = centsText(x, true)
			b.Lines[i].Quantity = strconv.FormatInt(q, 10)
			net += n
			tax += x
			quantity += q
		}
		s.Amount = centsText(net, true)
		c.Envelope.Result.Total = s.Amount
		b.ExpectedNet = s.Amount
		b.ExpectedTax = centsText(tax, true)
		sub, total, err := documentSubmission(b, c, s, "first")
		if err != nil {
			t.Fatal(iteration, err)
		}
		if total.Net != centsText(net, false) || total.Tax != centsText(tax, false) || total.Quantity != fmt.Sprintf("%d.00", quantity) {
			t.Fatal(iteration, total)
		}
		b.Lines[0], b.Lines[1] = b.Lines[1], b.Lines[0]
		permuted, again, err := documentSubmission(b, c, s, "second")
		if err != nil || again != total || permuted.Lines[0].Source != sub.Lines[1].Source || permuted.Lines[1].Source != sub.Lines[0].Source {
			t.Fatal("permutation", iteration, err)
		}
		c.Envelope.Input.ID = "recreated-candidate"
		b.Candidate = c.Envelope.Input.ID
		s.ID = "recreated-request"
		b.Request = s.ID
		for i := range c.Envelope.Input.Lines {
			old := c.Envelope.Input.Lines[i].ID
			next := "recreated-" + strconv.Itoa(i)
			c.Envelope.Input.Lines[i].ID = next
			for j := range c.Envelope.Input.Normalized.Rows {
				if c.Envelope.Input.Normalized.Rows[j].Line == old {
					c.Envelope.Input.Normalized.Rows[j].Line = next
				}
			}
			for j := range c.Envelope.Result.Charges {
				if c.Envelope.Result.Charges[j].Line == old {
					c.Envelope.Result.Charges[j].Line = next
				}
			}
			for j := range b.Lines {
				if b.Lines[j].Line == old {
					b.Lines[j].Line = next
				}
			}
		}
		recreated, _, err := documentSubmission(b, c, s, "third")
		if err != nil || recreated.Lines[0].Source != permuted.Lines[0].Source || recreated.Lines[1].Source != permuted.Lines[1].Source {
			t.Fatal("unstable economic identity", iteration, err)
		}
	}
}
