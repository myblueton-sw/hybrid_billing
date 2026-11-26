package server

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"

	"github.com/myblueton-sw/hybrid_billing/modules/intake"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
)

type sourceArtifact struct {
	ID       string
	Document string
	Revision uint64
	Kind     intake.KeyKind
	Sums     map[string]string
	Rows     []normalize.Row
}

type sourcePacket struct {
	Mode          string
	CostField     string
	CurrencyField string
	Run           normalize.RunSpec
	Artifacts     []sourceArtifact
}

type sourceCell struct {
	Name string
	Text string
	Null bool
}

type sourceRawRow struct {
	Key   string
	Cells []sourceCell
}

type sourceAdmission struct {
	Difference string `json:"difference"`
	Latest     uint64 `json:"latest_manifest"`
	Staging    int    `json:"staged_artifacts"`
	Behind     int    `json:"behind_revisions"`
	SameBytes  int    `json:"same_bytes"`
	Blockers   int    `json:"blockers"`
}

type sourceCondition struct {
	Reason   string
	Observed sourceAdmission
	Expected uint64
}

func (e sourceCondition) Error() string { return "unsupported source assembly condition: " + e.Reason }

type candidateSourceRow struct {
	Kind     string `json:"kind"`
	Document string `json:"document"`
	Key      string `json:"key"`
	Artifact string `json:"artifact"`
	Digest   string `json:"digest"`
	Amount   string `json:"source_amount"`
	Currency string `json:"source_currency"`
}

type candidateSource struct {
	Workspace   string               `json:"workspace"`
	Provider    string               `json:"provider"`
	Scope       string               `json:"scope"`
	Period      string               `json:"period"`
	Version     string               `json:"source_version"`
	CostField   string               `json:"source_cost_field"`
	Currency    string               `json:"source_currency_field"`
	Manifest    uint64               `json:"manifest"`
	Content     string               `json:"content_digest"`
	Attribution string               `json:"attribution_digest"`
	Admission   sourceAdmission      `json:"recorded_admission"`
	Rows        []candidateSourceRow `json:"rows"`
}

type sourceAssembly struct {
	Print      string
	Packet     []byte
	Base       uint64
	Manifest   uint64
	Complete   bool
	Normalized bool
	Artifacts  map[string]bool
}

type sourcePrepared struct {
	packet  sourcePacket
	bytes   []byte
	print   string
	subject intake.Subject
	specs   []intake.Spec
	records [][]intake.Record
	rows    []normalize.Row
	origins map[normalize.RowKey]candidateSourceRow
}

func sourceCells(cells map[string]normalize.Cell) []sourceCell {
	keys := make([]string, 0, len(cells))
	for key := range cells {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]sourceCell, 0, len(keys))
	for _, key := range keys {
		out = append(out, sourceCell{key, cells[key].Text, cells[key].Null})
	}
	return out
}

func sourcePacketText(packet sourcePacket) bool {
	if len(packet.Artifacts) < 1 || len(packet.Artifacts) > 1000 || len(packet.Run.Expected) > 20 {
		return false
	}
	rows := 0
	for _, artifact := range packet.Artifacts {
		rows += len(artifact.Rows)
		if len(artifact.Rows) < 1 || rows > 1000 || len(artifact.Sums) > 20 {
			return false
		}
		for _, row := range artifact.Rows {
			if len(row.Cells) < 1 || len(row.Cells) > 20 {
				return false
			}
		}
	}
	remaining := candidateBytes
	if !candidateText(reflect.ValueOf(packet), &remaining) {
		return false
	}
	check := func(s string) bool { return candidateText(reflect.ValueOf(s), &remaining) }
	for field, sums := range packet.Run.Expected {
		if len(sums) > 20 || !check(field) {
			return false
		}
		for currency, amount := range sums {
			if !check(currency) || !check(amount) {
				return false
			}
		}
	}
	for _, artifact := range packet.Artifacts {
		for currency, amount := range artifact.Sums {
			if !check(currency) || !check(amount) {
				return false
			}
		}
		for _, row := range artifact.Rows {
			for name, cell := range row.Cells {
				if !check(name) || !check(cell.Text) {
					return false
				}
			}
		}
	}
	return true
}

