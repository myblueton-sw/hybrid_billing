package server

import (
	"errors"
	"strings"
	"testing"
)

var errSessionBodyReader = errors.New("synthetic session body read failure")

type brokenSessionBodyReader struct{}

func (brokenSessionBodyReader) Read([]byte) (int, error) { return 0, errSessionBodyReader }

func TestSessionBodyReadAndTruncationFailuresAreNotObjects(t *testing.T) {
	if _, err := sessionObject(brokenSessionBodyReader{}, "session"); !errors.Is(err, errSessionBodyReader) {
		t.Fatal("body reader failure was lost")
	}
	for _, raw := range []string{`{"`, `{"session"`, `{"session":`, `{"session":"known"`, `{"session":"known"]`, `{"session":"known"} []`} {
		if _, err := sessionObject(strings.NewReader(raw), "session"); err == nil {
			t.Fatal("truncated or trailing body was accepted as one object")
		}
	}
}

func TestSessionEndMalformedVariantsHaveNoOperationOrSessionEffects(t *testing.T) {
	f, old, middle, current := queryActors(t)
	selection := "session-selection-" + strings.Repeat("a", 32)
	digest := strings.Repeat("b", 64)
	validSelection := `"selection":"` + selection + `","digest":"` + digest + `","count":1`
	bodies := []string{
		`{"session":true}`, `{"session":{}}`, `{"generation":1}`, `{"handles":["` + handleOfActor(old) + `"]}`,
		`{"digest":"` + digest + `"}`, `{"count":1}`,
		`{"selection":null,"digest":"` + digest + `","count":1}`,
		`{"selection":"wrong","digest":"` + digest + `","count":1}`,
		`{"selection":"` + selection + `","digest":null,"count":1}`,
		`{"selection":"` + selection + `","digest":"bad","count":1}`,
		`{"selection":"` + selection + `","digest":"` + digest + `","count":0}`,
		`{"selection":"` + selection + `","digest":"` + digest + `","count":100001}`,
		`{"selection":"` + selection + `","digest":"` + digest + `","count":"1"}`,
		`{"selection":"` + selection + `","digest":"` + digest + `","count":1.5}`,
		`{` + validSelection + `,"session":"` + handleOfActor(old) + `"}`,
		`{` + validSelection + `,"session":""}`,
		`{` + validSelection + `,"session":null}`,
		`{"selection":"` + selection + `","digest":"` + digest + `","session":"` + handleOfActor(old) + `"}`,
		`{` + validSelection + `,"generation":"` + strings.Repeat("c", 32) + `"}`,
		`{` + validSelection + `,"generation":null}`,
		`{` + validSelection + `,"handles":null}`,
	}
	before, operations, records, units := f.s.Revision(), len(f.s.selfEnds[SampleMember]), len(f.s.sessionRecords), f.s.sessionRecordUnits
	for i, raw := range bodies {
		if _, _, err := decodeEnd(strings.NewReader(raw)); err == nil {
			t.Fatal("mixed, missing or wrongly typed end body was accepted", i)
		}
		got := f.selfPost(current, "sessions/end", raw)
		if got.status != 422 || got.body["operation"] != nil || got.body["sessions"] != nil || got.body["rows"] != nil {
			t.Fatal("invalid end body reached execution", i, got.status)
		}
		if f.s.Revision() != before || len(f.s.selfEnds[SampleMember]) != operations || len(f.s.sessionRecords) != records || f.s.sessionRecordUnits != units {
			t.Fatal("invalid end body changed operation or accounting state", i)
		}
	}
	if f.me(old).status != 200 || f.me(middle).status != 200 || f.me(current).status != 200 {
		t.Fatal("invalid end body ended a real session")
	}
}
