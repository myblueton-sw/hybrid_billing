package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

var errServiceViewChanged = errors.New("service query source changed")

type serviceAggregate struct {
	Total    uint64    `json:"total"`
	Active   uint64    `json:"active"`
	Expired  uint64    `json:"expired"`
	Revoked  uint64    `json:"revoked"`
	Earliest time.Time `json:"earliest"`
	Soon     bool      `json:"soon"`
}

type serviceCursor struct {
	Area            string           `json:"area"`
	Target          string           `json:"target"`
	Owner           sessionOwner     `json:"owner"`
	Caller          string           `json:"caller"`
	Query           string           `json:"query"`
	Basis           string           `json:"basis"`
	Reference       time.Time        `json:"reference"`
	Expires         time.Time        `json:"expires"`
	After           string           `json:"after"`
	AfterService    string           `json:"after_service"`
	PendingService  string           `json:"pending_service"`
	AfterCredential string           `json:"after_credential"`
	Completed       uint64           `json:"completed"`
	Matched         uint64           `json:"matched"`
	Aggregate       serviceAggregate `json:"aggregate"`
}

type serviceBasis struct {
	Held          bool
	Stranded      bool
	Identity      identity.ServiceQueryVersion
	Organizations uint64
	Contracts     uint64
	Catalog       string
	Revision      string
	Directory     uint64
	Delivery      uint64
}

func (s *Server) serviceMAC(domain string, body []byte) []byte {
	mac := hmac.New(sha256.New, s.serviceQueryKey)
	mac.Write([]byte(domain))
	mac.Write([]byte{0})
	mac.Write(body)
	return mac.Sum(nil)
}

func (s *Server) sealServiceCursor(cursor serviceCursor) (string, bool) {
	body, err := json.Marshal(cursor)
	if err != nil {
		return "", false
	}
	return base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(s.serviceMAC("service-query-v1", body)), true
}

func (s *Server) openServiceCursor(raw string) (serviceCursor, bool) {
	var out serviceCursor
	if len(raw) > 4096 {
		return out, false
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return out, false
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return out, false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, s.serviceMAC("service-query-v1", body)) {
		return out, false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&out) != nil {
		return out, false
	}
	if _, err = decoder.Token(); err != io.EOF {
		return out, false
	}
	for _, id := range []string{out.After, out.AfterService, out.PendingService, out.AfterCredential} {
		if id != "" && !validWorkspace(id) {
			return out, false
		}
	}
	if out.Aggregate.Active > out.Aggregate.Total || out.Aggregate.Expired > out.Aggregate.Total || out.Aggregate.Revoked > out.Aggregate.Total || out.Reference.IsZero() || !out.Reference.Before(out.Expires) {
		return out, false
	}
	return out, true
}

func (s *Server) serviceBasis(actor identity.Verifier, area string) (string, error) {
	identityBasis, err := s.identity.Workspace().ServiceQueryBasis(actor, s.bootWorkspace)
	if err != nil {
		return "", err
	}
	basis := serviceBasis{Held: s.held(), Stranded: len(s.stranded) > 0, Identity: identityBasis, Catalog: s.serviceCatalog.Version(), Revision: s.Revision(), Directory: s.dir.version}
	if area == "organizations" || area == "scopes" || area == "service_detail" || area == "organization_detail" || area == "scope_detail" {
		if s.faulty("organizations") {
			return "", identity.ErrUnavailable
		}
		basis.Organizations, err = s.orgs.QueryVersion(s.bootWorkspace)
		if err != nil {
			return "", err
		}
	}
	if area == "scopes" || area == "service_detail" || area == "scope_detail" {
		basis.Contracts, err = s.lifecycles.QueryVersion(s.bootWorkspace)
		if err != nil {
			return "", err
		}
	}
	if area == "credentials" || area == "credential_detail" {
		if s.dir.deliveryVersion == math.MaxUint64 {
			return "", identity.ErrUnavailable
		}
		basis.Delivery = s.dir.deliveryVersion
	}
	body, err := json.Marshal(basis)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(s.serviceMAC("service-source-v1", body)), nil
}

type serviceWalk struct {
	s            *Server
	actor        identity.Verifier
	query        serviceQuery
	cursor       serviceCursor
	scanned      int
	stopped      bool
	availability string
	total        uint64
}

