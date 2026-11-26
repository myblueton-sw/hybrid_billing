package server

import (
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	pickupOpen      = "open"
	pickupCollected = "collected"
	pickupNever     = "never_collected"
	servicesTarget  = "services"
	servicePrefix   = "svc-"
	secretPrefix    = "hbsk_"
	maxServices     = 200
)

type credentialRow struct {
	ID         string `json:"id"`
	NotBefore  string `json:"not_before"`
	NotAfter   string `json:"not_after"`
	State      string `json:"state"`
	Usable     bool   `json:"usable"`
	Pickup     string `json:"secret_pickup,omitempty"`
	PickupEnd  string `json:"secret_pickup_until,omitempty"`
	Replaces   string `json:"replaces,omitempty"`
	ReplacedBy string `json:"replaced_by,omitempty"`
	Rotation   string `json:"rotation,omitempty"`
}

type serviceLimits struct {
	CommandValueBytes     string `json:"command_value_bytes"`
	RationaleBytes        int    `json:"rationale_bytes"`
	BodyBytes             int    `json:"body_bytes"`
	CredentialDays        int    `json:"credential_days"`
	CredentialsPerService string `json:"credentials_per_service"`
	Services              int    `json:"services"`
	ReplaySeconds         int    `json:"replay_seconds"`
	OverlapMinutes        int    `json:"credential_overlap_minutes"`
	OverlapMaxMinutes     int    `json:"credential_overlap_max_minutes"`
	PickupMinutes         int    `json:"secret_pickup_minutes"`
}

const (
	rotationReplacing = "replacing"
	rotationEnding    = "ending"
)

func (s *Server) overlapBounds() (int, int, int) {
	low := int(pickupValidity / time.Minute)
	high := s.cfg.Access.CredentialOverlapMax
	if longest := int(s.cfg.Identity.CredentialValidity / time.Minute); longest < high {
		high = longest
	}
	if high < low {
		high = low
	}
	usual := s.cfg.Access.CredentialOverlap
	if usual > high {
		usual = high
	}
	if usual < low {
		usual = low
	}
	return low, usual, high
}

func wholeNumber(raw string, low, high int) (int, bool) {
	if len(raw) == 0 || len(raw) > 6 || (len(raw) > 1 && raw[0] == '0') || strings.TrimLeft(raw, "0123456789") != "" {
		return 0, false
	}
	n, _ := strconv.Atoi(raw)
	return n, n >= low && n <= high
}

func serviceState(state identity.UserState) string {
	switch state {
	case identity.Suspended:
		return "suspended"
	case identity.Departed:
		return "retired"
	}
	return "active"
}

func credentialState(c identity.CredentialInfo, now time.Time) string {
	switch {
	case c.Revoked:
		return "revoked"
	case !now.Before(c.Validity.NotAfter):
		return "expired"
	}
	return "active"
}

func (s *Server) serviceUsers(actor identity.Verifier) ([]identity.ServiceInfo, error) {
	if s.faulty("services") {
		return nil, identity.ErrUnavailable
	}
	out := []identity.ServiceInfo{}
	after := ""
	for {
		batch, err := s.identity.Workspace().ServiceUsers(actor, s.bootWorkspace, after, s.cfg.Identity.PageSize)
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
		if len(batch) < s.cfg.Identity.PageSize {
			return out, nil
		}
		after = batch[len(batch)-1].ID
	}
}

func (s *Server) credentials(actor identity.Verifier, service string) ([]identity.CredentialInfo, error) {
	if s.faulty("credentials") {
		return nil, identity.ErrUnavailable
	}
	return s.identity.Workspace().Credentials(actor, s.bootWorkspace, service)
}

func orderCredentials(list []identity.CredentialInfo, now time.Time) {
	sort.SliceStable(list, func(a, b int) bool {
		x, y := credentialState(list[a], now) == "active", credentialState(list[b], now) == "active"
		if x != y {
			return x
		}
		return list[a].Validity.NotAfter.After(list[b].Validity.NotAfter)
	})
}

func (s *Server) credentialRows(actor identity.Verifier, svc identity.ServiceInfo, now time.Time) ([]credentialRow, error) {
	list, err := s.credentials(actor, svc.ID)
	if err != nil {
		return nil, err
	}
	orderCredentials(list, now)
	active := map[string]bool{}
	for _, c := range list {
		active[c.ID] = credentialState(c, now) == "active"
	}
	out := []credentialRow{}
	for _, c := range list {
		state := credentialState(c, now)
		row := credentialRow{ID: c.ID, NotBefore: stamp(c.Validity.NotBefore), NotAfter: stamp(c.Validity.NotAfter), State: state,
			Usable: state == "active" && svc.State == identity.Active, Replaces: c.Replaces, ReplacedBy: c.ReplacedBy}
		switch {
		case active[c.ID] && c.ReplacedBy != "":
			row.Rotation = rotationEnding
		case active[c.ID] && active[c.Replaces]:
			row.Rotation = rotationReplacing
		}
		out = append(out, row)
	}
	return out, nil
}

