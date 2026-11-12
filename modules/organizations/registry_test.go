package organizations

import (
	"errors"
	"sync"
	"testing"
)

func seeded(t testing.TB, profile OperatingProfile) *Registry {
	t.Helper()
	r, err := NewRegistry("synthetic-installation")
	if err != nil {
		t.Fatal(err)
	}
	must := func(_ uint64, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(r.PutWorkspace(Workspace{ID: "w1", Profile: profile, State: Active}, 0))
	must(r.PutWorkspace(Workspace{ID: "w2", Profile: profile, State: Active}, 0))
	for _, id := range []string{"seller", "customer-a", "customer-b", "agent"} {
		must(r.PutParty(Party{ID: id, WorkspaceID: "w1", Kind: LegalEntity, State: Active}, 0))
	}
	must(r.PutParty(Party{ID: "unit-1", WorkspaceID: "w1", Kind: InternalUnit, ParentID: "customer-a", State: Active}, 0))
	must(r.PutParty(Party{ID: "unit-2", WorkspaceID: "w1", Kind: InternalUnit, ParentID: "customer-a", State: Active}, 0))
	must(r.PutParty(Party{ID: "unit-b", WorkspaceID: "w1", Kind: InternalUnit, ParentID: "customer-b", State: Active}, 0))
	must(r.PutParty(Party{ID: "foreign", WorkspaceID: "w2", Kind: LegalEntity, State: Active}, 0))
	return r
}

func external(id, issuer, buyer, payer, currency string) BillingProfile {
	return BillingProfile{ID: id, WorkspaceID: "w1", IssuerID: issuer, BuyerID: buyer, PayerID: payer, Path: ExternalInvoice, Currency: currency, State: Active}
}

func internal(id, issuer, buyer string) BillingProfile {
	return BillingProfile{ID: id, WorkspaceID: "w1", IssuerID: issuer, BuyerID: buyer, Path: InternalChargeback, Currency: "USD", State: Active}
}

func TestBillingProfilesSeparateIssuerBuyerPayerAndPath(t *testing.T) {
	r := seeded(t, ProfileMSP)
	cases := []struct {
		name    string
		profile BillingProfile
		want    error
	}{
		{"external invoice between two legal entities", external("p1", "seller", "customer-a", "customer-a", "USD"), nil},
		{"payer is a third legal entity", external("p2", "seller", "customer-a", "agent", "USD"), nil},
		{"recipient differs from buyer", func() BillingProfile {
			p := external("p3", "seller", "customer-a", "customer-a", "KRW")
			p.RecipientID = "agent"
			return p
		}(), nil},
		{"internal chargeback between units of one legal entity", internal("p4", "unit-1", "unit-2"), nil},
		{"internal chargeback from a legal entity to its own unit", internal("p5", "customer-a", "unit-1"), nil},
		{"external invoice to oneself", external("n1", "seller", "seller", "seller", "USD"), ErrInvalid},
		{"external invoice issued by an internal unit", external("n2", "unit-1", "customer-b", "customer-b", "USD"), ErrInvalid},
		{"external invoice bought by an internal unit", external("n3", "seller", "unit-1", "customer-a", "USD"), ErrInvalid},
		{"external invoice paid by an internal unit", external("n4", "seller", "customer-a", "unit-1", "USD"), ErrInvalid},
		{"external invoice without a payer", external("n5", "seller", "customer-a", "", "USD"), ErrInvalid},
		{"payer is the issuer", external("n17", "seller", "customer-a", "seller", "USD"), ErrInvalid},
		{"recipient is the issuer", func() BillingProfile {
			p := external("n18", "seller", "customer-a", "customer-a", "USD")
			p.RecipientID = "seller"
			return p
		}(), ErrInvalid},
		{"recipient is an internal unit", func() BillingProfile {
			p := external("n6", "seller", "customer-a", "customer-a", "USD")
			p.RecipientID = "unit-1"
			return p
		}(), ErrInvalid},
		{"chargeback between units of different legal entities", internal("n7", "unit-1", "unit-b"), ErrIntercompanyUndecided},
		{"chargeback between two legal entities", internal("n8", "customer-a", "customer-b"), ErrIntercompanyUndecided},
		{"chargeback with a payer", func() BillingProfile { p := internal("n9", "unit-1", "unit-2"); p.PayerID = "customer-a"; return p }(), ErrInvalid},
		{"chargeback with a document recipient", func() BillingProfile {
			p := internal("n10", "unit-1", "unit-2")
			p.RecipientID = "customer-a"
			return p
		}(), ErrInvalid},
		{"party of another workspace", external("n11", "seller", "foreign", "foreign", "USD"), ErrNotFound},
		{"unknown party", external("n12", "seller", "nobody", "nobody", "USD"), ErrNotFound},
		{"lowercase currency", external("n13", "seller", "customer-a", "customer-a", "usd"), ErrInvalid},
		{"unknown path", func() BillingProfile {
			p := external("n14", "seller", "customer-a", "customer-a", "USD")
			p.Path = "barter"
			return p
		}(), ErrInvalid},
		{"identifier with a space", external("n 15", "seller", "customer-a", "customer-a", "USD"), ErrInvalid},
		{"unknown workspace", func() BillingProfile {
			p := external("n16", "seller", "customer-a", "customer-a", "USD")
			p.WorkspaceID = "w9"
			return p
		}(), ErrNotFound},
	}
	for _, c := range cases {
		version, err := r.PutBillingProfile(c.profile, 0)
		if !errors.Is(err, c.want) || (c.want == nil && (err != nil || version != 1)) {
			t.Errorf("%s: version %d error %v; want %v", c.name, version, err, c.want)
		}
	}
	for id, want := range map[string]bool{"p1": true, "p4": false} {
		profile, _, err := r.BillingProfile("w1", id)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := RequiresExternalInvoice(profile); err != nil || got != want {
			t.Errorf("RequiresExternalInvoice(%s) = %v, %v; want %v", id, got, err, want)
		}
	}
	if _, err := RequiresExternalInvoice(BillingProfile{}); !errors.Is(err, ErrInvalid) {
		t.Errorf("missing path error = %v", err)
	}
}

func TestOperatingProfileIsOnlyALabel(t *testing.T) {
	var outcomes []string
	for _, profile := range []OperatingProfile{ProfileMSP, ProfileSingleEnterprise, ProfileGroup} {
		r := seeded(t, profile)
		outcome := ""
		for _, candidate := range []BillingProfile{
			external("a", "seller", "customer-a", "customer-a", "USD"),
			internal("b", "unit-1", "unit-2"),
			internal("c", "customer-a", "customer-b"),
			external("d", "unit-1", "customer-b", "customer-b", "USD"),
		} {
			_, err := r.PutBillingProfile(candidate, 0)
			if err == nil {
				outcome += "ok;"
			} else {
				outcome += err.Error() + ";"
			}
		}
		outcomes = append(outcomes, outcome)
	}
	want := "ok;ok;" + ErrIntercompanyUndecided.Error() + ";" + ErrInvalid.Error() + ";"
	for _, outcome := range outcomes {
		if outcome != want {
			t.Errorf("outcome %q; want %q under every operating profile", outcome, want)
		}
	}
	r := seeded(t, ProfileSingleEnterprise)
	if _, err := r.PutBillingProfile(internal("only-internal", "customer-a", "unit-1"), 0); err != nil {
		t.Errorf("a single enterprise settles internally without any seller party: %v", err)
	}
	for _, workspace := range []Workspace{{ID: "w3", Profile: "reseller", State: Active}, {ID: "w3", Profile: ProfileMSP, State: "closed"}, {ID: "", Profile: ProfileMSP, State: Active}} {
		if _, err := r.PutWorkspace(workspace, 0); !errors.Is(err, ErrInvalid) {
			t.Errorf("PutWorkspace(%+v) error = %v", workspace, err)
		}
	}
}

func TestPartiesKeepKindParentAndWorkspace(t *testing.T) {
	r := seeded(t, ProfileGroup)
	cases := []struct {
		name     string
		party    Party
		expected uint64
		want     error
	}{
		{"legal entity with a parent", Party{ID: "x1", WorkspaceID: "w1", Kind: LegalEntity, ParentID: "seller", State: Active}, 0, ErrInvalid},
		{"internal unit without a parent", Party{ID: "x2", WorkspaceID: "w1", Kind: InternalUnit, State: Active}, 0, ErrInvalid},
		{"internal unit as its own parent", Party{ID: "x3", WorkspaceID: "w1", Kind: InternalUnit, ParentID: "x3", State: Active}, 0, ErrInvalid},
		{"internal unit under an internal unit", Party{ID: "x4", WorkspaceID: "w1", Kind: InternalUnit, ParentID: "unit-1", State: Active}, 0, ErrInvalid},
		{"parent in another workspace", Party{ID: "x5", WorkspaceID: "w1", Kind: InternalUnit, ParentID: "foreign", State: Active}, 0, ErrNotFound},
		{"unknown kind", Party{ID: "x6", WorkspaceID: "w1", Kind: "department", State: Active}, 0, ErrInvalid},
		{"changing the kind", Party{ID: "customer-b", WorkspaceID: "w1", Kind: InternalUnit, ParentID: "customer-a", State: Active}, 1, ErrInvalid},
		{"changing the parent", Party{ID: "unit-1", WorkspaceID: "w1", Kind: InternalUnit, ParentID: "customer-b", State: Active}, 1, ErrInvalid},
		{"stale version", Party{ID: "agent", WorkspaceID: "w1", Kind: LegalEntity, State: Suspended}, 0, ErrConflict},
		{"suspending a party", Party{ID: "agent", WorkspaceID: "w1", Kind: LegalEntity, State: Suspended}, 1, nil},
	}
	for _, c := range cases {
		if _, err := r.PutParty(c.party, c.expected); !errors.Is(err, c.want) {
			t.Errorf("%s: error %v; want %v", c.name, err, c.want)
		}
	}
	if _, err := r.PutBillingProfile(external("s1", "seller", "customer-a", "agent", "USD"), 0); !errors.Is(err, ErrSuspended) {
		t.Errorf("profile naming a suspended payer error = %v", err)
	}
	if _, _, err := r.Party("w2", "seller"); !errors.Is(err, ErrNotFound) {
		t.Errorf("reading a party through another workspace error = %v", err)
	}
	if _, _, err := r.Party("w1", "nobody"); !errors.Is(err, ErrNotFound) {
		t.Errorf("reading an unknown party error = %v", err)
	}
	if entity, err := r.LegalEntityOf("w1", "unit-2"); err != nil || entity != "customer-a" {
		t.Errorf("LegalEntityOf(unit-2) = %q, %v", entity, err)
	}
	if _, err := r.PutWorkspace(Workspace{ID: "w1", Profile: ProfileGroup, State: Suspended}, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PutParty(Party{ID: "late", WorkspaceID: "w1", Kind: LegalEntity, State: Active}, 0); !errors.Is(err, ErrSuspended) {
		t.Errorf("write into a suspended workspace error = %v", err)
	}
	if _, err := NewRegistry("bad id"); !errors.Is(err, ErrInvalid) {
		t.Errorf("invalid installation error = %v", err)
	}
}

func revision(path TransactionPath, seller string, components ...Component) ContractRevision {
	return ContractRevision{ContractID: "c1", Revision: 1, WorkspaceID: "w1", ProviderRef: "provider-x", CustomerID: "customer-a", SellerID: seller, Path: path, Components: components}
}

func TestContractRevisionsCarryPathAndProviderResponsibility(t *testing.T) {
	usage := func(responsible string, billable bool) Component {
		return Component{ID: "usage", Kind: ProviderCharge, ProviderPaymentResponsibleID: responsible, SellerBillable: billable}
	}
	fee := Component{ID: "managed-service", Kind: SellerService, SellerBillable: true}
	cases := []struct {
		name     string
		revision ContractRevision
		want     error
	}{
		{"direct with the customer responsible and a seller service", revision(Direct, "seller", usage("customer-a", false), fee), nil},
		{"direct without any seller", revision(Direct, "", usage("customer-a", false)), nil},
		{"resale with the seller responsible and billable", revision(Resale, "seller", usage("seller", true), fee), nil},
		{"resale with a component outside the eligible scope", revision(Resale, "seller", usage("seller", false)), nil},
		{"direct marking provider usage billable", revision(Direct, "seller", usage("customer-a", true)), ErrInvalid},
		{"direct with the seller responsible to the provider", revision(Direct, "seller", usage("seller", false)), ErrInvalid},
		{"resale without a seller", revision(Resale, "", usage("customer-a", false)), ErrInvalid},
		{"billable although the customer pays the provider", revision(Resale, "seller", usage("customer-a", true)), ErrInvalid},
		{"provider charge without a responsible party", revision(Resale, "seller", usage("", false)), ErrInvalid},
		{"provider charge with an outside responsible party", revision(Resale, "seller", usage("agent", false)), ErrInvalid},
		{"seller service without a seller", revision(Direct, "", fee), ErrInvalid},
		{"seller service with a provider responsibility", revision(Direct, "seller", Component{ID: "fee", Kind: SellerService, ProviderPaymentResponsibleID: "seller"}), ErrInvalid},
		{"unknown component kind", revision(Direct, "seller", Component{ID: "x", Kind: "tax"}), ErrInvalid},
		{"repeated component", revision(Resale, "seller", usage("seller", true), usage("seller", false)), ErrInvalid},
		{"no component", revision(Direct, "seller"), ErrInvalid},
		{"customer is an internal unit", func() ContractRevision {
			c := revision(Direct, "", usage("unit-1", false))
			c.CustomerID = "unit-1"
			return c
		}(), ErrInvalid},
		{"seller equals customer", revision(Resale, "customer-a", usage("customer-a", false)), ErrInvalid},
		{"seller in another workspace", revision(Resale, "foreign", usage("foreign", true)), ErrNotFound},
		{"unknown path", revision("barter", "seller", usage("seller", false)), ErrInvalid},
		{"missing provider reference", func() ContractRevision {
			c := revision(Direct, "", usage("customer-a", false))
			c.ProviderRef = ""
			return c
		}(), ErrInvalid},
	}
	for _, c := range cases {
		r := seeded(t, ProfileMSP)
		if err := r.AppendContractRevision(c.revision); !errors.Is(err, c.want) {
			t.Errorf("%s: error %v; want %v", c.name, err, c.want)
		}
	}
	r := seeded(t, ProfileMSP)
	first := revision(Direct, "seller", usage("customer-a", false), fee)
	if err := r.AppendContractRevision(first); err != nil {
		t.Fatal(err)
	}
	first.Components[0].SellerBillable = true
	stored, err := r.ContractRevision("w1", "c1", 1)
	if err != nil || stored.Components[0].SellerBillable {
		t.Errorf("stored revision changed with the caller's slice: %+v, %v", stored, err)
	}
	stored.Components[0].ID = "tampered"
	again, _ := r.ContractRevision("w1", "c1", 1)
	if again.Components[0].ID != "usage" {
		t.Error("a returned revision must be a copy")
	}
	for _, number := range []uint64{1, 3, 0} {
		next := revision(Resale, "seller", usage("seller", true))
		next.Revision = number
		if err := r.AppendContractRevision(next); !errors.Is(err, ErrConflict) {
			t.Errorf("appending revision %d error = %v", number, err)
		}
	}
	second := revision(Resale, "seller", usage("seller", true))
	second.Revision = 2
	if err := r.AppendContractRevision(second); err != nil {
		t.Fatal(err)
	}
	earlier, err := r.ContractRevision("w1", "c1", 1)
	if err != nil || earlier.Path != Direct || len(earlier.Components) != 2 {
		t.Errorf("revision 1 after appending revision 2: %+v, %v", earlier, err)
	}
	for _, lookup := range []struct {
		workspace string
		number    uint64
	}{{"w2", 1}, {"w1", 3}, {"w1", 0}} {
		if _, err := r.ContractRevision(lookup.workspace, "c1", lookup.number); !errors.Is(err, ErrNotFound) {
			t.Errorf("ContractRevision(%s, %d) error = %v", lookup.workspace, lookup.number, err)
		}
	}
}

func TestSellingEligibilityBlocksRebillingWithoutProviderResponsibility(t *testing.T) {
	components := func(responsible string, billable bool) []Component {
		return []Component{
			{ID: "usage", Kind: ProviderCharge, ProviderPaymentResponsibleID: responsible, SellerBillable: billable},
			{ID: "support", Kind: ProviderCharge, ProviderPaymentResponsibleID: responsible, SellerBillable: false},
			{ID: "managed-service", Kind: SellerService, SellerBillable: true},
			{ID: "onboarding", Kind: SellerService, SellerBillable: false},
		}
	}
	direct := ContractRevision{CustomerID: "customer-a", SellerID: "seller", Path: Direct, Components: components("customer-a", false)}
	resale := ContractRevision{CustomerID: "customer-a", SellerID: "seller", Path: Resale, Components: components("seller", true)}
	cases := []struct {
		name      string
		revision  ContractRevision
		component string
		issuer    string
		want      Decision
	}{
		{"direct: provider usage is not the seller's to bill", direct, "usage", "seller", Decision{false, NoPaymentResponsibility}},
		{"direct: provider support likewise", direct, "support", "seller", Decision{false, NoPaymentResponsibility}},
		{"direct: the seller's own service is eligible", direct, "managed-service", "seller", Decision{true, EligiblePendingApproval}},
		{"direct: a service outside the scope", direct, "onboarding", "seller", Decision{false, OutsideEligibleScope}},
		{"resale: usage the seller pays for and may bill", resale, "usage", "seller", Decision{true, EligiblePendingApproval}},
		{"resale: usage outside the eligible scope", resale, "support", "seller", Decision{false, OutsideEligibleScope}},
		{"resale: the seller's own service", resale, "managed-service", "seller", Decision{true, EligiblePendingApproval}},
		{"an issuer that is not the seller", resale, "usage", "agent", Decision{false, NotSeller}},
		{"the customer as issuer", resale, "managed-service", "customer-a", Decision{false, NotSeller}},
		{"a component the revision does not have", resale, "storage", "seller", Decision{false, UnknownComponent}},
		{"an unknown component asked by a non-seller", resale, "storage", "agent", Decision{false, NotSeller}},
		{"direct with the seller named responsible is still not sellable", ContractRevision{CustomerID: "customer-a", SellerID: "seller", Path: Direct, Components: components("seller", true)}, "usage", "seller", Decision{false, NoPaymentResponsibility}},
		{"an unknown path", ContractRevision{CustomerID: "customer-a", SellerID: "seller", Path: "barter", Components: components("seller", true)}, "usage", "seller", Decision{false, InvalidContract}},
		{"a missing path", ContractRevision{CustomerID: "customer-a", SellerID: "seller", Components: components("seller", true)}, "usage", "seller", Decision{false, InvalidContract}},
		{"a component without a kind", ContractRevision{SellerID: "seller", Path: Direct, Components: []Component{{ID: "usage", SellerBillable: true}}}, "usage", "seller", Decision{false, InvalidContract}},
		{"an unknown component kind", ContractRevision{SellerID: "seller", Path: Resale, Components: []Component{{ID: "usage", Kind: "tax", SellerBillable: true}}}, "usage", "seller", Decision{false, InvalidContract}},
		{"a provider charge without a responsible party", ContractRevision{SellerID: "seller", Path: Resale, Components: []Component{{ID: "usage", Kind: ProviderCharge, SellerBillable: true}}}, "usage", "seller", Decision{false, InvalidContract}},
		{"a service that names a provider responsibility", ContractRevision{SellerID: "seller", Path: Resale, Components: []Component{{ID: "fee", Kind: SellerService, ProviderPaymentResponsibleID: "seller", SellerBillable: true}}}, "fee", "seller", Decision{false, InvalidContract}},
		{"a repeated component identifier", ContractRevision{CustomerID: "customer-a", SellerID: "seller", Path: Direct, Components: []Component{{ID: "usage", Kind: SellerService, SellerBillable: true}, {ID: "usage", Kind: ProviderCharge, ProviderPaymentResponsibleID: "customer-a"}}}, "usage", "seller", Decision{false, InvalidContract}},
		{"an empty component identifier", ContractRevision{CustomerID: "customer-a", SellerID: "seller", Path: Resale, Components: []Component{{Kind: SellerService, SellerBillable: true}}}, "", "seller", Decision{false, InvalidContract}},
		{"a seller that is the customer", ContractRevision{CustomerID: "seller", SellerID: "seller", Path: Resale, Components: components("seller", true)}, "usage", "seller", Decision{false, InvalidContract}},
		{"resale charge the customer pays the provider for", ContractRevision{CustomerID: "customer-a", SellerID: "seller", Path: Resale, Components: components("customer-a", false)}, "usage", "seller", Decision{false, NoPaymentResponsibility}},
		{"a contract without a seller", ContractRevision{CustomerID: "customer-a", Path: Direct, Components: components("customer-a", false)}, "usage", "customer-a", Decision{false, NotSeller}},
		{"an empty issuer against a contract without a seller", ContractRevision{CustomerID: "customer-a", Path: Direct, Components: components("customer-a", false)}, "usage", "", Decision{false, NotSeller}},
	}
	for _, c := range cases {
		if got := SellingEligibility(c.revision, c.component, c.issuer); got != c.want {
			t.Errorf("%s: %+v; want %+v", c.name, got, c.want)
		}
	}
	eligible := func(revision ContractRevision) (names []string) {
		for _, component := range revision.Components {
			if SellingEligibility(revision, component.ID, "seller").Eligible {
				names = append(names, component.ID)
			}
		}
		return names
	}
	if got := eligible(direct); len(got) != 1 || got[0] != "managed-service" {
		t.Errorf("direct contract eligible components = %v; want only the seller's service", got)
	}
	if got := eligible(resale); len(got) != 2 || got[0] != "usage" || got[1] != "managed-service" {
		t.Errorf("resale contract eligible components = %v", got)
	}
}

func TestDocumentsAreNotCombinedAcrossPartiesCurrenciesOrPaths(t *testing.T) {
	r := seeded(t, ProfileGroup)
	for _, profile := range []BillingProfile{
		external("usd-a", "seller", "customer-a", "customer-a", "USD"),
		external("usd-a-again", "seller", "customer-a", "customer-a", "USD"),
		external("krw-a", "seller", "customer-a", "customer-a", "KRW"),
		external("usd-b", "seller", "customer-b", "customer-b", "USD"),
		external("usd-a-agent", "seller", "customer-a", "agent", "USD"),
		external("usd-a-other-issuer", "agent", "customer-a", "customer-a", "USD"),
		internal("internal-1", "unit-1", "unit-2"),
		internal("internal-2", "customer-a", "unit-1"),
	} {
		if _, err := r.PutBillingProfile(profile, 0); err != nil {
			t.Fatal(err)
		}
	}
	key := func(id string) DocumentKey {
		t.Helper()
		found, err := r.DocumentKey("w1", id)
		if err != nil {
			t.Fatal(err)
		}
		return found
	}
	if got := key("internal-1"); got != (DocumentKey{WorkspaceID: "w1", IssuerLegalEntityID: "customer-a", BuyerLegalEntityID: "customer-a", Currency: "USD", Path: InternalChargeback}) {
		t.Errorf("internal units resolve to their legal entity: %+v", got)
	}
	if err := SameDocument(key("usd-a"), key("usd-a-again")); err != nil {
		t.Errorf("same issuer, buyer, payer, currency and path: %v", err)
	}
	if err := SameDocument(key("internal-1"), key("internal-2")); err != nil {
		t.Errorf("internal lines within one legal entity: %v", err)
	}
	for name, other := range map[string]string{
		"different currency": "krw-a", "different buyer": "usd-b", "different payer": "usd-a-agent",
		"different issuer": "usd-a-other-issuer", "different path": "internal-1",
	} {
		if err := SameDocument(key("usd-a"), key(other)); !errors.Is(err, ErrMixedDocument) {
			t.Errorf("%s: error %v", name, err)
		}
	}
	if err := SameDocument(); !errors.Is(err, ErrMixedDocument) {
		t.Errorf("no lines error = %v", err)
	}
	if err := SameDocument(DocumentKey{}, DocumentKey{}); !errors.Is(err, ErrMixedDocument) {
		t.Errorf("missing keys error = %v", err)
	}
	whole := key("usd-a")
	for name, change := range map[string]func(*DocumentKey){
		"no workspace":           func(k *DocumentKey) { k.WorkspaceID = "" },
		"external without payer": func(k *DocumentKey) { k.PayerID = "" },
		"unknown path":           func(k *DocumentKey) { k.Path = "barter" },
		"chargeback with payer":  func(k *DocumentKey) { k.Path = InternalChargeback },
		"no currency":            func(k *DocumentKey) { k.Currency = "" },
		"no issuer":              func(k *DocumentKey) { k.IssuerLegalEntityID = "" },
		"no buyer":               func(k *DocumentKey) { k.BuyerLegalEntityID = "" },
	} {
		broken := whole
		change(&broken)
		if err := SameDocument(broken, broken); !errors.Is(err, ErrMixedDocument) {
			t.Errorf("%s: error %v", name, err)
		}
	}
	for _, id := range []string{"seller", "customer-a"} {
		if _, err := r.PutParty(Party{ID: id, WorkspaceID: "w2", Kind: LegalEntity, State: Active}, 0); err != nil {
			t.Fatal(err)
		}
	}
	twin := external("usd-a", "seller", "customer-a", "customer-a", "USD")
	twin.WorkspaceID = "w2"
	if _, err := r.PutBillingProfile(twin, 0); err != nil {
		t.Fatal(err)
	}
	other, err := r.DocumentKey("w2", "usd-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := SameDocument(key("usd-a"), other); !errors.Is(err, ErrMixedDocument) {
		t.Errorf("identical identifiers in two workspaces must not combine: %v", err)
	}
	if _, err := r.DocumentKey("w2", "krw-a"); !errors.Is(err, ErrNotFound) {
		t.Errorf("key through another workspace error = %v", err)
	}
}

func TestConcurrentWritersWithOneExpectedVersion(t *testing.T) {
	r := seeded(t, ProfileMSP)
	const writers = 64
	var wg sync.WaitGroup
	results := make(chan error, writers)
	for index := 0; index < writers; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.PutParty(Party{ID: "seller", WorkspaceID: "w1", Kind: LegalEntity, State: Suspended}, 1)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	succeeded, conflicted := 0, 0
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrConflict):
			conflicted++
		default:
			t.Errorf("unexpected error %v", err)
		}
	}
	if succeeded != 1 || conflicted != writers-1 {
		t.Errorf("succeeded %d conflicted %d", succeeded, conflicted)
	}
}

