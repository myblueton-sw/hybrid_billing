package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

const (
	contractsShown     = 200
	contractSeedActor  = "seed"
	contractSeedReason = "sample seed"
)

type contractMeta struct {
	workspace string
	id        string
	name      string
	profile   string
	timezone  string
	cycle     string
}

type writeMeta struct {
	actor  string
	reason string
}

type contractStore struct {
	byKey  map[string]*contractMeta
	writes map[string]writeMeta
}

func newContractStore() *contractStore {
	return &contractStore{byKey: map[string]*contractMeta{}, writes: map[string]writeMeta{}}
}

func (c *contractStore) get(workspace, id string) *contractMeta {
	return c.byKey[planKey(workspace, id)]
}

func (c *contractStore) note(workspace, id, write, actor, reason string) {
	c.writes[planKey(workspace, id)+"\x00"+write] = writeMeta{actor: actor, reason: reason}
}

func (c *contractStore) written(workspace, id, write string) writeMeta {
	return c.writes[planKey(workspace, id)+"\x00"+write]
}

func ContractLimits() contracts.Limits {
	return contracts.Limits{Contracts: contractsShown, Writes: 200, Reserve: 20, Batch: 8}
}

func contractDay(year, month, day int) contracts.Day {
	d, _ := contracts.NewDay(year, month, day)
	return d
}

type sampleContract struct {
	id        string
	name      string
	profile   string
	customer  string
	seller    string
	revisions [][]organizations.Component
	events    []contracts.Event
	after     func(s *Server, ws, id string) error
}

func resaleComponents(extra ...string) []organizations.Component {
	out := []organizations.Component{
		{ID: "cloud-usage", Kind: organizations.ProviderCharge, ProviderPaymentResponsibleID: "sample-le-seller", SellerBillable: true},
		{ID: "managed-service", Kind: organizations.SellerService},
	}
	for _, id := range extra {
		out = append(out, organizations.Component{ID: id, Kind: organizations.SellerService})
	}
	return out
}

func sampleContracts() []sampleContract {
	ev := func(id string, kind contracts.Kind, from contracts.Day, revision uint64) contracts.Event {
		return contracts.Event{ID: id, Kind: kind, From: from, Revision: revision}
	}
	return []sampleContract{
		{id: "sample-contract-01", name: "North managed cloud (sample)", profile: "sample-profile-north", customer: "sample-le-north", seller: "sample-le-seller",
			revisions: [][]organizations.Component{resaleComponents(), resaleComponents("support-plus")},
			events:    []contracts.Event{ev("ev-01-start", contracts.Start, contractDay(2026, 1, 1), 1), ev("ev-01-change", contracts.Change, contractDay(2026, 7, 1), 2)}},
		{id: "sample-contract-02", name: "North analytics (sample)", profile: "sample-profile-north", customer: "sample-le-north", seller: "sample-le-seller",
			revisions: [][]organizations.Component{resaleComponents()},
			events:    []contracts.Event{ev("ev-02-start", contracts.Start, contractDay(2026, 1, 1), 1), ev("ev-02-suspend", contracts.Suspend, contractDay(2026, 9, 1), 0)}},
		{id: "sample-contract-03", name: "South web hosting (sample)", profile: "sample-profile-south", customer: "sample-le-south", seller: "sample-le-seller",
			revisions: [][]organizations.Component{resaleComponents()},
			events:    []contracts.Event{ev("ev-03-start", contracts.Start, contractDay(2026, 3, 1), 1), ev("ev-03-end", contracts.End, contractDay(2026, 10, 1), 0)},
			after: func(s *Server, ws, id string) error {
				return s.seedWrite(ws, id, "cw-03-gap", func(seq uint64) (contracts.Outcome, error) {
					return s.lifecycles.RecordEvidence(ws, id, seq, "cw-03-gap", contracts.Evidence{Kind: contracts.Missing, Reason: contracts.SourceUnavailable,
						OwnerID: SampleAdmin, FollowUp: "sample-followup-03"})
				})
			}},
		{id: "sample-contract-04", name: "South archive storage (sample)", profile: "sample-profile-south", customer: "sample-le-south", seller: "sample-le-seller",
			revisions: [][]organizations.Component{resaleComponents()},
			events:    []contracts.Event{ev("ev-04-start", contracts.Start, contractDay(2026, 2, 1), 1), ev("ev-04-end", contracts.End, contractDay(2026, 8, 1), 0)},
			after: func(s *Server, ws, id string) error {
				if err := s.seedWrite(ws, id, "cw-04-final", func(seq uint64) (contracts.Outcome, error) {
					return s.lifecycles.RecordEvidence(ws, id, seq, "cw-04-final", contracts.Evidence{Kind: contracts.Secured, Reference: "sample-final-04",
						CoveredUntil: contractDay(2026, 8, 1)})
				}); err != nil {
					return err
				}
				return s.seedWrite(ws, id, "cw-04-settle", func(seq uint64) (contracts.Outcome, error) {
					return s.lifecycles.Settle(ws, id, seq, "cw-04-settle", contractDay(2026, 8, 1))
				})
			}},
		{id: "sample-contract-05", name: "North data chargeback (sample)", profile: "sample-profile-north-cb", customer: "sample-le-north",
			revisions: [][]organizations.Component{{{ID: "cloud-usage", Kind: organizations.ProviderCharge, ProviderPaymentResponsibleID: "sample-le-north"}}},
			events:    []contracts.Event{ev("ev-05-start", contracts.Start, contractDay(2027, 1, 1), 1)}},
	}
}

func (s *Server) seedWrite(ws, id, write string, call func(seq uint64) (contracts.Outcome, error)) error {
	seq, err := s.lifecycles.Sequence(ws, id)
	if err != nil {
		return err
	}
	if _, err := call(seq); err != nil {
		return err
	}
	s.contracts.note(ws, id, write, contractSeedActor, contractSeedReason)
	return nil
}

func (s *Server) seedContracts() error {
	ws := s.Workspace()
	for _, c := range sampleContracts() {
		path := organizations.Resale
		if c.seller == "" {
			path = organizations.Direct
		}
		for index, components := range c.revisions {
			if err := s.orgs.AppendContractRevision(organizations.ContractRevision{ContractID: c.id, Revision: uint64(index + 1), WorkspaceID: ws,
				ProviderRef: "sample-provider-" + c.id[len(c.id)-2:], CustomerID: c.customer, SellerID: c.seller, Path: path, Components: components}); err != nil {
				return err
			}
		}
		write := "cw-" + c.id[len(c.id)-2:] + "-seed"
		if _, err := s.lifecycles.Amend(ws, c.id, 0, write, nil, c.events, ""); err != nil {
			return err
		}
		s.contracts.byKey[planKey(ws, c.id)] = &contractMeta{workspace: ws, id: c.id, name: c.name, profile: c.profile, timezone: "UTC", cycle: "monthly"}
		s.contracts.note(ws, c.id, write, contractSeedActor, contractSeedReason)
		if c.after != nil {
			if err := c.after(s, ws, c.id); err != nil {
				return err
			}
		}
	}
	return nil
}
