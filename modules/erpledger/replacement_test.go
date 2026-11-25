package erpledger

import (
	"reflect"
	"sync"
	"testing"
)

func replacementFixture(t *testing.T) (fixture, ResultView, Document, []Link) {
	t.Helper()
	f := resultFixture(t)
	resultDocument(f, "document", 1, "10", "1", "P-1")
	resultLink(f, "document", "key", "line", "10", "1")
	before := must(f.r.ResultOf("w1", "key"))
	next := Document{Connection: "api", External: "document", Revision: 2, Rows: []Row{{ID: "next-row", Product: "P-1", Net: "10", Tax: "0", Quantity: "1"}}}
	links := []Link{{Connection: "api", External: "document", Row: "next-row", Key: "key", Line: "line", Net: "10", Tax: "0", Quantity: "1"}}
	return f, before, next, links
}

func TestReplaceResultCompleteHistoryAndCopies(t *testing.T) {
	f, before, next, links := replacementFixture(t)
	got := must(f.r.ReplaceResult("w1", "key", before, next, links, human("replacement-actor")))
	if !got.Reconciliation.Matched || got.Documents[0].Document.Revision != 2 || len(got.Documents[0].Links) != 2 || !got.Documents[0].Links[0].Stale || got.Documents[0].Links[1].Stale || got.Documents[0].Links[1].Author != "replacement-actor" || !reflect.DeepEqual(got.Submission, before.Submission) {
		t.Fatal("replacement lost complete provenance")
	}
	old := before.Documents[0].Links[0]
	old.Stale = true
	if got.Documents[0].Links[0] != old || before.Documents[0].Links[0].Stale {
		t.Fatal("predecessor history changed")
	}
	next.Rows[0].Product = "changed"
	links[0].Net = "9"
	got.Documents[0].Document.Rows[0].Product = "changed"
	got.Documents[0].Links[1].Link.Net = "9"
	retained := must(f.r.ResultOf("w1", "key"))
	if retained.Documents[0].Document.Rows[0].Product != "P-1" || retained.Documents[0].Links[1].Link.Net != "10" {
		t.Fatal("replacement retained caller alias")
	}
	expect(t, "no automatic replay", second(f.r.ReplaceResult("w1", "key", before, retained.Documents[0].Document, []Link{retained.Documents[0].Links[1].Link}, human("replacement-actor"))), "conflict")
}

func TestReplaceResultRejectionsDoNotWrite(t *testing.T) {
	for _, scenario := range []string{"stale-cas", "submission-cas", "product", "partial", "duplicate", "revision", "scope", "assistant", "system", "capacity", "mapping", "state"} {
		t.Run(scenario, func(t *testing.T) {
			f, before, next, links := replacementFixture(t)
			actor := human("replacement-actor")
			switch scenario {
			case "stale-cas":
				before.Documents[0].Author = "foreign"
			case "submission-cas":
				before.Submission.Author = "foreign"
			case "product":
				next.Rows[0].Product = "P-2"
			case "partial":
				links[0].Net = "9"
			case "duplicate":
				links = append(links, links[0])
			case "revision":
				next.Revision = 1
			case "scope":
				next.External = "other"
			case "assistant":
				actor = assistant("assistant")
			case "system":
				actor = system("system")
			case "capacity":
				f.r.limits.Links = 1
			case "mapping":
				must(f.r.Map("w1", Mapping{"api", "product", "compute", "P-9"}, human("admin")))
			case "state":
				must(f.r.Command("w1", "key", CommandSubmit, human("biller")))
				before = must(f.r.ResultOf("w1", "key"))
			}
			unchanged := must(f.r.ResultOf("w1", "key"))
			if _, err := f.r.ReplaceResult("w1", "key", before, next, links, actor); err == nil {
				t.Fatal("invalid replacement accepted")
			}
			if !reflect.DeepEqual(unchanged, must(f.r.ResultOf("w1", "key"))) {
				t.Fatal("rejected replacement wrote state")
			}
		})
	}
}

