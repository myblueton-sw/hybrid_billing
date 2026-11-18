package plugins

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLicenseForm(t *testing.T) {
	accepted := []string{"MIT", "MIT-", "0BSD", "Zlib", "zlib", "Z", "z", "9", "A0", "LicenseRef-z9Z", "Apache-2.0", "LicenseRef-acme", "LicenseRef-a", "LicenseRef-.", "a",
		strings.Repeat("x", maxLicenseBytes), licenseRef + strings.Repeat("r", maxLicenseBytes-len(licenseRef)), "Licenseref", "LicenseRefX"}
	for _, text := range accepted {
		if !validLicense(text) {
			t.Errorf("%q refused", text)
		}
	}
	refused := []string{"", strings.Repeat("x", maxLicenseBytes+1), licenseRef + strings.Repeat("r", maxLicenseBytes-len(licenseRef)+1), "LicenseRef-",
		"licenseref-x", "LICENSEREF-x", "licenseRef-x", "-", ".", "-MIT", ".MIT", "MIT OR Apache-2.0", "(MIT)", "GPL-2.0+", "MIT_1", "MIT:1",
		"Überblick", "MIT\n", "\x00", "MIT/2", " MIT"}
	for _, text := range refused {
		if validLicense(text) {
			t.Errorf("%q accepted", text)
		}
	}
	for _, text := range refused {
		m := manifest("ledger", "1.0.0")
		m.License = text
		if m.valid() {
			t.Errorf("manifest with %q valid", text)
		}
		expect(t, "propose "+text, second(standard(t).r.Propose(signed(m), ann)), ErrInvalid)
	}
}

