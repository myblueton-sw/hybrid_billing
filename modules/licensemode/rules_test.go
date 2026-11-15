package licensemode

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestKeys(t *testing.T) {
	f := setup(t)
	k := keySpec(f, "k1", "owner")
	priv := f.keys["k1"]
	private := k
	private.Public = hex.EncodeToString(priv)
	reason(t, "private", second(f.r.Propose(Change{Kind: ChangeKey, Key: private}, ann)), ReasonPrivateKey)
	seedAsPublic := k
	seedAsPublic.Public = hex.EncodeToString(priv.Seed())
	reason(t, "seed pasted", f.try(Change{Kind: ChangeKey, Key: seedAsPublic}), ReasonNoProof)
	swapped := k
	swapped.Proof = hex.EncodeToString(ed25519.Sign(priv, proofMessage("k", "1owner")))
	reason(t, "proof domain", f.try(Change{Kind: ChangeKey, Key: swapped}), ReasonNoProof)
	forms := map[string]func(*Key){
		"id": func(k *Key) { k.ID = "" }, "issuer": func(k *Key) { k.Issuer = "" }, "short": func(k *Key) { k.Public = k.Public[:62] },
		"upper": func(k *Key) { k.Public = strings.ToUpper(k.Public) }, "hex": func(k *Key) { k.Public = "zz" + k.Public[2:] },
		"proof": func(k *Key) { k.Proof = k.Proof[:126] }, "window": func(k *Key) { k.NotAfter = k.NotBefore },
		"accept": func(k *Key) { k.AcceptUntil = k.NotAfter - 1 }, "zero": func(k *Key) { k.NotBefore = 0 }, "huge": func(k *Key) { k.AcceptUntil = maxTime + 1 },
	}
	for name, mutate := range forms {
		x := k
		mutate(&x)
		expect(t, name, second(f.r.Propose(Change{Kind: ChangeKey, Key: x}, ann)), ErrInvalid)
	}
	f.approve(Change{Kind: ChangeKey, Key: k})
	expect(t, "repeat", f.try(Change{Kind: ChangeKey, Key: k}), ErrConflict)
	if v, err := f.r.Key("k1"); err != nil || v.Key != k || v.Compromised || v.Retired {
		t.Errorf("key read %+v %v", v, err)
	}
	expect(t, "key missing", second(f.r.Key("nope")), ErrNotFound)
	reason(t, "retire past", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: "k1", At: start.Add(-time.Second).Unix()}}), ReasonRetirePast)
	expect(t, "retire after not after", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: "k1", At: k.NotAfter + 1}}), ErrInvalid)
	reason(t, "retire at not before", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: "k1", At: k.NotBefore}}), ReasonRetirePast)
	later := keySpec(f, "k5", "owner")
	later.NotBefore = start.Add(time.Hour).Unix()
	f.approve(Change{Kind: ChangeKey, Key: later})
	expect(t, "retire before not before", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: "k5", At: later.NotBefore}}), ErrInvalid)
	expect(t, "retire missing", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: "nope", At: start.Unix()}}), ErrNotFound)
	f.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: "k1", At: start.Add(time.Hour).Unix()}})
	if v, _ := f.r.Key("k1"); !v.Retired || v.Key.NotAfter != start.Add(time.Hour).Unix() || v.Key.AcceptUntil != k.AcceptUntil || f.r.active != 1 {
		t.Errorf("retired %+v", v)
	}
	reason(t, "retire twice", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: "k1", At: start.Add(time.Hour).Unix()}}), ReasonRetired)
	f.approve(Change{Kind: ChangeCompromise, Compromise: "k1"})
	f.approve(Change{Kind: ChangeKey, Key: keySpec(f, "k6", "owner")})
	f.approve(Change{Kind: ChangeCompromise, Compromise: "k6"})
	reason(t, "retire compromised", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: "k6", At: start.Add(time.Hour).Unix()}}), ReasonCompromised)
	reason(t, "compromise twice", f.try(Change{Kind: ChangeCompromise, Compromise: "k1"}), ReasonCompromised)
	expect(t, "compromise missing", f.try(Change{Kind: ChangeCompromise, Compromise: "nope"}), ErrNotFound)
	f.approve(Change{Kind: ChangeKey, Key: keySpec(f, "k2", "owner")})
	f.approve(Change{Kind: ChangeCompromise, Compromise: "k2"})
	if f.r.active != 1 {
		t.Errorf("active %d", f.r.active)
	}
	f.auth.set(time.Time{}, false)
	f.approve(Change{Kind: ChangeKey, Key: keySpec(f, "k3", "owner")})
	reason(t, "retire without time", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: "k3", At: start.Add(time.Hour).Unix()}}), OutcomeTimeUnavailable)
	for _, c := range []Change{{Kind: "other"}, {Kind: ChangeRetire}, {Kind: ChangeCompromise}, {Kind: ChangeReset}, {Kind: ChangeReset, Reset: Reset{At: 1}}} {
		if _, err := clean(c); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s accepted", c.Kind)
		}
	}
}

