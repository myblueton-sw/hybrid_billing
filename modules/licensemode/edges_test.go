package licensemode

import (
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func TestTextAndIDEdges(t *testing.T) {
	for _, id := range []string{"A", "Z", "a", "z", "0", "9", "a.b_c:d-e", strings.Repeat("a", 128)} {
		if !validID(id) {
			t.Errorf("id %q refused", id)
		}
	}
	for _, id := range []string{"", "!a", "a b", "a@", "a[", "a`", "a{", "a/", strings.Repeat("a", 129)} {
		if validID(id) {
			t.Errorf("id %q accepted", id)
		}
	}
	for _, text := range []string{"Acme Ltd", strings.Repeat("a", 200), " x", "~"} {
		if !validText(text) {
			t.Errorf("text %q refused", text)
		}
	}
	for _, text := range []string{"", strings.Repeat("a", 201), "a\x1f", "a\x7f", "a\u0080", "a\u009f", "a\xff", strings.Repeat("a", 201) + "\xff"} {
		if validText(text) {
			t.Errorf("text %q accepted", text)
		}
	}
	if hexLower("AF") != "af" || hexLower("@G") != "@G" {
		t.Error("hexLower")
	}
	for _, c := range []string{"0123456789abcdef"} {
		if !validHexBytes(c) {
			t.Error("hex")
		}
	}
	for _, c := range []string{"0A", "0F", "0g", "0"} {
		if validHexBytes(c) {
			t.Errorf("hex %q accepted", c)
		}
	}
	for _, text := range []string{"a1", "1a", "01", ""} {
		if digitsOnly(text, 12) {
			t.Errorf("digits %q", text)
		}
	}
}

func TestTimeAndBodyEdges(t *testing.T) {
	c := commercial("e1", "agr-1", start, start.Add(time.Hour))
	c.Start, c.End, c.IssuedAt = 1, 253402300799, 1
	if _, ok := Decode(Encode(c)); !ok {
		t.Error("time edges refused")
	}
	c.End = 253402300800
	if _, ok := Decode(Encode(c)); ok {
		t.Error("time past maximum")
	}
	var e encoder
	for _, v := range []string{claimsHeader, DocCommercial, "e1", "owner", "k1", "1000000000000"} {
		e.text(v)
	}
	if _, ok := Decode(e.out.Bytes()); ok {
		t.Error("13-digit time")
	}
	body := Encode(evaluation("g1"))
	exact := make([]byte, len(body))
	copy(exact, body)
	if _, ok := Decode(exact); !ok {
		t.Error("exact capacity")
	}
	big := commercial("e1", "agr-1", start, start.Add(time.Hour))
	big.Scope = make([]string, 32)
	for i := range big.Scope {
		big.Scope[i] = "s" + itoa(10+i)
	}
	for i := 0; len(Encode(big)) < 4096; i = (i + 1) % 32 {
		if len(big.Scope[i]) < 128 {
			big.Scope[i] += "x"
		}
	}
	if n := len(Encode(big)); n != 4096 {
		t.Fatalf("body %d", n)
	}
	if _, ok := Decode(Encode(big)); !ok {
		t.Error("4096-byte body refused")
	}
	for i := range big.Scope {
		if len(big.Scope[i]) < 128 {
			big.Scope[i] += "x"
			break
		}
	}
	if _, ok := Decode(Encode(big)); ok {
		t.Error("4097-byte body accepted")
	}
	scope33 := commercial("e1", "agr-1", start, start.Add(time.Hour))
	scope33.Scope = nil
	for i := 0; i < 33; i++ {
		scope33.Scope = append(scope33.Scope, "s"+itoa(10+i))
	}
	if _, ok := Decode(Encode(scope33)); ok {
		t.Error("33 scope entries accepted")
	}
}

func TestKeyWindowEdges(t *testing.T) {
	f := setup(t)
	k := keySpec(f, "k1", "owner")
	k.AcceptUntil = k.NotAfter
	f.approve(Change{Kind: ChangeKey, Key: k})
	f.must(f.r.Install("i1", "hybrid-billing", ann))
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	upper := keySpec(f, "k2", "owner")
	for _, from := range []string{"a", "f"} {
		x := upper
		i := strings.Index(x.Public, from)
		if i < 0 {
			continue
		}
		x.Public = x.Public[:i] + strings.ToUpper(from) + x.Public[i+1:]
		expect(t, "upper "+from, second(f.r.Propose(Change{Kind: ChangeKey, Key: x}, ann)), ErrInvalid)
	}
	at := commercial("e1", "agr-1", start, start.Add(time.Hour))
	at.IssuedAt = k.NotBefore
	if out, _ := f.r.Register("i1", f.sign(at), ann); out != OutcomeAccepted {
		t.Errorf("issued at not before %s", out)
	}
	expect(t, "retire form", second(f.r.Propose(Change{Kind: ChangeRetire, Retire: Retire{Key: "k1", At: 0}}, ann)), ErrInvalid)
	f.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: "k1", At: start.Unix()}})
	g := setup(t)
	gk := keySpec(g, "k1", "owner")
	g.approve(Change{Kind: ChangeKey, Key: gk})
	g.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: "k1", At: gk.NotAfter}})
	h := setup(t)
	hk := keySpec(h, "k1", "owner")
	hk.NotBefore = start.Unix()
	h.approve(Change{Kind: ChangeKey, Key: hk})
	h.must(h.r.Install("i1", "hybrid-billing", ann))
	h.must(h.r.SelectMode("i1", ModeCommercial, ann))
	now := commercial("e1", "agr-1", start, start.Add(time.Hour))
	now.IssuedAt = start.Unix()
	if out, _ := h.r.Register("i1", h.sign(now), ann); out != OutcomeAccepted {
		t.Errorf("now at not before %s", out)
	}
	m := setup(t)
	mk := keySpec(m, "k1", "owner")
	mk.NotAfter, mk.AcceptUntil = start.Add(time.Hour).Unix(), start.Add(2*time.Hour).Unix()
	m.approve(Change{Kind: ChangeKey, Key: mk})
	m.must(m.r.Install("i1", "hybrid-billing", ann))
	m.must(m.r.SelectMode("i1", ModeCommercial, ann))
	m.advance(2 * time.Hour)
	late := commercial("e1", "agr-1", start, start.Add(10*time.Hour))
	late.IssuedAt = start.Add(time.Hour).Unix() - 1
	if out, _ := m.r.Register("i1", m.sign(late), ann); out != OutcomeUnverifiable {
		t.Errorf("at accept until %s", out)
	}
	n := setup(t)
	nk := keySpec(n, "k1", "owner")
	nk.NotAfter = start.Add(time.Hour).Unix()
	n.approve(Change{Kind: ChangeKey, Key: nk})
	n.must(n.r.Install("i1", "hybrid-billing", ann))
	n.must(n.r.SelectMode("i1", ModeCommercial, ann))
	n.advance(2 * time.Hour)
	edge := commercial("e1", "agr-1", start, start.Add(10*time.Hour))
	edge.IssuedAt = nk.NotAfter
	if out, _ := n.r.Register("i1", n.sign(edge), ann); out != OutcomeUnverifiable {
		t.Errorf("issued at not after %s", out)
	}
	p := setup(t)
	p.approve(Change{Kind: ChangeKey, Key: keySpec(p, "k1", "owner")})
	p.approve(Change{Kind: ChangeKey, Key: keySpec(p, "k2", "owner")})
	p.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: "k1", At: start.Add(time.Hour).Unix()}})
	p.approve(Change{Kind: ChangeCompromise, Compromise: "k1"})
	if p.r.active != 1 {
		t.Errorf("active after compromising a retired key %d", p.r.active)
	}
	_ = hex.EncodeToString
}

