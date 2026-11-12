package identity

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"math"
	"time"
)

var (
	ErrInvalid           = errors.New("request is incomplete or inconsistent")
	ErrRefused           = errors.New("request is refused")
	ErrConflict          = errors.New("record conflicts with the stored state")
	ErrLastAdministrator = errors.New("workspace would be left without an administrator")
	ErrLimit             = errors.New("a configured limit is reached")
	ErrUnavailable       = errors.New("identity registry is unavailable")
)

type Reason string

const (
	ReasonNone             Reason = ""
	ReasonSession          Reason = "session_not_current"
	ReasonNotAdministrator Reason = "actor_not_administrator"
	ReasonUnknown          Reason = "record_unknown_here"
	ReasonState            Reason = "record_state"
	ReasonVerifier         Reason = "verifier_mismatch"
	ReasonExpired          Reason = "outside_validity"
	ReasonUser             Reason = "user_not_eligible"
	ReasonNotInvitee       Reason = "not_the_invited_user"
	ReasonInviter          Reason = "inviter_authority_changed"
	ReasonMember           Reason = "already_a_member"
	ReasonMembership       Reason = "no_active_membership"
	ReasonIdentity         Reason = "identity_not_linked"
	ReasonLastIdentity     Reason = "last_identity"
	ReasonLaterIdentity    Reason = "session_of_a_later_identity"
	ReasonBootstrap        Reason = "bootstrap_state"
	ReasonReauthentication Reason = "session_too_old"
	ReasonAnswer           Reason = "answer_not_current"
	ReasonOperatorBlock    Reason = "operator_block"
	ReasonOwner            Reason = "owner_not_eligible"
)

type refusal struct{ reason Reason }

func (refusal) Error() string { return ErrRefused.Error() }

func (refusal) Is(target error) bool { return target == ErrRefused }

func refused(reason Reason) error { return refusal{reason: reason} }

func ReasonOf(err error) Reason {
	var found refusal
	if errors.As(err, &found) {
		return found.reason
	}
	return ReasonNone
}

type Verifier [32]byte

var emptyDigest = Verifier(sha256.Sum256(nil))

func (v Verifier) valid() bool { return v != Verifier{} && v != emptyDigest }

func (v Verifier) matches(other Verifier) bool {
	return subtle.ConstantTimeCompare(v[:], other[:]) == 1
}

type Kind string

const (
	Human   Kind = "human"
	Service Kind = "service"
)

type UserState string

const (
	Active    UserState = "active"
	Suspended UserState = "suspended"
	Departed  UserState = "departed"
)

type Role string

const (
	Member        Role = "member"
	Administrator Role = "administrator"
	ServiceRole   Role = "service"
)

type MembershipState string

const (
	MembershipActive    MembershipState = "active"
	MembershipSuspended MembershipState = "suspended"
	MembershipRemoved   MembershipState = "removed"
)

const (
	maxIDBytes    = 128
	maxNameBytes  = 512
	maxEmailBytes = 320
	maxTextBytes  = 256
)

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

type Triple struct {
	Connection string
	Issuer     string
	Subject    string
}

func (t Triple) valid() bool {
	for _, part := range []string{t.Connection, t.Issuer, t.Subject} {
		if part == "" || len(part) > maxNameBytes {
			return false
		}
	}
	return true
}

type Validity struct {
	NotBefore time.Time
	NotAfter  time.Time
}

func (v Validity) valid(longest time.Duration) bool {
	return !v.NotBefore.IsZero() && v.NotBefore.Before(v.NotAfter) && v.NotAfter.Sub(v.NotBefore) <= longest
}

func (v Validity) at(now time.Time) bool { return !now.Before(v.NotBefore) && now.Before(v.NotAfter) }

type Operator struct {
	Reference string
	Reason    string
}

func (o Operator) valid() bool {
	return o.Reference != "" && o.Reason != "" && len(o.Reference) <= maxTextBytes && len(o.Reason) <= maxTextBytes
}

type Limits struct {
	BootstrapValidity         time.Duration
	InvitationValidity        time.Duration
	CredentialValidity        time.Duration
	SessionIdle               time.Duration
	SessionAbsolute           time.Duration
	Reauthentication          time.Duration
	Attempts                  int
	PendingInvitations        int
	SessionsPerUser           int
	CredentialsPerServiceUser int
	PageSize                  int
}

func (l Limits) valid() bool {
	for _, length := range []time.Duration{l.BootstrapValidity, l.InvitationValidity, l.CredentialValidity, l.SessionIdle, l.SessionAbsolute, l.Reauthentication} {
		if length <= 0 {
			return false
		}
	}
	for _, count := range []int{l.Attempts, l.PendingInvitations, l.SessionsPerUser, l.CredentialsPerServiceUser, l.PageSize} {
		if count < 1 {
			return false
		}
	}
	return true
}

func bump(counter *uint64) bool {
	if *counter == math.MaxUint64 {
		return false
	}
	*counter++
	return true
}

type Answer struct {
	issued       bool
	instance     [16]byte
	installation string
	workspace    string
	user         string
	kind         Kind
	epoch        uint64
	role         Role
	notBefore    time.Time
	notAfter     time.Time
	version      uint64
	serial       uint64
	identity     Triple
	credential   string
}

func (a Answer) Installation() string      { return a.installation }
func (a Answer) Workspace() string         { return a.workspace }
func (a Answer) User() string              { return a.user }
func (a Answer) Kind() Kind                { return a.kind }
func (a Answer) Epoch() uint64             { return a.epoch }
func (a Answer) Role() Role                { return a.role }
func (a Answer) NotAfter() time.Time       { return a.notAfter }
func (a Answer) Connection() string        { return a.identity.Connection }
func (a Answer) MembershipVersion() uint64 { return a.version }

type Resolved struct {
	User  string
	Email string
}

type UserInfo struct {
	ID        string
	Kind      Kind
	State     UserState
	Epoch     uint64
	Workspace string
}

type MemberInfo struct {
	User      string
	Role      Role
	State     MembershipState
	Version   uint64
	UserState UserState
}

type ServiceInfo struct {
	ID      string
	State   UserState
	Epoch   uint64
	Creator string
	Owner   string
}