func (s *Server) beginServiceQuery(w http.ResponseWriter, r *http.Request, area, target string) (*serviceWalk, bool) {
	request := r.Header.Get("X-Request-Id")
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return nil, false
	}
	if answer.Workspace() != s.bootWorkspace || found.owner != (sessionOwner{answer.User(), answer.Workspace(), answer.Epoch()}) || found.handle == "" {
		failCode(w, commands.Unauthenticated, request)
		return nil, false
	}
	if len(r.URL.RawQuery) > 8192 {
		invalidFields(w, request, "query")
		return nil, false
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		invalidFields(w, request, "query")
		return nil, false
	}
	query, after, field := parseServiceQuery(values, area, s.bootWorkspace, s.cfg.Access.ServiceQueryPage)
	if field != "" {
		invalidFields(w, request, field)
		return nil, false
	}
	basis, err := s.serviceBasis(found.verifier, area)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return nil, false
	}
	now := s.clock()
	expiry := earlier(answer.NotAfter(), now.Add(time.Duration(s.cfg.Access.ServiceQuerySeconds)*time.Second))
	if !found.idleUntil.IsZero() {
		expiry = earlier(expiry, found.idleUntil)
	}
	cursor := serviceCursor{Area: area, Target: target, Owner: found.owner, Caller: found.handle, Query: query.digest(area, target), Basis: basis, Reference: now, Expires: expiry}
	if after != "" {
		prior, valid := s.openServiceCursor(after)
		if !valid || prior.Area != cursor.Area || prior.Target != target || prior.Owner != cursor.Owner || prior.Caller != cursor.Caller || prior.Query != cursor.Query || prior.Basis != basis || !now.Before(prior.Expires) || prior.Reference.After(now) {
			failRead(w, commands.RevisionConflict, request, "after")
			return nil, false
		}
		cursor = prior
	}
	return &serviceWalk{s: s, actor: found.verifier, query: query, cursor: cursor}, true
}

func (walk *serviceWalk) reserve(units int) bool {
	if units > walk.remaining() {
		walk.stopped = true
		return false
	}
	walk.scanned += units
	return true
}

func (walk *serviceWalk) remaining() int { return walk.s.cfg.Access.ServiceQueryScan - walk.scanned }

func (walk *serviceWalk) matched() error {
	if walk.cursor.Matched == math.MaxUint64 {
		return identity.ErrUnavailable
	}
	walk.cursor.Matched++
	return nil
}

func serviceQueryFailure(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, errServiceViewChanged) {
		failRead(w, commands.RevisionConflict, r.Header.Get("X-Request-Id"), "after")
		return
	}
	failCode(w, commands.DependencyUnavailable, r.Header.Get("X-Request-Id"))
}

func writeServiceQueryPage[T any](w http.ResponseWriter, r *http.Request, walk *serviceWalk, rows []T, more bool, order string) {
	s := walk.s
	if _, _, admitted := s.admitted(w, r); !admitted {
		return
	}
	basis, err := s.serviceBasis(walk.actor, walk.cursor.Area)
	if err != nil {
		serviceQueryFailure(w, r, err)
		return
	}
	if basis != walk.cursor.Basis {
		serviceQueryFailure(w, r, errServiceViewChanged)
		return
	}
	next := ""
	certainty := "exact"
	if more {
		var valid bool
		next, valid = s.sealServiceCursor(walk.cursor)
		if !valid {
			failCode(w, commands.InternalError, r.Header.Get("X-Request-Id"))
			return
		}
		certainty = "lower_bound"
	}
	page := serviceQueryPage[T]{Availability: walk.availability, Rows: rows, Query: walk.query, Next: next, Reference: walk.cursor.Reference.UTC().Format(time.RFC3339Nano), Observed: s.clock().UTC().Format(time.RFC3339Nano), Revision: s.Revision(), Order: order,
		Count: serviceQueryCount{Value: strconv.FormatUint(walk.cursor.Matched, 10), Certainty: certainty}, Scanned: walk.scanned, ScanStopped: more && (walk.remaining() == 0 || walk.stopped),
		Settings: append(s.rowsOf(serviceQuerySettings), s.rowsOf(serviceDeclarationSettings)...), Limits: s.queryServiceLimits(), CatalogVersion: s.serviceCatalog.Version()}
	if walk.cursor.Area == "accounts" {
		page.Admission = &serviceAdmission{Recorded: strconv.FormatUint(walk.total, 10), Certainty: "exact", Limit: strconv.Itoa(maxServices)}
	}
	writeJSON(w, http.StatusOK, r.Header.Get("X-Request-Id"), page)
}
