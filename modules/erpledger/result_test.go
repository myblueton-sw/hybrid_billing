package erpledger

import (
	"reflect"
	"strconv"
	"sync"
	"testing"
)

func resultFixture(t *testing.T) fixture {
	t.Helper()
	f := newFixture(t)
	must(f.r.Prepare("w1", invoice("key", "api", line("line", "source", "10")), human("biller")))
	return f
}

func resultDocument(f fixture, external string, revision int, net, quantity, product string) DocumentView {
	return must(f.r.Document("w1", Document{Connection: "api", External: external, Revision: revision, Rows: []Row{{ID: "row", Product: product, Net: net, Tax: "0", Quantity: quantity}}}, system("synthetic-result")))
}

func resultLink(f fixture, external, key, line, net, quantity string) DocumentView {
	return must(f.r.Link("w1", Link{Connection: "api", External: external, Row: "row", Key: key, Line: line, Net: net, Tax: "0", Quantity: quantity}, human("reconciler")))
}

func TestResultOfIdentityAndEmpty(t *testing.T) {
	f := resultFixture(t)
	for _, pair := range [][2]string{{"", "key"}, {"w1", "bad key"}, {"other", "key"}, {"w1", "absent"}} {
		expect(t, "missing identity", second(f.r.ResultOf(pair[0], pair[1])), "not_found")
	}
	before := must(f.r.SubmissionOf("w1", "key"))
	v := must(f.r.ResultOf("w1", "key"))
	if !reflect.DeepEqual(v.Submission, before) || v.Reconciliation.Matched || len(v.Documents) != 0 || len(v.Reconciliation.Documents) != 0 || v.Reconciliation.Lines[0].Linked.Net != "0" {
		t.Fatal("unlinked preparation presented as result")
	}
	if !reflect.DeepEqual(v.Reconciliation, must(f.r.Reconcile("w1", "key"))) {
		t.Fatal("reconciliation changed")
	}
	if !reflect.DeepEqual(before, must(f.r.SubmissionOf("w1", "key"))) {
		t.Fatal("read changed submission")
	}
}

func TestResultOfPartialCompleteAndChangedEvidence(t *testing.T) {
	f := resultFixture(t)
	resultDocument(f, "z", 1, "6", "1", "P-1")
	resultLink(f, "z", "key", "line", "6", "1")
	partial := must(f.r.ResultOf("w1", "key"))
	if partial.Reconciliation.Matched || partial.Reconciliation.Lines[0].Linked.Net != "6" {
		t.Fatal("partial result lost")
	}
	resultDocument(f, "a", 1, "4", "0", "P-1")
	resultLink(f, "a", "key", "line", "4", "0")
	complete := must(f.r.ResultOf("w1", "key"))
	if !complete.Reconciliation.Matched || len(complete.Documents) != 2 || complete.Documents[0].Document.External != "a" || complete.Documents[1].Document.External != "z" {
		t.Fatal("complete sorted evidence lost")
	}
	resultDocument(f, "z", 2, "6", "1", "P-2")
	changed := must(f.r.ResultOf("w1", "key"))
	if !changed.Reconciliation.Matched || changed.Documents[1].Document.Revision != 2 || changed.Documents[1].Document.Rows[0].Product != "P-2" {
		t.Fatal("equal totals concealed changed result")
	}
	if complete.Documents[1].Document.Revision != 1 || complete.Documents[1].Document.Rows[0].Product != "P-1" {
		t.Fatal("old snapshot rewritten")
	}
	resultDocument(f, "z", 3, "5", "1", "P-2")
	stale := must(f.r.ResultOf("w1", "key"))
	if stale.Reconciliation.Matched || len(stale.Documents) != 2 || !stale.Documents[1].Links[0].Stale || stale.Reconciliation.Documents[1].Attributed.Net != "0" {
		t.Fatal("stale evidence concealed")
	}
	if !reflect.DeepEqual(stale.Reconciliation, must(f.r.Reconcile("w1", "key"))) {
		t.Fatal("legacy reconciliation changed")
	}
}