func (s *Server) pickupLive(p pickup) (time.Time, bool) {
	found, ok := s.sessions.get(p.session)
	if !ok || s.sessions.isDenied(p.session) || s.stale(found) {
		return time.Time{}, false
	}
	answer, idle, err := s.identity.Boundary().Peek(p.session, found.workspace)
	if err != nil {
		return time.Time{}, false
	}
	if fresh, _ := s.freshness(answer, found); !fresh {
		return time.Time{}, false
	}
	decision := s.gate.Decide(gate.Answer(answer), ConsoleClient, gate.Request{Action: gate.AdministerInstallation, Kind: gate.InstallationKind,
		Resource: gate.Resource{WorkspaceID: answer.Workspace()}})
	if !decision.Allowed {
		return time.Time{}, false
	}
	until := p.expires
	for _, end := range []time.Time{answer.NotAfter(), idle} {
		if end.Before(until) {
			until = end
		}
	}
	return until, true
}

func (s *Server) loginOf(user string) string {
	if login, ok := s.dir.loginOf(user); ok {
		return login
	}
	return ""
}

func validServiceName(name string) bool {
	if len(name) < 3 || len(name) > 48 || name[0] == '-' || name[len(name)-1] == '-' {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		case c == '-' && name[i-1] != '-':
		default:
			return false
		}
	}
	return true
}

type serviceCommand struct {
	s    *Server
	kind string
}

func (s *Server) findService(id string) (identity.ServiceInfo, bool, error) {
	if s.faulty("services") {
		return identity.ServiceInfo{}, false, identity.ErrUnavailable
	}
	found, err := s.identity.Workspace().ServiceUser(s.caller, s.bootWorkspace, id)
	if err != nil && !errors.Is(err, identity.ErrUnavailable) {
		return identity.ServiceInfo{}, false, nil
	}
	return found, err == nil, err
}

func (s *Server) findCredential(id string) (bool, error) {
	if s.faulty("credentials") {
		return false, identity.ErrUnavailable
	}
	_, _, err := s.identity.Workspace().Credential(s.caller, s.bootWorkspace, id)
	if err != nil && !errors.Is(err, identity.ErrUnavailable) {
		return false, nil
	}
	return err == nil, err
}