func TestLicenseListForm(t *testing.T) {
	clock := func() time.Time { return start }
	config := func(list []string) Config {
		return Config{Core: "2.4.0", Destinations: &destinations{}, Random: &entropy{}, Licenses: list}
	}
	full := []string{}
	for index := 0; index < maxLicenses; index++ {
		full = append(full, "L"+strconv.Itoa(1000+index))
	}
	for name, list := range map[string][]string{"nil": nil, "empty": {}, "one": {"MIT"}, "sorted": {"Apache-2.0", "LicenseRef-x", "MIT"}, "full": full} {
		if _, err := NewRegistry(config(list), limits, clock); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, list := range map[string][]string{
		"unsorted": {"MIT", "Apache-2.0"}, "duplicate": {"MIT", "MIT"}, "malformed": {"MIT", "MIT OR ISC"}, "empty entry": {""},
		"over": append(append([]string{}, full...), "L9999"), "bare reference": {"LicenseRef-"},
	} {
		expect(t, name, second(NewRegistry(config(list), limits, clock)), ErrInvalid)
	}
	list := []string{"Apache-2.0", "MIT"}
	r, err := NewRegistry(config(list), limits, clock)
	if err != nil {
		t.Fatal(err)
	}
	list[0] = "Zlib"
	got := r.Licenses()
	if strings.Join(got, ",") != "Apache-2.0,MIT" {
		t.Errorf("licenses %v", got)
	}
	got[0] = "Zlib"
	if r.Licenses()[0] != "Apache-2.0" {
		t.Error("licenses read is not a copy")
	}
	empty, _ := NewRegistry(config(nil), limits, clock)
	if got := empty.Licenses(); got == nil || len(got) != 0 {
		t.Errorf("empty licenses %#v", got)
	}
}

func lengthPrefixed(values ...string) []byte {
	out := []byte{}
	for _, value := range values {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		out = append(out, size[:]...)
		out = append(out, value...)
	}
	return out
}

func TestManifestKnownAnswerIndependent(t *testing.T) {
	sum := sha256.Sum256([]byte("synthetic-package-ledger1.0.0"))
	digestText := hex.EncodeToString(sum[:])
	message := lengthPrefixed("hybrid-billing/plugin/v2", "ledger", "1.0.0", "acme", "k1", digestText, "MIT", "2.0.0", "2.9.9",
		"3", "call_command", "invoice.void", "read_events", "invoice.issued", "serve_screen", "overview",
		"1", "crm", "1", "api", "1000000000", "64", "2",
		"ledger-data", "acme", "30", "retain_for_evidence", "additive",
		"1", "overview", "table", "Overview <b>", "2", "total", "Total", "ledger-data", "event", "Event", "invoice.issued")
	answer := sha256.Sum256(message)
	if got := hex.EncodeToString(answer[:]); got != manifestAnswer {
		t.Errorf("independent answer %s; pinned %s", got, manifestAnswer)
	}
	if hex.EncodeToString(manifest("ledger", "1.0.0").encode()) != hex.EncodeToString(message) {
		t.Error("encode differs from the documented field order")
	}
}

func TestLicenseIsSigned(t *testing.T) {
	f := standard(t)
	m := manifest("audit", "1.0.0")
	c := signed(m)
	c.Manifest.License = "Apache-2.0"
	reason(t, "license swapped after signing", f.try(c), ReasonSignatureInvalid)
	f.version(m)
	if v, err := f.r.Version("audit", "1.0.0"); err != nil || v.Manifest.License != "MIT" || !v.Permitted {
		t.Errorf("version %+v %v", v, err)
	}
}

func licensed(plugin, number, license string) Manifest {
	m := manifest(plugin, number)
	m.License = license
	return m
}

func (f world) waiting() int {
	n := 0
	for _, s := range f.r.scopes {
		n += len(s.pending)
	}
	return n
}

func (f world) refusedAtProposal(name string, c Change) {
	f.t.Helper()
	before, count := f.snapshot(0), f.r.counter
	pending := f.waiting()
	_, err := f.r.Propose(c, ann)
	reason(f.t, name, err, ReasonLicenseNotPermitted)
	if f.snapshot(0) != before || f.r.counter != count || f.waiting() != pending {
		f.t.Errorf("%s: refusal changed state", name)
	}
}

func TestLicenseRefusedBeforeActivation(t *testing.T) {
	f := standard(t)
	f.version(licensed("ledger", "0.9.0", "LicenseRef-closed"))
	f.version(licensed("ledger", "1.2.0", "GPL-3.0-only"))
	for _, number := range []string{"0.9.0", "1.2.0"} {
		v, err := f.r.Version("ledger", number)
		if err != nil || v.Permitted {
			t.Errorf("%s permitted %+v %v", number, v, err)
		}
	}
	f.refusedAtProposal("install", at(ChangeInstall, "w1", "ledger", "1.2.0"))
	f.refusedAtProposal("verification install", at(ChangeInstall, "v1", "ledger", "0.9.0"))
	f.install("w1", "ledger", "1.0.0")
	f.refusedAtProposal("upgrade", at(ChangeUpgrade, "w1", "ledger", "1.2.0"))
	f.refusedAtProposal("rollback", at(ChangeRollback, "w1", "ledger", "0.9.0"))
	f.r.config.Licenses = []string{"Apache-2.0"}
	f.refusedAtProposal("enable", at(ChangeEnable, "w1", "ledger", ""))
	f.r.config.Licenses = permittedLicenses
	f.enable("w1", "ledger")
	if v := f.state("w1", "ledger"); v.State != StateEnabled || v.Version != "1.0.0" {
		t.Errorf("permitted path %+v", v)
	}
	f.approve(at(ChangeUpgrade, "w1", "ledger", "1.1.0"))
}

func TestLicenseBackupChecksAtApproval(t *testing.T) {
	for _, c := range []Change{at(ChangeInstall, "w2", "ledger", "1.0.0"), at(ChangeUpgrade, "w1", "ledger", "1.1.0"), at(ChangeRollback, "w1", "ledger", "0.9.0"),
		at(ChangeEnable, "w1", "ledger", "")} {
		f := standard(t)
		f.version(manifest("ledger", "0.9.0"))
		f.install("w1", "ledger", "1.0.0")
		p, err := f.r.Propose(c, ann)
		f.must(err)
		f.must(f.r.Recheck(p.Number))
		f.r.config.Licenses = []string{"Apache-2.0"}
		f.agrees(c.Kind, p.Number, bob, Refusal{Reason: ReasonLicenseNotPermitted}.Error())
		if v, _ := f.r.Proposal(p.Number); v.State != ProposalPending {
			t.Errorf("%s: proposal %+v", c.Kind, v)
		}
	}
}

func TestLicenseBackupChecksAtRuntime(t *testing.T) {
	f := running(t)
	text := f.token("w1", run1, events)
	f.r.config.Licenses = []string{"Apache-2.0"}
	reason(t, "issue", third(f.r.IssueToken("w1", "ledger", []Capability{events}, time.Minute, run1)), ReasonLicenseNotPermitted)
	reason(t, "check", second(f.r.CheckToken(text, events)), ReasonStaleToken)
	reason(t, "check forgotten", second(f.r.CheckToken(text, events)), ReasonUnknownToken)
	reason(t, "egress", f.r.CheckEgress("w1", "ledger", "crm", run1), ReasonLicenseNotPermitted)
	f.r.config.Licenses = permittedLicenses
	f.dest.during = func() { f.r.config.Licenses = []string{"Apache-2.0"} }
	expect(t, "egress during dependency", f.r.CheckEgress("w1", "ledger", "crm", run1), ErrStale)
	f.dest.during = nil
	f.must(second(f.r.MarkVulnerable("ledger", "1.0.0", "Unrelated flaw", ann)))
	for _, w := range []string{"w1", "w2", "v1"} {
		if v := f.state(w, "ledger"); v.State != StateStopped || v.Reason != ReasonLicenseNotPermitted {
			t.Errorf("%s swept %+v", w, v)
		}
	}
}

func TestLicenseReasonOrder(t *testing.T) {
	f := standard(t)
	f.version(licensed("ledger", "1.2.0", "LicenseRef-closed"))
	f.must(second(f.r.MarkVulnerable("ledger", "1.2.0", "Known flaw", ann)))
	reason(t, "license before vulnerable", f.try(at(ChangeInstall, "w1", "ledger", "1.2.0")), ReasonLicenseNotPermitted)
	f.must(second(f.r.RevokePublisher("acme", "Compromised", ann)))
	reason(t, "revoked before license", f.try(at(ChangeInstall, "w1", "ledger", "1.2.0")), ReasonRevoked)
}

func TestLicenseSignatureCoversLicense(t *testing.T) {
	a := licensed("ledger", "1.0.0", "MIT")
	b := licensed("ledger", "1.0.0", "Apache-2.0")
	signature := ed25519.Sign(private("acmek1"), a.encode())
	if ed25519.Verify(public("acmek1"), b.encode(), signature) {
		t.Error("signature over license A verifies license B")
	}
}

func TestValidLicensesWrapper(t *testing.T) {
	for _, list := range [][]string{nil, {"MIT"}, {"Apache-2.0", "MIT"}, {"MIT", "Apache-2.0"}, {"MIT", "MIT"}, {"MIT OR ISC"}} {
		if ValidLicenses(list) != validLicenses(list) {
			t.Errorf("%v", list)
		}
	}
	if !ValidLicenses([]string{"Apache-2.0", "MIT"}) || ValidLicenses([]string{"MIT", "Apache-2.0"}) {
		t.Error("wrapper")
	}
}
