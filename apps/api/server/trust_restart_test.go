package server

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

func (f world) activated(t *testing.T, age string) (string, admins) {
	t.Helper()
	p := f.trustAdmins()
	id := f.approvedChange(p, age, "0")
	if got := f.trustActivate(p.c, id); got.status != 200 {
		t.Fatal(got.raw)
	}
	return id, p
}

func (f world) startWith(t *testing.T, change func(*Config), prepare func(*Server)) (world, error) {
	t.Helper()
	f.s.Close()
	cfg := f.s.cfg
	if change != nil {
		change(&cfg)
	}
	s, err := New(cfg)
	if err != nil {
		return world{}, err
	}
	if err := s.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820}); err != nil {
		t.Fatal(err)
	}
	s.manual = true
	if prepare != nil {
		prepare(s)
	}
	if err := s.Start(); err != nil {
		return world{}, err
	}
	return world{t: t, s: s, h: s.Handler(), now: f.now, mu: f.mu, value: f.value}, nil
}

func (f world) ledgerLines(t *testing.T) [][]byte {
	t.Helper()
	data, err := os.ReadFile(ledgerFile(f))
	if err != nil {
		t.Fatal(err)
	}
	return bytes.SplitAfter(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
}

func (f world) breakLine(t *testing.T, n int) {
	t.Helper()
	parts := f.ledgerLines(t)
	parts[n-1] = []byte("x\n")
	f.must(os.WriteFile(ledgerFile(f), bytes.Join(parts, nil), 0o600))
}

func (f world) appendGarbage(t *testing.T) {
	t.Helper()
	file, err := os.OpenFile(ledgerFile(f), os.O_WRONLY|os.O_APPEND, 0o600)
	f.must(err)
	_, err = file.Write([]byte("x\n"))
	f.must(errors.Join(err, file.Close()))
}

func (f world) trustNow(t *testing.T) map[string]any {
	t.Helper()
	return f.policyRead(f.as(AdminLogin)).body["trust"].(map[string]any)
}

type fixedReader struct{ data []byte }

func (r *fixedReader) Read(b []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, errors.New("exhausted")
	}
	n := copy(b, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestRestartReplaysTheLatestActivation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	id, _ := f.activated(t, "60")
	run, _ := trustchange.RunOf(id)
	raw := []byte{}
	for i := 0; i < len(run); i += 2 {
		n, _ := strconv.ParseUint(run[i:i+2], 16, 8)
		raw = append(raw, byte(n))
	}
	raw = append(raw, 0xab, 0xcd, 0xef, 0x01)
	g, err := f.startWith(t, nil, func(s *Server) { s.random = &fixedReader{data: raw} })
	f.must(err)
	current, _ := g.s.policy.Policy(SampleWorkspace)
	if current.Epoch != 2 || current.Number != 2 || current.Revision.MaxAge.Minutes() != 60 || current.Activator != SampleAdmin3 || g.s.trustOrigin.origin != originReplay {
		t.Errorf("replayed %+v %+v", current, g.s.trustOrigin)
	}
	if g.s.trust.Run() != "abcdef01" {
		t.Errorf("a colliding run prefix was not redrawn: %s", g.s.trust.Run())
	}
	a := g.as(AdminLogin)
	page := g.policyRead(a).body
	row := page["changes"].([]any)[0].(map[string]any)
	if row["id"] != id || row["earlier_run"] != true || row["run_started_at"] != stamp(start) || row["invalidation"].(map[string]any)["items"].([]any)[0].(map[string]any)["reason"] != "server_restarted" ||
		page["trust"].(map[string]any)["origin"] != originReplay || page["trust"].(map[string]any)["change"] != id {
		t.Errorf("read after restart %+v", row)
	}
	if _, err := g.startWith(t, nil, func(s *Server) { s.random = &fixedReader{} }); err == nil || !strings.Contains(err.Error(), "exhausted") {
		t.Error("a start without randomness")
	}
}

func TestIsolationPublishesTheVerifiedPrefix(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.activated(t, "60")
	p := admins{f.signedIn(AdminLogin, pw), f.signedIn(Admin2Login, pw), f.signedIn(Admin3Login, pw)}
	second := f.approvedChange(p, "90", "0")
	if got := f.trustActivate(p.c, second); got.status != 200 {
		t.Fatal(got.raw)
	}
	lines := len(f.ledgerLines(t))
	f.s.Close()
	f.breakLine(t, lines)
	g := f.restart(nil)
	current, _ := g.s.policy.Policy(SampleWorkspace)
	trust := g.trustNow(t)
	ledger := g.policyRead(g.as(AdminLogin)).body["ledger"].(map[string]any)
	if current.Epoch != 2 || current.Revision.MaxAge.Minutes() != 60 || trust["origin"] != originReplay || ledger["state"] != "isolated" || fmt.Sprint(ledger["problem_line"]) != strconv.Itoa(lines) {
		t.Errorf("isolated %+v %+v", current, ledger)
	}
	g.code(t, "isolated", g.trustPropose(g.as(AdminLogin), "30", "0"), 503, string(commands.DependencyUnavailable))
	h := setup(t, SeedSample).ready()
	h.s.Close()
	h.breakLine(t, 1)
	k := h.restart(nil)
	if current, _ := k.s.policy.Policy(SampleWorkspace); current.Epoch != 1 || k.s.trustOrigin.origin != originSeed {
		t.Errorf("no trust record in the prefix: seed genesis %+v", current)
	}
	m := setup(t, SeedSample).ready()
	m.activated(t, "60")
	m.s.Close()
	marker := filepath.Join(m.s.cfg.Data, decisionledger.MarkerName)
	m.must(os.WriteFile(marker, []byte(`{"ledger_id":"`+strings.Repeat("a", 32)+`","created_at":"x"}`), 0o600))
	n := m.restart(nil)
	if current, _ := n.s.policy.Policy(SampleWorkspace); current.Epoch != 2 || n.s.ledger.Status().Isolation != decisionledger.IsolationMismatch {
		t.Errorf("a mismatched ledger still publishes its verified record %+v", current)
	}
	if len(n.s.ledger.Decisions()) != 0 {
		t.Error("revocations were replayed from a mismatched ledger")
	}
}

func TestStartSyncFailureKeepsTheReplayButStops(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.activated(t, "60")
	g, err := f.startWith(t, nil, func(s *Server) { s.fault = func(step string) bool { return step == "ledger-start-sync" } })
	f.must(err)
	g.s.fault = nil
	trust := g.trustNow(t)
	if trust["epoch"] != "2" || trust["durable_unconfirmed"] != true || g.s.ledger.Status().State != decisionledger.StateStopped {
		t.Errorf("trust %+v", trust)
	}
	g.code(t, "stopped", g.trustPropose(g.as(AdminLogin), "30", "0"), 503, string(commands.DependencyUnavailable))
}

func replaceWith(access func(*AccessSettings)) func(*Config) {
	return func(c *Config) {
		c.Access.AcceptNewLedger = true
		access(&c.Access)
	}
}

func TestReplacementCarriesTheTrustConfiguration(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.activated(t, "60")
	oldID := f.s.ledger.Status().LedgerID
	f.s.Close()
	problem := len(f.ledgerLines(t)) + 1
	f.appendGarbage(t)
	if _, err := f.restartWith(replaceWith(func(a *AccessSettings) { a.AcceptTrustReset = true })); !errors.Is(err, ErrConfig) {
		t.Errorf("a reset over a carryable record %v", err)
	}
	if _, err := f.restartWith(replaceWith(func(a *AccessSettings) { a.TrustGeneration = 5 })); !errors.Is(err, ErrConfig) {
		t.Errorf("a stated generation that is known %v", err)
	}
	g := f.restart(func(c *Config) { c.Access.AcceptNewLedger = true })
	st := g.s.ledger.Status()
	current, _ := g.s.policy.Policy(SampleWorkspace)
	trust := g.trustNow(t)
	if st.Generation != 1 || st.Genesis.CarriedFrom != oldID || current.Epoch != 1<<32+1 || current.Revision.MaxAge.Minutes() != 60 || current.Number != 2 ||
		trust["origin"] != originCarried || !strings.HasPrefix(trust["carried_from"].(string), oldID+":") || fmt.Sprintf("%.0f", trust["epoch_generation"]) != "1" || fmt.Sprintf("%.0f", trust["epoch_number"]) != "1" ||
		fmt.Sprint(trust["carried_line"]) != fmt.Sprint(problem) || st.Genesis.ReplacedLine != problem {
		t.Errorf("carried %+v %+v %+v", st, current, trust)
	}
	h := g.restart(func(c *Config) { c.Access.AcceptNewLedger = false })
	if current, _ := h.s.policy.Policy(SampleWorkspace); current.Epoch != 1<<32+1 || h.s.trustOrigin.origin != originCarried {
		t.Errorf("restart after carry %+v", current)
	}
	p := h.trustAdmins()
	id := h.approvedChange(p, "90", "0")
	if got := h.trustActivate(p.c, id); got.body["fields"].(map[string]any)["epoch"] != strconv.FormatUint(1<<32+2, 10) {
		t.Errorf("the next activation after a carry %s", got.raw)
	}
}

func TestReplacementWithoutACarryableRecord(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.s.Close()
	f.breakLine(t, 1)
	if _, err := f.restartWith(func(c *Config) { c.Access.AcceptNewLedger = true }); !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), "accept-trust-reset") {
		t.Errorf("a replacement without a carryable record %v", err)
	}
	if _, err := f.restartWith(replaceWith(func(a *AccessSettings) { a.CarryTrustFrom = strings.Repeat("c", 32) })); !errors.Is(err, ErrConfig) {
		t.Errorf("a named ledger without a trust record %v", err)
	}
	g := f.restart(replaceWith(func(a *AccessSettings) { a.AcceptTrustReset = true }))
	trust := g.trustNow(t)
	if current, _ := g.s.policy.Policy(SampleWorkspace); current.Epoch != 1<<32+1 || trust["origin"] != originSeed || trust["reset_at"] == "" || !g.s.ledger.Status().Genesis.TrustReset {
		t.Errorf("reset %+v %+v", current, trust)
	}
	if next, ok := nextEpoch(5<<32, 5); !ok || next != 5<<32+1 {
		t.Errorf("the first activation of a generation without a policy %d", next)
	}
}