func TestReplaceResultRepeatedHistoryAndCapacity(t *testing.T) {
	f, before, next, links := replacementFixture(t)
	for revision := 2; revision <= 6; revision++ {
		next.Revision = revision
		before = must(f.r.ReplaceResult("w1", "key", before, next, links, human("replacement-actor")))
		if len(before.Documents[0].Links) != revision {
			t.Fatal("history discarded")
		}
		for i, link := range before.Documents[0].Links {
			if link.Stale != (i < revision-1) {
				t.Fatal("unexpected live history")
			}
		}
	}
	next.Revision = 7
	expect(t, "cumulative capacity", second(f.r.ReplaceResult("w1", "key", before, next, links, human("replacement-actor"))), "limit")
	if !reflect.DeepEqual(before, must(f.r.ResultOf("w1", "key"))) {
		t.Fatal("capacity refusal wrote")
	}
}

func TestReplaceResultConcurrentCASAndSnapshots(t *testing.T) {
	f, before, next, links := replacementFixture(t)
	var group sync.WaitGroup
	outcomes := make(chan error, 2)
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := f.r.ReplaceResult("w1", "key", before, next, links, human("replacement-actor"))
			outcomes <- err
		}()
	}
	for i := 0; i < 100; i++ {
		observed := must(f.r.ResultOf("w1", "key"))
		doc := observed.Documents[0]
		if !observed.Reconciliation.Matched || len(doc.Links) != doc.Document.Revision {
			t.Fatal("mixed replacement snapshot")
		}
		for j, l := range doc.Links {
			if l.Stale != (j < len(doc.Links)-1) {
				t.Fatal("mixed link state")
			}
		}
	}
	group.Wait()
	close(outcomes)
	success, conflict := 0, 0
	for err := range outcomes {
		if err == nil {
			success++
		} else if kind(err) == "conflict" {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("CAS admitted multiple writers")
	}
}

func TestReplaceResultForeignActiveAndStaleHistory(t *testing.T) {
	for _, stale := range []bool{false, true} {
		f := resultFixture(t)
		must(f.r.Prepare("w1", invoice("other", "api", line("other-line", "other-source", "10")), human("biller")))
		document := Document{Connection: "api", External: "shared", Revision: 1, Rows: []Row{{ID: "own", Product: "P-1", Net: "10", Tax: "0", Quantity: "1"}, {ID: "foreign", Product: "P-1", Net: "10", Tax: "0", Quantity: "1"}}}
		must(f.r.Document("w1", document, human("reporter")))
		own := Link{Connection: "api", External: "shared", Row: "own", Key: "key", Line: "line", Net: "10", Tax: "0", Quantity: "1"}
		foreign := Link{Connection: "api", External: "shared", Row: "foreign", Key: "other", Line: "other-line", Net: "10", Tax: "0", Quantity: "1"}
		must(f.r.Link("w1", own, human("reporter")))
		must(f.r.Link("w1", foreign, human("reporter")))
		if stale {
			document.Revision++
			document.Rows = document.Rows[:1]
			must(f.r.Document("w1", document, human("reporter")))
		}
		before := must(f.r.ResultOf("w1", "key"))
		other := must(f.r.ResultOf("w1", "other"))
		document.Revision++
		document.Rows = document.Rows[:1]
		if _, err := f.r.ReplaceResult("w1", "key", before, document, []Link{own}, human("replacement-actor")); err == nil {
			t.Fatal("foreign history accepted", stale)
		}
		if !reflect.DeepEqual(before, must(f.r.ResultOf("w1", "key"))) || !reflect.DeepEqual(other, must(f.r.ResultOf("w1", "other"))) {
			t.Fatal("foreign refusal changed a submission")
		}
	}
}

