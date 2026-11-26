import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import {
  AdminError,
  type AccessSetting,
  type Outcome,
} from "../../../../service/admin";
import {
  admitsCommand,
  admitsDeclaration,
  commandIntents,
  declarationFields,
  type CommandDraft,
  type DeclarationDraft,
} from "./command-intents";
import { readIdentity } from "./scoped-read";
import {
  accountKeys,
  clearOwned,
  cursorTrail,
  navigation,
  queryKey,
} from "./query-model";
import { serviceLimits } from "../../../../service/service-query-fixtures";

const limits = { ...serviceLimits, command_value_bytes: "9007199254740993" };
const settings = [
  { name: "service_declaration_purpose_bytes", value: 512 },
  { name: "service_declaration_scopes", value: 16 },
] as AccessSetting[];
const create: CommandDraft = {
  command: "service.create",
  target: "services",
  reason: "Synthetic valid rationale",
  fields: {
    name: "agent",
    owner: "owner",
    purpose: "처리 목적",
    organization: "org",
    scopes: '[{"audience":"api","scope":"read"}]',
    catalog_version: "a".repeat(64),
  },
  revision: "r1",
  replay: 300,
};
const accepted: Outcome = {
  status: "completed",
  state: "service_created",
  revision: "r2",
  job_id: "",
  fields: { declaration_version: "1" },
};
function fixture() {
  let current = true,
    time = 0,
    next = 0;
  const store = commandIntents(
    () => ({ request: "r" + ++next, key: "k" + next }),
    { mono: () => time, wall: () => time },
    () => current,
  );
  return {
    store,
    leave() {
      current = false;
    },
    advance(ms: number) {
      time += ms;
    },
  };
}

test("M04 navigation rejects duplicate or malformed owned state without replacing another view", () => {
  for (const query of [
    "svc_a_q=x&svc_a_q=y",
    "svc_a_organization=bad!",
    "svc_a_state=revoked",
    "svc_a_limit=01",
    "svc_a_limit=101",
    "svc_a_scope=read",
    "svc_a_after=a%20b",
    "svc_a_unknown=x",
  ])
    assert.equal(
      navigation(new URLSearchParams(query), "home", "accounts").valid,
      false,
      query,
    );
  const params = new URLSearchParams(
    "lang=ar&svc_a_selected=svc-known&svc_c_issue=svc-other&svc_c_selected=cred-1&svc_c_service=svc-filter",
  );
  const a = navigation(params, "home", "accounts"),
    c = navigation(params, "home", "credentials");
  assert.ok(a.valid && c.valid);
  assert.equal(a.selected, "svc-known");
  assert.equal(c.issue, "svc-other");
  assert.equal(c.selected, "cred-1");
  assert.equal(c.query.service, "svc-filter");
  assert.equal(a.query.workspace, "home");
  for (const q of [
    "ΟΣ",
    "İ",
    "\u0085Search\u0085",
    "\ufeffSearch\ufeff",
    "Ⱥ".repeat(32),
  ]) {
    const nav = navigation(
      new URLSearchParams({ svc_a_q: q }),
      "home",
      "accounts",
    );
    assert.ok(nav.valid);
    assert.equal(nav.query.q, q);
  }
  assert.equal(
    navigation(
      new URLSearchParams({ svc_a_q: "Ⱥ".repeat(33) }),
      "home",
      "accounts",
    ).valid,
    false,
  );
  assert.equal(accountKeys.includes("selected"), true);
});

test("M04 cursor trail holds only20 predecessor cursors and never rows", () => {
  const trail = cursorTrail(),
    key = queryKey({ workspace: "home", limit: 25, q: "a", after: "ignored" });
  for (let i = 0; i < 30; i++)
    assert.equal(trail.next(key, "c" + i, "c" + (i + 1)), true);
  for (let i = 29; i >= 10; i--) assert.equal(trail.previous(key), "c" + i);
  assert.equal(trail.previous(key), "");
  assert.equal(trail.next(key, "same", "same"), false);
  trail.next(key, "a", "b");
  trail.next("other-query", "x", "y");
  assert.equal(trail.previous(key), "");
});

test("M04 unknown retry preserves full copied intent and exact original identity across refresh", async () => {
  const { store } = fixture();
  const fields = { ...create.fields },
    proposed = { ...create, fields };
  let original = "",
    retried = "";
  await store.submit("create", proposed, limits, async (draft, tried) => {
    original = JSON.stringify({ draft, tried });
    throw new AdminError("unavailable");
  });
  fields.name = "changed";
  proposed.reason = "changed reason";
  proposed.revision = "r9";
  assert.equal(store.get("create")?.phase, "unknown");
  assert.equal(
    await store.submit(
      "create",
      { ...create, fields: { ...create.fields, name: "new" } },
      limits,
      async () => accepted,
    ),
    undefined,
  );
  const done = await store.submit(
    "create",
    undefined,
    undefined,
    async (draft, tried) => {
      retried = JSON.stringify({ draft, tried });
      return accepted;
    },
  );
  assert.deepEqual(done, accepted);
  assert.equal(retried, original);
  assert.equal(store.get("create")?.phase, "done");
});