func TestVerificationOrder(t *testing.T) {
	f := standard(t)
	f.must(f.r.SelectMode("i1", ModeCommercial, ann))
	register := func(s Signed) string {
		out, err := f.r.Register("i1", s, ann)
		f.must(err)
		return out
	}
	c := commercial("e1", "agr-1", start.Add(-time.Hour), start.Add(time.Hour))
	edited := f.sign(c)
	c2 := c
	c2.IssuedAt = start.Add(-2 * time.Hour).Unix()
	edited.Body = Encode(c2)
	if out := register(edited); out != OutcomeInvalid {
		t.Errorf("edited issued at %s", out)
	}
	trailing := f.sign(c)
	trailing.Body = append(trailing.Body, 0)
	if out := register(trailing); out != OutcomeInvalid {
		t.Errorf("trailing %s", out)
	}
	if out := register(Signed{Body: f.sign(c).Body, Signature: []byte{1}}); out != OutcomeInvalid {
		t.Errorf("short signature %s", out)
	}
	issuer := c
	issuer.Issuer = "mallory"
	if out := register(f.sign(issuer)); out != OutcomeUnverifiable {
		t.Errorf("issuer %s", out)
	}
	future := c
	future.IssuedAt = start.Add(time.Minute).Unix()
	if out := register(f.sign(future)); out != OutcomeUnverifiable {
		t.Errorf("future issued %s", out)
	}
	before := c
	before.IssuedAt = start.Add(-48 * time.Hour).Unix()
	if out := register(f.sign(before)); out != OutcomeUnverifiable {
		t.Errorf("before key %s", out)
	}
	product := c
	product.Product = "other"
	if out := register(f.sign(product)); out != OutcomeScopeMismatch {
		t.Errorf("product %s", out)
	}
	f.auth.set(start.Add(-time.Minute), true)
	if out := register(f.sign(c)); out != OutcomeClockAnomaly {
		t.Errorf("anomaly before key checks %s", out)
	}
	f.auth.set(start.Add(25*time.Hour), true)
	if out := register(f.sign(c)); out != ReasonLocalClockBehind {
		t.Errorf("ahead %s", out)
	}
	f.auth.set(start, true)
	if out := register(f.sign(c)); out != OutcomeAccepted {
		t.Errorf("accepted %s", out)
	}
	if out := register(f.sign(c)); out != OutcomeRepeat {
		t.Errorf("repeat %s", out)
	}
	changed := c
	changed.Subject = "Acme Limited"
	_, err := f.r.Register("i1", f.sign(changed), ann)
	expect(t, "same id other body", err, ErrConflict)
	g := standard(t)
	g.must(g.r.SelectMode("i1", ModeCommercial, ann))
	g.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: "k1", At: start.Add(time.Hour).Unix()}})
	backdated := commercial("e9", "agr-9", start, start.Add(48*time.Hour))
	backdated.IssuedAt = start.Add(30 * time.Minute).Unix()
	g.advance(2 * time.Hour)
	if out, _ := g.r.Register("i1", g.sign(backdated), ann); out != OutcomeAccepted {
		t.Errorf("signed inside window, accepted until AcceptUntil %s", out)
	}
	late := commercial("e10", "agr-10", start, start.Add(48*time.Hour))
	late.IssuedAt = start.Add(90 * time.Minute).Unix()
	if out, _ := g.r.Register("i1", g.sign(late), ann); out != OutcomeUnverifiable {
		t.Errorf("signed after retirement %s", out)
	}
}

