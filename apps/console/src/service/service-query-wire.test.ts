import { test } from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { adminClient } from "./admin";
import type { PageQuery } from "./service-query-types";

test("M04 Console consumes all12 actual HTTP producer samples", { skip: !process.env.HB_SERVICE_QUERY_SAMPLES }, async () => {
  const data = JSON.parse(await readFile(join(process.env.HB_SERVICE_QUERY_SAMPLES!, "m04-wire.json"), "utf8")) as {
    ids: { workspace: string; service: string; credential: string; owner: string; organization: string; audience: string; scope: string };
    samples: Record<string, { path: string; body: unknown }>;
    unicode_queries: { input: string; body: unknown }[];
    equivalent_continuation: { query: PageQuery; body: unknown };
  };
  const ids = data.ids;
  const query = { workspace: ids.workspace };
  const methods = {
    accounts: (api: ReturnType<typeof adminClient>) => api.serviceAccounts(query),
    account: (api: ReturnType<typeof adminClient>) => api.serviceAccount(ids.service, ids.workspace),
    credentials: (api: ReturnType<typeof adminClient>) => api.serviceCredentials(query),
    credential: (api: ReturnType<typeof adminClient>) => api.serviceCredential(ids.credential, ids.workspace),
    owners: (api: ReturnType<typeof adminClient>) => api.serviceOwners(query),
    owner: (api: ReturnType<typeof adminClient>) => api.serviceOwner(ids.owner, ids.workspace),
    organizations: (api: ReturnType<typeof adminClient>) => api.serviceOrganizations(query),
    organization: (api: ReturnType<typeof adminClient>) => api.serviceOrganization(ids.organization, ids.workspace),
    audiences: (api: ReturnType<typeof adminClient>) => api.serviceAudiences(query),
    scopes: (api: ReturnType<typeof adminClient>) => api.serviceScopes(ids.audience, query),
    scope: (api: ReturnType<typeof adminClient>) => api.serviceScope(ids.audience, ids.scope, ids.workspace),
    history: (api: ReturnType<typeof adminClient>) => api.serviceDeclarations(ids.service, query),
  };
  assert.equal(Object.keys(data.samples).length, 12);
  for (const [name, invoke] of Object.entries(methods)) {
    const sample = data.samples[name];
    let calls = 0;
    const api = adminClient("csrf", (async (path, init) => {
      assert.equal(String(path), sample.path, name);
      assert.equal(init?.method, "GET", name);
      calls++;
      return new Response(JSON.stringify(sample.body));
    }) as typeof fetch);
    assert.deepEqual(await invoke(api), sample.body, name);
    assert.equal(calls, 1, name);
  }
  for (const { input, body } of data.unicode_queries) {
    const api = adminClient("csrf", (async () => new Response(JSON.stringify(body))) as typeof fetch);
    assert.deepEqual(await api.serviceOwners({ workspace: ids.workspace, q: input }), body, input);
  }
  const continuation = data.equivalent_continuation;
  const api = adminClient("csrf", (async () => new Response(JSON.stringify(continuation.body))) as typeof fetch);
  assert.deepEqual(await api.serviceOwners(continuation.query), continuation.body);
});