func TestResultOfMixedAttributionAndCopies(t *testing.T) {
	f := resultFixture(t)
	must(f.r.Prepare("w1", invoice("other", "api", line("other-line", "other-source", "7")), human("biller")))
	resultDocument(f, "doc", 1, "17", "2", "P-1")
	resultLink(f, "doc", "key", "line", "10", "1")
	resultLink(f, "doc", "other", "other-line", "7", "1")
	original := must(f.r.ResultOf("w1", "key"))
	if !original.Reconciliation.Matched || original.Reconciliation.Documents[0].Attributed.Net != "17" || len(original.Documents[0].Links) != 2 {
		t.Fatal("mixed attribution hidden")
	}
	changed := must(f.r.ResultOf("w1", "key"))
	changed.Submission.Submission.Lines[0].Net = "999"
	changed.Submission.Mappings[0] = "changed"
	changed.Documents[0].Document.Rows[0].Net = "999"
	changed.Documents[0].Links[0].Link.Net = "999"
	changed.Reconciliation.Lines[0].Linked.Net = "999"
	changed.Reconciliation.Documents[0].Total.Net = "999"
	if !reflect.DeepEqual(original, must(f.r.ResultOf("w1", "key"))) {
		t.Fatal("mutable returned alias")
	}
	if len(original.Submission.Commands) != 0 || original.Submission.State != StatePrepared {
		t.Fatal("result read performed command")
	}
}

func TestResultOfConcurrentCoherence(t *testing.T) {
	f := resultFixture(t)
	resultDocument(f, "doc", 1, "10", "1", "P-1")
	resultLink(f, "doc", "key", "line", "10", "1")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for revision := 2; revision <= 100; revision++ {
			if _, err := f.r.Document("w1", Document{Connection: "api", External: "doc", Revision: revision, Rows: []Row{{ID: "row", Product: "P-" + strconv.Itoa(revision), Net: strconv.Itoa(revision + 10), Tax: "0", Quantity: "1"}}}, system("synthetic-result")); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for i := 0; i < 200; i++ {
		v, err := f.r.ResultOf("w1", "key")
		if err != nil {
			t.Fatal(err)
		}
		d := v.Documents[0]
		if d.Document.Rows[0].Net != d.Total.Net || d.Total != v.Reconciliation.Documents[0].Total || d.Document.Rows[0].Product != "P-"+strconv.Itoa(d.Document.Revision) || !v.Reconciliation.Matched || v.Reconciliation.Lines[0].Linked.Net != "10" || v.Reconciliation.Documents[0].Attributed.Net != "10" {
			t.Fatal("mixed snapshot")
		}
	}
	wg.Wait()
}

func TestResultOfLifecycleCopies(t *testing.T) {
	f := newFixture(t)
	f.send(t, invoice("key", "api", line("line", "source", "10")))
	f.observe(t, "api", "key", "synthetic-document", 1, StatePosted, SourceWebhook)
	f.observe(t, "api", "key", "synthetic-document", 2, StateRejected, SourceWebhook)
	must(f.r.Resolve("w1", "key", Resolution{Outcome: ResolveKeep, Evidence: "synthetic-resolution"}, human("lead")))
	before := must(f.r.SubmissionOf("w1", "key"))
	v := must(f.r.ResultOf("w1", "key"))
	if v.Submission.Watch == nil || v.Submission.Resolution == nil || len(v.Submission.Commands) == 0 {
		t.Fatal("lifecycle fixture incomplete")
	}
	v.Submission.Watch.Owner = "changed"
	v.Submission.Resolution.Resolution.Evidence = "changed"
	v.Submission.Commands[0].Actor = "changed"
	if !reflect.DeepEqual(before, must(f.r.ResultOf("w1", "key")).Submission) {
		t.Fatal("lifecycle aliases or read mutation")
	}
}
