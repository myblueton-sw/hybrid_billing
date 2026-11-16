package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

const orgSeedPlan = "sample-seed"

var orgNames = map[string]string{
	"sample-le-seller":        "Sample Seller Ltd (sample)",
	"sample-le-north":         "North Holdings (sample)",
	"sample-le-south":         "South Trading (sample)",
	"sample-unit-north-ops":   "North operations (sample)",
	"sample-unit-north-data":  "North data platform (sample)",
	"sample-unit-south-web":   "South web services (sample)",
	"sample-profile-north":    "North invoice, USD (sample)",
	"sample-profile-south":    "South invoice, USD (sample)",
	"sample-profile-north-cb": "North internal chargeback (sample)",
	"sample-acct-01":          "Account 01 (sample)",
	"sample-acct-02":          "Account 02 (sample)",
	"sample-acct-03":          "Account 03 (sample)",
	"sample-acct-04":          "Account 04 (sample)",
	"sample-acct-05":          "Account 05 (sample)",
	"sample-acct-06":          "Account 06 (sample)",
	"sample-pool-shared":      "Shared platform pool (sample)",
	"sample-view-north":       "North view group (sample)",
	"sample-view-south":       "South view group (sample)",
}

func orgName(id string) string { return orgNames[id] }

func (s *Server) orgWorkspace() error {
	_, err := s.orgs.PutWorkspace(organizations.Workspace{ID: s.Workspace(), Profile: organizations.ProfileMSP, State: organizations.Active}, 0)
	return err
}

func sampleDay(year, month, day int) relationships.Day {
	d, _ := relationships.NewDay(year, month, day)
	return d
}

func sampleTimelines() []relationships.Timeline {
	start := sampleDay(2026, 1, 1)
	move := sampleDay(2026, 10, 1)
	open := func(target string) []relationships.Entry {
		return []relationships.Entry{{Target: target, Interval: relationships.From(start)}}
	}
	line := func(kind relationships.Kind, member string, entries []relationships.Entry) relationships.Timeline {
		return relationships.Timeline{Kind: kind, Member: member, Entries: entries}
	}
	return []relationships.Timeline{
		line(relationships.Hierarchy, "sample-unit-north-ops", open("sample-le-north")),
		line(relationships.Hierarchy, "sample-unit-north-data", open("sample-le-north")),
		line(relationships.Hierarchy, "sample-unit-south-web", open("sample-le-south")),
		line(relationships.Hierarchy, "sample-acct-01", open("sample-unit-north-ops")),
		line(relationships.Hierarchy, "sample-acct-02", []relationships.Entry{
			{Target: "sample-unit-north-ops", Interval: relationships.Between(start, move)},
			{Target: "sample-unit-north-data", Interval: relationships.From(move)},
		}),
		line(relationships.Hierarchy, "sample-acct-03", open("sample-unit-north-data")),
		line(relationships.Hierarchy, "sample-acct-04", open("sample-unit-south-web")),
		line(relationships.Hierarchy, "sample-acct-05", open("sample-unit-south-web")),
		line(relationships.Hierarchy, "sample-acct-06", open("sample-unit-south-web")),
		line(relationships.Payer, "sample-acct-01", open("sample-le-north")),
		line(relationships.Payer, "sample-acct-02", open("sample-le-north")),
		line(relationships.Payer, "sample-acct-03", open("sample-le-north")),
		line(relationships.Payer, "sample-acct-04", open("sample-le-south")),
		line(relationships.Payer, "sample-acct-05", open("sample-le-south")),
		line(relationships.Payer, "sample-acct-06", open("sample-le-south")),
		line(relationships.AllocationPool, "sample-acct-03", open("sample-pool-shared")),
		line(relationships.AllocationPool, "sample-acct-06", open("sample-pool-shared")),
		line(relationships.ViewGroup, "sample-acct-03", []relationships.Entry{
			{Target: "sample-view-north", Interval: relationships.From(start)},
			{Target: "sample-view-south", Interval: relationships.From(start)},
		}),
		line(relationships.ViewGroup, "sample-acct-04", open("sample-view-south")),
	}
}

func (s *Server) seedOrganizations() error {
	ws := s.Workspace()
	party := func(id string, kind organizations.PartyKind, parent string) func() error {
		return func() error {
			_, err := s.orgs.PutParty(organizations.Party{ID: id, WorkspaceID: ws, Kind: kind, ParentID: parent, State: organizations.Active}, 0)
			return err
		}
	}
	profile := func(p organizations.BillingProfile) func() error {
		return func() error {
			p.WorkspaceID, p.State = ws, organizations.Active
			_, err := s.orgs.PutBillingProfile(p, 0)
			return err
		}
	}
	steps := []func() error{
		party("sample-le-seller", organizations.LegalEntity, ""),
		party("sample-le-north", organizations.LegalEntity, ""),
		party("sample-le-south", organizations.LegalEntity, ""),
		party("sample-unit-north-ops", organizations.InternalUnit, "sample-le-north"),
		party("sample-unit-north-data", organizations.InternalUnit, "sample-le-north"),
		party("sample-unit-south-web", organizations.InternalUnit, "sample-le-south"),
		profile(organizations.BillingProfile{ID: "sample-profile-north", IssuerID: "sample-le-seller", BuyerID: "sample-le-north", PayerID: "sample-le-north",
			RecipientID: "sample-le-north", Path: organizations.ExternalInvoice, Currency: "USD"}),
		profile(organizations.BillingProfile{ID: "sample-profile-south", IssuerID: "sample-le-seller", BuyerID: "sample-le-south", PayerID: "sample-le-south",
			RecipientID: "sample-le-south", Path: organizations.ExternalInvoice, Currency: "USD"}),
		profile(organizations.BillingProfile{ID: "sample-profile-north-cb", IssuerID: "sample-le-north", BuyerID: "sample-unit-north-data",
			Path: organizations.InternalChargeback, Currency: "USD"}),
		func() error { return s.rels.Draft(ws, orgSeedPlan, 0) },
		func() error { return s.rels.AddChunk(ws, orgSeedPlan, 0, sampleTimelines()) },
		func() error {
			preview, err := s.rels.Seal(ws, orgSeedPlan, 1)
			if err == nil {
				_, err = s.rels.Activate(ws, orgSeedPlan, preview.Digest, 0)
			}
			return err
		},
	}
	return run(steps...)
}