func TestLatchAndStatusEdges(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	reg(t, f, commercial("e1", "agr-1", start.Add(-time.Hour), start.Add(time.Hour)))
	if v := status(t, f, "i1"); v.LastOutcome != OutcomeAccepted {
		t.Errorf("one submission %+v", v)
	}
	f.advance(time.Hour)
	if v := status(t, f, "i1"); v.State != StateCommercialExpired {
		t.Errorf("at end %+v", v)
	}
	f.approve(Change{Kind: ChangeReset, Reset: Reset{At: start.Unix(), Reason: "test"}})
	f.auth.set(start.Add(30*time.Minute), true)
	if v := status(t, f, "i1"); v.State != StateCommercialExpired {
		t.Errorf("latched at exact end %+v", v)
	}
	g := standard(t)
	g.must(g.r.SelectMode("i1", ModeCommercial, ann))
	p := commercial("p", "agr-p", start, start)
	p.End, p.Perpetual = 0, true
	reg(t, g, p)
	if v := status(t, g, "i1"); v.State != StateCommercialActive {
		t.Errorf("perpetual %+v", v)
	}
	h := standard(t)
	h.must(h.r.SelectMode("i1", ModeCommercial, ann))
	cur := commercial("c1", "agr-c", start.Add(-time.Hour), start.Add(10*time.Hour))
	cur.IssuedAt = start.Add(-time.Minute).Unix()
	reg(t, h, cur)
	tie := commercial("c2", "agr-d", start, start.Add(10*time.Hour))
	tie.SubjectID, tie.IssuedAt = "other", cur.IssuedAt
	if out := reg(t, h, tie); out != OutcomeSuperseded {
		t.Errorf("equal issued at %s", out)
	}
	before := commercial("c3", "agr-e", start.Add(-3*time.Hour), start.Add(-time.Hour))
	before.IssuedAt = start.Add(-time.Second).Unix()
	if out := reg(t, h, before); out != OutcomeExpired {
		t.Errorf("before %s", out)
	}
	k := standard(t)
	k.approve(Change{Kind: ChangeKey, Key: keySpec(k, "leak", "owner")})
	k.must(k.r.SelectMode("i1", ModeCommercial, ann))
	reg(t, k, commercial("old", "agr-o", start.Add(-time.Hour), start.Add(time.Minute)))
	k.advance(time.Minute)
	bad := commercial("bad", "agr-b", start.Add(time.Minute), start.Add(time.Hour))
	bad.KeyID, bad.IssuedAt = "leak", start.Add(time.Minute).Unix()
	reg(t, k, bad)
	k.approve(Change{Kind: ChangeCompromise, Compromise: "leak"})
	if v := status(t, k, "i1"); v.State != StateUnverified {
		t.Errorf("compromised starting now %+v", v)
	}
	k.advance(59 * time.Minute)
	if v := status(t, k, "i1"); v.State != StateCommercialExpired {
		t.Errorf("compromised at its end %+v", v)
	}
	m := standard(t)
	m.must(m.r.SelectMode("i1", ModeCommercial, ann))
	reg(t, m, commercial("x1", "agr-x", start.Add(time.Hour), start.Add(2*time.Hour)))
	early := commercial("x0", "agr-y", start.Add(-time.Hour), start.Add(time.Hour))
	early.IssuedAt = start.Unix()
	m.advance(0)
	if out := reg(t, m, early); out != OutcomeAccepted && out != OutcomeNotYetValid {
		t.Errorf("adjacent before %s", out)
	}
}

