package gate

import "testing"

func TestServiceTermsGatePreservesFullValidationAndNoGrantEffect(t *testing.T) {
	f := setup(t)
	terms := Terms{Action: View, Kind: "usage_statement", Fields: []string{"quantity"}, Scope: Scope{WorkspaceID: "w1", Customer: One("customer"), LegalEntity: All(), Contract: All()}, Period: AnyPeriod()}
	before := f.g.permissions.Generation()
	if !f.g.DescribesServiceTerms(terms) {
		t.Fatal("gate refused valid service declaration vocabulary")
	}
	terms.Scope.Customer = Selector{}
	if f.g.DescribesServiceTerms(terms) {
		t.Fatal("gate ignored missing service scope selector")
	}
	terms.Scope.Customer = All()
	terms.Action, terms.Fields = Approve, nil
	if f.g.DescribesServiceTerms(terms) {
		t.Fatal("gate admitted human approval declaration")
	}
	if f.g.permissions.Generation() != before || len(f.g.permissions.Grants()) != 0 {
		t.Fatal("gate declaration inspection created authority")
	}
	for _, unavailable := range []*Gate{nil, {}} {
		if unavailable.DescribesServiceTerms(terms) {
			t.Fatal("unavailable gate accepted terms")
		}
	}
}
