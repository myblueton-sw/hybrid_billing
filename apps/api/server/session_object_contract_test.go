package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestSessionObjectEntryRejectsUnknownDuplicateNonobjectAndTrailingInputsWithoutEffects(t *testing.T) {
	f, target, other, caller := queryActors(t)
	handle := handleOfActor(target)
	valid := `{"session":"` + handle + `"}`
	units, counter := f.s.sessionRecordUnits, f.s.counter
	for _, body := range []string{
		`{"session":"` + handle + `","unexpected":null}`,
		`{"session":"` + handle + `","session":"` + handle + `"}`,
		`[` + valid + `]`,
		valid + ` {}`,
	} {
		got := f.selfPost(caller, "sessions/end", body)
		if got.status != 422 || got.body["code"] != string(commands.InvalidRequest) {
			t.Fatal("strict session JSON entry accepted an invalid object", got.status)
		}
		if f.s.sessionRecordUnits != units || len(f.s.sessionOperations) != 0 || f.s.counter != counter || f.me(target).status != 200 || f.me(other).status != 200 || f.me(caller).status != 200 {
			t.Fatal("strict session JSON refusal changed sessions or accounting")
		}
	}
	unknown := f.selfPost(caller, "sessions/selections", `{"unexpected":null}`)
	if unknown.status != 422 || unknown.body["code"] != string(commands.InvalidRequest) {
		t.Fatal("strict selection JSON entry accepted an unknown key", unknown.status)
	}
	if f.s.sessionRecordUnits != units || len(f.s.sessionRecords) != 0 || len(f.s.sessionOperations) != 0 || f.s.counter != counter || f.me(target).status != 200 || f.me(other).status != 200 || f.me(caller).status != 200 {
		t.Fatal("unknown preparation key changed sessions or accounting")
	}
	prepared := f.selfPost(caller, "sessions/selections", valid)
	if prepared.status != 200 || prepared.body["kind"] != kindOne || prepared.body["state"] != "sealed" || countOf(prepared.body["count"]) != 1 {
		t.Fatal("strict object parsing refused a healthy actual single-target preparation")
	}
	assertSessionRecordAccounting(t, f)
}
