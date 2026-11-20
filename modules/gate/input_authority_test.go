package gate

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestInputAuthorityUsesActualSessionAndSeparateCeiling(t *testing.T) {
	for _, change := range []string{"ended session", "revoked ceiling", "suspended client", "revoked user"} {
		t.Run(change, func(t *testing.T) {
			f := setup(t)
			request := ask(ConfirmInput)
			f.expect("administrator without grant", f.g.Decide(f.answer("root"), "", request), false, ReasonNoGrant)
			f.member("ann")
			userGrant := f.grant("ann", ConfirmInput)
			session := f.login("ann")
			answer, err := f.b.Access(session, "w1")
			f.must(err)
			f.must(second(f.a.PutClient(Client{ID: "app", State: Active}, 0)))
			f.expect("missing client ceiling", f.g.Decide(answer, "app", request), false, ReasonClient)
			f.must(f.a.AddGrant(Grant{ID: "ceiling", ClientID: "app", Terms: terms(ConfirmInput), Validity: always}))
			first := f.expect("explicit intersection", f.g.Decide(answer, "app", request), true, ReasonAllowed)
			f.expect("unchanged recheck", f.g.Recheck(first.Stamp, "app", request), true, ReasonAllowed)
			switch change {
			case "ended session":
				f.must(f.b.EndSession(session))
			case "revoked ceiling":
				f.must(f.a.RevokeGrant("ceiling", 1))
			case "suspended client":
				f.must(second(f.a.PutClient(Client{ID: "app", State: Suspended}, 1)))
			case "revoked user":
				f.must(f.a.RevokeGrant(userGrant, 1))
			}
			if f.g.Decide(answer, "app", request).Allowed || f.g.Recheck(first.Stamp, "app", request).Allowed {
				t.Fatal("invalid authority remained usable")
			}
		})
	}
}

func TestInputAuthorityRefusesActualServiceCredential(t *testing.T) {
	f := setup(t)
	f.member("ann")
	f.must(f.w.CreateServiceUser(f.login("root"), "w1", "svc", "ann"))
	f.must(f.w.AddCredential(f.login("root"), "w1", "svc", "c1", v("c1"), identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
	answer, err := f.b.AccessByCredential("c1", v("c1"))
	f.must(err)
	for _, client := range []string{"", "app"} {
		f.expect("service cannot confirm", f.g.Decide(answer, client, ask(ConfirmInput)), false, ReasonPrincipal)
	}
}
