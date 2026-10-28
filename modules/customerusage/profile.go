package customerusage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	ca "github.com/myblueton-sw/hybrid_billing/modules/customeraccess"
)

type Profile struct {
	Kind         string            `json:"kind"`
	Installation string            `json:"installation"`
	Source       string            `json:"source"`
	Actor        string            `json:"actor"`
	CompanyID    string            `json:"company_id"`
	TenantID     string            `json:"tenant_id"`
	Snapshot     string            `json:"snapshot"`
	Start        string            `json:"start"`
	End          string            `json:"end"`
	Grain        string            `json:"grain"`
	Services     map[string]string `json:"services"`
	Units        map[string]string `json:"units"`
}

type Budget struct {
	Rows    int
	Bytes   int64
	Timeout time.Duration
}

type Page struct {
	Rows            []ca.Row
	ProfileRevision string
	Snapshot        string
	Start           string
	End             string
	Timezone        string
	AmountBasis     string
	Currency        string
	Classification  string
	RangeBasis      string
	Coverage        string
}

var uuidPattern = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
var snapshotPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func Fields() []string {
	return []string{"service", "quantity", "unit", "usage_start", "usage_end", "measurement_state"}
}

func validProfile(p Profile) bool {
	if p.Kind != "source-usage-v1" || !uuidPattern.MatchString(p.Installation) || !uuidPattern.MatchString(p.TenantID) || !snapshotPattern.MatchString(p.Snapshot) || (p.Grain != "day" && p.Grain != "month") {
		return false
	}
	for _, value := range []string{p.Source, p.Actor, p.CompanyID} {
		if value == "" || len(value) > 256 || strings.TrimSpace(value) != value {
			return false
		}
	}
	start, err := time.Parse(time.RFC3339Nano, p.Start)
	end, endErr := time.Parse(time.RFC3339Nano, p.End)
	if err != nil || endErr != nil || !strings.HasSuffix(p.Start, "Z") || !strings.HasSuffix(p.End, "Z") || !start.Before(end) {
		return false
	}
	for _, labels := range []map[string]string{p.Services, p.Units} {
		if len(labels) == 0 || len(labels) > 100 {
			return false
		}
		for source, label := range labels {
			if source == "" || len(source) > 4096 || label == "" || len(label) > 128 {
				return false
			}
			for _, r := range label {
				if unicode.IsControl(r) {
					return false
				}
			}
		}
	}
	return true
}

func NewGateway(verify ca.VerifyFunc, store *ca.Store, command []string, profile Profile, budget Budget) (*Gateway, error) {
	if len(command) == 0 || !filepath.IsAbs(command[0]) || !validProfile(profile) || budget.Rows < 1 || budget.Rows > 100 || budget.Bytes < 128 || budget.Bytes > 1048576 || budget.Timeout <= 0 || budget.Timeout > 30*time.Second {
		return nil, ca.ErrInvalidPolicy
	}
	if _, err := ca.NewService(verify, store, func(_ctx context.Context, _selection ca.Selection) ([]ca.Row, error) { return nil, nil }, ca.Limits{MaxRows: budget.Rows, MaxScopes: 1, MaxFields: len(Fields())}); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(profile)
	if err != nil || len(raw) > 60000 {
		return nil, ca.ErrInvalidPolicy
	}
	var owned Profile
	if json.Unmarshal(raw, &owned) != nil {
		return nil, ca.ErrInvalidPolicy
	}
	digest := sha256.Sum256(raw)
	return &Gateway{verify: verify, store: store, command: append([]string(nil), command...), profile: owned, raw: string(raw), revision: hex.EncodeToString(digest[:]), budget: budget}, nil
}
