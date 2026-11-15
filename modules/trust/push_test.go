package trust

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func pushes(t *testing.T) (world, Admission) {
	f := setup(t)
	a := f.connect("w1", "c1", print("a"))
	f.key("w1", "c1", "v1", "one", start.Add(-time.Hour), start.Add(24*time.Hour))
	return f, a
}

func TestMessageEncoding(t *testing.T) {
	p := Push{EventIDs: []string{"e1", "e2"}, KeyVersion: "v1", Signed: time.Unix(0, 1700000000000000000), Body: []byte("body")}
	message := Message("w1", "c1", p)
	digest := sha256.Sum256([]byte("body"))
	want := "000000000000000d" + hex.EncodeToString([]byte(pushTag)) + "0000000000000002" + hex.EncodeToString([]byte("w1")) + "0000000000000002" +
		hex.EncodeToString([]byte("c1")) + "0000000000000002" + hex.EncodeToString([]byte("v1")) + "0000000000000001" + hex.EncodeToString([]byte("2")) +
		"0000000000000002" + hex.EncodeToString([]byte("e1")) + "0000000000000002" + hex.EncodeToString([]byte("e2")) + "0000000000000013" +
		hex.EncodeToString([]byte("1700000000000000000")) + "0000000000000020" + hex.EncodeToString(digest[:])
	if hex.EncodeToString(message) != want {
		t.Errorf("message %x", message)
	}
}

