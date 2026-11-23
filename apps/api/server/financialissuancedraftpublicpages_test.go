package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
)

func TestFinancialDraftPublicActualListAndLinePages(t *testing.T) {
	f, _, configuration, _ := applicationWorld(t)
	a := f.s.financialIssuance
	login := activateProtectedMember(f, "public-page-reader")
	grantConfiguredDraftCapability(f, "page-reader-", draftauthority.Inspect, "public-page-reader")
	producer, adapter, ledger := f.s.chargeProducer, f.s.documentERP, f.s.documentERP.ledger
	var expected []string
	var long financialDraftRecord
	for i := 0; i < 11; i++ {
		prefix := fmt.Sprintf("public-page-%02d", i)
		from := time.Date(2026, time.Month(11+i), 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, 0)
		var rows []invoiceRowOracle
		if i == 0 {
			rows = []invoiceRowOracle{{"2.5", "0.15", "1"}, {"2.5", "0.15", "1"}, {"2.5", "0.15", "1"}, {"2.5", "0.15", "1"}, {"2.5", "0.15", "1"}, {"2.5", "0.25", "1"}}
		}
		maker, checker := f.as(MemberLogin), f.as(Admin3Login)
		candidate, digest := captureInvoiceInWorld(t, f, maker, checker, prefix, from.Format("2006-01"), from, to, rows...)
		closeInvoiceActors(f, maker, checker)
		final, finalDigest := finalizeInvoiceInWorld(t, f, prefix, from.Format("2006-01"), candidate, digest, rows...)
		spec := financialDraftSpec{ID: prefix + "-draft", Language: "en", Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: configurationReference(configuration), FinalRequest: final, FinalDigest: finalDigest}}
		submitter := f.as(MemberLogin)
		draft, err := submitDraftAs(f, submitter, spec)
		closeInvoiceActors(f, submitter)
		f.must(err)
		expected = append(expected, spec.ID)
		if i == 0 {
			long = draft
		}
		f.advance(time.Second)
	}
	if f.s.chargeProducer != producer || f.s.documentERP != adapter || f.s.documentERP.ledger != ledger || len(a.configurations) != 1 || len(a.drafts) != 11 || len(long.Draft.Lines) != 12 {
		t.Fatal("paging fixture replaced world or lacks real records")
	}
	member := f.as(MemberLogin)
	revision := f.s.Revision()
	first := documentRead(f, member, "issuance_draft.list", SampleWorkspace, map[string]string{})
	if first.status != 200 || field(first, "count") != "11" || field(first, "next") == "" || field(first, "row_09") == "" || field(first, "row_10") != "" {
		t.Fatal(first.raw)
	}
	second := documentRead(f, member, "issuance_draft.list", SampleWorkspace, map[string]string{"after": field(first, "next"), "snapshot": field(first, "snapshot")})
	if second.status != 200 || field(second, "next") != "" || field(second, "snapshot") != field(first, "snapshot") || field(second, "row_01") != "" {
		t.Fatal(second.raw)
	}
	var actual []string
	for _, page := range []reply{first, second} {
		for i := 0; i < 10; i++ {
			raw := field(page, fmt.Sprintf("row_%02d", i))
			if raw == "" {
				break
			}
			var row map[string]string
			f.must(json.Unmarshal([]byte(raw), &row))
			actual = append(actual, row["draft"])
		}
	}
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatal("paged IDs missing, duplicated or unordered", actual)
	}
	one := documentRead(f, member, "issuance_draft.inspect", long.Spec.ID, map[string]string{})
	if one.status != 200 || field(one, "line_count") != "12" || field(one, "next_line") != "9" || field(one, "line_09") == "" {
		t.Fatal(one.raw)
	}
	two := documentRead(f, member, "issuance_draft.inspect", long.Spec.ID, map[string]string{"after_line": field(one, "next_line"), "snapshot": field(one, "snapshot")})
	if two.status != 200 || field(two, "next_line") != "" || field(two, "line_01") == "" || field(two, "line_02") != "" {
		t.Fatal(two.raw)
	}
	index := 0
	for _, page := range []reply{one, two} {
		for i := 0; i < 10; i++ {
			raw := field(page, fmt.Sprintf("line_%02d", i))
			if raw == "" {
				break
			}
			var line struct {
				Index, Text, Amount string
				Tax                 bool
			}
			f.must(json.Unmarshal([]byte(raw), &line))
			expectedLine := long.Draft.Lines[index]
			if line.Index != strconv.Itoa(index) || line.Text != expectedLine.Text || line.Amount != expectedLine.Amount || line.Tax != expectedLine.Tax {
				t.Fatal("line page changed exact monetary evidence")
			}
			index++
		}
	}
	if index != 12 || f.s.Revision() != revision {
		t.Fatal("read pagination omitted lines or mutated revision")
	}
	for _, fields := range []map[string]string{{"after_line": "9"}, {"snapshot": field(one, "snapshot")}, {"after_line": "09", "snapshot": field(one, "snapshot")}} {
		rejected := documentRead(f, member, "issuance_draft.inspect", long.Spec.ID, fields)
		f.code(t, "invalid cursor pair", rejected, 422, string(commands.InvalidRequest))
		if rejected.body["fields"] != nil {
			t.Fatal("invalid cursor disclosed partial page")
		}
	}
	f.s.fault = func(at string) bool { return at == "issuance-draft-lookup" }
	stale := documentRead(f, member, "issuance_draft.inspect", long.Spec.ID, map[string]string{"after_line": "9", "snapshot": field(one, "snapshot")})
	f.code(t, "changed observation invalidates line cursor", stale, 409, string(commands.EvidenceStale))
	stale = documentRead(f, member, "issuance_draft.list", SampleWorkspace, map[string]string{"after": field(first, "next"), "snapshot": field(first, "snapshot")})
	f.code(t, "changed observation invalidates list cursor", stale, 409, string(commands.EvidenceStale))
	f.s.fault = nil
	other := f.as(login)
	cross := documentRead(f, other, "issuance_draft.inspect", long.Spec.ID, map[string]string{"after_line": "9", "snapshot": field(one, "snapshot")})
	f.code(t, "another inspector cannot reuse cursor", cross, 409, string(commands.EvidenceStale))
	if cross.body["fields"] != nil {
		t.Fatal("foreign-viewer cursor disclosed a page")
	}
}
