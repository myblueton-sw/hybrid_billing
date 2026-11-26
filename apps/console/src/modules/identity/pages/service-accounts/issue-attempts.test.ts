import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, type Attempt, type Outcome } from "../../../../service/admin";
import { leaveDeliberately, stayAgain } from "../../../../shared/unload-intent";
import { issueAttempts as createIssueAttempts, type Exec } from "./issue-attempts";

function issueAttempts(...args: Parameters<typeof createIssueAttempts>) {
  const store = createIssueAttempts(...args);
  store.activate();
  return store;
}

const counter = () => {
  let n = 0;
  return (): Attempt => { n += 1; return { request: `r${n}`, key: `k${n}` }; };
};

function clocks() {
  const now = { mono: 1000, wall: 1_800_000_000_000 };
  return { now, clocks: { mono: () => now.mono, wall: () => now.wall } };
}

function target() {
  const added: string[] = [];
  const removed: string[] = [];
  let handler: ((e: Event) => void) | null = null;
  return {
    added, removed,
    fire() {
      const e = new Event("beforeunload", { cancelable: true });
      handler?.(e);
      return e.defaultPrevented;
    },
    t: {
      addEventListener: (type: string, fn: EventListenerOrEventListenerObject) => { added.push(type); handler = fn as (e: Event) => void; },
      removeEventListener: (type: string) => { removed.push(type); handler = null; },
    } as unknown as Pick<Window, "addEventListener" | "removeEventListener">,
  };
}

const ok = (credential = "cred-0123456789abcdef", ref = "R".repeat(22)): Outcome => ({ status: "completed", state: "credential_issued", job_id: "", fields: { credential, secret_ref: ref } });
const fail = (failure: string, code = "") => new AdminError(failure as AdminError["failure"], [], 0, "", code);

function scripted(answers: (Outcome | Error)[]) {
  const sent: { attempt: Attempt; expected: string }[] = [];
  const exec: Exec = async (attempt, expected) => {
    sent.push({ attempt, expected });
    const next = answers.shift();
    if (!next || next instanceof Error) throw next ?? fail("unavailable");
    return next;
  };
  return { sent, exec };
}

function deferred() {
  let resolve!: (o: Outcome) => void;
  let reject!: (e: unknown) => void;
  const promise = new Promise<Outcome>((y, n) => { resolve = y; reject = n; });
  return { promise, resolve, reject };
}

const values = { days: "1", reason: "why" };

const observed = { service: "svc-a", id: "cred-0123456789abcdef", current_state: "active" as const, secret_pickup: "open" as const };

test("incomplete pages and not-open observations preserve the issued reference", async () => {
  for (const row of [null, { ...observed, secret_pickup: "never_collected" as const }]) {
    const store = issueAttempts(counter(), clocks().clocks);
    await store.submit("u", "svc-a", values, "rev-1", 3600, scripted([ok()]).exec);
    store.observe("u", row, store.readStarted());
    assert.equal(store.get("u", "svc-a")?.issued?.ref, "R".repeat(22), "page absence or current not-open state is not terminal evidence");
  }
});

test("an unknown outcome keeps the attempt, values and expected revision across a remount, and the retry resends them", async () => {
  const { clocks: c } = clocks();
  const store = issueAttempts(counter(), c);
  const { sent, exec } = scripted([fail("busy"), ok()]);
  assert.deepEqual(await store.submit("u", "svc-a", values, "rev-1", 3600, exec), { kind: "unknown", error: fail("busy") });
  const again = store.get("u", "svc-a");
  assert.equal(again?.phase, "unknown");
  assert.equal(again?.days, "1");
  assert.equal(again?.reason, "why");
  const result = await store.submit("u", "svc-a", { days: "9", reason: "changed" }, "rev-7", 3600, exec);
  assert.equal(result.kind, "issued");
  assert.deepEqual(sent[1], sent[0]);
  assert.equal(sent[1].expected, "rev-1");
  assert.equal(store.get("u", "svc-a")?.phase, "issued");
});

