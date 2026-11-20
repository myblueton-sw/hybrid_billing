package server

import (
	"encoding/base64"
	"errors"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/licensemode"
)

type licenseHandler struct {
	s    *Server
	kind string
}

func (h licenseHandler) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || target != licenseTarget || h.s.cfg.License.Installation == "" {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}
func decodeLicense(fields map[string]string) (licensemode.Signed, error) {
	body, tail, signature := fields["body"], fields["body_tail"], fields["signature"]
	if len(body) > 4096 || len(tail) > 1368 || tail != "" && len(body) != 4096 || len(signature) != 88 {
		return licensemode.Signed{}, commands.Fail(commands.InvalidRequest, "body", "body_tail", "signature")
	}
	encoded := body + tail
	bytes, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(bytes) == 0 || len(bytes) > 4096 || base64.StdEncoding.EncodeToString(bytes) != encoded {
		return licensemode.Signed{}, commands.Fail(commands.InvalidRequest, "body", "body_tail")
	}
	sig, err := base64.StdEncoding.Strict().DecodeString(signature)
	if err != nil || len(sig) != 64 || base64.StdEncoding.EncodeToString(sig) != signature {
		return licensemode.Signed{}, commands.Fail(commands.InvalidRequest, "signature")
	}
	return licensemode.Signed{Body: bytes, Signature: sig}, nil
}
func (h licenseHandler) Apply(ctx commands.Context) (commands.Outcome, error) {
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
	actor := licensemode.Actor{ID: ctx.User, Kind: licensemode.KindHuman}
	var err error
	state := ""
	switch h.kind {
	case "initialize":
		if s.license != nil {
			state = "already_initialized"
			break
		}
		s.license, err = licensemode.CreateDurable(s.licenseDir(), s.cfg.License.Installation, licenseProduct, actor, s.licenseConfig(), s.clock)
		if err != nil {
			s.licenseFailed = true
		}
		state = "initialized"
	default:
		if s.license == nil {
			return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
		}
		switch h.kind {
		case "select_mode":
			err = s.license.SelectMode(ctx.Fields["mode"], actor)
			state = "mode_selected"
		case "register", "activate":
			signed, decodeErr := decodeLicense(ctx.Fields)
			if decodeErr != nil {
				return commands.Outcome{}, decodeErr
			}
			if h.kind == "register" {
				state, err = s.license.Register(signed, actor)
			} else {
				_, err = s.license.Activate(signed, licensemode.Acceptance{Actor: ctx.User, Terms: ctx.Fields["terms"]}, actor)
				state = "activation_recorded"
			}
		}
	}
	if err != nil {
		var refusal licensemode.Refusal
		switch {
		case errors.Is(err, licensemode.ErrStorage), errors.Is(err, licensemode.ErrOwned), errors.Is(err, licensemode.ErrCheckpoint):
			return commands.Outcome{}, err
		case errors.As(err, &refusal):
			state = refusal.Reason
		case errors.Is(err, licensemode.ErrInvalid):
			return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
		case errors.Is(err, licensemode.ErrLimit):
			return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
		case errors.Is(err, licensemode.ErrConflict), errors.Is(err, licensemode.ErrNotFound):
			return commands.Outcome{}, commands.Fail(commands.InvalidStage)
		default:
			return commands.Outcome{}, err
		}
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: state, Fields: map[string]string{"outcome": state}}, nil
}
func (s *Server) licenseDefinitions() []commands.Definition {
	define := func(kind string, fields ...commands.Field) commands.Definition {
		return commands.Definition{ID: "licenses." + kind, Permission: commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}, Mutating: true, Rationale: true, Fields: fields, Handler: licenseHandler{s, kind}}
	}
	signed := []commands.Field{{Name: "body", Type: commands.Text, Required: true}, {Name: "body_tail", Type: commands.Text}, {Name: "signature", Type: commands.Text, Required: true}}
	return []commands.Definition{
		define("initialize"),
		define("select_mode", commands.Field{Name: "mode", Type: commands.Enum, Required: true, Values: []string{licensemode.ModeNoncommercial, licensemode.ModeEvaluation, licensemode.ModeCommercial}}),
		define("register", signed...),
		define("activate", append(append([]commands.Field{}, signed...), commands.Field{Name: "terms", Type: commands.Text, Required: true})...),
	}
}
