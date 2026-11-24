package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (f world) credentialRowOf(a actor, service, id string) map[string]any {
	f.t.Helper()
	for _, c := range f.service(a, service)["credentials"].([]any) {
		if m := c.(map[string]any); m["id"] == id {
			return m
		}
	}
	return nil
}

func (f world) secretOf(a actor, ref string) []byte {
	f.t.Helper()
	got := f.collect(a, ref)
	if got.status != 200 {
		f.t.Fatalf("collect %+v", got)
	}
	return secretBytes(f.t, got.body["secret"].(string))
}

func (f world) rotated(a actor, credential string, fields map[string]string) (string, string, reply) {
	f.t.Helper()
	got := f.run(a, "service.rotate_credential", credential, fields)
	if got.status != 200 || got.body["business_state"] != "credential_rotated" {
		f.t.Fatalf("rotate %+v", got)
	}
	out := got.body["fields"].(map[string]any)
	return out["credential"].(string), out["secret_ref"].(string), got
}

func TestCredentialRotationIssuesTheNewOneAndBoundsTheOldOne(t *testing.T) {
	var logs bytes.Buffer
	f := setupWith(t, func(c *Config) { c.Log = slog.New(slog.NewTextHandler(&logs, nil)) })
	root := f.as(AdminLogin)
	svc := f.newService(root, "rotate")
	old, oldRef, _ := f.issued(root, svc, "30")
	oldSecret := f.secretOf(root, oldRef)
	limits := f.servicesRead(root).body["limits"].(map[string]any)
	if fmt.Sprint(limits["credential_overlap_minutes"]) != "1440" || fmt.Sprint(limits["credential_overlap_max_minutes"]) != "10080" || fmt.Sprint(limits["secret_pickup_minutes"]) != "10" {
		t.Fatalf("limits %+v", limits)
	}
	for name, fields := range map[string]map[string]string{
		"no days": {}, "days 0": {"days": "0"}, "days 31": {"days": "31"}, "days x": {"days": "x"},
	} {
		f.failed(t, name, f.run(root, "service.rotate_credential", old, fields), commands.InvalidRequest, "days")
	}
	for _, overlap := range []string{"", "9", "10081", "x", "010", "1440.5", "-10", "1000000"} {
		f.failed(t, "overlap "+overlap, f.run(root, "service.rotate_credential", old, map[string]string{"days": "5", "overlap_minutes": overlap}), commands.InvalidRequest, "overlap_minutes")
	}
	if rows := f.service(root, svc)["credentials"].([]any); len(rows) != 1 || rows[0].(map[string]any)["not_after"] != stamp(start.Add(30*24*time.Hour)) {
		t.Fatalf("a refused rotation changed the credentials: %+v", rows)
	}
	next, ref, result := f.rotated(root, old, map[string]string{"days": "5"})
	end := start.Add(24 * time.Hour)
	fields := result.body["fields"].(map[string]any)
	if len(fields) != 4 || fields["replaced"] != old || fields["replaced_until"] != stamp(end) || !strings.HasPrefix(next, "cred-") || next == old {
		t.Fatalf("result %+v", fields)
	}
	before, after := f.credentialRowOf(root, svc, old), f.credentialRowOf(root, svc, next)
	if before["not_after"] != stamp(end) || before["replaced_by"] != next || before["rotation"] != rotationEnding || before["replaces"] != nil || before["state"] != "active" {
		t.Errorf("old row %+v", before)
	}
	if after["not_after"] != stamp(start.Add(5*24*time.Hour)) || after["not_before"] != stamp(start) || after["replaces"] != old || after["rotation"] != rotationReplacing || after["replaced_by"] != nil ||
		after["secret_pickup"] != pickupOpen {
		t.Errorf("new row %+v", after)
	}
	f.code(t, "another session collecting the new secret", f.collect(f.as(AdminLogin), ref), 403, string(commands.PermissionDenied))
	collected := f.collect(root, ref)
	if collected.status != 200 || collected.body["credential"] != next {
		t.Fatalf("collect %+v", collected)
	}
	secret := collected.body["secret"].(string)
	newSecret := secretBytes(t, secret)
	journal, _ := json.Marshal(f.s.commands.Journal(0, 1000))
	for name, text := range map[string]string{"result": result.raw, "journal": string(journal), "logs": logs.String(), "read": f.servicesRead(root).raw} {
		if strings.Contains(text, secret) || strings.Contains(text, strings.TrimPrefix(secret, secretPrefix)) {
			t.Errorf("secret in %s", name)
		}
	}
	boundary := f.s.identity.Boundary()
	if _, err := boundary.AccessByCredential(next, verifierOf(newSecret)); err != nil {
		t.Errorf("the new credential: %v", err)
	}
	f.advance(24*time.Hour - time.Second)
	if _, err := boundary.AccessByCredential(old, verifierOf(oldSecret)); err != nil {
		t.Errorf("the old credential one second before its new end: %v", err)
	}
	f.advance(time.Second)
	if _, err := boundary.AccessByCredential(old, verifierOf(oldSecret)); err == nil {
		t.Error("the old credential answers at its new end")
	}
	if _, err := boundary.AccessByCredential(next, verifierOf(newSecret)); err != nil {
		t.Errorf("the new credential after the old one ended: %v", err)
	}
	root = f.as(AdminLogin)
	before, after = f.credentialRowOf(root, svc, old), f.credentialRowOf(root, svc, next)
	if before["state"] != "expired" || before["rotation"] != nil || before["replaced_by"] != next || after["rotation"] != nil || after["replaces"] != old {
		t.Errorf("after the old end: old %+v new %+v", before, after)
	}
	if raw := f.integrations(root).raw; strings.Contains(raw, "replaced_by") || strings.Contains(raw, `"rotation"`) || strings.Contains(raw, `"replaces"`) {
		t.Error("the integrations read carries rotation fields")
	}
}