func TestSuspensionAndFrozenFields(t *testing.T) {
	r := seeded(t, ProfileMSP)
	if _, err := r.PutBillingProfile(external("p", "seller", "customer-a", "agent", "USD"), 0); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*BillingProfile){
		"issuer":    func(p *BillingProfile) { p.IssuerID = "agent"; p.PayerID = "customer-a" },
		"buyer":     func(p *BillingProfile) { p.BuyerID = "customer-b" },
		"payer":     func(p *BillingProfile) { p.PayerID = "customer-a" },
		"currency":  func(p *BillingProfile) { p.Currency = "KRW" },
		"path":      func(p *BillingProfile) { p.Path = InternalChargeback; p.PayerID = "" },
		"recipient": func(p *BillingProfile) { p.RecipientID = "customer-b" },
	} {
		changed := external("p", "seller", "customer-a", "agent", "USD")
		change(&changed)
		if _, err := r.PutBillingProfile(changed, 1); !errors.Is(err, ErrInvalid) {
			t.Errorf("changing the %s of a billing profile error = %v", name, err)
		}
	}
	if _, err := r.PutParty(Party{ID: "agent", WorkspaceID: "w1", Kind: LegalEntity, State: Suspended}, 1); err != nil {
		t.Fatal(err)
	}
	suspended := external("p", "seller", "customer-a", "agent", "USD")
	suspended.State = Suspended
	if version, err := r.PutBillingProfile(suspended, 1); err != nil || version != 2 {
		t.Errorf("suspending a profile whose payer is suspended = %d, %v", version, err)
	}
	if _, err := r.PutBillingProfile(external("p", "seller", "customer-a", "agent", "USD"), 2); !errors.Is(err, ErrSuspended) {
		t.Errorf("reactivating a profile whose payer is suspended error = %v", err)
	}
	if _, err := r.PutParty(Party{ID: "agent", WorkspaceID: "w1", Kind: LegalEntity, State: Active}, 2); err != nil {
		t.Fatal(err)
	}
	if version, err := r.PutBillingProfile(external("p", "seller", "customer-a", "agent", "USD"), 2); err != nil || version != 3 {
		t.Errorf("reactivating after the payer is active again = %d, %v", version, err)
	}
	if _, err := r.PutParty(Party{ID: "customer-a", WorkspaceID: "w1", Kind: LegalEntity, State: Suspended}, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PutBillingProfile(internal("via-units", "unit-1", "unit-2"), 0); !errors.Is(err, ErrSuspended) {
		t.Errorf("chargeback between active units of a suspended legal entity error = %v", err)
	}
	if version, err := r.PutParty(Party{ID: "unit-1", WorkspaceID: "w1", Kind: InternalUnit, ParentID: "customer-a", State: Suspended}, 1); err != nil || version != 2 {
		t.Errorf("suspending a unit under a suspended legal entity = %d, %v", version, err)
	}
	if _, err := r.PutParty(Party{ID: "unit-1", WorkspaceID: "w1", Kind: InternalUnit, ParentID: "customer-a", State: Active}, 2); !errors.Is(err, ErrSuspended) {
		t.Errorf("reactivating a unit under a suspended legal entity error = %v", err)
	}
	if _, err := r.PutBillingProfile(internal("after", "unit-2", "customer-a"), 0); !errors.Is(err, ErrSuspended) {
		t.Errorf("chargeback inside a suspended legal entity error = %v", err)
	}
	if _, err := r.PutParty(Party{ID: "unit-3", WorkspaceID: "w1", Kind: InternalUnit, ParentID: "customer-a", State: Active}, 0); !errors.Is(err, ErrSuspended) {
		t.Errorf("new unit under a suspended legal entity error = %v", err)
	}
	roles := []struct {
		name    string
		profile BillingProfile
	}{
		{"issuer", external("r1", "customer-a", "customer-b", "customer-b", "USD")},
		{"buyer", external("r2", "seller", "customer-a", "customer-b", "USD")},
		{"payer", external("r3", "seller", "customer-b", "customer-a", "USD")},
		{"recipient", func() BillingProfile {
			p := external("r4", "seller", "customer-b", "customer-b", "USD")
			p.RecipientID = "customer-a"
			return p
		}()},
	}
	for _, role := range roles {
		if _, err := r.PutBillingProfile(role.profile, 0); !errors.Is(err, ErrSuspended) {
			t.Errorf("suspended %s error = %v", role.name, err)
		}
	}
	usage := Component{ID: "usage", Kind: ProviderCharge, ProviderPaymentResponsibleID: "customer-a"}
	if err := r.AppendContractRevision(revision(Direct, "", usage)); !errors.Is(err, ErrSuspended) {
		t.Errorf("contract for a suspended customer error = %v", err)
	}
	asSeller := ContractRevision{ContractID: "c2", Revision: 1, WorkspaceID: "w1", ProviderRef: "provider-x", CustomerID: "customer-b", SellerID: "customer-a", Path: Resale, Components: []Component{{ID: "usage", Kind: ProviderCharge, ProviderPaymentResponsibleID: "customer-a", SellerBillable: true}}}
	if err := r.AppendContractRevision(asSeller); !errors.Is(err, ErrSuspended) {
		t.Errorf("contract with a suspended seller error = %v", err)
	}
	if _, err := r.PutWorkspace(Workspace{ID: "w1", Profile: ProfileMSP, State: Suspended}, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PutBillingProfile(external("late", "seller", "customer-b", "customer-b", "USD"), 0); !errors.Is(err, ErrSuspended) {
		t.Errorf("profile in a suspended workspace error = %v", err)
	}
	late := ContractRevision{ContractID: "c3", Revision: 1, WorkspaceID: "w1", ProviderRef: "provider-x", CustomerID: "customer-b", Path: Direct, Components: []Component{{ID: "usage", Kind: ProviderCharge, ProviderPaymentResponsibleID: "customer-b"}}}
	if err := r.AppendContractRevision(late); !errors.Is(err, ErrSuspended) {
		t.Errorf("contract in a suspended workspace error = %v", err)
	}
	late.WorkspaceID = "w9"
	if err := r.AppendContractRevision(late); !errors.Is(err, ErrNotFound) {
		t.Errorf("contract in an unknown workspace error = %v", err)
	}
	if key, err := r.DocumentKey("w1", "p"); err != nil || key.PayerID != "agent" {
		t.Errorf("a stored profile stays readable in a suspended workspace: %+v, %v", key, err)
	}
	if _, err := r.PutWorkspace(Workspace{ID: "w1", Profile: ProfileMSP, State: Active}, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PutBillingProfile(external("late", "seller", "customer-b", "customer-b", "USD"), 0); err != nil {
		t.Errorf("profile after the workspace is active again: %v", err)
	}
}

func TestContractKeepsItsCustomerAndProvider(t *testing.T) {
	r := seeded(t, ProfileMSP)
	usage := func(responsible string) Component {
		return Component{ID: "usage", Kind: ProviderCharge, ProviderPaymentResponsibleID: responsible}
	}
	if err := r.AppendContractRevision(revision(Direct, "", usage("customer-a"))); err != nil {
		t.Fatal(err)
	}
	otherCustomer := revision(Direct, "", usage("customer-b"))
	otherCustomer.Revision, otherCustomer.CustomerID = 2, "customer-b"
	if err := r.AppendContractRevision(otherCustomer); !errors.Is(err, ErrInvalid) {
		t.Errorf("revision with another customer error = %v", err)
	}
	otherProvider := revision(Direct, "", usage("customer-a"))
	otherProvider.Revision, otherProvider.ProviderRef = 2, "provider-y"
	if err := r.AppendContractRevision(otherProvider); !errors.Is(err, ErrInvalid) {
		t.Errorf("revision with another provider error = %v", err)
	}
	withSeller := revision(Direct, "seller", usage("customer-a"), Component{ID: "managed-service", Kind: SellerService, SellerBillable: true})
	withSeller.Revision = 2
	if err := r.AppendContractRevision(withSeller); err != nil {
		t.Errorf("a later revision may add a seller and its service: %v", err)
	}
	stored, err := r.ContractRevision("w1", "c1", 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := SellingEligibility(stored, "managed-service", "seller"); got != (Decision{true, EligiblePendingApproval}) {
		t.Errorf("registry-stored revision decision = %+v", got)
	}
	if got := SellingEligibility(stored, "usage", "seller"); got != (Decision{false, NoPaymentResponsibility}) {
		t.Errorf("registry-stored direct usage decision = %+v", got)
	}
}