func TestReplaceResultZeroAndSignedExactAmounts(t *testing.T) {
	for _, values := range []Amounts{{Net: "0", Tax: "0", Quantity: "0"}, {Net: "-10", Tax: "-1", Quantity: "-2"}} {
		f := newFixture(t)
		l := line("line", "source", values.Net)
		l.Tax = values.Tax
		l.Quantity = values.Quantity
		must(f.r.Prepare("w1", invoice("key", "api", l), human("biller")))
		document := Document{Connection: "api", External: "document", Revision: 1, Rows: []Row{{ID: "row", Product: "P-1", Net: values.Net, Tax: values.Tax, Quantity: values.Quantity}}}
		allocation := Link{Connection: "api", External: "document", Row: "row", Key: "key", Line: "line", Net: values.Net, Tax: values.Tax, Quantity: values.Quantity}
		must(f.r.Document("w1", document, human("reporter")))
		must(f.r.Link("w1", allocation, human("reporter")))
		before := must(f.r.ResultOf("w1", "key"))
		document.Revision++
		got := must(f.r.ReplaceResult("w1", "key", before, document, []Link{allocation}, human("replacement-actor")))
		if !got.Reconciliation.Matched || got.Documents[0].Total != values {
			t.Fatal("exact signed or zero value changed")
		}
	}
}

func TestReplaceResultEqualTotalProductSwap(t *testing.T) {
	f := newFixture(t)
	one, two := line("compute-line", "compute-source", "10"), line("storage-line", "storage-source", "10")
	two.Product = "storage"
	must(f.r.Prepare("w1", invoice("key", "api", one, two), human("biller")))
	document := Document{Connection: "api", External: "document", Revision: 1, Rows: []Row{{ID: "compute-row", Product: "P-1", Net: "10", Tax: "0", Quantity: "1"}, {ID: "storage-row", Product: "P-2", Net: "10", Tax: "0", Quantity: "1"}}}
	links := []Link{{Connection: "api", External: "document", Row: "compute-row", Key: "key", Line: "compute-line", Net: "10", Tax: "0", Quantity: "1"}, {Connection: "api", External: "document", Row: "storage-row", Key: "key", Line: "storage-line", Net: "10", Tax: "0", Quantity: "1"}}
	must(f.r.Document("w1", document, human("reporter")))
	for _, l := range links {
		must(f.r.Link("w1", l, human("reporter")))
	}
	before := must(f.r.ResultOf("w1", "key"))
	document.Revision++
	links[0].Line, links[1].Line = links[1].Line, links[0].Line
	if _, err := f.r.ReplaceResult("w1", "key", before, document, links, human("replacement-actor")); err == nil {
		t.Fatal("equal-total swapped products accepted")
	}
	if !reflect.DeepEqual(before, must(f.r.ResultOf("w1", "key"))) {
		t.Fatal("product swap rejection wrote state")
	}
	links[0].Line, links[1].Line = links[1].Line, links[0].Line
	got := must(f.r.ReplaceResult("w1", "key", before, document, links, human("replacement-actor")))
	if !got.Reconciliation.Matched || len(got.Documents[0].Links) != 4 {
		t.Fatal("complete product-bound replacement failed")
	}
}

func TestReplaceResultPreflightSharesValidationWithoutPublishing(t *testing.T) {
	f, before, next, links := replacementFixture(t)
	if err := f.r.CheckResultReplacement("w1", "key", before, next, links, human("replacement-actor")); err != nil {
		t.Fatal(err)
	}
	observed := must(f.r.ResultOf("w1", "key"))
	if !reflect.DeepEqual(before, observed) {
		t.Fatal("preflight published replacement")
	}
	f.r.limits.Links = 1
	if err := f.r.CheckResultReplacement("w1", "key", before, next, links, human("replacement-actor")); err != ErrLimit {
		t.Fatalf("capacity preflight: %v", err)
	}
	if !reflect.DeepEqual(before, must(f.r.ResultOf("w1", "key"))) {
		t.Fatal("refused preflight mutated ledger")
	}
}
