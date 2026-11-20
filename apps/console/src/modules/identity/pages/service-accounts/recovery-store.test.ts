import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, type Outcome, type Secret } from "../../../../service/admin";
import { issueAttempts } from "./issue-attempts";
import { leaveDeliberately, stayAgain } from "../../../../shared/unload-intent";

const credential = "cred-0123456789abcdef";
const secret: Secret = { credential, secret: "hbsk_" + "A".repeat(43) };
const outcome: Outcome = { status: "completed", state: "credential_issued", job_id: "", fields: { credential, secret_ref: "R".repeat(22) } };
const pickup = { credential, ref: "R".repeat(22) };
const values = { days: "1", reason: "Original reason" };
const failure = (name: string) => (e: unknown) => e instanceof AdminError && e.failure === name;
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
function setup() {
  const gate = { current: true, visible: true };
  const attached = new Map<string, EventListenerOrEventListenerObject>();
  const store = issueAttempts(() => ({ request: "request", key: "key" }), { mono: () => 100, wall: () => Date.UTC(2026, 10, 20) }, {
    addEventListener: (name: string, listener: EventListenerOrEventListenerObject) => attached.set(name, listener), removeEventListener: (name: string) => attached.delete(name),
  }, () => gate.current, () => gate.visible);
  const issue = async () => { assert.equal((await store.submit("u", "svc-a", values, "rev-1", 300, async () => outcome)).kind, "issued"); };
  return { store, gate, attached, issue };
}

test("construction is inert and disposal is terminal, idempotent and unsubscribed", async () => {
  const { store, attached, issue } = setup();
  assert.equal(store.current(), false);
  assert.equal((await store.submit("u", "svc-a", values, "rev-1", 300, async () => outcome)).kind, "refused");
  store.mark("u", credential, "svc-a");
  assert.equal(attached.size, 0);
  leaveDeliberately();
  assert.equal(store.disposed(), false, "inert construction registered a global leave listener");
  stayAgain();
  assert.equal(store.activate(), true);
  await issue();
  assert.equal(attached.size, 1);
  store.dispose(); store.dispose();
  assert.equal(attached.size, 0);
  assert.equal(store.activate(), false);
  assert.equal(store.open("u", "svc-a"), null);
  assert.deepEqual(store.list("u"), []);
  store.mark("u", credential, "svc-a");
  assert.deepEqual(store.marks("u"), []);
  let notified = false;
  store.subscribe(() => { notified = true; });
  store.release();
  assert.equal(notified, false);
});

test("authoritative scope loss fences captured stores before cleanup and late issuance", async () => {
  const { store, gate, issue } = setup();
  store.activate();
  await issue();
  gate.current = false;
  assert.equal(store.open("u", "svc-a"), null);
  assert.equal(store.get("u", "svc-a"), undefined);
  assert.deepEqual(store.list("u"), []);
  store.mark("u", credential, "svc-a");
  assert.deepEqual(store.marks("u"), []);
  assert.equal(store.readStarted(), 0);
  assert.equal(store.discard("u", "svc-a"), false);
  let calls = 0;
  await assert.rejects(store.collect("u", "svc-a", pickup, async () => { calls++; return secret; }), failure("unauthenticated"));
  assert.equal(calls, 0);
  store.dispose();
  const next = setup(); next.store.activate();
  const wait = deferred<Outcome>();
  const pending = next.store.submit("u", "svc-a", values, "rev-1", 300, () => wait.promise);
  next.store.dispose(); wait.resolve(outcome);
  assert.equal((await pending).kind, "refused");
  assert.equal(next.attached.size, 0);
  const beforeCleanup = setup(); beforeCleanup.store.activate();
  const late = deferred<Outcome>();
  const inFlight = beforeCleanup.store.submit("u", "svc-a", values, "rev-1", 300, () => late.promise);
  beforeCleanup.gate.current = false;
  late.resolve(outcome);
  assert.equal((await inFlight).kind, "refused", "scope loss before cleanup published late issuance");
  beforeCleanup.store.dispose();
});

test("unknown issuance retry supplies immutable original values to the actual executor", async () => {
  const { store } = setup(); store.activate();
  const first = { ...values };
  await store.submit("u", "svc-a", first, "rev-1", 300, async () => { throw new AdminError("busy"); });
  first.days = "9"; first.reason = "Mutated";
  let sent: unknown;
  await store.submit("u", "svc-a", first, "rev-2", 300, async (attempt, expected, frozen) => {
    sent = { attempt, expected, values: frozen };
    assert.equal(Object.isFrozen(frozen), true);
    return outcome;
  });
  assert.deepEqual(sent, { attempt: { request: "request", key: "key" }, expected: "rev-1", values });
  store.dispose();
});

test("two observers share one pending consume and keep only canonical unstored metadata", async () => {
  const { store, issue } = setup(); store.activate(); await issue();
  const wait = deferred<Secret>(); let calls = 0;
  const collect = async (reference: string) => { assert.equal(reference, "R".repeat(22)); calls++; return wait.promise; };
  const first = store.collect("u", "svc-a", pickup, collect);
  const replay = store.collect("u", "svc-a", pickup, collect);
  assert.equal(first, replay);
  await Promise.resolve(); assert.equal(calls, 1);
  assert.equal(store.collectionPhase("u", credential), "pending");
  wait.resolve(secret);
  assert.deepEqual(await first, secret);
  assert.equal(store.collectionPhase("u", credential), "idle");
  assert.equal(store.open("u", "svc-a"), null);
  assert.deepEqual(store.marks("u"), [{ service: "svc-a", credential }]);
  assert.equal(JSON.stringify(store.list("u")).includes(secret.secret), false);
  store.unmark("u", credential);
  store.observe("u", { service: "svc-a", id: credential, current_state: "active", secret_pickup: "collected" }, store.readStarted());
  assert.deepEqual(store.marks("u"), []);
  await assert.rejects(store.collect("u", "svc-a", pickup, collect), failure("unavailable"));
  assert.equal(calls, 1);
  store.dispose();
});

