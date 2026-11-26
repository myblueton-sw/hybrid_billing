import { test } from "node:test";
import assert from "node:assert/strict";
import { adminClient, AdminError } from "./admin";

const every = { any: true, id: "" };
const grant = { source: "grant", id: "approval-sample-admin", subject: "sample-admin", subject_login: "admin@example.invalid", client: "", delegator: "", delegator_login: "", revocation_requested: false, parent: "", redelegate: false, action: "view", kind: "approval_review", fields: ["approval_evidence"],
  scope: { customer: every, legal_entity: every, contract: every, period: "any" }, not_before: "2026-11-26T08:45:14Z", not_after: "2026-12-26T08:45:14Z", state: "active", effective: true, version: 1 };
const clientGrant = { ...grant, id: "approval-console", subject: "", subject_login: "", client: "console" };
const delegation = { ...grant, source: "delegation", id: "del-1", subject: "sample-member", subject_login: "member@example.invalid", delegator: "sample-admin", delegator_login: "admin@example.invalid", parent: "", scope: { ...grant.scope, customer: { any: false, id: "c1" } } };
const echo = { q: "", subject: "", action: "", customer: "", legal_entity: "", source: "", state: "", expires_before: "", limit: 25 };
const pageOf = (rows: unknown[], change: Record<string, unknown> = {}) => ({ rows, search_input: "", query: echo, next: "", reference_at: "2026-11-26T08:45:18.206196209Z", observed_at: "2026-11-26T08:45:18.211902709Z",
  revision: "r27", order: "source_then_id", count: { value: String(rows.length), certainty: "exact" }, work_units: rows.length, scan_stopped: false, ...change });

function client(body: unknown, inspect?: (path: string, init: RequestInit) => void) {
  return adminClient("csrf", (async (path, init) => { inspect?.(String(path), init!); return new Response(JSON.stringify(body)); }) as typeof fetch);
}
const invalid = (e: unknown) => e instanceof AdminError && e.failure === "invalid";

test("M03 permissions: a page as the service answers it is accepted and sent as one bounded read", async () => {
  let calls = 0;
  const body = pageOf([clientGrant, grant, delegation]);
  const got = await client(body, (path, init) => {
    calls++;
    assert.equal(path, "/api/v1/identity/access/permissions");
    assert.equal(init.method, "GET");
  }).accessPermissions({});
  assert.equal(calls, 1);
  assert.deepEqual(got.rows.map((r) => [r.source, r.id]), [["grant", "approval-console"], ["grant", "approval-sample-admin"], ["delegation", "del-1"]]);
  assert.deepEqual(got.rows[0].scope.customer, { any: true });
  assert.equal((await client(pageOf([{ ...grant, revocation_requested: true }])).accessPermissions({})).rows[0].revocation_requested, true);
  assert.equal((await client(pageOf([{ ...grant, subject_login: "" }])).accessPermissions({})).rows[0].subject_login, "", "a holder the directory does not know");
  assert.deepEqual(got.rows[2].scope.customer, { any: false, id: "c1" });
  let asked = "";
  await client(pageOf([delegation], { search_input: "Del", query: { ...echo, q: "del", source: "delegation", customer: "c1", limit: 5 } }), (path) => { asked = path; })
    .accessPermissions({ q: " Del ", source: "delegation", customer: "c1", limit: 5, state: undefined });
  assert.equal(asked, "/api/v1/identity/access/permissions?q=Del&customer=c1&source=delegation&limit=5");
  const chained = { ...delegation, id: "del-2", parent: "del-1", subject: "sample-admin2", delegator: "sample-member" };
  assert.equal((await client(pageOf([delegation, chained])).accessPermissions({})).rows[1].parent, "del-1");
  assert.equal((await client(pageOf([], { reference_at: "2026-11-26T08:45:18Z" })).accessPermissions({ after: "a.b" })).rows.length, 0, "an empty last page");
  await assert.rejects(client(pageOf([grant], { query: { ...echo, q: "x" } })).accessPermissions({}), invalid, "an echo of a search that was not asked");
  await assert.rejects(client(pageOf([grant], { search_input: "appr" })).accessPermissions({ q: "appr" }), invalid, "a search that was asked and not echoed");
});