test("a late issuance remains recorded and opening Reveal does not consume the reference (F6)", async () => {
  const { clocks: c } = clocks();
  const store = issueAttempts(counter(), c);
  const d = deferred();
  const pending = store.submit("u", "svc-a", values, "rev-1", 3600, () => d.promise);
  assert.equal(store.get("u", "svc-a")?.phase, "sending");
  assert.deepEqual(await store.submit("u", "svc-a", values, "rev-1", 3600, () => d.promise), { kind: "refused" });
  assert.equal(store.discard("u", "svc-a"), false);
  d.resolve(ok());
  await pending;
  const e = store.get("u", "svc-a");
  assert.equal(e?.phase, "issued");
  assert.equal(e?.issued?.ref, "R".repeat(22));
  assert.deepEqual(await store.submit("u", "svc-a", values, "rev-1", 3600, () => d.promise), { kind: "refused" });
  assert.equal(store.discard("u", "svc-a"), false);
  assert.equal(store.open("u", "svc-a")?.credential, "cred-0123456789abcdef");
  assert.equal(store.get("u", "svc-a")?.phase, "issued");
  await store.submit("u", "svc-b", values, "rev-1", 3600, scripted([ok("cred-00000000000000bb")]).exec);
  store.revoked("u", "cred-00000000000000bb");
  assert.equal(store.get("u", "svc-b"), undefined);
});

test("a malformed success is treated as an unknown outcome", async () => {
  const store = issueAttempts(counter(), clocks().clocks);
  const result = await store.submit("u", "svc-a", values, "rev-1", 3600, scripted([ok("bad", "x")]).exec);
  assert.equal(result.kind, "unknown");
  assert.equal(store.get("u", "svc-a")?.phase, "unknown");
});

test("definitive answers remove the entry and record the refusal; retries are marked (F2)", async () => {
  const store = issueAttempts(counter(), clocks().clocks);
  const first = scripted([fail("conflict", "revision_conflict")]);
  assert.equal((await store.submit("u", "svc-a", values, "rev-1", 3600, first.exec)).kind, "removed");
  assert.equal(store.get("u", "svc-a"), undefined);
  assert.equal(store.refusal("u", "svc-a")?.retry, false);
  const retried = scripted([fail("busy"), fail("refused")]);
  await store.submit("u", "svc-a", values, "rev-1", 3600, retried.exec);
  assert.equal(store.refusal("u", "svc-a"), undefined);
  assert.equal((await store.submit("u", "svc-a", values, "rev-1", 3600, retried.exec)).kind, "removed");
  assert.equal(store.refusal("u", "svc-a")?.retry, true);
  const next = scripted([ok()]);
  await store.submit("u", "svc-a", values, "rev-2", 3600, next.exec);
  assert.notDeepEqual(next.sent[0].attempt, retried.sent[0].attempt);
});

test("ambiguous and expired answers lapse instead of claiming nothing was issued (F2)", async () => {
  for (const [answers, label] of [
    [[fail("stale", "idempotency_conflict")], "idempotency_conflict on a first send"],
    [[fail("busy"), fail("stale", "idempotency_key_expired")], "expired key on a retry"],
    [[fail("busy"), fail("partial", "internal_error")], "partial on a retry"],
    [[fail("busy"), fail("denied")], "denied on a retry"],
  ] as [Error[], string][]) {
    const store = issueAttempts(counter(), clocks().clocks);
    const count = answers.length;
    const s = scripted(answers);
    for (let i = 0; i < count; i++) await store.submit("u", "svc-a", values, "rev-1", 3600, s.exec);
    assert.equal(store.get("u", "svc-a")?.phase, "lapsed", label);
    assert.deepEqual(await store.submit("u", "svc-a", values, "rev-1", 3600, s.exec), { kind: "refused" }, label);
    assert.equal(store.discard("u", "svc-a"), true, label);
  }
});

test("limited, unauthenticated and a cancelled password confirmation keep a held retry unknown", async () => {
  for (const f of [fail("limited"), fail("unauthenticated"), fail("reauth_cancelled"), fail("reauth", "reauthentication_required")]) {
    const store = issueAttempts(counter(), clocks().clocks);
    const s = scripted([fail("busy"), f]);
    await store.submit("u", "svc-a", values, "rev-1", 3600, s.exec);
    await store.submit("u", "svc-a", values, "rev-1", 3600, s.exec);
    assert.equal(store.get("u", "svc-a")?.phase, "unknown", f.failure);
  }
  const store = issueAttempts(counter(), clocks().clocks);
  assert.equal((await store.submit("u", "svc-a", values, "rev-1", 3600, scripted([fail("reauth_cancelled")]).exec)).kind, "removed");
});