func (h serviceCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	if workspace != s.bootWorkspace {
		return commands.Target{}, commands.NotFound
	}
	if h.kind == "create" {
		if target == servicesTarget {
			return commands.Target{Revision: s.Revision()}, commands.Found
		}
		return commands.Target{}, commands.NotFound
	}
	var found bool
	var err error
	if h.kind == "revoke_credential" || h.kind == "rotate_credential" {
		found, err = s.findCredential(target)
	} else {
		_, found, err = s.findService(target)
	}
	switch {
	case err != nil:
		return commands.Target{}, commands.Unavailable
	case !found:
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

func (h serviceCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
	}
	now := s.clock()
	s.dir.prune(now)
	var state string
	var fields map[string]string
	var err error
	switch h.kind {
	case "create":
		state, fields, err = h.create(ctx, now)
	case "declare":
		state, fields, err = h.declare(ctx, now)
	case "revoke_credential":
		state, err = h.revoke(ctx)
	case "rotate_credential":
		state, fields, err = h.rotate(ctx, now)
	default:
		var svc identity.ServiceInfo
		svc, _, err = s.findService(ctx.Target)
		if err == nil {
			state, fields, err = h.step(ctx, svc, now)
		}
	}
	if err != nil {
		var failure *commands.Failure
		if errors.As(err, &failure) || errors.Is(err, errPartial) {
			return commands.Outcome{}, err
		}
		return commands.Outcome{}, mapIdentity(err)
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: state, Fields: fields}, nil
}

func (h serviceCommand) create(ctx commands.Context, now time.Time) (string, map[string]string, error) {
	s := h.s
	name := ctx.Fields["name"]
	if !validServiceName(name) {
		return "", nil, commands.Fail(commands.InvalidRequest, "name")
	}
	owner := ctx.Fields["owner"]
	eligible, err := s.eligibleServiceOwner(s.caller, owner)
	if err != nil {
		return "", nil, err
	}
	if !eligible {
		return "", nil, commands.Fail(commands.InvalidRequest, "owner")
	}
	if s.faulty("services") {
		return "", nil, identity.ErrUnavailable
	}
	count, err := s.identity.Workspace().ServiceCount(s.caller, s.bootWorkspace)
	if err != nil {
		return "", nil, err
	}
	if count >= maxServices {
		return "", nil, commands.Fail(commands.PrerequisiteFailed)
	}
	id := servicePrefix + name
	if _, exists, err := s.findService(id); err != nil {
		return "", nil, err
	} else if exists {
		return "", nil, commands.Fail(commands.InvalidRequest, "name")
	}
	prepared, err := s.prepareServiceDeclaration(id, ctx.Fields, 0, now)
	if err != nil {
		return "", nil, err
	}
	if err := s.identity.Workspace().CreateServiceUser(s.caller, s.bootWorkspace, id, owner); err != nil {
		if errors.Is(err, identity.ErrConflict) {
			err = commands.Fail(commands.InvalidRequest, "name")
		}
		return "", nil, err
	}
	s.serviceDeclarations[prepared.key] = prepared.history
	return "service_created", map[string]string{"declaration_version": "1"}, nil
}

func (h serviceCommand) step(ctx commands.Context, svc identity.ServiceInfo, now time.Time) (string, map[string]string, error) {
	s := h.s
	ws := s.identity.Workspace()
	switch h.kind {
	case "suspend":
		if err := ws.SuspendServiceUser(s.caller, s.bootWorkspace, svc.ID, svc.Epoch); err != nil {
			return "", nil, err
		}
		s.dir.dropPickups(func(p pickup) bool { return p.target == svc.ID })
		return "service_suspended", nil, nil
	case "reinstate":
		if err := s.requireServiceDeclaration(svc, now); err != nil {
			return "", nil, err
		}
		return "service_active", nil, ws.ReinstateServiceUser(s.caller, s.bootWorkspace, svc.ID, svc.Epoch)
	case "retire":
		if err := ws.RetireServiceUser(s.caller, s.bootWorkspace, svc.ID, svc.Epoch); err != nil {
			return "", nil, err
		}
		s.dir.dropPickups(func(p pickup) bool { return p.target == svc.ID })
		creds, err := s.credentials(s.caller, svc.ID)
		for _, c := range creds {
			if err == nil && credentialState(c, now) == "active" {
				if err = ws.RevokeCredential(s.caller, s.bootWorkspace, c.ID); err == nil {
					s.dir.dropDelivery(c.ID)
				}
			}
		}
		if err != nil || s.faulty("retire") {
			return "", nil, errPartial
		}
		return "service_retired", nil, nil
	}
	return h.issue(ctx, svc, now)
}

func (h serviceCommand) issue(ctx commands.Context, svc identity.ServiceInfo, now time.Time) (string, map[string]string, error) {
	s := h.s
	if svc.State == identity.Suspended {
		return "", nil, commands.Fail(commands.PrerequisiteFailed)
	}
	days, ok := minutes(ctx.Fields["days"], 1, int(s.cfg.Identity.CredentialValidity/(24*time.Hour)))
	if !ok {
		return "", nil, commands.Fail(commands.InvalidRequest, "days")
	}
	if err := s.requireServiceDeclaration(svc, now); err != nil {
		return "", nil, err
	}
	creds, err := s.credentials(s.caller, svc.ID)
	if err != nil {
		return "", nil, err
	}
	active := map[string]bool{}
	for _, c := range creds {
		if credentialState(c, now) == "active" {
			active[c.ID] = true
		}
	}
	s.dir.forget(svc.ID, active)
	live := len(active)
	if live >= s.cfg.Identity.CredentialsPerServiceUser {
		return "", nil, commands.Fail(commands.PrerequisiteFailed)
	}
	secret := random(tokenBytes)
	id := "cred-" + hex.EncodeToString(random(8))
	validity := identity.Validity{NotBefore: now, NotAfter: now.Add(time.Duration(days) * 24 * time.Hour)}
	if err := s.identity.Workspace().AddCredential(s.caller, s.bootWorkspace, svc.ID, id, verifierOf(secret), validity); err != nil {
		return "", nil, err
	}
	ref := encode(random(16))
	s.dir.putPickup(ref, pickup{secret: secretPrefix + encode(secret), credential: id, session: s.caller, target: svc.ID, expires: now.Add(pickupValidity)})
	s.dir.putDelivery(id, delivery{service: svc.ID})
	return "credential_issued", map[string]string{"credential": id, "secret_ref": ref}, nil
}

func (h serviceCommand) rotate(ctx commands.Context, now time.Time) (string, map[string]string, error) {
	s := h.s
	ws := s.identity.Workspace()
	if s.faulty("credentials") {
		return "", nil, identity.ErrUnavailable
	}
	owner, old, err := ws.Credential(s.caller, s.bootWorkspace, ctx.Target)
	if err != nil {
		return "", nil, err
	}
	svc, known, err := s.findService(owner)
	if err != nil {
		return "", nil, err
	}
	if !known || svc.State == identity.Suspended {
		return "", nil, commands.Fail(commands.PrerequisiteFailed)
	}
	days, ok := minutes(ctx.Fields["days"], 1, int(s.cfg.Identity.CredentialValidity/(24*time.Hour)))
	if !ok {
		return "", nil, commands.Fail(commands.InvalidRequest, "days")
	}
	low, overlap, high := s.overlapBounds()
	if raw, given := ctx.Fields["overlap_minutes"]; given {
		if overlap, ok = wholeNumber(raw, low, high); !ok {
			return "", nil, commands.Fail(commands.InvalidRequest, "overlap_minutes")
		}
	}
	if credentialState(old, now) != "active" || now.Before(old.Validity.NotBefore) {
		return "", nil, commands.Fail(commands.InvalidStage)
	}
	if err := s.requireServiceDeclaration(svc, now); err != nil {
		return "", nil, err
	}
	creds, err := s.credentials(s.caller, owner)
	if err != nil {
		return "", nil, err
	}
	active := map[string]bool{}
	for _, c := range creds {
		if credentialState(c, now) == "active" {
			active[c.ID] = true
		}
	}
	s.dir.forget(owner, active)
	if len(active) >= s.cfg.Identity.CredentialsPerServiceUser {
		return "", nil, commands.Fail(commands.PrerequisiteFailed)
	}
	secret := random(tokenBytes)
	id := "cred-" + hex.EncodeToString(random(8))
	validity := identity.Validity{NotBefore: now, NotAfter: now.Add(time.Duration(days) * 24 * time.Hour)}
	end := now.Add(time.Duration(overlap) * time.Minute)
	if err := ws.ReplaceCredential(s.caller, s.bootWorkspace, ctx.Target, id, verifierOf(secret), validity, end); err != nil {
		return "", nil, err
	}
	if old.Validity.NotAfter.Before(end) {
		end = old.Validity.NotAfter
	}
	ref := encode(random(16))
	s.dir.putPickup(ref, pickup{secret: secretPrefix + encode(secret), credential: id, session: s.caller, target: owner, expires: now.Add(pickupValidity)})
	s.dir.putDelivery(id, delivery{service: owner})
	return "credential_rotated", map[string]string{"credential": id, "secret_ref": ref, "replaced": ctx.Target, "replaced_until": stamp(end)}, nil
}

func (h serviceCommand) revoke(ctx commands.Context) (string, error) {
	s := h.s
	if err := s.identity.Workspace().RevokeCredential(s.caller, s.bootWorkspace, ctx.Target); err != nil {
		return "", err
	}
	s.dir.dropPickups(func(p pickup) bool { return p.credential == ctx.Target })
	s.dir.dropDelivery(ctx.Target)
	return "credential_revoked", nil
}

func (s *Server) serviceDefinitions() []commands.Definition {
	permission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	define := func(id, kind string, fields []commands.Field) commands.Definition {
		return commands.Definition{ID: id, Permission: permission, Mutating: true, Rationale: true, Fields: fields, Handler: serviceCommand{s: s, kind: kind}}
	}
	return []commands.Definition{
		define("service.create", "create", append([]commands.Field{{Name: "name", Type: commands.Text, Required: true}, {Name: "owner", Type: commands.ID, Required: true}}, declarationFields()...)),
		define("service.declare", "declare", append(declarationFields(), commands.Field{Name: "declaration_version", Type: commands.Integer, Required: true})),
		define("service.suspend", "suspend", nil),
		define("service.reinstate", "reinstate", nil),
		define("service.retire", "retire", nil),
		define("service.issue_credential", "issue", []commands.Field{{Name: "days", Type: commands.Integer, Required: true}}),
		define("service.revoke_credential", "revoke_credential", nil),
		define("service.rotate_credential", "rotate_credential", []commands.Field{{Name: "days", Type: commands.Integer, Required: true}, {Name: "overlap_minutes", Type: commands.Integer}}),
	}
}
