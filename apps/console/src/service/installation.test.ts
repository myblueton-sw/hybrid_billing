import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, adminClient } from "./admin";
import { installationOf, previewOf, proposalOf } from "./installation";

type Call = { url: string; init: RequestInit };

function fake(status: number, body: unknown) {
  const calls: Call[] = [];
  const f = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    return new Response(JSON.stringify(body), { status });
  }) as unknown as typeof fetch;
  return { f, calls };
}

const hash = "a".repeat(64);

test("the installation read parses every part and keeps unknowns visible", async () => {
  const r = fake(200, {
    revision: "r7",
    now: "2026-11-17T09:00:00Z",
    state: "configured",
    site: { id: "sample-site", platform: "linux/amd64", probe: "p", config_overrides: [] },
    observation: { state: "fresh", observed_at: "2026-11-17T08:00:00Z", expires_at: "2026-11-18T08:00:00Z" },
    installed: { state: "not_installed", services: [], volumes: [], history: [], backups: [] },
    releases: [
      { version: "1.1.0", key_version: "k1", upgrade_from: ["1.0.0"], platforms: ["linux/amd64"], services: [{ name: "api", artifact: "api", ports: ["8443/tcp", 1], volumes: ["config"] }], volumes: [{ name: "ledger", data: true }, { name: "config", data: "no" }], artifacts: [{ name: "api", kind: "container_image" }], mark: null },
      null,
      { version: "1.0.0", upgrade_from: "x", mark: { by: "u1", reason: "bad" } },
      { version: "0.9.0" },
      { version: "0.8.0", mark: "bad" },
    ],
    preview: { state: "available", kind: "install", version: "1.0.0", hash, services: [{ kind: "service", target: "api", action: "create", conflict: "" }], ports: [], volumes: [], data_effects: [{ volume: "/v", effect: "new" }], permissions: ["run_service", 3], leftovers: [], conflicts: [] },
    upgrade: { state: "not_applicable", reason: "" },
    rollback: "bogus",
    proposals: [{ number: 5, target: "installation-proposal-5", kind: "destination", state: "pending", summary: { host: "h.example.invalid", port: 443, ranges: ["198.51.100.0/24"] }, can_approve: true, approve_blocked: "", can_withdraw: false, withdraw_blocked: "not_proposer" }],
    destinations: [{ id: "d1", host: "h", port: "443", ranges: ["192.0.2.0/24"] }],
    reports: [{ destination: "d1", kind: "tls", failure: "unknown_authority", address: "192.0.2.10", via: "", cause: "untrusted_issuer", steps: ["verify_chain", null] }],
    runs: "none",
  });
  const got = await adminClient("t", r.f).installation();
  assert.equal(r.calls[0].url, "/api/v1/installation");
  assert.equal(r.calls[0].init.method, "GET");
  assert.equal(got.state, "configured");
  assert.equal(got.site.id, "sample-site");
  assert.equal(got.releases.length, 4);
  assert.equal(got.releases[2].mark, "unknown");
  assert.equal(got.releases[3].mark, "unknown");
  assert.deepEqual(got.releases[0].services?.[0].ports, ["8443/tcp"]);
  assert.deepEqual(got.releases[0].volumes?.map((v) => v.data), [true, null]);
  assert.equal(got.releases[1].upgrade_from, null);
  assert.equal(got.releases[1].services, null);
  assert.deepEqual(got.releases[1].mark, { by: "u1", by_login: "", reason: "bad", confirmed_by: "", confirmed_by_login: "" });
  assert.equal(got.releases[0].mark, null);
  assert.equal(got.preview.hash, hash);
  assert.deepEqual(got.preview.permissions, ["run_service"]);
  assert.equal(got.upgrade.services, null);
  assert.equal(got.rollback.state, "");
  assert.equal(got.rollback.conflicts, null);
  assert.equal(got.destinations[0].port, null);
  assert.deepEqual(got.reports[0].steps, ["verify_chain"]);
  assert.equal(got.proposals[0].summary.port, 443);
  assert.equal(got.installed.history, 0);
});

