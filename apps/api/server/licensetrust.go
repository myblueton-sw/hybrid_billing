package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/licensemode"
)

const licenseChangePrefix = "license-change-"

func licenseChangeNumber(raw string) (uint64, bool) {
	n, err := strconv.ParseUint(raw, 10, 64)
	return n, err == nil && n > 0 && strconv.FormatUint(n, 10) == raw
}
func licenseChangeTarget(n uint64) string { return licenseChangePrefix + strconv.FormatUint(n, 10) }
func licenseKeyFingerprint(key licensemode.Key) string {
	public, _ := hex.DecodeString(key.Public)
	sum := sha256.Sum256(public)
	return hex.EncodeToString(sum[:])
}

type licenseKeyRow struct {
	ID          string `json:"id"`
	Issuer      string `json:"issuer"`
	Fingerprint string `json:"fingerprint_sha256"`
	NotBefore   int64  `json:"not_before"`
	NotAfter    int64  `json:"not_after"`
	AcceptUntil int64  `json:"accept_until"`
}

func licenseKeyOf(k licensemode.Key) licenseKeyRow {
	return licenseKeyRow{k.ID, k.Issuer, licenseKeyFingerprint(k), k.NotBefore, k.NotAfter, k.AcceptUntil}
}

type licenseChangeRow struct {
	Number    string         `json:"number"`
	Target    string         `json:"target"`
	Kind      string         `json:"kind"`
	State     string         `json:"state"`
	Proposer  string         `json:"proposer"`
	Approver  string         `json:"approver"`
	Key       *licenseKeyRow `json:"key,omitempty"`
	TargetKey string         `json:"target_key,omitempty"`
	RetireAt  int64          `json:"retire_at,omitempty"`
}

