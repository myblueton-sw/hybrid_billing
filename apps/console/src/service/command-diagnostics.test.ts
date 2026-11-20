import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, type AdminClient } from "./admin";
import { observeCommands, type CommandDiagnostic } from "./command-diagnostics";

function deferred() {
  let resolve!: (v: never) => void;
  let reject!: (e: unknown) => void;
  const promise = new Promise<never>((a, b) => { resolve = a; reject = b; });
  return { promise, resolve, reject };
}
const args = ["integrations.plugin_stop", "plugin:sample", "r1", "PRIVATE reason", { secret: "PRIVATE value" }, { request: "client-330", key: "PRIVATE key" }] as Parameters<AdminClient["command"]>;

test("closed-form diagnostics preserve response and client identities without request values", async () => {
  const error = new AdminError("refused", [], 0, "", "prerequisite_failed", "server-330");
  const api = { command: async () => { throw error; } } as unknown as AdminClient;
  const seen: Array<CommandDiagnostic | null> = [];
  const wrapped = observeCommands(api, (v) => seen.push(v), () => true);
  await assert.rejects(wrapped.command(...args), (e) => e === error);
  assert.deepEqual(seen, [null, { code: "prerequisite_failed", request: "server-330", client: "client-330", target: "plugin:sample", cancelled: false }]);
  assert.ok(!JSON.stringify(seen).includes("PRIVATE"));
});

test("late older errors and success cannot replace a newer attempt, and obsolete scopes publish nothing", async () => {
  const first = deferred(); const second = deferred();
  let calls = 0; let current = true;
  const api = { command: () => (++calls === 1 ? first.promise : second.promise) } as unknown as AdminClient;
  const seen: Array<CommandDiagnostic | null> = [];
  const wrapped = observeCommands(api, (v) => seen.push(v), () => current);
  const a = wrapped.command(...args).catch((e) => e);
  const b = wrapped.command(...args).catch((e) => e);
  first.reject(new AdminError("refused", [], 0, "", "prerequisite_failed", "old")); await a;
  assert.deepEqual(seen, [null, null]);
  current = false;
  second.reject(new AdminError("refused", [], 0, "", "prerequisite_failed", "other-scope")); await b;
  assert.deepEqual(seen, [null, null]);
});

test("transport errors contain no invented server metadata and cancellation is marked local", async () => {
  for (const error of [new Error("PRIVATE stack"), new AdminError("reauth_cancelled", [], 0, "", "reauthentication_required", "challenge-330")]) {
    let got: CommandDiagnostic | null = null;
    const api = { command: async () => { throw error; } } as unknown as AdminClient;
    await assert.rejects(observeCommands(api, (v) => { got = v; }, () => true).command(...args));
    assert.deepEqual(got, { code: error instanceof AdminError ? error.code : "", request: error instanceof AdminError ? error.request : "", client: "client-330", target: "plugin:sample", cancelled: error instanceof AdminError && error.failure === "reauth_cancelled" });
  }
});

test("an older success cannot erase a later request failure", async () => {
  const first = deferred();
  let calls = 0;
  const failure = new AdminError("refused", [], 0, "", "prerequisite_failed", "newer-330");
  const api = { command: () => ++calls === 1 ? first.promise : Promise.reject(failure) } as unknown as AdminClient;
  const seen: Array<CommandDiagnostic | null> = [];
  const wrapped = observeCommands(api, (v) => seen.push(v), () => true);
  const old = wrapped.command(...args);
  await assert.rejects(wrapped.command(...args), (e) => e === failure);
  first.resolve({ state: "stopped" } as never);
  await old;
  assert.equal(seen.length, 3);
  assert.equal(seen[2]?.request, "newer-330");
});
