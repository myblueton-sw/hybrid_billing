package server

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/licensemode"
)

const licenseProduct = "hybrid-billing"
const licenseTarget = "license-installation"

type LicenseSettings struct {
	Installation string
	Authority    licensemode.Authority
}

type unavailableLicenseTime struct{}

func (unavailableLicenseTime) Now() (time.Time, bool) { return time.Time{}, false }

func (s *Server) licenseConfig() licensemode.Config {
	authority := s.cfg.License.Authority
	if authority == nil {
		authority = unavailableLicenseTime{}
	}
	return licensemode.Config{Authority: authority}
}
func (s *Server) licenseDir() string { return filepath.Join(s.cfg.Data, "license-state") }
func (s *Server) openLicense() error {
	if s.cfg.License.Installation == "" {
		return nil
	}
	_, err := os.Lstat(s.licenseDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return ErrStart
	}
	s.license, err = licensemode.OpenDurable(s.licenseDir(), s.cfg.License.Installation, licenseProduct, s.licenseConfig(), s.clock)
	if err != nil {
		return errors.Join(ErrStart, err)
	}
	return nil
}

type licenseStatusRow struct {
	Mode             string                 `json:"mode"`
	State            string                 `json:"state"`
	Reason           string                 `json:"reason"`
	Warning          bool                   `json:"warning"`
	Actions          []string               `json:"actions"`
	RemainingSeconds int64                  `json:"remaining_seconds"`
	LastOutcome      string                 `json:"last_outcome"`
	LastVerified     string                 `json:"last_verified"`
	LastVerifiedAt   string                 `json:"last_verified_at"`
	Entitlement      *licenseEntitlementRow `json:"entitlement"`
	Evaluation       *licenseEvaluationRow  `json:"evaluation"`
}
type licenseEntitlementRow struct {
	ID        string   `json:"id"`
	SubjectID string   `json:"subject_id"`
	Subject   string   `json:"subject"`
	Agreement string   `json:"agreement"`
	Scope     []string `json:"scope"`
	Channel   string   `json:"channel"`
	Metric    string   `json:"metric"`
	Start     int64    `json:"start"`
	End       int64    `json:"end"`
	Perpetual bool     `json:"perpetual"`
}
type licenseEvaluationRow struct {
	Grant   string   `json:"grant"`
	Subject string   `json:"subject"`
	Terms   string   `json:"terms"`
	Scope   []string `json:"scope"`
	Start   string   `json:"start"`
	End     string   `json:"end"`
}
type licensePage struct {
	Revision      string            `json:"revision"`
	ObservedAt    string            `json:"observed_at"`
	State         string            `json:"state"`
	Installation  string            `json:"installation,omitempty"`
	Product       string            `json:"product,omitempty"`
	TimeAuthority string            `json:"time_authority,omitempty"`
	Status        *licenseStatusRow `json:"status,omitempty"`
}

func (s *Server) readLicense(w http.ResponseWriter, r *http.Request) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	request := r.Header.Get("X-Request-Id")
	if r.URL.RawQuery != "" {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	page := licensePage{Revision: s.Revision(), ObservedAt: stamp(s.clock()), State: stateNotConfigured}
	if answer.Workspace() != s.bootWorkspace || s.cfg.License.Installation == "" {
		writeJSON(w, 200, request, page)
		return
	}
	page.Installation, page.Product, page.State = s.cfg.License.Installation, licenseProduct, "uninitialized"
	page.TimeAuthority = "not_connected"
	if s.cfg.License.Authority != nil {
		page.TimeAuthority = "configured"
	}
	if s.license != nil {
		v, err := s.license.Status()
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		page.State = "configured"
		page.Status = &licenseStatusRow{Mode: v.Mode, State: v.State, Reason: v.Reason, Warning: v.Warning, Actions: v.Actions, RemainingSeconds: int64(v.Remaining / time.Second), LastOutcome: v.LastOutcome, LastVerified: v.LastVerified, LastVerifiedAt: stampOrEmpty(v.LastVerifiedAt)}
		if v.Entitlement != nil {
			c := v.Entitlement
			page.Status.Entitlement = &licenseEntitlementRow{ID: c.ID, SubjectID: c.SubjectID, Subject: c.Subject, Agreement: c.Agreement, Scope: c.Scope, Channel: c.Channel, Metric: c.Metric, Start: c.Start, End: c.End, Perpetual: c.Perpetual}
		}
		if v.Activation != nil {
			a := v.Activation
			page.Status.Evaluation = &licenseEvaluationRow{Grant: a.Grant, Subject: a.Subject, Terms: a.Terms, Scope: a.Scope, Start: stamp(a.Start), End: stamp(a.End)}
		}
	}
	writeJSON(w, 200, request, page)
}