func TestCredentialRotationOverlapAndEnds(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "overlap")
	short, _, _ := f.issued(root, svc, "1")
	_, _, got := f.rotated(root, short, map[string]string{"days": "2", "overlap_minutes": "2880"})
	if until := got.body["fields"].(map[string]any)["replaced_until"]; until != stamp(start.Add(24*time.Hour)) || f.credentialRowOf(root, svc, short)["not_after"] != until {
		t.Errorf("an old credential with less time left than the overlap ends at %v", until)
	}
	long, _, _ := f.issued(root, svc, "30")
	next, _, got := f.rotated(root, long, map[string]string{"days": "30", "overlap_minutes": "10"})
	if until := got.body["fields"].(map[string]any)["replaced_until"]; until != stamp(start.Add(10*time.Minute)) {
		t.Errorf("the shortest overlap ends at %v", until)
	}
	widest, _, _ := f.issued(root, svc, "30")
	_, _, got = f.rotated(root, widest, map[string]string{"days": "1", "overlap_minutes": "10080"})
	if until := got.body["fields"].(map[string]any)["replaced_until"]; until != stamp(start.Add(7*24*time.Hour)) {
		t.Errorf("the longest overlap ends at %v", until)
	}
	f.failed(t, "a second rotation of the same credential", f.run(root, "service.rotate_credential", long, map[string]string{"days": "1"}), commands.InvalidStage)
	if _, _, got := f.rotated(root, next, map[string]string{"days": "1"}); got.status != 200 {
		t.Errorf("rotating the replacement %+v", got)
	}
	if row := f.credentialRowOf(root, svc, long); row["not_after"] != stamp(start.Add(10*time.Minute)) || row["replaced_by"] != next {
		t.Errorf("rotating the replacement changed the first credential %+v", row)
	}
	if row := f.credentialRowOf(root, svc, next); row["rotation"] != rotationEnding || row["replaces"] != long || row["not_after"] != stamp(start.Add(24*time.Hour)) {
		t.Errorf("a replacement that is itself being replaced %+v", row)
	}
}