func licenseChangeOf(p licensemode.ProposalView) licenseChangeRow {
	row := licenseChangeRow{Number: strconv.FormatUint(p.Number, 10), Target: licenseChangeTarget(p.Number), Kind: p.Change.Kind, State: p.State, Proposer: p.Proposer, Approver: p.Approver}
	switch p.Change.Kind {
	case licensemode.ChangeKey:
		k := licenseKeyOf(p.Change.Key)
		row.Key = &k
	case licensemode.ChangeRetire:
		row.TargetKey, row.RetireAt = p.Change.Retire.Key, p.Change.Retire.At
	case licensemode.ChangeCompromise:
		row.TargetKey = p.Change.Compromise
	}
	return row
}
func (s *Server) readLicenseTrust(w http.ResponseWriter, r *http.Request) {
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
	state := stateNotConfigured
	if answer.Workspace() != s.bootWorkspace || s.cfg.License.Installation == "" {
		writeJSON(w, 200, request, map[string]string{"state": state})
		return
	}
	if s.licenseFailed {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	if s.license == nil {
		writeJSON(w, 200, request, map[string]string{"state": "uninitialized"})
		return
	}
	var value any
	var err error
	if id := r.PathValue("key"); id != "" {
		if !validRequestID(id) {
			failCode(w, commands.InvalidRequest, request)
			return
		}
		var key licensemode.KeyView
		key, err = s.license.Key(id)
		value = struct {
			Key         licenseKeyRow `json:"key"`
			Retired     bool          `json:"retired"`
			Compromised bool          `json:"compromised"`
		}{licenseKeyOf(key.Key), key.Retired, key.Compromised}
	} else {
		number, ok := licenseChangeNumber(r.PathValue("number"))
		if !ok {
			failCode(w, commands.InvalidRequest, request)
			return
		}
		var proposal licensemode.ProposalView
		proposal, err = s.license.Proposal(number)
		value = licenseChangeOf(proposal)
	}
	if errors.Is(err, licensemode.ErrNotFound) {
		writeJSON(w, 200, request, map[string]string{"state": "not_found"})
		return
	}
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	writeJSON(w, 200, request, map[string]any{"state": "configured", "revision": s.Revision(), "record": value})
}

type licenseTrustHandler struct {
	s    *Server
	kind string
}

func (h licenseTrustHandler) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || h.s.cfg.License.Installation == "" {
		return commands.Target{}, commands.NotFound
	}
	if h.kind == "approve_key_change" || h.kind == "withdraw_key_change" {
		raw, found := strings.CutPrefix(target, licenseChangePrefix)
		_, ok := licenseChangeNumber(raw)
		if !found || !ok {
			return commands.Target{}, commands.NotFound
		}
	} else if target != licenseTarget {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}
func licenseUnix(raw string) (int64, bool) {
	n, err := strconv.ParseInt(raw, 10, 64)
	return n, err == nil && n > 0 && n <= 253402300799 && strconv.FormatInt(n, 10) == raw
}
func licenseTrustChange(kind string, fields map[string]string) (licensemode.Change, error) {
	switch kind {
	case "propose_key":
		before, b := licenseUnix(fields["not_before"])
		after, a := licenseUnix(fields["not_after"])
		until, u := licenseUnix(fields["accept_until"])
		if !b || !a || !u {
			return licensemode.Change{}, commands.Fail(commands.InvalidRequest, "not_before", "not_after", "accept_until")
		}
		return licensemode.Change{Kind: licensemode.ChangeKey, Key: licensemode.Key{ID: fields["key_id"], Issuer: fields["issuer"], Public: fields["public_key"], Proof: fields["proof"], NotBefore: before, NotAfter: after, AcceptUntil: until}}, nil
	case "propose_retirement":
		at, ok := licenseUnix(fields["at"])
		if !ok {
			return licensemode.Change{}, commands.Fail(commands.InvalidRequest, "at")
		}
		return licensemode.Change{Kind: licensemode.ChangeRetire, Retire: licensemode.Retire{Key: fields["key_id"], At: at}}, nil
	case "propose_compromise":
		return licensemode.Change{Kind: licensemode.ChangeCompromise, Compromise: fields["key_id"]}, nil
	}
	return licensemode.Change{}, commands.Fail(commands.UnsupportedCapability)
}
func (h licenseTrustHandler) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
	}
	if err := s.humanActor(ctx.User); err != nil {
		return commands.Outcome{}, err
	}
	if s.licenseFailed {
		return commands.Outcome{}, licensemode.ErrStorage
	}
	if s.license == nil {
		return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
	}
	actor := licensemode.Actor{ID: ctx.User, Kind: licensemode.KindHuman}
	var proposal licensemode.ProposalView
	var err error
	switch h.kind {
	case "approve_key_change", "withdraw_key_change":
		raw, found := strings.CutPrefix(ctx.Target, licenseChangePrefix)
		number, ok := licenseChangeNumber(raw)
		if !found || !ok {
			return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
		}
		if h.kind == "approve_key_change" {
			proposal, err = s.license.Approve(number, actor)
		} else {
			proposal, err = s.license.Withdraw(number, actor)
		}
	default:
		change, parseErr := licenseTrustChange(h.kind, ctx.Fields)
		if parseErr != nil {
			return commands.Outcome{}, parseErr
		}
		proposal, err = s.license.Propose(change, actor)
	}
	fields := map[string]string{}
	state := proposal.State
	if err != nil {
		var refusal licensemode.Refusal
		switch {
		case errors.Is(err, licensemode.ErrStorage), errors.Is(err, licensemode.ErrCheckpoint), errors.Is(err, licensemode.ErrOwned):
			return commands.Outcome{}, err
		case errors.As(err, &refusal):
			switch refusal.Reason {
			case licensemode.ReasonSelfApproval:
				return commands.Outcome{}, commands.Fail(commands.SelfApprovalDenied)
			case licensemode.ReasonNotProposer:
				return commands.Outcome{}, commands.Fail(commands.PermissionDenied)
			}
			state = refusal.Reason
		case errors.Is(err, licensemode.ErrInvalid):
			return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
		case errors.Is(err, licensemode.ErrLimit):
			return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
		case errors.Is(err, licensemode.ErrNotFound), errors.Is(err, licensemode.ErrConflict):
			return commands.Outcome{}, commands.Fail(commands.InvalidStage)
		default:
			return commands.Outcome{}, err
		}
	} else {
		fields["proposal"], fields["target"] = strconv.FormatUint(proposal.Number, 10), licenseChangeTarget(proposal.Number)
	}
	fields["outcome"] = state
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: state, Fields: fields}, nil
}
func (s *Server) licenseTrustDefinitions() []commands.Definition {
	define := func(kind string, fields ...commands.Field) commands.Definition {
		return commands.Definition{ID: "licenses." + kind, Permission: commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}, Mutating: true, Rationale: true, Refusals: []commands.Code{commands.SelfApprovalDenied}, Fields: fields, Handler: licenseTrustHandler{s, kind}}
	}
	field := func(name string, kind commands.FieldType) commands.Field {
		return commands.Field{Name: name, Type: kind, Required: true}
	}
	return []commands.Definition{
		define("propose_key", field("key_id", commands.ID), field("issuer", commands.ID), field("public_key", commands.Text), field("proof", commands.Text), field("not_before", commands.Integer), field("not_after", commands.Integer), field("accept_until", commands.Integer)),
		define("propose_retirement", field("key_id", commands.ID), field("at", commands.Integer)),
		define("propose_compromise", field("key_id", commands.ID)), define("approve_key_change"), define("withdraw_key_change"),
	}
}
