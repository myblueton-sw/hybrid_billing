package server

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func (s *Server) seedDocumentERP() error {
	c, err := s.chargeProducer.candidates.read(s.Workspace(), "sample-charge-01")
	if err != nil {
		return err
	}
	numbers, err := decimal.NewProfile(candidateProfile())
	if err != nil {
		return err
	}
	r, err := erpledger.NewRegistry(numbers, erpledger.Limits{Workspaces: 2, Connections: 10, Mappings: 1000, Anchors: 100, Submissions: 100, Lines: 100, Documents: 100, Rows: 100, Observations: 100, Links: 100, Surveys: 100}, s.clock)
	if err != nil {
		return err
	}
	actor := erpledger.Actor{ID: "synthetic-document-author", Kind: erpledger.KindHuman}
	connection := "synthetic-document-erp"
	_, err = r.Connect(s.Workspace(), erpledger.Connection{ID: connection, Entity: c.Envelope.Input.Normalized.Document.IssuerLegalEntityID, System: "synthetic-erp", Transport: erpledger.TransportAPI, Capabilities: []string{erpledger.CommandPrepare, erpledger.CommandMapping}, Currency: "USD", Scale: 2, Interval: time.Hour, Owner: actor.ID}, actor)
	if err != nil {
		return err
	}
	for _, mapping := range []erpledger.Mapping{{Connection: connection, Kind: "customer", Platform: c.Envelope.Input.Customer, External: "synthetic-customer"}, {Connection: connection, Kind: "product", Platform: "document-product", External: "synthetic-product"}, {Connection: connection, Kind: "tax", Platform: "document-tax", External: "synthetic-tax"}} {
		if _, err = r.Map(s.Workspace(), mapping, actor); err != nil {
			return err
		}
	}
	_, err = r.Survey(s.Workspace(), erpledger.Survey{Connection: connection, Customer: c.Envelope.Input.Customer, Period: "2026-10", Outcome: erpledger.SurveyComplete, Evidence: "synthetic-survey"}, actor)
	if err != nil {
		return err
	}
	s.documentERP, err = newFinancialERP(s, r)
	return err
}
