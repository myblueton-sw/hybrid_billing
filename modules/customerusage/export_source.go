package customerusage

import (
	"context"

	ca "github.com/myblueton-sw/hybrid_billing/modules/customeraccess"
)

func (g *Gateway) exportSelection(maxRows int) ca.Selection {
	return ca.Selection{Scopes: []ca.Scope{{CompanyID: g.profile.CompanyID, TenantID: g.profile.TenantID}}, Fields: Fields(), Limit: maxRows}
}

func (g *Gateway) exportService(verify ca.VerifyFunc, read ca.ReadFunc, maxRows int) (*ca.Service, error) {
	return ca.NewServiceForAction(verify, g.store, read, ca.Limits{MaxRows: maxRows, MaxScopes: 1, MaxFields: len(Fields())}, ca.UsageExport)
}

func (g *Gateway) exportSource(ctx context.Context, checkpoint *ca.Checkpoint, maxRows int, consume func([]ca.Row) error) error {
	ctx, cancel := context.WithTimeout(ctx, g.budget.Timeout)
	defer cancel()
	var held *bridge
	defer func() { held.abort() }()
	service, err := g.exportService(g.verify, func(ctx context.Context, selection ca.Selection) ([]ca.Row, error) {
		var rows []ca.Row
		var err error
		held, rows, err = g.start(ctx, selection, "export")
		if err != nil {
			return nil, err
		}
		if err := consume(rows); err != nil {
			return nil, err
		}
		return rows, nil
	}, maxRows)
	if err != nil {
		return err
	}
	if _, err := service.QueryPinned(ctx, g.exportSelection(maxRows), checkpoint); err != nil {
		return err
	}
	if held == nil || held.release() != nil {
		return ca.ErrUnavailable
	}
	return ctx.Err()
}
