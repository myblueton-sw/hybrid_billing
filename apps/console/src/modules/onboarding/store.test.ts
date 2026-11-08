import { test } from "node:test";
import assert from "node:assert/strict";
import {
  createStore,
  draftKey,
  canStep,
  resumeTarget,
  blockedProfile,
  configurationReady,
} from "./store";
import { getProfile } from "../../profiles";

function fakeTimer() {
  const queue: (() => void)[] = [];
  return {
    timer: (fn: () => void) => queue.push(fn),
    flush: () => queue.splice(0).forEach((fn) => fn()),
  };
}
function toConnectors(store: ReturnType<typeof createStore>, key: string) {
  store.example(key, 1);
  assert.equal(store.submit(key, 1), "setup-attribution");
  store.example(key, 2);
  assert.equal(store.submit(key, 2), "setup-connectors");
  store.example(key, 3);
}
function toReadtest(store: ReturnType<typeof createStore>, key: string) {
  toConnectors(store, key);
  assert.equal(store.reviewPlan(key), "ok");
  store.setPlanAck(key, true);
  assert.equal(store.connectionNext(key), "setup-readtest");
}

test("draft key normalizes scope like the shell and separates modes", () => {
  assert.equal(draftKey("sample-b", "synthetic"), "sample-b:synthetic");
  assert.equal(draftKey("x", "synthetic"), "sample-a:synthetic");
  assert.equal(draftKey(null, "unconnected"), "sample-a:unconnected");
});

test("drafts are isolated by scope and mode", () => {
  const s = createStore(fakeTimer().timer);
  s.example("sample-a:synthetic", 1);
  assert.equal(s.get("sample-b:synthetic").values.entity, undefined);
  assert.equal(s.get("sample-a:unconnected").values.entity, undefined);
});

test("gating requires the previous step and resume finds the first gap", () => {
  const s = createStore(fakeTimer().timer),
    k = "sample-a:synthetic",
    d = s.get(k);
  assert(canStep(d, 1) && !canStep(d, 2));
  assert.equal(s.submit(k, 2), null);
  assert.equal(s.submit(k, 1), null);
  s.example(k, 1);
  s.submit(k, 1);
  assert(canStep(d, 2));
  assert.equal(resumeTarget(d), "setup-attribution");
});

test("input invalidates the current and later steps", () => {
  const s = createStore(fakeTimer().timer),
    k = "sample-a:synthetic";
  toConnectors(s, k);
  s.setField(k, 1, "entity", "Changed LLC");
  const d = s.get(k);
  assert(!d.done.has("setup-profile") && !d.done.has("setup-attribution"));
});

test("AWS account must be 12 digits and new route skips provider fields", () => {
  const s = createStore(fakeTimer().timer),
    k = "sample-a:synthetic";
  toConnectors(s, k);
  const v = s.get(k).values;
  assert(configurationReady(v));
  s.setField(k, 3, "account", "12345");
  assert(!configurationReady(v));
  s.setField(k, 3, "account", "123456789012");
  s.setField(k, 3, "route", "new");
  s.setField(k, 3, "role", "");
  assert(configurationReady(v));
});

test("provider change resets provider-specific values", () => {
  const s = createStore(fakeTimer().timer),
    k = "sample-a:synthetic";
  toConnectors(s, k);
  s.setField(k, 3, "provider", "azure");
  const v = s.get(k).values;
  assert.equal(v.account, "");
  assert.equal(v.identity, "");
  assert.equal(v.role, undefined);
  assert.equal(v.route, "existing");
});

test("blocked profiles never pass plan review", () => {
  const cases: Record<string, string | boolean>[] = [
    { provider: "naver" },
    { provider: "azure", agreement: "MOSP" },
    { provider: "gcp", preview: true, location: "ASIA" },
    { provider: "gcp", preview: false, location: "US" },
    { provider: "alibaba", preview: false },
    { provider: "ibm", ibmRoute: "api" },
  ];
  for (const c of cases) assert(blockedProfile(c), JSON.stringify(c));
  assert(!blockedProfile({ provider: "azure", agreement: "EA" }));
  const s = createStore(fakeTimer().timer),
    k = "sample-a:synthetic";
  toConnectors(s, k);
  s.setField(k, 3, "provider", "kt");
  assert.equal(s.reviewPlan(k), "blocked");
});