func TestCredentialRotationRefusalsChangeNothing(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	svc := f.newService(root, "refusals")
	old, _, _ := f.issued(root, svc, "30")
	days := map[string]string{"days": "1"}
	snapshot := func() string {
		return fmt.Sprint(f.service(root, svc)["credentials"], f.collectedEntries(), len(f.s.dir.pickups))
	}
	was := snapshot()
	unchanged := func(name string) {
		t.Helper()
		if now := snapshot(); now != was {
			t.Errorf("%s changed something:\n%s\nwas\n%s", name, now, was)
		}
	}
	f.failed(t, "a member", f.runWith(f.as(MemberLogin), f.revision(root), "service.rotate_credential", old, days), commands.PermissionDenied)
	f.failed(t, "an unknown credential", f.run(root, "service.rotate_credential", "cred-ghost", days), commands.PermissionDenied)
	f.failed(t, "a service account as the target", f.run(root, "service.rotate_credential", svc, days), commands.PermissionDenied)
	stale := f.revision(root)
	other := f.newService(root, "elsewhere")
	f.failed(t, "a stale revision", f.runWith(root, stale, "service.rotate_credential", old, days), commands.RevisionConflict)
	f.s.fault = func(at string) bool { return at == "credentials" }
	f.failed(t, "credentials unavailable", f.run(root, "service.rotate_credential", old, days), commands.DependencyUnavailable)
	f.s.fault = func(at string) bool { return at == "services" }
	f.failed(t, "services unavailable", f.run(root, "service.rotate_credential", old, days), commands.DependencyUnavailable)
	f.s.fault = nil
	unchanged("refusals by standing, target, revision and faults")

	f.run(root, "service.suspend", svc, nil)
	was = snapshot()
	f.failed(t, "a suspended account", f.run(root, "service.rotate_credential", old, days), commands.PrerequisiteFailed)
	unchanged("a suspended account")
	f.run(root, "service.reinstate", svc, nil)

	ids := []string{old}
	for len(ids) < 10 {
		id, _, _ := f.issued(root, svc, "30")
		ids = append(ids, id)
	}
	was = snapshot()
	f.failed(t, "the limit, counted with the old credential", f.run(root, "service.rotate_credential", old, days), commands.PrerequisiteFailed)
	unchanged("the limit")
	f.run(root, "service.revoke_credential", ids[9], nil)
	was = snapshot()
	f.failed(t, "a revoked old credential", f.run(root, "service.rotate_credential", ids[9], days), commands.InvalidStage)
	unchanged("a revoked old credential")
	if _, _, got := f.rotated(root, old, days); got.status != 200 {
		t.Fatalf("a rotation below the limit %+v", got)
	}
	expiring, _, _ := f.issued(root, other, "1")
	f.advance(24 * time.Hour)
	root = f.as(AdminLogin)
	f.failed(t, "an expired old credential", f.run(root, "service.rotate_credential", expiring, days), commands.InvalidStage)
	kept, _, _ := f.issued(root, other, "30")
	f.run(root, "service.retire", other, nil)
	if row := f.credentialRowOf(root, other, kept); row["state"] != "revoked" {
		t.Fatalf("retiring did not revoke the live credential %+v", row)
	}
	f.failed(t, "a credential revoked by retiring its account", f.run(root, "service.rotate_credential", kept, days), commands.InvalidStage)
}

func TestCredentialRotationReplay(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "replayed")
	old, _, _ := f.issued(root, svc, "30")
	body := fmt.Sprintf(`{"request_id":"r1","target_id":%q,"idempotency_key":"k1","expected_revision":%q,"rationale":"x","body":{"days":"3","overlap_minutes":"10"}}`, old, f.revision(root))
	first := f.call("service.rotate_credential", body, root.cookie, root.csrf)
	again := f.call("service.rotate_credential", body, root.cookie, root.csrf)
	if first.status != 200 || again.status != 200 || again.body["replay"] != true || fmt.Sprint(first.body["fields"]) != fmt.Sprint(again.body["fields"]) {
		t.Fatalf("replay %s %s", first.raw, again.raw)
	}
	if n := len(f.service(root, svc)["credentials"].([]any)); n != 2 || f.collectedEntries() != 2 {
		t.Fatalf("after a replay: %d credentials, %d pickup entries", n, f.collectedEntries())
	}
	fields := first.body["fields"].(map[string]any)
	next, ref := fields["credential"].(string), fields["secret_ref"].(string)
	elsewhere := f.as(AdminLogin)
	fromOther := f.call("service.rotate_credential", body, elsewhere.cookie, elsewhere.csrf)
	if fromOther.status != 200 || fmt.Sprint(fromOther.body["fields"]) != fmt.Sprint(first.body["fields"]) {
		t.Fatalf("replay from another session of the same person %s", fromOther.raw)
	}
	f.code(t, "the replayed reference collected from the other session", f.collect(elsewhere, ref), 403, string(commands.PermissionDenied))
	if got := f.collect(root, ref); got.status != 200 || got.body["credential"] != next {
		t.Fatalf("collect by the issuing session after replays %+v", got)
	}
	f.advance(11 * time.Minute)
	root = f.as(AdminLogin)
	late := f.call("service.rotate_credential", body, root.cookie, root.csrf)
	if late.status != 200 || late.body["replay"] != true || fmt.Sprint(late.body["fields"]) != fmt.Sprint(first.body["fields"]) {
		t.Fatalf("replay after the old credential ended %s", late.raw)
	}
	if rows := f.service(root, svc)["credentials"].([]any); len(rows) != 2 || f.credentialRowOf(root, svc, old)["state"] != "expired" {
		t.Errorf("after the late replay %+v", rows)
	}
	changed := strings.Replace(body, `"days":"3"`, `"days":"4"`, 1)
	if got := f.call("service.rotate_credential", changed, root.cookie, root.csrf); got.status == 200 {
		t.Errorf("the same key with another body was accepted %s", got.raw)
	}
}