func TestLiteralCaps(t *testing.T) {
	f := setup(t)
	for len(f.r.keys) < 99 {
		f.r.keys["z"+itoa(len(f.r.keys))] = &key{}
		f.r.active++
	}
	f.approve(Change{Kind: ChangeKey, Key: keySpec(f, "k-fit", "owner")})
	expect(t, "101st key", f.try(Change{Kind: ChangeKey, Key: keySpec(f, "k-over", "owner")}), ErrLimit)
	g := setup(t)
	for len(g.r.installations) < 9999 {
		g.r.installations["x"+itoa(len(g.r.installations))] = &installation{}
	}
	g.must(g.r.Install("fit", "p", ann))
	expect(t, "10001st installation", g.r.Install("over", "p", ann), ErrLimit)
	h := standard(t)
	h.must(h.r.SelectMode("i1", ModeCommercial, ann))
	for len(h.r.documents) < 99999 {
		h.r.documents["x"+itoa(len(h.r.documents))] = &entitlement{}
	}
	if out := reg(t, h, commercial("fit", "agr-fit", start, start.Add(time.Hour))); out != OutcomeAccepted {
		t.Errorf("100000th %s", out)
	}
	over := commercial("over", "agr-over", start, start.Add(time.Hour))
	over.SubjectID, over.IssuedAt = "other", start.Unix()
	_, err := h.r.Register("i1", h.sign(over), ann)
	expect(t, "100001st", err, ErrLimit)
	k := standard(t)
	k.must(k.r.SelectMode("i1", ModeEvaluation, ann))
	a, err := k.r.Activate("i1", k.sign(evaluation("g1")), Acceptance{Actor: "ann", Terms: "eval-terms-v1"}, ann)
	k.must(err)
	k.approve(Change{Kind: ChangeReset, Reset: Reset{At: a.Start.Unix(), Reason: "exact"}})
	m := standard(t)
	m.must(m.r.SelectMode("i1", ModeCommercial, ann))
	reg(t, m, commercial("later", "agr-l", start.Add(time.Hour), start.Add(48*time.Hour)))
	before := commercial("before", "agr-b", start.Add(-time.Hour), start.Add(time.Hour))
	before.IssuedAt = start.Unix()
	if out := reg(t, m, before); out != OutcomeAccepted {
		t.Errorf("new ending at existing start %s", out)
	}
}