func TestReplacementOfAnotherLedger(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.activated(t, "60")
	oldID := f.s.ledger.Status().LedgerID
	f.s.Close()
	marker := filepath.Join(f.s.cfg.Data, decisionledger.MarkerName)
	f.must(os.WriteFile(marker, []byte(`{"ledger_id":"`+strings.Repeat("a", 32)+`","created_at":"x","g":3}`), 0o600))
	if _, err := f.restartWith(func(c *Config) { c.Access.AcceptNewLedger = true }); !errors.Is(err, ErrConfig) {
		t.Errorf("an ID mismatch carries nothing without a flag %v", err)
	}
	for name, access := range map[string]func(*AccessSettings){
		"a wrong ledger ID": func(a *AccessSettings) { a.CarryTrustFrom = strings.Repeat("b", 32) },
		"both flags":        func(a *AccessSettings) { a.CarryTrustFrom, a.AcceptTrustReset = oldID, true },
		"neither flag":      func(a *AccessSettings) { a.CarryTrustFrom = "" },
	} {
		if _, err := f.restartWith(replaceWith(access)); !errors.Is(err, ErrConfig) {
			t.Errorf("%s %v", name, err)
		}
	}
	g := f.restart(replaceWith(func(a *AccessSettings) { a.CarryTrustFrom = oldID }))
	st := g.s.ledger.Status()
	if current, _ := g.s.policy.Policy(SampleWorkspace); st.Generation != 4 || current.Epoch != 4<<32+1 || current.Revision.MaxAge.Minutes() != 60 || st.Genesis.CarriedFrom != oldID {
		t.Errorf("carried from a named ledger %+v %+v", st, current)
	}
}

