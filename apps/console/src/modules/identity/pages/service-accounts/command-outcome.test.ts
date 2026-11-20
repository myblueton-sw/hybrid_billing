import { test } from "node:test";
import assert from "node:assert/strict";
import {
  AdminError,
  type AdminFailure,
  type Outcome,
} from "../../../../service/admin";
import { commandIntents, type CommandDraft } from "./command-intents";
import { serviceLimits } from "../../../../service/service-query-fixtures";
const draft: CommandDraft = {
  command: "service.suspend",
  target: "svc-a",
  reason: "Synthetic outcome history check",
  fields: {},
  revision: "r1",
  replay: 300,
};
const accepted: Outcome = {
  status: "completed",
  state: "service_suspended",
  revision: "r2",
  job_id: "",
  fields: {},
};
function fixture() {
  let time = 0,
    sequence = 0;
  const store = commandIntents(
    () => ({ request: "req-" + ++sequence, key: "key-" + sequence }),
    { mono: () => time, wall: () => time },
    () => true,
  );
  return {
    store,
    advance: () => {
      time += 400000;
    },
  };
}
test("a refused retry preserves original uncertainty and the exact retained request", async () => {
  for (const failure of [
    "conflict",
    "invalid",
    "refused",
    "stage",
    "denied",
  ] as AdminFailure[]) {
    const { store } = fixture();
    let original = "",
      replay = "";
    await store.submit(
      "state",
      draft,
      serviceLimits,
      async (value, attempt) => {
        original = JSON.stringify({ value, attempt });
        throw new AdminError("unavailable");
      },
    );
    assert.equal(store.get("state")!.phase, "unknown");
    await store.submit(
      "state",
      undefined,
      serviceLimits,
      async (value, attempt) => {
        replay = JSON.stringify({ value, attempt });
        throw new AdminError(failure);
      },
    );
    assert.equal(replay, original);
    assert.equal(
      store.get("state")!.hadUnknown,
      true,
      "refused replay erased the original uncertain outcome",
    );
    assert.equal(
      store.get("state")!.phase,
      failure === "denied" ? "lapsed" : "refused",
    );
  }
});
test("first definitive refusal differs from unknown history and discard starts a fresh history", async () => {
  const { store } = fixture();
  await store.submit("state", draft, serviceLimits, async () => {
    throw new AdminError("conflict");
  });
  assert.equal(store.get("state")!.phase, "refused");
  assert.equal(store.get("state")!.hadUnknown, false);
  store.discard("state");
  await store.submit("state", draft, serviceLimits, async () => {
    throw new AdminError("unavailable");
  });
  assert.equal(store.get("state")!.hadUnknown, true);
  await store.submit("state", undefined, serviceLimits, async () => accepted);
  assert.equal(store.get("state")!.phase, "done");
  assert.equal(store.get("state")!.hadUnknown, true);
  store.discard("state");
  await store.submit("state", draft, serviceLimits, async () => accepted);
  assert.equal(store.get("state")!.hadUnknown, false);
});
test("elapsed replay admission retains uncertainty without dispatching a request", async () => {
  const { store, advance } = fixture();
  await store.submit("state", draft, serviceLimits, async () => {
    throw new AdminError("unavailable");
  });
  advance();
  let calls = 0;
  await store.submit("state", undefined, serviceLimits, async () => {
    calls++;
    return accepted;
  });
  assert.equal(calls, 0);
  assert.equal(store.get("state")!.phase, "lapsed");
  assert.equal(store.get("state")!.hadUnknown, true);
});