test("new configuration requires administrator read-back", () => {
  const s = createStore(fakeTimer().timer),
    k = "sample-a:synthetic";
  toConnectors(s, k);
  s.setField(k, 3, "route", "new");
  assert.equal(s.reviewPlan(k), "ok");
  s.setPlanAck(k, true);
  assert.equal(s.connectionNext(k), null);
  s.adminReadback(k);
  assert.equal(s.connectionNext(k), "setup-readtest");
});

test("plan acknowledgment toggling invalidates and restores the plan", () => {
  const s = createStore(fakeTimer().timer),
    k = "sample-a:synthetic";
  toReadtest(s, k);
  s.setPlanAck(k, false);
  const d = s.get(k);
  assert(d.plan && !d.values.planAck && !d.done.has("setup-connectors"));
});

test("diagnostic scenarios, stale runs and scenario invalidation", () => {
  const t = fakeTimer(),
    s = createStore(t.timer),
    k = "sample-a:synthetic";
  toReadtest(s, k);
  const d = s.get(k);
  s.setScenario(k, "denied");
  s.runDiagnostic(k);
  assert.equal(d.diagnostic, "running");
  t.flush();
  assert.equal(d.diagnostic, "failed");
  s.setScenario(k, "outcome");
  assert.equal(d.diagnostic, "idle");
  s.runDiagnostic(k);
  s.setScenario(k, "normal");
  t.flush();
  assert.equal(d.diagnostic, "idle");
  s.runDiagnostic(k);
  t.flush();
  assert.equal(d.diagnostic, "pass");
  assert.equal(s.next(k, 4), "setup-schedule");
});

test("partial scenario passes diagnostics but holds the receipt", () => {
  const t = fakeTimer(),
    s = createStore(t.timer),
    k = "sample-a:synthetic";
  toReadtest(s, k);
  s.setScenario(k, "partial");
  s.runDiagnostic(k);
  t.flush();
  s.next(k, 4);
  s.example(k, 5);
  assert.equal(s.submit(k, 5), "setup-first");
  s.validateReceipt(k);
  t.flush();
  const d = s.get(k);
  assert.equal(d.diagnostic, "pass");
  assert.equal(d.receipt, "partial");
  assert.equal(s.next(k, 6), null);
});

test("receipt requires a diagnostic pass", () => {
  const s = createStore(fakeTimer().timer),
    k = "sample-a:synthetic";
  s.validateReceipt(k);
  assert.equal(s.get(k).receipt, "idle");
});

test("dispose cancels running checks without leaving them stuck", () => {
  const t = fakeTimer(),
    s = createStore(t.timer),
    k = "sample-a:synthetic";
  toReadtest(s, k);
  s.runDiagnostic(k);
  s.dispose();
  const d = s.get(k);
  assert.equal(d.diagnostic, "idle");
  t.flush();
  assert.equal(d.diagnostic, "idle");
});

test("both installation profiles register onboarding", () => {
  for (const id of ["standard", "evidence"])
    assert(getProfile(id).modules.includes("onboarding"));
});

test("cadence must match the provider's allowed modes", () => {
  const s = createStore(fakeTimer().timer),
    k = "sample-a:synthetic";
  toReadtest(s, k);
  const d = s.get(k);
  d.done.add("setup-readtest");
  s.example(k, 5);
  assert.equal(d.values.cadence, "daily");
  d.values.provider = "ibm";
  assert.equal(s.submit(k, 5), null);
  d.values.cadence = "once";
  assert.equal(s.submit(k, 5), "setup-first");
});

test("input invalidation also clears dependent check results", () => {
  const t = fakeTimer(),
    s = createStore(t.timer),
    k = "sample-a:synthetic";
  toReadtest(s, k);
  s.runDiagnostic(k);
  t.flush();
  const d = s.get(k);
  assert.equal(d.diagnostic, "pass");
  s.setField(k, 3, "account", "210987654321");
  assert.equal(d.diagnostic, "idle");
  assert.equal(d.receipt, "idle");
  assert(!d.plan);
});