test("a site that is not configured has empty parts, never zeros", () => {
  const got = installationOf({ revision: "r1", now: "2026-11-17T09:00:00Z", state: "not_configured" });
  assert.equal(got.state, "not_configured");
  assert.deepEqual([got.releases, got.proposals, got.destinations, got.reports], [[], [], [], []]);
  assert.equal(got.installed.history, null);
  assert.equal(got.installed.services, null);
  assert.equal(got.preview.conflicts, null);
  assert.equal(got.observation.state, "");
});

test("wrong types become unknown rather than plausible values", () => {
  const p = previewOf({ state: 3, hash: 5, services: "x", conflicts: [null, 1, { target: "t", conflict: "port_in_use" }] });
  assert.equal(p.state, "");
  assert.equal(p.hash, "");
  assert.equal(p.services, null);
  assert.equal(p.conflicts?.length, 1);
  const got = installationOf({ state: "configured", proposals: [{ number: "3" }, { number: 0 }, { number: 2.5 }, { number: -1 }] });
  assert.deepEqual(got.proposals.map((x) => x.number), [null, null, null, null]);
});

test("approve and withdraw hints fail closed", () => {
  const ok = proposalOf({ can_approve: true, approve_blocked: "", can_withdraw: true, withdraw_blocked: "" });
  assert.deepEqual([ok.can_approve, ok.approve_blocked, ok.can_withdraw, ok.withdraw_blocked], [true, "", true, ""]);
  const odd = proposalOf({ can_approve: true, approve_blocked: "conflicts", can_withdraw: "yes", withdraw_blocked: "" });
  assert.deepEqual([odd.can_approve, odd.approve_blocked, odd.can_withdraw, odd.withdraw_blocked], [false, "conflicts", false, "missing"]);
  const bare = proposalOf({});
  assert.deepEqual([bare.can_approve, bare.approve_blocked, bare.can_withdraw, bare.withdraw_blocked], [false, "missing", false, "missing"]);
  const blank = proposalOf({ can_approve: false, approve_blocked: "" });
  assert.equal(blank.approve_blocked, "missing");
});

test("commands send the envelope and resend with the same identity", async () => {
  const r = fake(200, { business_state: "applied", fields: { proposal: 5, target: "installation-proposal-5" } });
  const api = adminClient("csrf", r.f);
  const tried = { request: "m01-req", key: "m01-key" };
  const out = await api.command("installation.approve", "installation-proposal-5", "r7", "ok", {}, tried);
  assert.deepEqual({ state: out.state, fields: out.fields }, { state: "applied", fields: { proposal: "5", target: "installation-proposal-5" } });
  await api.command("installation.approve", "installation-proposal-5", "r7", "ok", {}, tried);
  assert.equal(r.calls[0].url, "/api/v1/commands/installation.approve");
  const first = JSON.parse(String(r.calls[0].init.body));
  const second = JSON.parse(String(r.calls[1].init.body));
  assert.deepEqual(first, { request_id: "m01-req", target_id: "installation-proposal-5", idempotency_key: "m01-key", expected_revision: "r7", rationale: "ok", body: {} });
  assert.deepEqual(second, first);
  assert.equal((r.calls[0].init.headers as Record<string, string>)["X-Request-Id"], "m01-req");
  await api.command("installation.propose_plan", "installation", "r7", "why", { version: "1.0.0", hash }, tried);
  assert.equal(r.calls[2].url, "/api/v1/commands/installation.propose_plan");
  assert.deepEqual(JSON.parse(String(r.calls[2].init.body)).body, { version: "1.0.0", hash });
});

test("refusals map to failures with fields", async () => {
  const cases: [number, Record<string, unknown>, string, string[]][] = [
    [422, { code: "invalid_request", fields: ["hash"] }, "invalid", ["hash"]],
    [409, { code: "invalid_stage" }, "stage", []],
    [409, { code: "prerequisite_failed" }, "refused", []],
    [403, { code: "permission_denied" }, "denied", []],
    [409, { code: "revision_conflict" }, "conflict", []],
    [503, { code: "dependency_unavailable" }, "busy", []],
  ];
  for (const [status, body, failure, fields] of cases) {
    try {
      await adminClient("t", fake(status, body).f).command("installation.withdraw", "installation-proposal-1", "r", "x", {}, { request: "a", key: "b" });
      assert.fail("expected a failure");
    } catch (e) {
      assert.ok(e instanceof AdminError);
      assert.equal(e.failure, failure);
      assert.deepEqual(e.fields, fields);
    }
  }
});
