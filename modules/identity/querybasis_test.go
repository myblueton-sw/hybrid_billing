package identity

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestMemberPositionsBoundExaminedGapsWithoutChangingLegacyMembers(t *testing.T) {
	f := setup(t)
	for _, id := range []string{"ann", "bob", "cat", "dan"} {
		f.member(id, Member)
	}
	admin := f.login("root")
	f.must(f.w.RemoveMembership(admin, "w1", "bob", 1))
	f.must(f.w.RemoveMembership(admin, "w1", "cat", 1))
	before, err := f.w.ServiceQueryBasis(admin, "w1")
	f.must(err)
	after := ""
	ids := []string{}
	scanned := 0
	for _, want := range []struct {
		after, id string
		scanned   int
		more      bool
	}{{"bob", "ann", 2, true}, {"dan", "dan", 2, true}, {"root", "root", 1, false}} {
		page, err := f.w.MemberPositions(admin, "w1", after, 2)
		f.must(err)
		if page.After != want.after || page.Scanned != want.scanned || page.More != want.more || len(page.Rows) != 1 || page.Rows[0].User.ID != want.id || page.Rows[0].Member == nil {
			t.Fatal("member scan filled past removed positions", page, want)
		}
		ids = append(ids, page.Rows[0].User.ID)
		scanned += page.Scanned
		after = page.After
	}
	if scanned != 5 || !reflect.DeepEqual(ids, []string{"ann", "dan", "root"}) {
		t.Fatal("bounded member traversal lost records", ids, scanned)
	}
	legacy, err := f.w.Members(admin, "w1", "", 2)
	f.must(err)
	if len(legacy) != 2 || legacy[0].User != "ann" || legacy[1].User != "dan" {
		t.Fatal("legacy filled-page contract changed")
	}
	current, err := f.w.ServiceQueryBasis(admin, "w1")
	f.must(err)
	if before != current {
		t.Fatal("query read changed its source")
	}
	page, err := f.w.MemberPositions(admin, "w1", "root", 2)
	f.must(err)
	if page.Scanned != 0 || len(page.Rows) != 0 || page.More {
		t.Fatal("past-end member scan", page)
	}
}

func TestServiceQueryBasisCredentialsAreScopedAndOriginalGenerationIsPreserved(t *testing.T) {
	f := setup(t)
	f.seed("w2", "adm2")
	admin, other := f.login("root"), f.login("adm2")
	f.must(f.w.CreateServiceUser(admin, "w1", "svc-one", "root"))
	f.must(f.w.CreateServiceUser(other, "w2", "svc-two", "adm2"))
	original := f.r.Subjects().Generation()
	first, err := f.w.ServiceQueryBasis(admin, "w1")
	f.must(err)
	foreign, err := f.w.ServiceQueryBasis(other, "w2")
	f.must(err)
	f.must(f.w.AddCredential(admin, "w1", "svc-one", "cred", v("cred"), Validity{start, start.Add(time.Hour)}))
	added, err := f.w.ServiceQueryBasis(admin, "w1")
	f.must(err)
	if added.Subjects != first.Subjects || added.Credentials == first.Credentials {
		t.Fatal("credential insertion omitted its source fence")
	}
	f.must(f.w.RevokeCredential(admin, "w1", "cred"))
	revoked, err := f.w.ServiceQueryBasis(admin, "w1")
	f.must(err)
	if revoked.Credentials == added.Credentials || revoked.Subjects != added.Subjects {
		t.Fatal("revocation without pruning omitted its fence")
	}
	if err := f.w.RevokeCredential(admin, "w1", "cred"); err == nil {
		t.Fatal("failed-write premise")
	}
	failed, err := f.w.ServiceQueryBasis(admin, "w1")
	f.must(err)
	foreignAfter, err := f.w.ServiceQueryBasis(other, "w2")
	f.must(err)
	if failed != revoked || foreignAfter != foreign || f.r.Subjects().Generation() != original {
		t.Fatal("credential query bookkeeping changed unrelated/public generation")
	}
	f.r.mu.Lock()
	moved := *f.r.credentials["cred"]
	moved.user = "svc-two"
	f.r.putCredential("cred", &moved)
	f.r.mu.Unlock()
	old, err := f.w.ServiceQueryBasis(admin, "w1")
	f.must(err)
	next, err := f.w.ServiceQueryBasis(other, "w2")
	f.must(err)
	if old.Credentials == revoked.Credentials || next.Credentials == foreign.Credentials {
		t.Fatal("replacement omitted old/new workspace fences")
	}
	rows, err := f.w.CredentialPage(admin, "w1", "svc-one", "", 2)
	f.must(err)
	movedRows, err := f.w.CredentialPage(other, "w2", "svc-two", "", 2)
	f.must(err)
	if len(rows) != 0 || len(movedRows) != 1 || movedRows[0].ID != "cred" {
		t.Fatal("replacement left stale credential membership")
	}
}

