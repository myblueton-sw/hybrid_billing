package server

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestServiceQueryAggregateBoundaryNeverWrapsOrChangesAfterExhaustion(t *testing.T) {
	for _, tc := range []struct {
		name       string
		credential identity.CredentialInfo
		initial    serviceAggregate
		want       serviceAggregate
	}{
		{"active", identity.CredentialInfo{Validity: identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}}, serviceAggregate{Total: math.MaxUint64 - 1, Active: math.MaxUint64 - 1}, serviceAggregate{Total: math.MaxUint64, Active: math.MaxUint64, Earliest: start.Add(time.Hour), Soon: true}},
		{"expired", identity.CredentialInfo{Validity: identity.Validity{NotBefore: start.Add(-time.Hour), NotAfter: start}}, serviceAggregate{Total: math.MaxUint64 - 1, Expired: math.MaxUint64 - 1}, serviceAggregate{Total: math.MaxUint64, Expired: math.MaxUint64}},
		{"revoked", identity.CredentialInfo{Revoked: true}, serviceAggregate{Total: math.MaxUint64 - 1, Revoked: math.MaxUint64 - 1}, serviceAggregate{Total: math.MaxUint64, Revoked: math.MaxUint64}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			aggregate := tc.initial
			if err := aggregate.add(tc.credential, start, 14); err != nil || aggregate != tc.want {
				t.Fatal("last representable aggregate was not exact")
			}
			before := aggregate
			if err := aggregate.add(tc.credential, start, 14); !errors.Is(err, identity.ErrUnavailable) || aggregate != before {
				t.Fatal("exhausted aggregate wrapped or partially changed")
			}
		})
	}
}

func TestServiceQueryMatchedBoundaryNeverWrapsOrChangesAfterExhaustion(t *testing.T) {
	walk := serviceWalk{cursor: serviceCursor{Matched: math.MaxUint64 - 1}}
	if err := walk.matched(); err != nil || walk.cursor.Matched != math.MaxUint64 {
		t.Fatal("last representable match count was not exact")
	}
	before := walk.cursor
	if err := walk.matched(); !errors.Is(err, identity.ErrUnavailable) || walk.cursor != before {
		t.Fatal("exhausted match count wrapped or partially changed")
	}
}