func TestCodec(t *testing.T) {
	c := commercial("e1", "agr-1", start, start.Add(time.Hour))
	if got, ok := Decode(Encode(c)); !ok || got.ID != "e1" || got.End != c.End {
		t.Fatalf("roundtrip %+v %v", got, ok)
	}
	p := c
	p.End, p.Perpetual = 0, true
	if _, ok := Decode(Encode(p)); !ok {
		t.Error("perpetual")
	}
	e := evaluation("g1")
	e.ActivateBy = start.Unix()
	if _, ok := Decode(Encode(e)); !ok {
		t.Error("evaluation")
	}
	bad := map[string]func(*Claims){
		"control": func(c *Claims) { c.Subject = "Acme\nLtd" },
		"kind":    func(c *Claims) { c.Kind = "trial" }, "id": func(c *Claims) { c.ID = "" }, "issuer": func(c *Claims) { c.Issuer = "" },
		"key": func(c *Claims) { c.KeyID = "" }, "subject id": func(c *Claims) { c.SubjectID = "" }, "subject": func(c *Claims) { c.Subject = "" },
		"product": func(c *Claims) { c.Product = "" }, "installation": func(c *Claims) { c.Installation = "" }, "scope": func(c *Claims) { c.Scope = nil },
		"scope order": func(c *Claims) { c.Scope = []string{"b", "a"} }, "scope dup": func(c *Claims) { c.Scope = []string{"a", "a"} },
		"scope bad": func(c *Claims) { c.Scope = []string{"a b"} }, "agreement": func(c *Claims) { c.Agreement = "" },
		"supersedes": func(c *Claims) { c.Supersedes = "a b" }, "channel": func(c *Claims) { c.Channel = "reseller" },
		"metric": func(c *Claims) { c.Metric = "cloud_spend" }, "start": func(c *Claims) { c.Start = 0 }, "end": func(c *Claims) { c.End = c.Start },
		"both": func(c *Claims) { c.Perpetual = true }, "terms": func(c *Claims) { c.Terms = "x" }, "activate by": func(c *Claims) { c.ActivateBy = 1 },
		"issued": func(c *Claims) { c.IssuedAt = 0 }, "late": func(c *Claims) { c.End = maxTime + 1 },
		"many": func(c *Claims) {
			c.Scope = nil
			for i := 0; i <= maxScope; i++ {
				c.Scope = append(c.Scope, "s"+strings.Repeat("0", 2-len(itoa(i)))+itoa(i))
			}
		},
	}
	for name, mutate := range bad {
		x := commercial("e1", "agr-1", start, start.Add(time.Hour))
		x.Scope = []string{"a"}
		mutate(&x)
		if _, ok := Decode(Encode(x)); ok {
			t.Errorf("commercial %s accepted", name)
		}
	}
	evalBad := map[string]func(*Claims){
		"terms": func(c *Claims) { c.Terms = "" }, "agreement": func(c *Claims) { c.Agreement = "a" }, "supersedes": func(c *Claims) { c.Supersedes = "a" },
		"channel": func(c *Claims) { c.Channel = ChannelDirect }, "metric": func(c *Claims) { c.Metric = MetricSeats }, "start": func(c *Claims) { c.Start = 1 },
		"end": func(c *Claims) { c.End = 1 }, "perpetual": func(c *Claims) { c.Perpetual = true },
	}
	for name, mutate := range evalBad {
		x := evaluation("g1")
		mutate(&x)
		if _, ok := Decode(Encode(x)); ok {
			t.Errorf("evaluation %s accepted", name)
		}
	}
	full := commercial("e1", "agr-1", start, start.Add(time.Hour))
	full.Scope = nil
	for i := 0; i < maxScope; i++ {
		full.Scope = append(full.Scope, "s"+strings.Repeat("0", 2-len(itoa(i)))+itoa(i))
	}
	if _, ok := Decode(Encode(full)); !ok {
		t.Error("scope cap refused")
	}
	body := Encode(c)
	raw := map[string][]byte{
		"empty":    nil,
		"short":    body[:5],
		"header":   append([]byte{0, 0, 0, 0, 0, 0, 0, 1, 'x'}, body[41:]...),
		"overlong": append(append([]byte{}, body[:8]...), 0xff),
		"large":    make([]byte, maxBody+1),
	}
	for name, b := range raw {
		if _, ok := Decode(b); ok {
			t.Errorf("raw %s accepted", name)
		}
	}
	var e2 encoder
	for _, v := range []string{claimsHeader, DocCommercial, "e1", "owner", "k1", "0100"} {
		e2.text(v)
	}
	if _, ok := Decode(e2.out.Bytes()); ok {
		t.Error("leading zero")
	}
	mutateField := func(index int, value string) []byte {
		fields := splitFields(body)
		fields[index] = value
		var out encoder
		for _, f := range fields {
			out.text(f)
		}
		return out.out.Bytes()
	}
	for name, b := range map[string][]byte{
		"count text":  mutateField(12, "x"),
		"count zero":  mutateField(12, "0"),
		"count lead":  mutateField(12, "01"),
		"perpetual 2": mutateField(len(splitFields(body))-3, "2"),
		"time sign":   mutateField(5, "-5"),
		"time text":   mutateField(5, "1a"),
	} {
		if _, ok := Decode(b); ok {
			t.Errorf("%s accepted", name)
		}
	}
	for _, text := range []string{"", "01", "1a", strings.Repeat("9", 13)} {
		if digitsOnly(text, 12) {
			t.Errorf("digits %q", text)
		}
	}
	if !digitsOnly("0", 1) || !digitsOnly(strings.Repeat("9", 12), 12) {
		t.Error("digits edges")
	}
}

func splitFields(body []byte) []string {
	d := &decoder{in: body}
	out := []string{}
	for len(d.in) > 0 {
		out = append(out, d.text())
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for ; n > 0; n /= 10 {
		out = string(rune('0'+n%10)) + out
	}
	return out
}
