package customeraccess

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
)

var (
	ErrDenied        = errors.New("access denied")
	ErrUnavailable   = errors.New("query unavailable")
	ErrInvalidPolicy = errors.New("invalid authority configuration")
	ErrConflict      = errors.New("authority version conflict")
)

type Action string

const (
	UsageRead   Action = "usage.read"
	UsageExport Action = "usage.export"
)

type Scope struct {
	CompanyID string
	TenantID  string
}

type Company struct {
	ID     string
	Active bool
}

type Tenant struct {
	ID        string
	CompanyID string
	Active    bool
}

type Grant struct {
	Scope     Scope
	Action    Action
	Fields    []string
	ExpiresAt time.Time
}

type Subject struct {
	ID     string
	Active bool
	Epoch  uint64
	Grants []Grant
}

type Principal struct {
	Installation string
	SubjectID    string
	Epoch        uint64
	ValidUntil   time.Time
}

type Selection struct {
	Scopes []Scope
	Fields []string
	Limit  int
}

type Row struct {
	Scope  Scope
	ID     string
	Values map[string]string
}

type Limits struct {
	MaxRows   int
	MaxScopes int
	MaxFields int
}

type VerifyFunc func(context.Context) (Principal, error)
type ReadFunc func(context.Context, Selection) ([]Row, error)

func validID(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value || strings.Contains(value, "*") {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) || r == unicode.ReplacementChar {
			return false
		}
	}
	return true
}

func copySelection(in Selection) Selection {
	return Selection{Scopes: append([]Scope(nil), in.Scopes...), Fields: append([]string(nil), in.Fields...), Limit: in.Limit}
}
