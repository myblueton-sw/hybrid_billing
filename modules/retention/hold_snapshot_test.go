package retention

import "testing"

func TestHoldsCoveringReturnsDetachedScopeSnapshots(t *testing.T) {
	f := setup(t)
	f.put("artifact", ClassIdentity, 1, []string{"t1"})
	for _, h := range []Hold{
		{ID: "z", Owner: "legal", Reason: "case", Tenants: []string{"t1"}, Classes: []string{ClassIdentity}, Artifacts: []string{"artifact"}},
		{ID: "a", Owner: "legal", Reason: "case", Tenants: []string{"t1"}},
		{ID: "other", Owner: "legal", Reason: "case", Tenants: []string{"t1"}, Classes: []string{ClassSource}},
	} {
		_, err := f.r.PlaceHold(h, cat)
		f.must(err)
	}
	rows := f.r.HoldsCovering(ClassIdentity)
	if len(rows) != 2 || rows[0].Hold.ID != "a" || rows[1].Hold.ID != "z" {
		t.Fatalf("class filter or order: %+v", rows)
	}
	rows[1].Hold.Tenants[0] = "other-tenant"
	rows[1].Hold.Classes[0] = ClassSource
	rows[1].Hold.Artifacts[0] = "other-artifact"
	hold, err := f.r.Hold("z")
	f.must(err)
	if !hold.Hold.Covers("t1", ClassIdentity) || hold.Hold.Covers("other-tenant", ClassIdentity) || hold.Hold.Artifacts[0] != "artifact" {
		t.Fatalf("snapshot mutation changed legal hold: %+v", hold)
	}
}