func TestCredentialRotationDuringTheOverlap(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "during")
	a, _, _ := f.issued(root, svc, "30")
	b, _, _ := f.rotated(root, a, map[string]string{"days": "30", "overlap_minutes": "60"})
	end := stamp(start.Add(time.Hour))
	if got := f.run(root, "service.revoke_credential", b, nil); got.status != 200 {
		t.Fatalf("revoking the new credential %+v", got)
	}
	if row := f.credentialRowOf(root, svc, a); row["not_after"] != end || row["state"] != "active" || row["rotation"] != rotationEnding {
		t.Errorf("revoking the new credential changed the old one %+v", row)
	}
	c, _, _ := f.issued(root, svc, "30")
	d, dRef, _ := f.rotated(root, c, map[string]string{"days": "30", "overlap_minutes": "60"})
	if got := f.run(root, "service.revoke_credential", c, nil); got.status != 200 {
		t.Fatalf("revoking the old credential %+v", got)
	}
	if row := f.credentialRowOf(root, svc, d); row["state"] != "active" || row["rotation"] != nil || row["replaces"] != c || row["secret_pickup"] != pickupOpen {
		t.Errorf("revoking the old credential changed the new one %+v", row)
	}
	e, _, _ := f.issued(root, svc, "30")
	g, gRef, _ := f.rotated(root, e, map[string]string{"days": "30", "overlap_minutes": "60"})
	f.run(root, "service.suspend", svc, nil)
	f.code(t, "the new secret after suspension", f.collect(root, gRef), 403, string(commands.PermissionDenied))
	f.code(t, "an earlier new secret after suspension", f.collect(root, dRef), 403, string(commands.PermissionDenied))
	f.run(root, "service.reinstate", svc, nil)
	if row := f.credentialRowOf(root, svc, e); row["not_after"] != end {
		t.Errorf("reinstating moved the old end %+v", row)
	}
	f.code(t, "the new secret after reinstating", f.collect(root, gRef), 403, string(commands.PermissionDenied))
	if row := f.credentialRowOf(root, svc, g); row["secret_pickup"] != pickupNever || row["usable"] != true {
		t.Errorf("the new credential after suspend and reinstate %+v", row)
	}
}

func TestCredentialOverlapSettings(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.Access.CredentialOverlap, c.Access.CredentialOverlapMax = 30, 60 })
	root := f.as(AdminLogin)
	limits := f.servicesRead(root).body["limits"].(map[string]any)
	if fmt.Sprint(limits["credential_overlap_minutes"]) != "30" || fmt.Sprint(limits["credential_overlap_max_minutes"]) != "60" {
		t.Fatalf("limits %+v", limits)
	}
	svc := f.newService(root, "settings")
	old, _, _ := f.issued(root, svc, "30")
	f.failed(t, "above the configured maximum", f.run(root, "service.rotate_credential", old, map[string]string{"days": "1", "overlap_minutes": "61"}), commands.InvalidRequest, "overlap_minutes")
	if _, _, got := f.rotated(root, old, map[string]string{"days": "1"}); got.body["fields"].(map[string]any)["replaced_until"] != stamp(start.Add(30*time.Minute)) {
		t.Errorf("the configured default was not applied %+v", got.body["fields"])
	}
	settings := DefaultAccessSettings()
	if settings.CredentialOverlap != 1440 || settings.CredentialOverlapMax != 10080 || settings.validate() != nil {
		t.Fatalf("defaults %d %d %v", settings.CredentialOverlap, settings.CredentialOverlapMax, settings.validate())
	}
	for name, change := range map[string]func(*AccessSettings){
		"a default above the maximum": func(s *AccessSettings) { s.CredentialOverlap, s.CredentialOverlapMax = 120, 60 },
		"a default below the pickup":  func(s *AccessSettings) { s.CredentialOverlap = 9 },
		"a maximum above the bound":   func(s *AccessSettings) { s.CredentialOverlapMax = 43201 },
	} {
		s := DefaultAccessSettings()
		change(&s)
		if s.validate() == nil {
			t.Errorf("%s is accepted", name)
		}
	}
	if list := CredentialSettingList(); len(list) != 2 || list[0].Name != "credential_overlap_minutes" || list[1].Flag != "credential-overlap-max-minutes" {
		t.Errorf("setting list %+v", list)
	}
	short := setupWith(t, func(c *Config) { c.Identity.CredentialValidity = 2 * 24 * time.Hour })
	limits = short.servicesRead(short.as(AdminLogin)).body["limits"].(map[string]any)
	if fmt.Sprint(limits["credential_overlap_max_minutes"]) != "2880" || fmt.Sprint(limits["credential_overlap_minutes"]) != "1440" {
		t.Errorf("the maximum is not held to the credential validity %+v", limits)
	}
	brief := setupWith(t, func(c *Config) { c.Identity.CredentialValidity = 12 * time.Hour })
	limits = brief.servicesRead(brief.as(AdminLogin)).body["limits"].(map[string]any)
	if fmt.Sprint(limits["credential_overlap_max_minutes"]) != "720" || fmt.Sprint(limits["credential_overlap_minutes"]) != "720" {
		t.Errorf("the default is not held to a maximum below it %+v", limits)
	}
}