test("the wall clock counts even when the monotonic clock stood still (N1)", async () => {
  const { now, clocks: c } = clocks();
  const store = issueAttempts(counter(), c);
  const s = scripted([fail("busy")]);
  await store.submit("u", "svc-a", values, "rev-1", 3600, s.exec);
  now.wall += 3540 * 1000;
  assert.deepEqual(await store.submit("u", "svc-a", values, "rev-1", 3600, s.exec), { kind: "lapsed" });
  assert.equal(s.sent.length, 1);
  const back = clocks();
  const other = issueAttempts(counter(), back.clocks);
  const t = scripted([fail("busy")]);
  await other.submit("u", "svc-a", values, "rev-1", 3600, t.exec);
  back.now.wall -= 86400000;
  back.now.mono += 3540 * 1000;
  assert.deepEqual(await other.submit("u", "svc-a", values, "rev-1", 3600, t.exec), { kind: "lapsed" });
  assert.equal(t.sent.length, 1);
});

test("a refused retry near or after the window lapses, and an unknown window lapses it too", async () => {
  const { now, clocks: c } = clocks();
  const store = issueAttempts(counter(), c);
  const s = scripted([fail("busy"), fail("conflict")]);
  await store.submit("u", "svc-a", values, "rev-1", 3600, s.exec);
  const d = deferred();
  const retry = store.submit("u", "svc-a", values, "rev-1", 3600, () => d.promise);
  now.mono += 3600 * 1000;
  d.reject(fail("conflict"));
  assert.deepEqual(await retry, { kind: "lapsed" });
  const unknown = issueAttempts(counter(), clocks().clocks);
  const u = scripted([fail("busy"), fail("conflict")]);
  await unknown.submit("u", "svc-a", values, "rev-1", 0, u.exec);
  await unknown.submit("u", "svc-a", values, "rev-1", 0, u.exec);
  assert.equal(unknown.get("u", "svc-a")?.phase, "lapsed");
  assert.equal(u.sent.length, 2);
});

test("only post-issuance matching terminal observations retire the issued reference", async () => {
  const store = issueAttempts(counter(), clocks().clocks);
  const stale = store.readStarted();
  await store.submit("u", "svc-a", values, "rev-1", 3600, scripted([ok()]).exec);
  store.observe("u", { ...observed, secret_pickup: "collected" }, stale);
  assert.equal(store.get("u", "svc-a")?.phase, "issued");
  const fresh = store.readStarted();
  for (const row of [observed, { ...observed, id: "other" }, { ...observed, service: "svc-other" }, { ...observed, secret_pickup: undefined }]) store.observe("u", row, fresh);
  assert.equal(store.get("u", "svc-a")?.phase, "issued");
  for (const row of [{ ...observed, current_state: "revoked" as const }, { ...observed, current_state: "expired" as const }, { ...observed, secret_pickup: "collected" as const }]) {
    const s = issueAttempts(counter(), clocks().clocks);
    await s.submit("u", "svc-a", values, "rev-1", 3600, scripted([ok()]).exec);
    s.observe("u", row, s.readStarted());
    assert.equal(s.get("u", "svc-a"), undefined);
    if (row.secret_pickup === "collected") {
      assert.deepEqual(s.marks("u"), [{ service: "svc-a", credential: observed.id }]);
      s.unmark("u", observed.id);
      s.observe("u", row, s.readStarted());
      assert.deepEqual(s.marks("u"), []);
    }
  }
});

test("the unload guard follows sending, unknown, issued and unstored marks, and a deliberate leave releases it (F5)", async () => {
  stayAgain();
  const w = target();
  const store = issueAttempts(counter(), clocks().clocks, w.t);
  assert.equal(store.guarded(), false);
  const d = deferred();
  const pending = store.submit("u", "svc-a", values, "rev-1", 3600, () => d.promise);
  assert.equal(store.listening(), true);
  assert.equal(w.fire(), true);
  d.reject(fail("busy"));
  await pending;
  assert.equal(w.fire(), true);
  store.discard("u", "svc-a");
  assert.equal(store.listening(), false);
  await store.submit("u", "svc-a", values, "rev-1", 3600, scripted([fail("stale", "idempotency_key_expired")]).exec);
  assert.equal(store.get("u", "svc-a")?.phase, "lapsed");
  assert.equal(store.guarded(), false);
  store.mark("u", "cred-1", "svc-a");
  assert.equal(store.listening(), true);
  store.unmark("u", "cred-1");
  assert.equal(store.listening(), false);
  await store.submit("u", "svc-b", values, "rev-1", 3600, scripted([ok()]).exec);
  assert.equal(store.listening(), true);
  store.observe("u", null, 0);
  assert.equal(store.listening(), true, "a read that started earlier keeps the issued entry");
  store.mark("u", "cred-2", "svc-b");
  leaveDeliberately();
  assert.equal(w.fire(), false);
  assert.equal(store.guarded(), false);
  assert.equal(store.listening(), false);
  assert.deepEqual(store.list("u"), []);
  assert.deepEqual(store.marks("u"), []);
  assert.deepEqual(w.added, ["beforeunload", "beforeunload", "beforeunload", "beforeunload"]);
  stayAgain();
});

