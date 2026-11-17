package decisionledger

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func lastLine(t testing.TB, o Options) ([]byte, uint64) {
	t.Helper()
	parts := lines(t, o)
	n := uint64(0)
	var last []byte
	for _, p := range parts {
		if len(p) > 0 {
			last, n = bytes.TrimSuffix(p, []byte("\n")), n+1
		}
	}
	return last, n
}

func nextRecord(t testing.TB, o Options) line {
	t.Helper()
	last, n := lastLine(t, o)
	return line{V: FormatV1, Seq: n + 1, Prev: hashOf(last), At: stamp(start), Kind: KindDecision, Workspace: "w", Decision: RevokeGrant, Target: "g", Revocation: "r", RequestedBy: "u"}
}

func appendRaw(t testing.TB, o Options, data []byte) {
	t.Helper()
	f, _ := os.OpenFile(ledgerPath(o), os.O_WRONLY|os.O_APPEND, 0o600)
	f.Write(data)
	f.Close()
}

func TestEachRecordConditionIsCheckedOnItsOwn(t *testing.T) {
	for name, change := range map[string]func(*line){
		"format":        func(l *line) { l.V = 2 },
		"sequence":      func(l *line) { l.Seq++ },
		"previous hash": func(l *line) { l.Prev = strings.Repeat("1", 64) },
		"genesis at two": func(l *line) {
			*l = line{V: l.V, Seq: l.Seq, Prev: l.Prev, At: l.At, Kind: KindGenesis, LedgerID: strings.Repeat("a", 32), CreatedAt: l.At}
		},
		"time":  func(l *line) { l.At = "2026-11-18T18:00:00+09:00" },
		"shape": func(l *line) { l.Workspace = "" },
	} {
		t.Run(name, func(t *testing.T) {
			o := options(t)
			existing(t, &o)
			rec := nextRecord(t, o)
			change(&rec)
			appendRaw(t, o, append(encode(rec), '\n'))
			l := must(t, o)
			defer l.Close()
			if st := l.Status(); st.State != StateIsolated || st.Isolation != IsolationIntegrity || st.ProblemLine != 2 {
				t.Errorf("%+v", st)
			}
		})
	}
	o := options(t)
	existing(t, &o)
	appendRaw(t, o, append(encode(nextRecord(t, o)), '\n'))
	if l := must(t, o); l.Status().State != StateOK {
		t.Errorf("the unchanged crafted record is valid %+v", l.Status())
	} else {
		l.Close()
	}
}

func TestAnOverlongLineIsNotReadAsARecord(t *testing.T) {
	o := options(t)
	existing(t, &o)
	rec := nextRecord(t, o)
	base := len(encode(rec))
	rec.Target = strings.Repeat("t", LineLimit-base+1)
	data := encode(rec)
	if len(data) != LineLimit {
		t.Fatalf("crafted %d", len(data))
	}
	appendRaw(t, o, append(data, 'x', '\n'))
	l := must(t, o)
	defer l.Close()
	if st := l.Status(); st.State != StateIsolated || st.ProblemLine != 2 || len(l.Decisions()) != 0 {
		t.Errorf("%+v %d", st, len(l.Decisions()))
	}
}

func TestExactlyOneRecordOfRoomIsEnough(t *testing.T) {
	o := options(t)
	l := must(t, o)
	commit(t, l, "1")
	l.Close()
	size := int64(len(bytes.Join(lines(t, o), nil)))
	o.MaxBytes = size + RecordLimit
	at := must(t, o)
	if at.Status().State != StateOK {
		t.Errorf("exactly one record of room starts %s", at.Status().State)
	}
	at.Close()
	appendRaw(t, o, []byte(`{"v":1`))
	o.MaxBytes = size + RecordLimit
	rec := must(t, o)
	defer rec.Close()
	if st := rec.Status(); st.Recovery == nil || st.Records != 3 {
		t.Errorf("a torn tail with exactly one record of room is recovered with its record %+v", st)
	}
}
