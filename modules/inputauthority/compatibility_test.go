package inputauthority

import "testing"

func TestLegacyAuthorityIdentity(t *testing.T) {
	for _, capability := range []string{Confirm, Inspect} {
		w := setup(t)
		p := w.proposal(Console)
		p.Spec.Capability = capability
		r, err := w.registry.Propose(p)
		if err != nil {
			t.Fatal(err)
		}
		got, err := w.registry.Approve(intent(r))
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{Confirm: "542fb224998e845c17e8bfd524d4a05cb6076409f4b85946581bd4cc91327ea3", Inspect: "55e99936bb4b93cb84830ef4e729c38bb0d1a1e9540402922831ee3a08e458c2"}[capability]
		if r.Digest != want || got.Decision.GrantID != "input-grant-"+want {
			t.Fatal("legacy identity changed", capability)
		}
	}
}
