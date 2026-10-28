package customerusage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	ca "github.com/myblueton-sw/hybrid_billing/modules/customeraccess"
)

type ExportManifest struct {
	Format          string   `json:"format"`
	ProfileRevision string   `json:"profile_revision"`
	Snapshot        string   `json:"snapshot"`
	Scope           ca.Scope `json:"scope"`
	Start           string   `json:"start"`
	End             string   `json:"end"`
	Timezone        string   `json:"timezone"`
	Fields          []string `json:"fields"`
	AmountBasis     string   `json:"amount_basis"`
	Currency        string   `json:"currency"`
	Classification  string   `json:"classification"`
	RangeBasis      string   `json:"range_basis"`
	Completion      string   `json:"completion"`
	RowCount        int      `json:"row_count"`
	PayloadSHA256   string   `json:"payload_sha256"`
}

type ExportPackage struct {
	Manifest ExportManifest `json:"manifest"`
	Rows     []ca.Row       `json:"rows"`
}

type Download struct {
	Filename    string
	ContentType string
	SHA256      string
	Bytes       []byte
}

func hashBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func (g *Gateway) packageRows(rows []ca.Row, maxBytes int) ([]byte, error) {
	payload, err := json.Marshal(rows)
	if err != nil || len(payload) > maxBytes {
		return nil, ca.ErrUnavailable
	}
	manifest := ExportManifest{Format: "usage-export-v1", ProfileRevision: g.revision, Snapshot: g.profile.Snapshot,
		Scope: ca.Scope{CompanyID: g.profile.CompanyID, TenantID: g.profile.TenantID}, Start: g.profile.Start, End: g.profile.End,
		Timezone: "UTC", Fields: Fields(), AmountBasis: "usage-only", Currency: "not-applicable", Classification: "exploratory-query-only",
		RangeBasis: "delivery-window-overlap; charge-interval-contained; no-proration", Completion: "complete-supported-selection",
		RowCount: len(rows), PayloadSHA256: hashBytes(payload)}
	data, err := json.Marshal(ExportPackage{Manifest: manifest, Rows: rows})
	if err != nil || len(data) > maxBytes {
		return nil, ca.ErrUnavailable
	}
	return data, nil
}