test("M04 concurrent submission, stale completion, malformed accepted result and retry horizon", async () => {
  const f = fixture();
  let resolve!: (value: Outcome) => void;
  const pending = f.store.submit(
    "create",
    create,
    limits,
    () =>
      new Promise<Outcome>((done) => {
        resolve = done;
      }),
  );
  assert.equal(
    await f.store.submit("create", undefined, undefined, async () => accepted),
    undefined,
  );
  f.leave();
  resolve(accepted);
  assert.equal(await pending, undefined);
  assert.equal(f.store.get("create"), undefined);
  const g = fixture();
  await g.store.submit("create", create, limits, async () => ({
    ...accepted,
    status: "accepted",
  }));
  assert.equal(g.store.get("create")?.phase, "unknown");
  g.advance(270000);
  assert.equal(
    await g.store.submit("create", undefined, undefined, async () => {
      assert.fail("lapsed retry dispatched");
    }),
    undefined,
  );
  assert.equal(g.store.get("create")?.phase, "lapsed");
  assert.equal(g.store.discard("create"), true);
});

test("M04 draft bounds count UTF8 and actual JSON escaping without rounding large limits", () => {
  const tried = { request: "request", key: "key" };
  assert.equal(admitsCommand(create, limits, tried), true);
  assert.equal(
    admitsCommand(
      { ...create, fields: { purpose: "ا".repeat(5) } },
      { ...limits, command_value_bytes: "9" },
      tried,
    ),
    false,
  );
  const escaped = { ...create, fields: { purpose: "\u0000".repeat(180000) } };
  assert.equal(admitsCommand(escaped, limits, tried), false);
  const draft: DeclarationDraft = {
    target: "",
    name: "agent",
    owner: "owner",
    purpose: "한".repeat(170),
    organization: "org",
    scopes: [{ audience: "api", scope: "read" }],
    catalog: "a".repeat(64),
    version: "0",
    reason: "Synthetic rationale",
  };
  assert.equal(admitsDeclaration(draft, limits, settings), true);
  assert.equal(
    admitsDeclaration(
      { ...draft, purpose: "한".repeat(171) },
      limits,
      settings,
    ),
    false,
  );
  assert.equal(admitsDeclaration(draft, limits, []), false);
  assert.equal(
    declarationFields({
      ...draft,
      target: "svc-agent",
      owner: "forbidden-transfer",
      version: "2",
    }).owner,
    undefined,
  );
});

test("M04 retains one edit target until an explicit discard and disposes session drafts", () => {
  const { store } = fixture();
  const draft: DeclarationDraft = {
    target: "svc-a",
    name: "",
    owner: "owner",
    purpose: "Purpose",
    organization: "org",
    scopes: [{ audience: "api", scope: "read" }],
    catalog: "a".repeat(64),
    version: "1",
    reason: "",
  };
  assert.equal(store.save("declare", draft, limits, settings), true);
  assert.equal(
    store.save("declare", { ...draft, target: "svc-b" }, limits, settings),
    false,
  );
  assert.equal(store.draft("declare")?.target, "svc-a");
  store.discardDraft("declare");
  assert.equal(
    store.save("declare", { ...draft, target: "svc-b" }, limits, settings),
    true,
  );
  store.dispose();
  assert.equal(store.draft("declare"), undefined);
  assert.equal(store.save("declare", draft, limits, settings), false);
});

test(
  "M04 navigation accepts an actual signed producer cursor",
  { skip: !process.env.HB_SERVICE_QUERY_SAMPLES },
  () => {
    const wire = JSON.parse(
      readFileSync(
        process.env.HB_SERVICE_QUERY_SAMPLES + "/m04-wire.json",
        "utf8",
      ),
    );
    const cursor = wire.equivalent_continuation.query.after;
    assert.ok(cursor.includes("."));
    const nav = navigation(
      new URLSearchParams({ svc_a_after: cursor }),
      "home",
      "accounts",
    );
    assert.ok(nav.valid);
    assert.equal(nav.query.after, cursor);
  },
);

test("M04 read identity separates targets from numeric refresh counters", () => {
  assert.notEqual(
    readIdentity("account", "home", "svc-a1", 2),
    readIdentity("account", "home", "svc-a", 12),
  );
});

test("M04 invalid owned keys can be cleared without changing another view", () => {
  const params = new URLSearchParams(
    "svc_a_typo=x&svc_a_q=a&svc_a_q=b&svc_c_issue=svc-other&lang=ar",
  );
  for (const key of Object.keys(clearOwned(params, "accounts")))
    params.delete(key);
  assert.ok(navigation(params, "home", "accounts").valid);
  assert.equal(params.get("svc_c_issue"), "svc-other");
  assert.equal(params.get("lang"), "ar");
});
