package customerusage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"reflect"

	ca "github.com/myblueton-sw/hybrid_billing/modules/customeraccess"
)

type Gateway struct {
	verify   ca.VerifyFunc
	store    *ca.Store
	command  []string
	profile  Profile
	raw      string
	revision string
	budget   Budget
}

type bridge struct {
	cmd     *exec.Cmd
	input   io.WriteCloser
	output  *bufio.Reader
	limited *io.LimitedReader
	reaped  bool
}

func decodeLine(reader *bufio.Reader, target any) error {
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return ca.ErrUnavailable
	}
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return ca.ErrUnavailable
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return ca.ErrUnavailable
	}
	return nil
}

func (b *bridge) abort() {
	if b == nil || b.reaped {
		return
	}
	b.input.Close()
	b.cmd.Process.Kill()
	b.cmd.Wait()
	b.reaped = true
}

func (b *bridge) release() error {
	if _, err := io.WriteString(b.input, "release\n"); err != nil {
		return ca.ErrUnavailable
	}
	if err := b.input.Close(); err != nil {
		return ca.ErrUnavailable
	}
	var acknowledgment struct {
		Status string `json:"status"`
	}
	if decodeLine(b.output, &acknowledgment) != nil || acknowledgment.Status != "released" {
		return ca.ErrUnavailable
	}
	if _, err := b.output.ReadByte(); err != io.EOF || b.limited.N == 0 {
		return ca.ErrUnavailable
	}
	err := b.cmd.Wait()
	b.reaped = true
	if err != nil {
		return ca.ErrUnavailable
	}
	return nil
}

func (g *Gateway) start(ctx context.Context, selection ca.Selection, mode string) (*bridge, []ca.Row, error) {
	cmd := exec.CommandContext(ctx, g.command[0], g.command[1:]...)
	cmd.Stderr = io.Discard
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, ca.ErrUnavailable
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		return nil, nil, ca.ErrUnavailable
	}
	if err := cmd.Start(); err != nil {
		input.Close()
		output.Close()
		return nil, nil, ca.ErrUnavailable
	}
	limited := &io.LimitedReader{R: output, N: g.budget.Bytes + 1}
	b := &bridge{cmd: cmd, input: input, output: bufio.NewReader(limited), limited: limited}
	request := struct {
		Profile string `json:"profile"`
		Limit   int    `json:"limit"`
		Mode    string `json:"mode,omitempty"`
	}{g.raw, selection.Limit, mode}
	if json.NewEncoder(input).Encode(request) != nil {
		b.abort()
		return nil, nil, ca.ErrUnavailable
	}
	var response struct {
		Status    string   `json:"status"`
		Revision  string   `json:"revision"`
		Rows      []ca.Row `json:"rows"`
		Selection string   `json:"selection"`
		RowCount  *int     `json:"row_count"`
	}
	if decodeLine(b.output, &response) != nil || response.Status != "ready" || response.Revision != g.revision || len(response.Rows) > selection.Limit {
		b.abort()
		return nil, nil, ca.ErrUnavailable
	}
	if mode == "export" && (response.Selection != "complete-supported-selection" || response.RowCount == nil || *response.RowCount != len(response.Rows)) {
		b.abort()
		return nil, nil, ca.ErrUnavailable
	}
	return b, response.Rows, nil
}

func (g *Gateway) Query(ctx context.Context, request ca.Selection) (*Page, error) {
	if ctx == nil || g == nil || len(request.Scopes) != 1 || request.Scopes[0] != (ca.Scope{CompanyID: g.profile.CompanyID, TenantID: g.profile.TenantID}) || !reflect.DeepEqual(request.Fields, Fields()) {
		return nil, ca.ErrDenied
	}
	ctx, cancel := context.WithTimeout(ctx, g.budget.Timeout)
	defer cancel()
	var held *bridge
	defer func() { held.abort() }()
	service, err := ca.NewService(func(ctx context.Context) (ca.Principal, error) {
		p, err := g.verify(ctx)
		if err != nil || p.Installation != g.profile.Installation {
			return ca.Principal{}, ca.ErrDenied
		}
		return p, nil
	}, g.store, func(ctx context.Context, selection ca.Selection) ([]ca.Row, error) {
		var rows []ca.Row
		var err error
		held, rows, err = g.start(ctx, selection, "page")
		return rows, err
	}, ca.Limits{MaxRows: g.budget.Rows, MaxScopes: 1, MaxFields: len(Fields())})
	if err != nil {
		return nil, err
	}
	rows, err := service.Query(ctx, request)
	if err != nil {
		return nil, err
	}
	if held == nil || held.release() != nil {
		return nil, ca.ErrUnavailable
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &Page{Rows: rows, ProfileRevision: g.revision, Snapshot: g.profile.Snapshot, Start: g.profile.Start, End: g.profile.End,
		Timezone: "UTC", AmountBasis: "usage-only", Currency: "not-applicable", Classification: "exploratory-query-only",
		RangeBasis: "delivery-window-overlap; charge-interval-contained; no-proration", Coverage: "bounded-page; no-completeness-claim"}, nil
}