test("M03 permissions: a page that contradicts itself or the request is refused", async () => {
  const cases: Record<string, unknown> = {
    "an unknown field": { ...pageOf([grant]), extra: 1 },
    "an unknown row field": pageOf([{ ...grant, note: "x" }]),
    "delegations before grants": pageOf([delegation, grant]),
    "identifiers out of order": pageOf([grant, clientGrant]),
    "a repeated row": pageOf([grant, grant]),
    "a grant with a delegator": pageOf([{ ...grant, delegator: "someone" }]),
    "a grant with a delegator's sign-in name": pageOf([{ ...grant, delegator_login: "x@example.invalid" }]),
    "a client grant with a holder's sign-in name": pageOf([{ ...clientGrant, subject_login: "x@example.invalid" }]),
    "a delegation with a pending revocation request": pageOf([{ ...delegation, revocation_requested: true }]),
    "a sign-in name with a control character": pageOf([{ ...grant, subject_login: "a\u0000b" }]),
    "a row without the request flag": pageOf([(({ revocation_requested: _omit, ...rest }) => rest)(grant)]),
    "a grant with both a subject and a client": pageOf([{ ...grant, client: "console" }]),
    "a grant with neither": pageOf([{ ...grant, subject: "" }]),
    "a grant marked ended": pageOf([{ ...grant, state: "ended", effective: false }]),
    "a delegation without a delegator": pageOf([{ ...delegation, delegator: "" }]),
    "a delegation to its own delegator": pageOf([{ ...delegation, delegator: "sample-member" }]),
    "effective but not active": pageOf([{ ...grant, state: "expired" }]),
    "a selector that is any and names one": pageOf([{ ...grant, scope: { ...grant.scope, customer: { any: true, id: "c1" } } }]),
    "a selector that is neither": pageOf([{ ...grant, scope: { ...grant.scope, customer: { any: false, id: "" } } }]),
    "a selector as text": pageOf([{ ...grant, scope: { ...grant.scope, customer: "all" } }]),
    "validity that ends before it starts": pageOf([{ ...grant, not_after: "2026-11-26T08:45:13Z" }]),
    "an unknown state": pageOf([{ ...grant, state: "closed", effective: false }]),
    "a count below the rows": pageOf([grant], { count: { value: "0", certainty: "exact" } }),
    "an exact count with a cursor": pageOf([grant], { next: "a.b" }),
    "a lower bound without a cursor": pageOf([grant], { count: { value: "1", certainty: "lower_bound" } }),
    "a first page whose count is not its rows": pageOf([grant], { next: "a.b", count: { value: "5", certainty: "lower_bound" } }),
    "a stopped scan without a cursor": pageOf([grant], { scan_stopped: true }),
    "another order": pageOf([grant], { order: "id" }),
    "a reference after the observation": pageOf([grant], { reference_at: "2026-11-26T08:45:19Z" }),
    "an echo of another limit": pageOf([grant], { query: { ...echo, limit: 7 } }),
    "more rows than the limit": pageOf([clientGrant, grant], { query: { ...echo, limit: 1 } }),
  };
  for (const [name, body] of Object.entries(cases)) {
    await assert.rejects(client(body).accessPermissions(name === "an echo of another limit" ? { limit: 25 } : {}), invalid, name);
  }
  const filtered = (query: Record<string, string>, rows: unknown[]) => client(pageOf(rows, { query: { ...echo, ...query }, search_input: query.q ?? "" })).accessPermissions(query);
  await assert.rejects(filtered({ source: "delegation" }, [grant]), invalid, "a row of another source");
  await assert.rejects(filtered({ subject: "someone" }, [grant]), invalid, "a row of another subject");
  await assert.rejects(filtered({ customer: "c2" }, [delegation]), invalid, "a row scoped to another customer");
  await assert.rejects(filtered({ state: "expired" }, [grant]), invalid, "a row in another state");
  await assert.rejects(filtered({ expires_before: "2026-12-01T00:00:00Z" }, [grant]), invalid, "a row that ends later");
  await assert.rejects(filtered({ q: "nothing" }, [grant]), invalid, "a row the search does not match");
  assert.equal((await filtered({ customer: "c2" }, [grant])).rows.length, 1, "a permission covering every customer applies to any customer");
  assert.equal((await filtered({ customer: "c1" }, [delegation])).rows.length, 1);
  await assert.rejects(client(pageOf([grant], { query: { ...echo, subject: "other" } })).accessPermissions({}), invalid, "an echo of a filter that was not asked");
});

test("M03 permissions: a request that could not be asked is not sent", async () => {
  let calls = 0;
  const c = client(pageOf([]), () => { calls++; });
  for (const query of [{ limit: 0 }, { limit: 101 }, { limit: 1.5 }, { subject: "a b" }, { customer: "a/b" }, { expires_before: "tomorrow" }, { q: "x".repeat(65) }, { q: "   " }, { after: "a".repeat(4097) }, { unknown: "1" }]) {
    await assert.rejects(c.accessPermissions(query as never), invalid, JSON.stringify(query));
  }
  await assert.rejects(c.accessPermission("grant", "a b"), invalid);
  await assert.rejects(c.accessPermission("other" as never, "g1"), invalid);
  assert.equal(calls, 0);
});

test("M03 permission detail: the answer must be the one asked for", async () => {
  const detail = { workspace: "sample-workspace", row: grant, would_end: ["del-1", "del-2"], revision: "r27", observed_at: "2026-11-26T08:45:18.211902709Z" };
  let asked = "";
  const got = await client(detail, (path) => { asked = path; }).accessPermission("grant", grant.id);
  assert.equal(asked, "/api/v1/identity/access/permissions/grant/approval-sample-admin");
  assert.equal(got.row.id, grant.id);
  await assert.rejects(client(detail).accessPermission("grant", "another"), invalid, "another identifier");
  await assert.rejects(client(detail).accessPermission("delegation", grant.id), invalid, "another source");
  await assert.rejects(client({ ...detail, eligibility: {} }).accessPermission("grant", grant.id), invalid, "an unknown field");
  await assert.rejects(client({ ...detail, observed_at: "now" }).accessPermission("grant", grant.id), invalid, "a time that is not one");
  assert.deepEqual(got.would_end, ["del-1", "del-2"]);
  await assert.rejects(client((({ would_end: _dropped, ...rest }) => rest)(detail)).accessPermission("grant", grant.id), invalid, "no list of what would end");
  await assert.rejects(client({ ...detail, would_end: ["del-2", "del-1"] }).accessPermission("grant", grant.id), invalid, "what would end out of order");
  await assert.rejects(client({ ...detail, would_end: ["del-1", "del-1"] }).accessPermission("grant", grant.id), invalid, "a delegation named twice");
  await assert.rejects(client({ ...detail, would_end: ["a b"] }).accessPermission("grant", grant.id), invalid, "something that is not an identifier");
  await assert.rejects(client({ ...detail, row: delegation, would_end: [delegation.id] }).accessPermission("delegation", delegation.id), invalid, "a delegation that would end itself");
  assert.deepEqual((await client({ ...detail, row: delegation, would_end: [] }).accessPermission("delegation", delegation.id)).would_end, []);
});