func (p *normalizedCharges) prepareSource(packet sourcePacket) (sourcePrepared, error) {
	out := sourcePrepared{}
	if packet.Mode != "synthetic" || !validRequestID(packet.CostField) || !validRequestID(packet.CurrencyField) || packet.Run.Input.Manifest != 0 || packet.Run.Rows != 0 || packet.Run.Chunks != 0 || !sourcePacketText(packet) {
		return out, errCandidateInvalid
	}
	data, err := json.Marshal(packet)
	if err != nil || len(data) > candidateBytes {
		return out, errCandidateInvalid
	}
	if err := json.Unmarshal(data, &out.packet); err != nil {
		return out, err
	}
	packet = out.packet
	out.bytes = data
	out.subject = intake.Subject{WorkspaceID: packet.Run.WorkspaceID, Provider: packet.Run.Mapping.Provider, Scope: packet.Run.Input.Scope, Period: packet.Run.Input.Period}
	revision, err := p.normal.Revision(packet.Run.Mapping, packet.Run.Revision)
	if err != nil {
		return out, err
	}
	out.print, err = normalizedDigest(struct {
		Packet  sourcePacket
		Mapping normalize.RevisionView
		Target  normalize.Target
	}{packet, revision, p.target})
	if err != nil {
		return out, err
	}
	out.origins = make(map[normalize.RowKey]candidateSourceRow)
	ids := make(map[string]bool)
	for _, artifact := range packet.Artifacts {
		if ids[artifact.ID] || !validRequestID(artifact.ID) || !validRequestID(artifact.Document) || artifact.Revision == 0 || artifact.Kind != intake.EventKeys && artifact.Kind != intake.LocatorKeys {
			return sourcePrepared{}, errCandidateInvalid
		}
		ids[artifact.ID] = true
		var records []intake.Record
		var raw []sourceRawRow
		for _, row := range artifact.Rows {
			if string(row.Key.Kind) != string(artifact.Kind) || !validRequestID(row.Key.Key) || artifact.Kind == intake.EventKeys && row.Key.Document != "" || artifact.Kind == intake.LocatorKeys && row.Key.Document != artifact.Document {
				return sourcePrepared{}, errCandidateInvalid
			}
			if _, duplicate := out.origins[row.Key]; duplicate {
				return sourcePrepared{}, errCandidateInvalid
			}
			cost, exists := row.Cells[packet.CostField]
			currency, named := row.Cells[packet.CurrencyField]
			if !exists || !named || cost.Null || currency.Null {
				return sourcePrepared{}, errCandidateInvalid
			}
			amount, err := p.sourceAmount(cost.Text)
			if err != nil {
				return sourcePrepared{}, err
			}
			cells := sourceCells(row.Cells)
			digest, err := normalizedDigest(struct {
				Schema                            string
				Subject                           intake.Subject
				Version, CostField, CurrencyField string
				Key                               normalize.RowKey
				Cells                             []sourceCell
			}{"hb_source_row_v1", out.subject, packet.Run.Mapping.SourceVersion, packet.CostField, packet.CurrencyField, row.Key, cells})
			if err != nil {
				return sourcePrepared{}, err
			}
			records = append(records, intake.Record{Key: row.Key.Key, Digest: digest, Amount: amount, Currency: currency.Text})
			raw = append(raw, sourceRawRow{row.Key.Key, cells})
			out.origins[row.Key] = candidateSourceRow{string(row.Key.Kind), row.Key.Document, row.Key.Key, artifact.ID, digest, amount, currency.Text}
			out.rows = append(out.rows, row)
		}
		body, err := json.Marshal(raw)
		if err != nil {
			return sourcePrepared{}, err
		}
		sum, err := normalizedDigest(raw)
		if err != nil {
			return sourcePrepared{}, err
		}
		out.specs = append(out.specs, intake.Spec{ID: artifact.ID, Subject: out.subject, Document: artifact.Document, Kind: artifact.Kind, Revision: artifact.Revision, Bytes: sum, Role: intake.Original, Holds: intake.Rows, Chunks: (len(records) + p.sourceChunk - 1) / p.sourceChunk, Rows: len(records), Sums: artifact.Sums, Totals: intake.SourceStated, Size: int64(len(body)), Expanded: int64(len(body)), Location: "synthetic-memory", Scan: intake.Passed, Syntax: intake.Passed, Bodies: intake.Passed})
		out.records = append(out.records, records)
	}
	return out, nil
}

func sourceAbsent(err error) bool { return errors.Is(err, intake.ErrNotFound) }