func TestServiceQueryBasisTracksHelpersAndSubjectCounterWrapWithoutChangingWrites(t *testing.T) {
	f := setup(t)
	admin := f.login("root")
	before, err := f.w.ServiceQueryBasis(admin, "w1")
	f.must(err)
	original := f.r.Subjects().Generation()
	f.r.mu.Lock()
	f.r.putUser("root", f.r.users["root"])
	f.r.putMembership("w1", "root", f.r.memberships["w1"]["root"])
	f.r.mu.Unlock()
	helpers, err := f.w.ServiceQueryBasis(admin, "w1")
	f.must(err)
	if helpers.Subjects == before.Subjects || f.r.Subjects().Generation() != original {
		t.Fatal("helper notifications changed original subject-generation semantics")
	}
	f.must(f.w.CreateServiceUser(admin, "w1", "svc", "root"))
	if f.r.Subjects().Generation() != original+1 {
		t.Fatal("service creation changed original generation delta")
	}
	f.r.mu.Lock()
	f.r.generation = math.MaxUint64 - 1
	f.r.querySubjects = math.MaxUint64 - 1
	f.r.mu.Unlock()
	f.must(f.w.SuspendServiceUser(admin, "w1", "svc", 1))
	if f.r.Subjects().Generation() != math.MaxUint64 {
		t.Fatal("original subject generation changed")
	}
	if _, err := f.w.ServiceQueryBasis(admin, "w1"); !errors.Is(err, ErrUnavailable) {
		t.Fatal("exhausted subject query fence reused", err)
	}
	f.must(f.w.ReinstateServiceUser(admin, "w1", "svc", 2))
	if f.r.Subjects().Generation() != 0 {
		t.Fatal("domain generation semantics were changed solely for querying")
	}
	if _, err := f.w.ServiceQueryBasis(admin, "w1"); !errors.Is(err, ErrUnavailable) {
		t.Fatal("original wrap reset sticky query exhaustion", err)
	}
}

func TestServiceQueryBasisAndMemberScanRequireCurrentAdministrator(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	admin, member := f.login("root"), f.login("ann")
	_, err := f.w.ServiceQueryBasis(member, "w1")
	f.refusal("member basis", err, ReasonNotAdministrator)
	_, err = f.w.MemberPositions(member, "w1", "", 1)
	f.refusal("member scan", err, ReasonNotAdministrator)
	for _, args := range []struct {
		workspace, after string
		limit            int
	}{{"", "", 1}, {"w1", "bad id", 1}, {"w1", "", 0}, {"w1", "", 3}} {
		_, err := f.w.MemberPositions(admin, args.workspace, args.after, args.limit)
		f.want("malformed member position", err, ErrInvalid)
	}
	_, err = f.w.ServiceQueryBasis(admin, "")
	f.want("malformed service basis", err, ErrInvalid)
	f.must(f.b.EndSession(admin))
	_, err = f.w.ServiceQueryBasis(admin, "w1")
	f.refusal("ended basis caller", err, ReasonSession)
	_, err = f.w.MemberPositions(admin, "w1", "", 1)
	f.refusal("ended scan caller", err, ReasonSession)
	var absent *Workspace
	_, err = absent.ServiceQueryBasis(admin, "w1")
	f.want("unavailable basis", err, ErrUnavailable)
	_, err = absent.MemberPositions(admin, "w1", "", 1)
	f.want("unavailable member positions", err, ErrUnavailable)
}

func TestServiceQueryCredentialCounterExhaustionRemainsSticky(t *testing.T) {
	f := setup(t)
	admin := f.login("root")
	f.must(f.w.CreateServiceUser(admin, "w1", "svc", "root"))
	f.r.mu.Lock()
	f.r.queryCredentials["w1"] = math.MaxUint64 - 1
	f.r.mu.Unlock()
	f.must(f.w.AddCredential(admin, "w1", "svc", "cred", v("cred"), Validity{start, start.Add(time.Hour)}))
	if _, err := f.w.ServiceQueryBasis(admin, "w1"); !errors.Is(err, ErrUnavailable) {
		t.Fatal("exhausted credential query fence reused", err)
	}
	f.must(f.w.RevokeCredential(admin, "w1", "cred"))
	if _, err := f.w.ServiceQueryBasis(admin, "w1"); !errors.Is(err, ErrUnavailable) {
		t.Fatal("credential query exhaustion was not sticky", err)
	}
}
