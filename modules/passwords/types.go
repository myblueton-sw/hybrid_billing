package passwords

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("request is incomplete or inconsistent")
	ErrLimit    = errors.New("a configured limit is reached")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
)

const (
	ReasonLinkInvalid  = "link_invalid"
	ReasonPasswordForm = "password_form"
	ReasonSameAsLogin  = "same_as_login"

	PurposeSet   = "set"
	PurposeReset = "reset"

	ResultOK     = "ok"
	ResultWrong  = "wrong"
	ResultLocked = "locked"
)

type Refusal struct {
	Reason string
}

func (r Refusal) Error() string { return ErrRejected.Error() + ": " + r.Reason }

func (r Refusal) Unwrap() error { return ErrRejected }

type Key struct {
	Issuer  string
	Subject string
}

type Params struct {
	Iterations  int
	SaltBytes   int
	KeyBytes    int
	MinLength   int
	MaxLength   int
	Attempts    int
	Lockout     time.Duration
	MinValidity time.Duration
	MaxValidity time.Duration
	Records     int
	Links       int
}

func DefaultParams() Params {
	return Params{Iterations: 600000, SaltBytes: 16, KeyBytes: 32, MinLength: 12, MaxLength: 256, Attempts: 5, Lockout: 15 * time.Minute,
		MinValidity: time.Hour, MaxValidity: 7 * 24 * time.Hour, Records: 100000, Links: 10000}
}

type Record struct {
	Iterations int
	Salt       []byte
	Hash       []byte
}

type Link struct {
	Key     Key
	Purpose string
	Expires time.Time
	Hold    time.Time
}
