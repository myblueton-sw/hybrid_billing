package commands

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"regexp"
	"sort"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

var (
	decimalForm = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)
	integerForm = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)
	instantForm = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?(Z|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`)
)

const digestTag = "hybrid-billing/commands/payload/v1"

func failure(code Code, request string, fields ...string) Result {
	rule := errorRules[code]
	return Result{Request: request, Err: &Error{Code: code, Status: rule.status, Retry: rule.retry, Next: rule.next, Request: request, Fields: fields}}
}

func validValue(field Field, value string) bool {
	switch field.Type {
	case ID:
		return validID(value)
	case Decimal:
		return decimalForm.MatchString(value)
	case Integer:
		return integerForm.MatchString(value)
	case Boolean:
		return value == "true" || value == "false"
	case Timestamp:
		_, err := time.Parse(time.RFC3339, value)
		return err == nil && instantForm.MatchString(value)
	case Enum:
		for _, allowed := range field.Values {
			if allowed == value {
				return true
			}
		}
		return false
	}
	return true
}

func (r *Registry) shape(definition Definition, call Call) (map[string]string, []string) {
	bad := map[string]bool{}
	if !validID(call.Request) {
		bad["request_id"] = true
	}
	if call.Client != "" && !validID(call.Client) {
		bad["client"] = true
	}
	if !validID(call.Target) {
		bad["target_id"] = true
	}
	if definition.Mutating != validID(call.Key) || (!definition.Mutating && call.Key != "") {
		bad["idempotency_key"] = true
	}
	if definition.Mutating != validID(call.Expected) || (!definition.Mutating && call.Expected != "") {
		bad["expected_revision"] = true
	}
	if definition.Rationale != (call.Rationale != "") || len(call.Rationale) > r.limits.Rationale {
		bad["rationale"] = true
	}
	declared := map[string]Field{}
	for _, field := range definition.Fields {
		declared[field.Name] = field
	}
	values := map[string]string{}
	excess := len(call.Fields) > len(definition.Fields)
	if excess {
		bad[Marker] = true
	}
	for _, given := range call.Fields {
		if excess {
			break
		}
		field, known := declared[given.Name]
		_, repeated := values[given.Name]
		switch {
		case !validID(given.Name):
			bad[Marker] = true
		case !known || repeated || len(given.Value) > r.limits.Value || !validValue(field, given.Value):
			bad[given.Name] = true
		default:
			values[given.Name] = given.Value
		}
	}
	for _, field := range definition.Fields {
		if _, given := values[field.Name]; field.Required && !given && !excess {
			bad[field.Name] = true
		}
	}
	names := make([]string, 0, len(bad))
	for name := range bad {
		names = append(names, name)
	}
	sort.Strings(names)
	return values, names
}

func payload(call Call, values map[string]string) []byte {
	out := []byte{}
	add := func(part string) {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(part)))
		out = append(out, size[:]...)
		out = append(out, part...)
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	add(digestTag)
	add(call.Command)
	add(call.Target)
	add(call.Expected)
	add(call.Rationale)
	var count [8]byte
	binary.BigEndian.PutUint64(count[:], uint64(len(names)))
	out = append(out, count[:]...)
	for _, name := range names {
		add(name)
		add(values[name])
	}
	return out
}

func (r *Registry) mapped(decision gate.Decision, request string) *Error {
	var code Code
	switch {
	case decision.Allowed:
		return nil
	case decision.Code == gate.AuthorityUnavailable, decision.Reason == gate.ReasonStamp:
		code = AuthorityUnavailable
	case decision.Reason == gate.ReasonIdentity:
		code = Unauthenticated
	case decision.Code == gate.SelfApprovalDenied:
		code = SelfApprovalDenied
	default:
		code = PermissionDenied
	}
	return failure(code, request).Err
}

func resolve(handler Handler, workspace, target string) (found Target, resolution Resolution, ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	found, resolution = handler.Resolve(workspace, target)
	return found, resolution, true
}

func apply(handler Handler, context Context) (outcome Outcome, err error, ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	outcome, err = handler.Apply(context)
	return outcome, err, true
}

func (r *Registry) validOutcome(definition Definition, outcome Outcome) bool {
	if len(outcome.Fields) > r.limits.Fields {
		return false
	}
	for name, value := range outcome.Fields {
		if !validID(name) || len(value) > r.limits.Value {
			return false
		}
	}
	if outcome.Accepted {
		return definition.Mutating && validID(outcome.JobID) && outcome.ResultRevision == "" && outcome.BusinessState == ""
	}
	if outcome.JobID != "" {
		return false
	}
	if !definition.Mutating {
		return outcome.ResultRevision == "" && outcome.BusinessState == ""
	}
	return validID(outcome.ResultRevision) && validID(outcome.BusinessState)
}

func copyFields(fields map[string]string) map[string]string {
	if fields == nil {
		return nil
	}
	copied := make(map[string]string, len(fields))
	for name, value := range fields {
		copied[name] = value
	}
	return copied
}

func replayed(stored Result, request string) Result {
	stored = detached(stored)
	stored.Request, stored.Replay = request, true
	if stored.Err != nil {
		stored.Err.Request = request
	}
	return stored
}

func (r *Registry) Execute(call Call) (result Result) {
	request := marked(call.Request)
	if r == nil || !r.built {
		return failure(AuthorityUnavailable, request)
	}
	switch r.gate.Admit(call.Answer) {
	case gate.Admitted:
	case gate.NotCurrent:
		return failure(Unauthenticated, request)
	default:
		return failure(AuthorityUnavailable, request)
	}
	if !r.reserve() {
		return failure(DependencyUnavailable, request)
	}
	locked := false
	entry := Record{Request: request, Command: marked(call.Command), User: call.Answer.User(), Client: marked(call.Client), Workspace: call.Answer.Workspace(),
		Target: marked(call.Target), Digest: Marker}
	if call.Client == "" {
		entry.Client = ""
	}
	defer func() {
		if recover() != nil {
			result = failure(InternalError, request)
		}
		entry.Outcome, entry.Operation, entry.Replay = string(result.Status), result.Operation, result.Replay
		if result.Err != nil {
			entry.Outcome, entry.Operation = string(result.Err.Code), result.Err.Operation
		}
		r.record(entry)
		if locked {
			r.mutations.Unlock()
		}
	}()
	return r.run(call, request, &entry, &locked)
}

func (r *Registry) run(call Call, request string, entry *Record, locked *bool) Result {
	r.mu.Lock()
	definition, registered := r.definitions[call.Command]
	r.mu.Unlock()
	if !registered {
		return failure(UnsupportedCommand, request)
	}
	if definition.Handler == nil {
		return failure(UnsupportedCapability, request)
	}
	values, bad := r.shape(definition, call)
	if len(bad) > 0 {
		return failure(InvalidRequest, request, bad...)
	}
	body := payload(call, values)
	keyed := hmac.New(sha256.New, r.secret[:])
	keyed.Write(body)
	entry.Digest = hex.EncodeToString(keyed.Sum(nil))
	if definition.Approval && call.Client != "" {
		return failure(PermissionDenied, request)
	}
	if definition.Mutating {
		r.mutations.Lock()
		*locked = true
	}
	workspace, user := call.Answer.Workspace(), call.Answer.User()
	target, resolution, sound := resolve(definition.Handler, workspace, call.Target)
	if !sound {
		return failure(InternalError, request)
	}
	if resolution == Unavailable {
		return failure(DependencyUnavailable, request)
	}
	if resolution != Found {
		target = Target{}
	}
	permitted := gate.Request{Action: definition.Permission.Action, Kind: definition.Permission.Kind, Fields: append([]string(nil), definition.Permission.Fields...),
		Resource: gate.Resource{WorkspaceID: workspace, CustomerID: target.CustomerID, LegalEntityID: target.LegalEntityID, ContractID: target.ContractID,
			Month: target.Month, HasMonth: target.HasMonth}}
	var decision gate.Decision
	if definition.Approval {
		decision = r.gate.CheckApproval(call.Answer, target.AuthorID, permitted)
	} else {
		decision = r.gate.Decide(call.Answer, call.Client, permitted)
	}
	refusal := r.mapped(decision, request)
	if resolution != Found && (refusal == nil || refusal.Code == SelfApprovalDenied) {
		return failure(PermissionDenied, request)
	}
	if refusal != nil {
		return Result{Request: request, Err: refusal}
	}
	if !validID(target.Revision) {
		return failure(InternalError, request)
	}
	digest := sha256.Sum256(body)
	identity := operationKey{installation: call.Answer.Installation(), client: call.Client, command: call.Command, key: call.Key}
	if definition.Mutating {
		r.mu.Lock()
		stored, full := r.operations[identity], len(r.operations) >= r.limits.Operations
		r.mu.Unlock()
		if stored != nil {
			if stored.workspace != workspace || stored.user != user || stored.digest != digest {
				return failure(IdempotencyConflict, request)
			}
			return replayed(stored.result, request)
		}
		if full {
			return failure(DependencyUnavailable, request)
		}
		if call.Expected != target.Revision {
			return failure(RevisionConflict, request)
		}
	}
	if definition.Approval {
		decision = r.gate.RecheckApproval(decision.Stamp, target.AuthorID, permitted)
	} else {
		decision = r.gate.Recheck(decision.Stamp, call.Client, permitted)
	}
	if refusal := r.mapped(decision, request); refusal != nil {
		return Result{Request: request, Err: refusal}
	}
	operationID := ""
	if definition.Mutating {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return failure(DependencyUnavailable, request)
		}
		operationID = "op-" + hex.EncodeToString(random[:])
	}
	at := r.clock()
	outcome, err, sound := apply(definition.Handler, Context{Workspace: workspace, User: user, Client: call.Client, Operation: operationID, Request: request,
		Target: call.Target, Resolved: target.Revision, Expected: call.Expected, Rationale: call.Rationale, Fields: copyFields(values), At: at})
	result, keep := r.finish(definition, call, request, operationID, target, outcome, err, sound, at)
	if keep {
		r.mu.Lock()
		r.operations[identity] = &operation{workspace: workspace, user: user, digest: digest, result: detached(result)}
		r.mu.Unlock()
	}
	return result
}

func detached(result Result) Result {
	result.Fields = copyFields(result.Fields)
	if result.Err != nil {
		failed := *result.Err
		failed.Fields = append([]string(nil), failed.Fields...)
		result.Err = &failed
	}
	return result
}

func (r *Registry) finish(definition Definition, call Call, request, operationID string, target Target, outcome Outcome, err error, sound bool, at time.Time) (Result, bool) {
	empty := !outcome.Accepted && outcome.JobID == "" && outcome.ResultRevision == "" && outcome.BusinessState == "" && len(outcome.Fields) == 0
	if named, isNamed := err.(*Failure); sound && isNamed && named != nil && handlerCodes[named.Code] && empty {
		var fields []string
		if named.Code == InvalidRequest {
			for _, declared := range definition.Fields {
				for _, name := range named.Fields {
					if name == declared.Name {
						fields = append(fields, name)
						break
					}
				}
			}
			sort.Strings(fields)
		}
		failed := failure(named.Code, request, fields...)
		if named.Code == RevisionConflict && !definition.Mutating {
			failed.Err.Retry = true
		}
		return failed, false
	}
	if !sound || err != nil || !r.validOutcome(definition, outcome) {
		failed := failure(InternalError, request)
		if definition.Mutating {
			failed.Err.Operation, failed.Err.Next = operationID, NextOperation
		}
		return failed, definition.Mutating
	}
	result := Result{Request: request, Status: Completed, Operation: operationID, Target: call.Target, Revision: target.Revision, Fields: copyFields(outcome.Fields), AsOf: at}
	if outcome.Accepted {
		result.Status, result.Job, result.Revision = Accepted, outcome.JobID, ""
	} else if definition.Mutating {
		result.Revision, result.State = outcome.ResultRevision, outcome.BusinessState
	}
	return result, definition.Mutating
}
