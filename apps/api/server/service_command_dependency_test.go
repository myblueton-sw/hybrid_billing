package server

import (
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestServiceCommandExecutionDependenciesPreservePopulatedResources(t *testing.T) {
	for _, kind := range []string{"declare", "create"} {
		t.Run(kind, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(AdminLogin, pw)
			caller := f.signedIn(AdminLogin, pw)
			service := f.newService(caller, "command-dependency")
			credential, _, _ := f.issued(caller, service, "1")
			d := detailBoundaryFixture{f: f, caller: caller, service: service, credential: credential}
			before := d.state(caller)
			verifier := caller.verifier(f)
			ctx := commands.Context{Target: service, Fields: f.declaredFields(map[string]string{"declaration_version": "1"})}
			if kind == "create" {
				ctx.Target = servicesTarget
				ctx.Fields = f.declaredFields(map[string]string{"name": "refused-dependency", "owner": SampleAdmin})
			}
			observations := 0
			outcome, err := func() (commands.Outcome, error) {
				f.s.domain.Lock()
				previousCaller, previousFault := f.s.caller, f.s.fault
				defer func() {
					f.s.caller, f.s.fault = previousCaller, previousFault
					f.s.domain.Unlock()
				}()
				f.s.caller = verifier
				f.s.fault = func(at string) bool {
					if at != "services" {
						return false
					}
					observations++
					return kind == "declare" || observations == 2
				}
				return (serviceCommand{s: f.s, kind: kind}).Apply(ctx)
			}()
			assertServiceReferenceFailure(t, err, commands.DependencyUnavailable, "")
			want := 1
			if kind == "create" {
				want = 2
			}
			if observations != want || !reflect.DeepEqual(outcome, commands.Outcome{}) {
				t.Fatal("execution dependency refusal observation or empty outcome contract")
			}
			if !reflect.DeepEqual(before, d.state(caller)) {
				t.Fatal("execution dependency refusal changed populated service resources")
			}
			if got := f.service(caller, service); got["id"] != service || f.me(caller).status != 200 {
				t.Fatal("execution dependency refusal lost canonical service or caller recovery")
			}
		})
	}
}

func TestServiceCreateStagedPublicInsertionPreservesCompetitorWithoutOuterEffects(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(AdminLogin, pw)
	caller := f.signedIn(AdminLogin, pw)
	service := f.newService(caller, "existing-competitor")
	credential, _, _ := f.issued(caller, service, "1")
	d := detailBoundaryFixture{f: f, caller: caller, service: service, credential: credential}
	before := d.state(caller)
	verifier := caller.verifier(f)
	id := "svc-staged-competitor"
	insertions := 0
	var competitor identity.ServiceInfo
	outcome, err := func() (commands.Outcome, error) {
		f.s.domain.Lock()
		previousCaller, previousFault := f.s.caller, f.s.fault
		defer func() {
			f.s.caller, f.s.fault = previousCaller, previousFault
			f.s.domain.Unlock()
		}()
		f.s.caller = verifier
		f.s.fault = func(at string) bool {
			if at == "service-declaration" {
				insertions++
				if insertions != 1 {
					t.Fatal("staged public insertion ran more than once")
				}
				f.must(f.s.identity.Workspace().CreateServiceUser(verifier, SampleWorkspace, id, SampleAdmin))
				var err error
				competitor, err = f.s.identity.Workspace().ServiceUser(verifier, SampleWorkspace, id)
				f.must(err)
			}
			return false
		}
		return (serviceCommand{s: f.s, kind: "create"}).Apply(commands.Context{Target: servicesTarget, Fields: f.declaredFields(map[string]string{"name": "staged-competitor", "owner": SampleAdmin})})
	}()
	assertServiceReferenceFailure(t, err, commands.InvalidRequest, "name")
	if insertions != 1 || !reflect.DeepEqual(outcome, commands.Outcome{}) || competitor.ID != id || competitor.Owner != SampleAdmin || competitor.Creator != SampleAdmin || competitor.Epoch == 0 || competitor.State != identity.Active {
		t.Fatal("staged competitor or empty outer outcome premise")
	}
	afterCompetitor, err := f.s.identity.Workspace().ServiceUser(verifier, SampleWorkspace, id)
	f.must(err)
	credentials, err := f.s.identity.Workspace().Credentials(verifier, SampleWorkspace, id)
	f.must(err)
	if !reflect.DeepEqual(competitor, afterCompetitor) || len(credentials) != 0 {
		t.Fatal("refused outer create changed competitor identity or created credentials")
	}
	if _, known := f.s.serviceDeclaration(SampleWorkspace, id, 0); known {
		t.Fatal("refused outer create published a competitor declaration")
	}
	after := d.state(caller)
	if after.count != before.count+1 {
		t.Fatal("staged public insertion did not increase service count exactly once")
	}
	after.count = before.count
	if !reflect.DeepEqual(before, after) {
		t.Fatal("refused outer create changed unrelated populated service resources")
	}
	if got := f.service(caller, service); got["id"] != service || f.me(caller).status != 200 {
		t.Fatal("staged conflict lost existing canonical service or caller recovery")
	}
}
