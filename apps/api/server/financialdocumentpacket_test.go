package server

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func packetFromBasis(b financialDocumentBasis) financialDocumentPacket {
	p := financialDocumentPacket{Mode: b.Mode, Profile: b.Profile, Evidence: b.Evidence, Request: b.Request, RequestDigest: b.RequestDigest, Connection: b.Connection.Connection.ID, ExpectedNet: b.ExpectedNet, ExpectedTax: b.ExpectedTax}
	for _, l := range b.Lines {
		p.Lines = append(p.Lines, financialDocumentPacketLine{Line: l.Line, Product: l.Product, Quantity: l.Quantity, Unit: l.Unit, TaxCode: l.TaxCode, Tax: l.Tax, Evidence: l.Evidence})
	}
	return p
}
func packetJSON(t *testing.T, p financialDocumentPacket) string {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func TestFinancialDocumentPacketStrictJSON(t *testing.T) {
	p := financialDocumentPacket{Mode: "synthetic", Profile: "synthetic_individual_net_v1", Evidence: "evidence", Request: "request", RequestDigest: "digest", Connection: "connection", ExpectedNet: "15", ExpectedTax: "0", Lines: []financialDocumentPacketLine{{Line: "line", Product: "product", Quantity: "1", Unit: "unit", TaxCode: "tax", Tax: "0", Evidence: "evidence"}}}
	raw := packetJSON(t, p)
	if got, err := parseDocumentPacket(raw); err != nil || !reflect.DeepEqual(got, p) {
		t.Fatal(got, err)
	}
	for name, invalid := range map[string]string{
		"unknown":           strings.Replace(raw, "{", `{"Document":{},`, 1),
		"duplicate":         strings.Replace(raw, "{", `{"Mode":"synthetic",`, 1),
		"escaped-duplicate": strings.Replace(raw, "{", `{"\u004dode":"synthetic",`, 1),
		"alias":             strings.Replace(raw, `"Mode"`, `"mode"`, 1),
		"null":              strings.Replace(raw, `"Tax":"0"`, `"Tax":null`, 1),
		"missing-tax":       strings.Replace(raw, `"Tax":"0",`, "", 1),
		"number":            strings.Replace(raw, `"Tax":"0"`, `"Tax":0`, 1),
		"surrogate":         strings.Replace(raw, `"evidence"`, `"\ud800"`, 1),
		"utf8":              strings.Replace(raw, "evidence", string([]byte{0xff}), 1),
		"trailing":          raw + " {}",
		"empty-lines":       strings.Replace(raw, packetJSONLine(t, p.Lines), "[]", 1),
		"missing-net":       strings.Replace(raw, `"ExpectedNet":"15",`, "", 1),
		"mode":              strings.Replace(raw, "synthetic\"", "live\"", 1),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseDocumentPacket(invalid)
			documentFailure(t, err, commands.InvalidRequest)
		})
	}
	p.Lines = append(p.Lines, p.Lines[0])
	_, err := parseDocumentPacket(packetJSON(t, p))
	documentFailure(t, err, commands.InvalidRequest)
	atLimit := raw + strings.Repeat(" ", pricingPolicyFieldBytes-len(raw))
	if _, err := parseDocumentPacket(atLimit); err != nil {
		t.Fatal(err)
	}
	_, err = parseDocumentPacket(atLimit + " ")
	documentFailure(t, err, commands.InvalidRequest)
}
func packetJSONLine(t *testing.T, lines []financialDocumentPacketLine) string {
	t.Helper()
	b, err := json.Marshal(lines)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestFinancialDocumentPacketDerivesOwnedSnapshots(t *testing.T) {
	w := explicitDocumentWorld(t)
	p := packetFromBasis(w.basis)
	before := w.f.s.Revision()
	build := func(login, raw string) (financialDocumentBasis, error) {
		var b financialDocumentBasis
		err := inputAuthorityAs(w.f, w.actors[login], func() error {
			var err error
			b, err = w.a.buildDocumentPacketLocked(SampleWorkspace, "built-document", w.f.s.answer.User(), raw)
			return err
		})
		return b, err
	}
	b, err := build(Admin2Login, packetJSON(t, p))
	w.f.must(err)
	want := cloneDocumentValue(t, w.basis)
	want.ID = "built-document"
	if !reflect.DeepEqual(b, want) {
		t.Fatalf("derived packet mismatch\ngot=%+v\nwant=%+v", b, want)
	}
	b.Lines[0].Tax = "99"
	b.Mappings[0].Mapping.External = "tampered"
	b.Connection.Connection.Capabilities[0] = "tampered"
	again, err := build(Admin2Login, packetJSON(t, p))
	w.f.must(err)
	if !reflect.DeepEqual(again, want) {
		t.Fatal("aliased snapshots or nondeterministic mapping order")
	}
	for _, request := range []string{p.Request, "missing"} {
		denied := p
		denied.Request = request
		_, err := build(AdminLogin, packetJSON(t, denied))
		documentFailure(t, err, commands.PermissionDenied)
	}
	for _, change := range []func(*financialDocumentPacket){
		func(p *financialDocumentPacket) { p.Request = "foreign" },
		func(p *financialDocumentPacket) { p.RequestDigest = "changed" },
		func(p *financialDocumentPacket) { p.Lines = p.Lines[:1] },
		func(p *financialDocumentPacket) { p.Lines[0].Quantity = "01" },
		func(p *financialDocumentPacket) { p.Lines[0].Tax = "0.250" },
		func(p *financialDocumentPacket) { p.Lines[0].Unit = "other" },
		func(p *financialDocumentPacket) { p.ExpectedTax = "2" },
	} {
		changed := packetFromBasis(w.basis)
		change(&changed)
		_, err := build(Admin2Login, packetJSON(t, changed))
		documentFailure(t, err, commands.EvidenceStale)
	}
	missing := packetFromBasis(w.basis)
	missing.Connection = "missing-connection"
	_, err = build(Admin2Login, packetJSON(t, missing))
	documentFailure(t, err, commands.DependencyUnavailable)
	missing = packetFromBasis(w.basis)
	missing.Lines[0].Product = "missing-product"
	_, err = build(Admin2Login, packetJSON(t, missing))
	documentFailure(t, err, commands.DependencyUnavailable)
	for _, grant := range w.f.s.admin.Grants() {
		if grant.SubjectID == SampleAdmin && grant.Terms.Action == gate.Approve {
			w.f.must(w.f.s.admin.RevokeGrant(grant.ID, grant.Version))
		}
	}
	_, err = build(Admin2Login, packetJSON(t, p))
	documentFailure(t, err, commands.PermissionDenied)
	if len(w.a.documents) != 1 || len(w.a.owners) != 0 || w.f.s.Revision() != before {
		t.Fatal("builder changed retained state")
	}
}