func TestStatedGeneration(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.activated(t, "60")
	f.s.Close()
	f.appendGarbage(t)
	g := f.restart(func(c *Config) { c.Access.AcceptNewLedger = true })
	g.s.Close()
	g.appendGarbage(t)
	h := g.restart(func(c *Config) { c.Access.AcceptNewLedger = true })
	if h.s.ledger.Status().Generation != 2 {
		t.Fatalf("generation %+v", h.s.ledger.Status())
	}
	h.s.Close()
	marker := filepath.Join(h.s.cfg.Data, decisionledger.MarkerName)
	h.must(os.WriteFile(marker, []byte("{"), 0o600))
	h.breakLine(t, 1)
	reset := replaceWith(func(a *AccessSettings) { a.AcceptTrustReset = true })
	if _, err := h.restartWith(reset); !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), "trust-generation") {
		t.Errorf("an unusable marker and ledger need a stated generation %v", err)
	}
	for _, n := range []uint64{1, 2} {
		if _, err := h.restartWith(replaceWith(func(a *AccessSettings) { a.AcceptTrustReset, a.TrustGeneration = true, n })); !errors.Is(err, ErrConfig) ||
			!strings.Contains(err.Error(), "at least 3") {
			t.Errorf("generation %d: the destroyed generation 2 ledger and its retained predecessors must not be reused %v", n, err)
		}
	}
	k := h.restart(replaceWith(func(a *AccessSettings) { a.AcceptTrustReset, a.TrustGeneration = true, 3000000 }))
	st := k.s.ledger.Status()
	trust := k.trustNow(t)
	ledger := k.policyRead(k.as(AdminLogin)).body["ledger"].(map[string]any)
	if st.Generation != 3000000 || !st.Genesis.GenerationStated || trust["epoch"] != strconv.FormatUint(3000000<<32+1, 10) || fmt.Sprintf("%.0f", trust["epoch_generation"]) != "3000000" ||
		fmt.Sprintf("%.0f", trust["epoch_number"]) != "1" || ledger["generation_stated"] != true {
		t.Errorf("stated %+v %+v", st, trust)
	}
}
