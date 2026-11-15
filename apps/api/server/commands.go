package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type callBody struct {
	RequestID string
	TargetID  string
	Key       string
	Expected  string
	Rationale string
	Fields    []commands.Value
}

var errShape = errors.New("body shape")

func stringValue(d *json.Decoder) (string, error) {
	t, err := d.Token()
	if err != nil {
		return "", err
	}
	v, ok := t.(string)
	if !ok {
		return "", errShape
	}
	return v, nil
}

func expectDelim(d *json.Decoder, want json.Delim) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	if t != want {
		return errShape
	}
	return nil
}

func decodeFields(d *json.Decoder) ([]commands.Value, error) {
	if err := expectDelim(d, '{'); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := []commands.Value{}
	for d.More() {
		name, err := stringValue(d)
		if err != nil {
			return nil, err
		}
		if seen[name] {
			return nil, errShape
		}
		seen[name] = true
		value, err := stringValue(d)
		if err != nil {
			return nil, err
		}
		out = append(out, commands.Value{Name: name, Value: value})
	}
	if err := expectDelim(d, '}'); err != nil {
		return nil, err
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Name < out[b].Name })
	return out, nil
}

func decodeCall(r io.Reader) (callBody, error) {
	d := json.NewDecoder(r)
	var c callBody
	if err := expectDelim(d, '{'); err != nil {
		return c, err
	}
	seen := map[string]bool{}
	for d.More() {
		name, err := stringValue(d)
		if err != nil {
			return c, err
		}
		if seen[name] {
			return c, errShape
		}
		seen[name] = true
		switch name {
		case "body":
			if c.Fields, err = decodeFields(d); err != nil {
				return c, err
			}
			continue
		case "request_id", "target_id", "idempotency_key", "expected_revision", "rationale":
		default:
			return c, errShape
		}
		value, err := stringValue(d)
		if err != nil {
			return c, err
		}
		switch name {
		case "request_id":
			c.RequestID = value
		case "target_id":
			c.TargetID = value
		case "idempotency_key":
			c.Key = value
		case "expected_revision":
			c.Expected = value
		default:
			c.Rationale = value
		}
	}
	if err := expectDelim(d, '}'); err != nil {
		return c, err
	}
	if _, err := d.Token(); err != io.EOF {
		if tooLarge(err) {
			return c, err
		}
		return c, errShape
	}
	if !seen["body"] {
		c.Fields = []commands.Value{}
	}
	return c, nil
}

func (s *Server) access(found session) (identity.Answer, commands.Code) {
	answer, err := s.identity.Boundary().Access(found.verifier, found.workspace)
	switch {
	case err == nil:
		return answer, ""
	case errors.Is(err, identity.ErrUnavailable):
		return answer, commands.AuthorityUnavailable
	case errors.Is(err, identity.ErrInvalid):
		return answer, commands.InternalError
	}
	s.sessions.drop(found.verifier)
	return answer, commands.Unauthenticated
}

func changed(mutating bool, result commands.Result) bool {
	if !mutating || result.Replay {
		return false
	}
	if result.Err == nil {
		return true
	}
	return result.Err.Operation != ""
}

func (s *Server) command(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	found, ok := s.lookup(r)
	if !ok {
		failCode(w, commands.Unauthenticated, request)
		return
	}
	if !sameToken(r.Header.Get(csrfHeader), found.csrf) {
		fail(w, http.StatusForbidden, CodeCSRF, request)
		return
	}
	body, err := decodeCall(r.Body)
	if err != nil {
		if tooLarge(err) {
			fail(w, http.StatusRequestEntityTooLarge, CodeBodyTooLarge, request)
			return
		}
		failCode(w, commands.InvalidRequest, request)
		return
	}
	answer, code := s.access(found)
	if code != "" {
		failCode(w, code, request)
		return
	}
	command := r.PathValue("command")
	call := commands.Call{Answer: gate.Answer(answer), Client: ConsoleClient, Request: body.RequestID, Command: command, Target: body.TargetID,
		Key: body.Key, Expected: body.Expected, Rationale: body.Rationale, Fields: body.Fields}
	writeResult(w, s.execute(command, call))
}

func (s *Server) execute(command string, call commands.Call) commands.Result {
	s.domain.Lock()
	defer s.domain.Unlock()
	result := s.commands.Execute(call)
	if changed(s.mutating[command], result) {
		s.counter++
	}
	return result
}