func TestKeys(t *testing.T) {
	f := setup(t)
	f.connect("w1", "c1", print("a"))
	good := Key{Version: "v1", Public: public("one"), NotBefore: start, NotAfter: start.Add(time.Hour)}
	for name, k := range map[string]Key{
		"no version":          {Public: good.Public, NotBefore: start, NotAfter: start.Add(time.Hour)},
		"no start":            {Version: "v1", Public: good.Public, NotAfter: start.Add(time.Hour)},
		"an empty validity":   {Version: "v1", Public: good.Public, NotBefore: start, NotAfter: start},
		"a reversed validity": {Version: "v1", Public: good.Public, NotBefore: start, NotAfter: start.Add(-time.Second)},
	} {
		expect(t, name, second(f.r.AddKey("w1", "c1", k, ann)), ErrInvalid)
	}
	expect(t, "no connection", second(f.r.AddKey("w1", "", good, ann)), ErrInvalid)
	reason(t, "an assistant adding", second(f.r.AddKey("w1", "c1", good, bot)), ReasonAINotAllowed)
	expect(t, "an unknown connection", second(f.r.AddKey("w1", "c9", good, ann)), ErrNotFound)
	weakKeys := append([][]byte{nil, make([]byte, 31), make([]byte, 33)}, smallOrder...)
	for _, text := range []string{"edffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f", "eeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "edffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"} {
		decoded, _ := hex.DecodeString(text)
		weakKeys = append(weakKeys, decoded)
	}
	for index, public := range weakKeys {
		reason(t, "weak key "+hex.EncodeToString(public), second(f.r.AddKey("w1", "c1", Key{Version: "weak" + string(rune('a'+index)), Public: public, NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)), ReasonWeakKey)
	}
	below, _ := hex.DecodeString("ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7e")
	if weak(below) {
		t.Errorf("a canonical encoding below the prime minus one is weak")
	}
	identity := smallOrder[0]
	forged := append(append([]byte(nil), identity...), make([]byte, 32)...)
	if !ed25519.Verify(identity, []byte("any message"), forged) {
		t.Errorf("the identity key no longer accepts the forged signature this rule guards against")
	}
	reason(t, "a validity above the key life", second(f.r.AddKey("w1", "c1", Key{Version: "v1", Public: good.Public, NotBefore: start, NotAfter: start.Add(limits.KeyLife + time.Nanosecond)}, ann)), ReasonKeyLife)
	f.must(second(f.r.AddKey("w1", "c1", Key{Version: "long", Public: public("long"), NotBefore: start, NotAfter: start.Add(limits.KeyLife)}, ann)))
	zone := time.FixedZone("x", 3600)
	view, err := f.r.AddKey("w1", "c1", Key{Version: "v1", Public: good.Public, NotBefore: start.In(zone), NotAfter: start.Add(time.Hour).In(zone)}, ann)
	if err != nil || view.State != StateProposed || view.Proposer != "ann" || view.Key.NotBefore.Location() != time.UTC {
		t.Fatalf("a key %+v %v", view, err)
	}
	view.Key.Public[0] ^= 1
	if again, err := f.r.AddKey("w1", "c1", good, ann); err != nil || again.Key.Public[0] != good.Public[0] {
		t.Errorf("the repeated key %+v %v", again, err)
	}
	expect(t, "another proposer", second(f.r.AddKey("w1", "c1", good, bob)), ErrConflict)
	expect(t, "another key under the version", second(f.r.AddKey("w1", "c1", Key{Version: "v1", Public: public("two"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)), ErrConflict)
	expect(t, "another validity", second(f.r.AddKey("w1", "c1", Key{Version: "v1", Public: good.Public, NotBefore: start, NotAfter: start.Add(2 * time.Hour)}, ann)), ErrConflict)
	expect(t, "another start", second(f.r.AddKey("w1", "c1", Key{Version: "v1", Public: good.Public, NotBefore: start.Add(time.Second), NotAfter: start.Add(time.Hour)}, ann)), ErrConflict)
	expect(t, "a third key", second(f.r.AddKey("w1", "c1", Key{Version: "v3", Public: public("three"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)), ErrLimit)
	for _, call := range []func() error{
		func() error { return second(f.r.ApproveKey("w1", "c1", "", bob)) }, func() error { return second(f.r.RetireKey("", "c1", "v1", bob)) },
	} {
		expect(t, "an invalid call", call(), ErrInvalid)
	}
	reason(t, "an assistant approving", second(f.r.ApproveKey("w1", "c1", "v1", bot)), ReasonAINotAllowed)
	expect(t, "an unknown key", second(f.r.ApproveKey("w1", "c1", "v9", bob)), ErrNotFound)
	expect(t, "a key of an unknown connection", second(f.r.ApproveKey("w1", "c9", "v1", bob)), ErrNotFound)
	reason(t, "the proposer approving", second(f.r.ApproveKey("w1", "c1", "v1", ann)), ReasonSelfApproval)
	if active, err := f.r.ApproveKey("w1", "c1", "v1", bob); err != nil || active.State != StateActive || active.Approver != "bob" {
		t.Errorf("approval %+v %v", active, err)
	}
	f.must(second(f.r.ApproveKey("w1", "c1", "v1", bob)))
	reason(t, "another approver", second(f.r.ApproveKey("w1", "c1", "v1", cat)), ReasonNotPending)
	if retired, err := f.r.RetireKey("w1", "c1", "long", cat); err != nil || retired.State != StateRetired {
		t.Errorf("retirement %+v %v", retired, err)
	}
	reason(t, "approving a retired key", second(f.r.ApproveKey("w1", "c1", "long", bob)), ReasonNotPending)
	f.must(second(f.r.AddKey("w1", "c1", Key{Version: "v3", Public: public("three"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)))
	f.must(second(f.r.ProposeConnection("w1", Connection{ID: "c2", Fingerprints: []Fingerprint{print("b")}, Endpoints: []string{"usage"}}, ann)))
	reason(t, "a key of a proposed connection", second(f.r.AddKey("w1", "c2", good, ann)), ReasonNotActive)
}

func TestVerify(t *testing.T) {
	f, a := pushes(t)
	p := sign(a, "one", "v1", start, "body", "e1")
	f.must(f.r.Verify(a, p))
	reason(t, "the same delivery again", f.r.Verify(a, p), ReasonAlreadyAccepted)
	reason(t, "the same identifier with another body", f.r.Verify(a, sign(a, "one", "v1", start, "other", "e1")), ReasonDuplicate)
	tampered := p
	tampered.Body = []byte("bodx")
	reason(t, "a tampered body", f.r.Verify(a, tampered), ReasonForged)
	moved := sign(a, "one", "v1", start, "body", "e9")
	moved.EventIDs = []string{"e8"}
	reason(t, "a changed identifier", f.r.Verify(a, moved), ReasonForged)
	reason(t, "a signature of another key", f.r.Verify(a, sign(a, "two", "v1", start, "body", "e2")), ReasonForged)
	reason(t, "an unknown version", f.r.Verify(a, sign(a, "one", "v9", start, "body", "e2")), ReasonUnknownKey)
	other := Admission{Workspace: "w1", Connection: "c1", Revision: 2}
	reason(t, "an admission of another revision", f.r.Verify(other, sign(other, "one", "v1", start, "body", "e2")), ReasonConnectionChanged)
	f.must(f.r.Verify(a, sign(a, "one", "v1", start.Add(-limits.Skew), "body", "e3")))
	f.must(f.r.Verify(a, sign(a, "one", "v1", start.Add(limits.Skew), "body", "e4")))
	reason(t, "a push signed too early", f.r.Verify(a, sign(a, "one", "v1", start.Add(-limits.Skew-time.Nanosecond), "body", "e5")), ReasonStale)
	reason(t, "a push signed in the future", f.r.Verify(a, sign(a, "one", "v1", start.Add(limits.Skew+time.Nanosecond), "body", "e5")), ReasonStale)
	entries, dropped, err := f.r.Quarantine("w1", 0, 10, ann)
	if err != nil || len(entries) != 3 || dropped != 0 {
		t.Fatalf("quarantine %+v %d %v", entries, dropped, err)
	}
	digest := sha256.Sum256([]byte("other"))
	first := Entry{Sequence: 1, Connection: "c1", EventIDs: []string{"e1"}, KeyVersion: "v1", Reason: ReasonDuplicate, Signed: start, Received: start, BodyDigest: digest}
	if entries[0].Sequence != first.Sequence || entries[0].Reason != first.Reason || entries[0].BodyDigest != first.BodyDigest || entries[0].EventIDs[0] != "e1" ||
		!entries[0].Signed.Equal(start) || !entries[0].Received.Equal(start) || entries[0].KeyVersion != "v1" || entries[0].Connection != "c1" || entries[2].Sequence != 3 {
		t.Errorf("entry %+v", entries[0])
	}
	entries[0].EventIDs[0] = "changed"
	if again, _, _ := f.r.Quarantine("w1", 0, 1, ann); len(again) != 1 || again[0].EventIDs[0] != "e1" {
		t.Errorf("the quarantine is shared with the caller")
	}
	if page, _, _ := f.r.Quarantine("w1", 2, 5, ann); len(page) != 1 || page[0].Sequence != 3 {
		t.Errorf("a later page %+v", page)
	}
	counts, err := f.r.Counts("w1", ann)
	if err != nil || counts[ReasonAlreadyAccepted] != 1 || counts[ReasonForged] != 3 || counts[ReasonUnknownKey] != 1 || counts[ReasonConnectionChanged] != 1 || counts[ReasonDuplicate] != 0 {
		t.Errorf("counts %+v %v", counts, err)
	}
	counts[ReasonForged] = 0
	if again, _ := f.r.Counts("w1", ann); again[ReasonForged] != 3 {
		t.Errorf("the counts are shared with the caller")
	}
}

func TestVerifyForms(t *testing.T) {
	f, a := pushes(t)
	good := sign(a, "one", "v1", start, "body", "e1")
	change := func(edit func(*Push)) Push {
		p := good
		edit(&p)
		return p
	}
	for name, p := range map[string]Push{
		"no identifier":         change(func(p *Push) { p.EventIDs = nil }),
		"too many identifiers":  change(func(p *Push) { p.EventIDs = []string{"a", "b", "c", "d"} }),
		"an identifier as text": change(func(p *Push) { p.EventIDs = []string{"a b"} }),
		"a repeated identifier": change(func(p *Push) { p.EventIDs = []string{"a", "a"} }),
		"no version":            change(func(p *Push) { p.KeyVersion = "" }),
		"no signing time":       change(func(p *Push) { p.Signed = time.Time{} }),
		"a long body":           change(func(p *Push) { p.Body = make([]byte, limits.Body+1) }),
		"a short signature":     change(func(p *Push) { p.Signature = p.Signature[1:] }),
	} {
		expect(t, name, f.r.Verify(a, p), ErrInvalid)
	}
	for name, admission := range map[string]Admission{"no workspace": {Connection: "c1", Revision: 1}, "no connection": {Workspace: "w1", Revision: 1},
		"no revision": {Workspace: "w1", Connection: "c1"}} {
		expect(t, name, f.r.Verify(admission, good), ErrInvalid)
	}
	expect(t, "an unknown workspace", f.r.Verify(Admission{Workspace: "w9", Connection: "c1", Revision: 1}, good), ErrNotFound)
	expect(t, "an unknown connection", f.r.Verify(Admission{Workspace: "w1", Connection: "c9", Revision: 1}, good), ErrNotFound)
	f.must(f.r.Verify(a, sign(a, "one", "v1", start, strings.Repeat("x", limits.Body), "a", "b", "c")))
	if entries, dropped, err := f.r.Quarantine("", 0, 1, ann); entries != nil || dropped != 0 || err != ErrInvalid {
		t.Errorf("an invalid quarantine read %v %d %v", entries, dropped, err)
	}
	if entries, dropped, err := f.r.Quarantine("w1", 0, 1, bot); entries != nil || dropped != 0 || err == nil {
		t.Errorf("an assistant quarantine read %v %d %v", entries, dropped, err)
	}
	for name, call := range map[string]func() error{
		"quarantine of no workspace": func() error { return third(f.r.Quarantine("", 0, 1, ann)) },
		"a negative page":            func() error { return third(f.r.Quarantine("w1", -1, 1, ann)) },
		"an empty page":              func() error { return third(f.r.Quarantine("w1", 0, 0, ann)) },
		"counts of no workspace":     func() error { return second(f.r.Counts("", ann)) },
	} {
		expect(t, name, call(), ErrInvalid)
	}
	reason(t, "an assistant reading the quarantine", third(f.r.Quarantine("w1", 0, 1, bot)), ReasonAINotAllowed)
	reason(t, "an assistant reading the counts", second(f.r.Counts("w1", bot)), ReasonAINotAllowed)
	if entries, dropped, err := f.r.Quarantine("w9", 0, 1, ann); entries != nil || dropped != 0 || err != nil {
		t.Errorf("an unknown workspace %v %d %v", entries, dropped, err)
	}
	if counts, err := f.r.Counts("w9", ann); len(counts) != 0 || err != nil {
		t.Errorf("counts of an unknown workspace %v %v", counts, err)
	}
}

func TestKeyValidity(t *testing.T) {
	f := setup(t)
	a := f.connect("w1", "c1", print("a"))
	f.key("w1", "c1", "old", "old", start.Add(-time.Hour), start)
	f.key("w1", "c1", "new", "new", start, start.Add(time.Hour))
	f.must(f.r.Verify(a, sign(a, "old", "old", start, "body", "e1")))
	f.must(f.r.Verify(a, sign(a, "new", "new", start, "body", "e2")))
	reason(t, "an old key after its end", f.r.Verify(a, sign(a, "old", "old", start.Add(time.Nanosecond), "body", "e3")), ReasonKeyExpired)
	reason(t, "a new key before its start", f.r.Verify(a, sign(a, "new", "new", start.Add(-time.Nanosecond), "body", "e3")), ReasonKeyExpired)
	expect(t, "a third open key", second(f.r.AddKey("w1", "c1", Key{Version: "next", Public: public("next"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)), ErrLimit)
	f.must(second(f.r.RetireKey("w1", "c1", "old", ann)))
	f.must(second(f.r.AddKey("w1", "c1", Key{Version: "next", Public: public("next"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)))
	reason(t, "a proposed key", f.r.Verify(a, sign(a, "next", "next", start, "body", "e4")), ReasonUnknownKey)
	reason(t, "a retired key", f.r.Verify(a, sign(a, "old", "old", start, "body", "e5")), ReasonUnknownKey)
	if entries, _, _ := f.r.Quarantine("w1", 0, 10, ann); len(entries) != 2 {
		t.Errorf("a retired key reached the quarantine %+v", entries)
	}
	for _, version := range []string{"r1", "r2"} {
		f.must(second(f.r.RetireKey("w1", "c1", "new", ann)))
		f.key("w1", "c1", version, version, start, start.Add(time.Hour))
		f.must(second(f.r.RetireKey("w1", "c1", version, ann)))
		if again, err := f.r.RetireKey("w1", "c1", version, ann); err != nil || again.State != StateRetired {
			t.Errorf("retiring again %+v %v", again, err)
		}
	}
	keys := f.r.workspaces["w1"].connections["c1"].keys
	if keys["old"] != nil || keys["new"] != nil || keys["r1"] == nil || keys["r2"] == nil || keys["next"] == nil || len(keys) != 3 {
		t.Errorf("retired keys beyond the bound were not pruned oldest first: %d", len(keys))
	}
	expect(t, "a pruned key", second(f.r.ApproveKey("w1", "c1", "old", bob)), ErrNotFound)
	f.must(second(f.r.RevokeConnection("w1", "c1", ann)))
	reason(t, "a revoked connection", f.r.Verify(a, sign(a, "new", "new", start, "body", "e6")), ReasonConnectionChanged)
}

func TestReplayWindow(t *testing.T) {
	f, a := pushes(t)
	p := sign(a, "one", "v1", start, "body", "e1", "e2")
	f.must(f.r.Verify(a, p))
	reason(t, "a batch overlapping an accepted one", f.r.Verify(a, sign(a, "one", "v1", start, "body", "e2", "e3")), ReasonDuplicate)
	f.must(f.r.Verify(a, sign(a, "one", "v1", start, "body", "e3")))
	f.advance(limits.Skew)
	reason(t, "a replay at the end of the window", f.r.Verify(a, p), ReasonAlreadyAccepted)
	f.advance(time.Nanosecond)
	reason(t, "a replay after the window", f.r.Verify(a, p), ReasonStale)
	f.must(f.r.Verify(a, sign(a, "one", "v1", *f.now, "fresh", "e1")))
	f.must(f.r.Verify(a, sign(a, "one", "v1", *f.now, "fresh", "e2")))
	if len(f.r.workspaces["w1"].connections["c1"].seen) != 3 {
		t.Errorf("expired identifiers were not reused")
	}
}

func TestReplayCapacity(t *testing.T) {
	f, a := pushes(t)
	b := f.connect("w2", "c1", print("b"))
	f.key("w2", "c1", "v1", "one", start.Add(-time.Hour), start.Add(24*time.Hour))
	f.must(f.r.Verify(a, sign(a, "one", "v1", start, "x", "e1", "e2", "e3")))
	f.must(f.r.Verify(a, sign(a, "one", "v1", start.Add(time.Second), "x", "e4")))
	reason(t, "a full store", f.r.Verify(a, sign(a, "one", "v1", start, "x", "e5")), ReasonReplayCapacity)
	f.must(f.r.Verify(b, sign(b, "one", "v1", start, "x", "e1", "e2", "e3")))
	f.advance(limits.Skew + time.Nanosecond)
	f.must(f.r.Verify(a, sign(a, "one", "v1", *f.now, "x", "e6", "e7", "e8")))
	reason(t, "a store full of live identifiers", f.r.Verify(a, sign(a, "one", "v1", *f.now, "x", "f1")), ReasonReplayCapacity)
	entries, _, _ := f.r.Quarantine("w1", 0, 10, ann)
	if len(entries) != 2 || entries[0].Reason != ReasonReplayCapacity {
		t.Errorf("quarantine %+v", entries)
	}
}

func TestQuarantineFlood(t *testing.T) {
	f, a := pushes(t)
	for index := 0; index < 20; index++ {
		forged := sign(a, "one", "v1", start, "body", "e1")
		forged.Signature[0] ^= byte(index + 1)
		reason(t, "a forged push", f.r.Verify(a, forged), ReasonForged)
	}
	for index, id := range []string{"s1", "s2", "s3", "s4", "s5"} {
		reason(t, "a stale push", f.r.Verify(a, sign(a, "one", "v1", start.Add(-time.Duration(index+2)*limits.Skew), "body", id)), ReasonStale)
	}
	entries, dropped, _ := f.r.Quarantine("w1", 0, 10, ann)
	if len(entries) != limits.Quarantine || dropped != 2 || entries[0].Reason != ReasonStale || entries[0].EventIDs[0] != "s1" {
		t.Errorf("quarantine %+v dropped %d", entries, dropped)
	}
	if counts, _ := f.r.Counts("w1", ann); counts[ReasonForged] != 20 {
		t.Errorf("counts %+v", counts)
	}
}
