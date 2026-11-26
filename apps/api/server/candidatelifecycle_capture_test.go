package server

import (
	"bytes"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
)

func TestNormalizedLifecycleAmendmentAndHistoricalCapture(t *testing.T) {
	p, b := normalizedFixture(t, false, nil)
	first, err := p.capture("workspace", "normalized", "first", b)
	if err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), p.candidates.records[[2]string{"workspace", "first"}].envelope...)
	p.domain.Lock()
	_, err = p.lifecycles.Amend("workspace", "contract", 1, "pause-write", nil, []contracts.Event{{ID: "pause", Kind: contracts.Suspend, From: contractDay(2026, 10, 15)}}, "")
	p.domain.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.capture("workspace", "normalized", "next", b); err == nil {
		t.Fatal("suspended interval captured")
	}
	if len(p.candidates.records) != 1 {
		t.Fatal("failed coverage reserved an ID")
	}
	history, err := p.candidates.read("workspace", "first")
	if err != nil || !reflect.DeepEqual(first, history) {
		t.Fatal("history followed current lifecycle", err)
	}
	p.domain.Lock()
	_, err = p.lifecycles.Amend("workspace", "contract", 2, "withdraw-pause", []string{"pause"}, nil, "")
	p.domain.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	next, err := p.capture("workspace", "normalized", "next", b)
	if err != nil || next.Envelope.Input.Normalized.Lifecycle.Sequence != 3 {
		t.Fatal("ID not reusable after corrected history", err)
	}
	if _, err := p.capture("workspace", "normalized", "first", b); !errors.Is(err, errCandidateConflict) {
		t.Fatal("changed provenance replayed old ID", err)
	}
	if !bytes.Equal(original, p.candidates.records[[2]string{"workspace", "first"}].envelope) {
		t.Fatal("historical bytes changed")
	}
	next.Envelope.Input.Normalized.Lifecycle.Segments[0].Revision = 99
	fresh, err := p.candidates.read("workspace", "next")
	if err != nil || fresh.Envelope.Input.Normalized.Lifecycle.Segments[0].Revision != 1 {
		t.Fatal("returned provenance aliases storage", err)
	}
}

func TestNormalizedLifecycleJournalSequenceBindsDigest(t *testing.T) {
	p, b := normalizedFixture(t, false, nil)
	q, other := normalizedFixture(t, false, nil)
	first, err := p.capture("workspace", "normalized", "same", b)
	if err != nil {
		t.Fatal(err)
	}
	q.domain.Lock()
	_, err = q.lifecycles.RecordEvidence("workspace", "contract", 1, "gap", contracts.Evidence{Kind: contracts.Missing, Reason: contracts.SourceUnavailable, OwnerID: "owner", FollowUp: "follow-up"})
	q.domain.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	second, err := q.capture("workspace", "normalized", "same", other)
	if err != nil {
		t.Fatal(err)
	}
	if first.Envelope.Input.Normalized.Lifecycle.Sequence != 1 || second.Envelope.Input.Normalized.Lifecycle.Sequence != 2 || first.Digest == second.Digest || first.Envelope.Input.Sources[0].Digest == second.Envelope.Input.Sources[0].Digest {
		t.Fatal("journal sequence absent from digest")
	}
	if !reflect.DeepEqual(first.Envelope.Result, second.Envelope.Result) {
		t.Fatal("evidence write changed rating result")
	}
	second.Envelope.Input.Normalized.Lifecycle.Sequence = 1
	second.Envelope.Input.Sources[0].Digest = first.Envelope.Input.Sources[0].Digest
	if !reflect.DeepEqual(first.Envelope.Input, second.Envelope.Input) {
		t.Fatal("unexpected differing inputs")
	}
}

func TestCandidateReadCannotPublish(t *testing.T) {
	f := setup(t, SeedSample).ready()
	defer f.s.Close()
	root := f.as(AdminLogin)
	count, counter := len(f.s.chargeProducer.candidates.records), f.s.counter
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		for _, path := range []string{"/api/v1/charge-candidates", "/api/v1/charge-candidates/sample-charge-01"} {
			got := f.do(method, path, `{}`, func(r *http.Request) {
				r.AddCookie(root.cookie)
				r.Header.Set(csrfHeader, root.csrf)
				r.Header.Set("Origin", "http://127.0.0.1:5173")
			})
			if got.status != 404 || got.body["code"] != "unsupported_command" {
				t.Fatalf("%s %s: %d %s", method, path, got.status, got.raw)
			}
		}
	}
	if len(f.s.chargeProducer.candidates.records) != count || f.s.counter != counter {
		t.Fatal("read routes published changes")
	}
}