test("unknown or malformed collection retains reference and requires explicit retry", async () => {
  for (const answer of [new AdminError("unavailable"), { ...secret, credential: "cred-0000000000000002" }, { ...secret, secret: "malformed" }]) {
    const { store, issue } = setup(); store.activate(); await issue();
    let calls = 0;
    const collect = async () => { calls++; if (answer instanceof Error) throw answer; return answer; };
    await assert.rejects(store.collect("u", "svc-a", pickup, collect), failure("unavailable"));
    assert.equal(store.collectionPhase("u", credential), "unknown");
    assert.equal(store.open("u", "svc-a")?.credential, credential);
    await assert.rejects(store.collect("u", "svc-a", pickup, collect), failure("unavailable"));
    assert.equal(calls, 1, "remount/observer automatically retried a consume");
    await store.collect("u", "svc-a", pickup, async () => { calls++; return secret; }, true);
    assert.equal(calls, 2);
    store.dispose();
  }
});

test("late or revoked collection cannot publish secret or recreate an acknowledged marker", async () => {
  for (const mode of ["scope", "dispose", "revoked", "acknowledged"] as const) {
    const { store, gate, issue } = setup(); store.activate(); await issue();
    const wait = deferred<Secret>();
    const pending = store.collect("u", "svc-a", pickup, () => wait.promise);
    await Promise.resolve();
    if (mode === "scope") gate.current = false;
    if (mode === "dispose") store.dispose();
    if (mode === "revoked") store.revoked("u", credential);
    if (mode === "acknowledged") {
      store.observe("u", { service: "svc-a", id: credential, current_state: "active", secret_pickup: "collected" }, store.readStarted());
      store.unmark("u", credential);
    }
    wait.resolve(secret);
    if (mode === "acknowledged") await pending;
    else await assert.rejects(pending, failure("unauthenticated"));
    assert.deepEqual(store.marks("u"), []);
    store.dispose();
  }
});

test("hidden initial or deferred dispatch never consumes or invents unknown delivery", async () => {
  for (const timing of ["initial", "deferred"] as const) {
    const { store, gate, issue } = setup(); store.activate(); await issue();
    let calls = 0;
    if (timing === "initial") gate.visible = false;
    const pending = store.collect("u", "svc-a", pickup, async () => { calls++; return secret; });
    if (timing === "deferred") gate.visible = false;
    await assert.rejects(pending, failure("unavailable"));
    assert.equal(calls, 0, "hidden transport consumed the one-time secret");
    assert.equal(store.collectionPhase("u", credential), "idle");
    assert.equal(store.open("u", "svc-a")?.ref, pickup.ref);
    gate.visible = true;
    await store.collect("u", "svc-a", pickup, async () => { calls++; return secret; });
    assert.equal(calls, 1);
    store.dispose();
  }
});

test("explicit release cancels pending collection before late success", async () => {
  const { store, issue } = setup(); store.activate(); await issue();
  const wait = deferred<Secret>();
  const pending = store.collect("u", "svc-a", pickup, () => wait.promise);
  await Promise.resolve();
  store.release("u");
  wait.resolve(secret);
  await assert.rejects(pending, failure("unauthenticated"));
  assert.deepEqual(store.marks("u"), []);
  store.dispose();
});

test("collection requires the exact retained reference before and during a pending read", async () => {
  const { store, issue } = setup(); store.activate(); await issue();
  let calls = 0;
  const wrong = { ...pickup, ref: "W".repeat(22) };
  await assert.rejects(store.collect("u", "svc-a", wrong, async () => { calls++; return secret; }), failure("unavailable"));
  assert.equal(calls, 0);
  const wait = deferred<Secret>();
  const pending = store.collect("u", "svc-a", pickup, async () => { calls++; return wait.promise; });
  await assert.rejects(store.collect("u", "svc-a", wrong, async () => { calls++; return secret; }), failure("unavailable"));
  assert.equal(calls, 1);
  wait.resolve(secret); await pending;
  store.dispose();
});

test("per-user release preserves another user's pending operation and canonical marker", async () => {
  const { store, issue } = setup(); store.activate(); await issue();
  const other = { credential: "cred-0000000000000002", ref: "S".repeat(22) };
  await store.submit("v", "svc-b", values, "rev-1", 300, async () => ({ ...outcome, fields: { credential: other.credential, secret_ref: other.ref } }));
  const first = deferred<Secret>(); const second = deferred<Secret>();
  const a = store.collect("u", "svc-a", pickup, () => first.promise);
  const b = store.collect("v", "svc-b", other, () => second.promise);
  await Promise.resolve();
  store.release("u");
  first.resolve(secret); second.resolve({ ...secret, credential: other.credential });
  await assert.rejects(a, failure("unauthenticated")); await b;
  assert.deepEqual(store.marks("u"), []);
  assert.deepEqual(store.marks("v"), [{ credential: other.credential, service: "svc-b" }]);
  store.dispose();
});
