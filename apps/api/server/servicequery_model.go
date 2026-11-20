package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type serviceQuery struct {
	Workspace    string `json:"workspace"`
	Q            string `json:"q"`
	Organization string `json:"organization"`
	Owner        string `json:"owner"`
	Audience     string `json:"audience"`
	Scope        string `json:"scope"`
	Expiry       string `json:"expiry"`
	State        string `json:"state"`
	Declaration  string `json:"declaration"`
	Service      string `json:"service"`
	Collection   string `json:"collection"`
	Version      string `json:"version"`
	Limit        int    `json:"limit"`
}

func parseServiceQuery(values url.Values, area, workspace string, limit int) (serviceQuery, string, string) {
	q := serviceQuery{Workspace: workspace, Limit: limit}
	switch area {
	case "accounts":
		q.Expiry = "all"
		q.State = "all"
		q.Declaration = "all"
	case "credentials":
		q.State = "all"
		q.Collection = "all"
	}
	after := ""
	keys := make([]string, 0, len(values))
	for name := range values {
		keys = append(keys, name)
	}
	slices.Sort(keys)
	for _, name := range keys {
		list := values[name]
		if len(list) != 1 || list[0] == "" {
			return q, "", name
		}
		raw := list[0]
		switch name {
		case "q":
			if area == "history" {
				return q, "", name
			}
			value, ok := parseSearch(raw)
			if !ok || strings.ContainsFunc(value, unicode.IsControl) {
				return q, "", name
			}
			q.Q = strings.ToLower(value)
		case "workspace":
			if raw != workspace {
				return q, "", name
			}
		case "limit":
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > 100 || strconv.Itoa(n) != raw {
				return q, "", name
			}
			q.Limit = n
		case "after":
			if len(raw) > 4096 {
				return q, "", name
			}
			after = raw
		case "organization", "owner", "audience", "scope":
			if area != "accounts" || !validWorkspace(raw) {
				return q, "", name
			}
			switch name {
			case "organization":
				q.Organization = raw
			case "owner":
				q.Owner = raw
			case "audience":
				q.Audience = raw
			case "scope":
				q.Scope = raw
			}
		case "service":
			if area != "credentials" || !validWorkspace(raw) {
				return q, "", name
			}
			q.Service = raw
		case "expiry":
			if area != "accounts" || !oneServiceValue(raw, "all", "none", "expired", "expiring", "valid") {
				return q, "", name
			}
			q.Expiry = raw
		case "state":
			if area == "accounts" && oneServiceValue(raw, "all", "active", "suspended", "retired") || area == "credentials" && oneServiceValue(raw, "all", "active", "expired", "revoked") {
				q.State = raw
			} else {
				return q, "", name
			}
		case "declaration":
			if area != "accounts" || !oneServiceValue(raw, "all", "recorded", "missing") {
				return q, "", name
			}
			q.Declaration = raw
		case "collection":
			if area != "credentials" || !oneServiceValue(raw, "all", "uncollected") {
				return q, "", name
			}
			q.Collection = raw
		case "version":
			version, err := strconv.ParseUint(raw, 10, 64)
			if area != "history" || err != nil || version == 0 || strconv.FormatUint(version, 10) != raw {
				return q, "", name
			}
			q.Version = raw
		default:
			return q, "", name
		}
	}
	if q.Version != "" && after != "" {
		return q, "", "after"
	}
	if q.Scope != "" && q.Audience == "" {
		return q, "", "audience"
	}
	return q, after, ""
}

func oneServiceValue(value string, options ...string) bool {
	for _, candidate := range options {
		if value == candidate {
			return true
		}
	}
	return false
}

func (q serviceQuery) digest(area, target string) string {
	raw, _ := json.Marshal(struct {
		Area, Target string
		Query        serviceQuery
	}{area, target, q})
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

type serviceQueryCount struct {
	Value     string `json:"value"`
	Certainty string `json:"certainty"`
}

type serviceQueryPage[T any] struct {
	Admission      *serviceAdmission `json:"admission,omitempty"`
	Availability   string            `json:"availability,omitempty"`
	Rows           []T               `json:"rows"`
	Query          serviceQuery      `json:"query"`
	Next           string            `json:"next"`
	Reference      string            `json:"reference_at"`
	Observed       string            `json:"observed_at"`
	Revision       string            `json:"revision"`
	Order          string            `json:"order"`
	Count          serviceQueryCount `json:"count"`
	Scanned        int               `json:"work_units"`
	ScanStopped    bool              `json:"scan_stopped"`
	Settings       []settingRow      `json:"settings"`
	Limits         serviceLimits     `json:"limits"`
	CatalogVersion string            `json:"catalog_version"`
}