test("users are isolated", async () => {
  const store = issueAttempts(counter(), clocks().clocks);
  await store.submit("a", "svc-a", values, "rev-1", 3600, scripted([fail("busy")]).exec);
  assert.equal(store.get("b", "svc-a"), undefined);
  assert.deepEqual(store.list("b"), []);
  store.mark("a", "cred-1", "svc-a");
  assert.deepEqual(store.marks("b"), []);
  assert.equal(store.discard("b", "svc-a"), false);
  assert.equal(store.get("a", "svc-a")?.phase, "unknown");
});


test("the original issuance replay window survives metadata refresh", async () => {
 const { now, clocks: c } = clocks();
 const store = issueAttempts(counter(), c);
 const s = scripted([fail("busy"), ok()]);
 await store.submit("u", "svc-a", values, "rev-1", 10, s.exec);
 now.mono += 9000;
 assert.deepEqual(await store.submit("u", "svc-a", values, "rev-new", 3600, s.exec), { kind: "lapsed" });
 assert.equal(s.sent.length, 1);
});

test("issuance accepts only a completed credential-issued result", async () => {
  for (const patch of [{ status: "accepted" }, { state: "other" }, { job_id: "pending-job" }]) {
    const store = issueAttempts(counter(), clocks().clocks);
    const result = await store.submit("u", "svc-a", values, "rev-1", 300, async () => ({ ...ok(), ...patch }));
    assert.equal(result.kind, "unknown");
    assert.equal(store.open("u", "svc-a"), null);
  }
});

test("a held rotation remembers which credential and overlap it named, and an issue attempt carries none", async () => {
  const { clocks: c } = clocks();
  const store = issueAttempts(counter(), c);
  const { exec, sent } = scripted([fail("busy"), { ...ok(), state: "credential_rotated" }]);
  const rotate = { credential: "cred-00000000000000aa", overlap: "60" };
  const given = { ...values, rotate };
  assert.equal((await store.submit("u", "svc-a", given, "rev-1", 3600, exec)).kind, "unknown");
  rotate.overlap = "999";
  assert.deepEqual(store.get("u", "svc-a")?.rotate, { credential: "cred-00000000000000aa", overlap: "60" });
  assert.equal((await store.submit("u", "svc-a", { ...values, rotate: { credential: "cred-00000000000000bb", overlap: "10" } }, "rev-2", 3600, exec)).kind, "issued");
  assert.deepEqual(store.get("u", "svc-a")?.rotate, { credential: "cred-00000000000000aa", overlap: "60" });
  const plain = issueAttempts(counter(), c);
  await plain.submit("u", "svc-a", values, "rev-1", 3600, scripted([fail("busy")]).exec);
  assert.equal(plain.get("u", "svc-a")?.rotate, undefined);
  assert.equal(sent.length, 2);
  const mixed = issueAttempts(counter(), c);
  assert.equal((await mixed.submit("u", "svc-a", given, "rev-1", 3600, scripted([ok()]).exec)).kind, "unknown", "a rotation is not settled by an issue result");
  const issue = issueAttempts(counter(), c);
  assert.equal((await issue.submit("u", "svc-a", values, "rev-1", 3600, scripted([{ ...ok(), state: "credential_rotated" }]).exec)).kind, "unknown", "an issue is not settled by a rotation result");
  const seen: unknown[] = [];
  const frozen = issueAttempts(counter(), c);
  await frozen.submit("u", "svc-a", { ...values, rotate: { credential: "cred-00000000000000aa", overlap: "60" } }, "rev-1", 3600, async (_tried, _expected, v) => { seen.push(v.rotate); return { ...ok(), state: "credential_rotated" }; });
  assert.deepEqual(seen, [{ credential: "cred-00000000000000aa", overlap: "60" }]);
});
