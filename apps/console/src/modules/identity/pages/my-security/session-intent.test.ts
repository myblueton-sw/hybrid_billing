import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, type AdminClient } from "../../../../service/admin";
import { clientFor } from "../../../../service/context";
import { intentStore, intentLocked, submitSelected, recoverSelected, replaySelected } from "./session-intent";

const fingerprint = { selection: "ref-a", digest: "digest-a", count: 170 };
const operation = { operation: "op-a", selection: "ref-a", digest: "digest-a", kind: "others", recorded_count: 170, state: "complete" };
const apiOf = (methods: Partial<AdminClient>) => methods as AdminClient;

test("uncertain submission survives remount identity and replay sends the identical target reference", async () => {
  const sent: unknown[] = [];
  let fail = true;
  const api = apiOf({ endSelectedSessions: async (value) => { sent.push(value); if (fail) throw new AdminError("unavailable"); return operation as never; },
    previewSessionSelection: async () => ({ ...fingerprint, kind: "others", operation: "op-a" }) as never, sessionOperation: async () => operation as never });
  const store = intentStore(api,"u:w:caller")!;
  await submitSelected(store,api,fingerprint,"others");
  assert.equal(store.get().intent?.state,"unknown");
  assert.equal(intentLocked(store.get()),true);
  assert.equal(intentStore(api,"u:w:caller"),store);
  await submitSelected(store,api,{ ...fingerprint,selection:"fresh" },"others");
  assert.equal(sent.length,1);
  fail = false;
  await replaySelected(store,api);
  assert.deepEqual(sent,[fingerprint,fingerprint]);
  assert.equal(store.get().intent?.operation?.operation,"op-a");
  assert.equal(intentLocked(store.get()),false);
});

test("read-back recovers an accepted operation without another effect request", async () => {
  let writes = 0;
  const api = apiOf({ endSelectedSessions: async () => { writes++; throw new Error("lost response"); },
    previewSessionSelection: async () => ({ ...fingerprint, kind:"others", operation:"op-a" }) as never, sessionOperation: async () => operation as never });
  const store = intentStore(api,"scope")!;
  await submitSelected(store,api,fingerprint,"others");
  await recoverSelected(store,api);
  assert.equal(writes,1);
  assert.equal(store.get().intent?.state,"accepted");
});

test("missing result and malformed accepted evidence never unlock a new target list", async () => {
  const api = apiOf({ endSelectedSessions: async () => ({ ...operation, selection:"wrong" }) as never,
    previewSessionSelection: async () => { throw new AdminError("unavailable",[],0,"","unsupported_command"); } });
  const store = intentStore(api,"scope")!;
  await submitSelected(store,api,fingerprint,"others");
  assert.equal(store.get().intent?.state,"unknown");
  await recoverSelected(store,api);
  assert.equal(store.get().intent?.state,"unavailable");
  assert.equal(intentLocked(store.get()),true);
  assert.equal(intentStore(api,"other-scope"),null);
  assert.equal(intentStore(apiOf({}),"scope")?.get().intent,null);
});

test("a confirmed first-admission refusal unlocks, but a refused replay preserves uncertainty", async () => {
  let first = true;
  const api = apiOf({ endSelectedSessions: async () => { if (first) throw new AdminError("limited",[],0,"","rate_limited"); throw new Error("transport"); } });
  const store = intentStore(api,"scope")!;
  await submitSelected(store,api,fingerprint,"others");
  assert.equal(store.get().intent,null);
  first = false;
  await submitSelected(store,api,fingerprint,"others");
  first = true;
  await replaySelected(store,api);
  assert.equal(intentLocked(store.get()),true);
});

test("authenticated client identity is stable across view/locale consumers and changes for a new login", () => {
  const challenge = async () => false;
  const a = clientFor("synthetic-session-a",challenge);
  const store = intentStore(a,"scope")!;
  assert.equal(clientFor("synthetic-session-a",challenge),a);
  assert.equal(intentStore(clientFor("synthetic-session-a",challenge),"scope"),store);
  assert.notEqual(clientFor("synthetic-session-b",challenge),a);
});

test("authentication loss during recovery or replay preserves the isolated uncertain request", async () => {
  let unauthorized=false;
  const rejected=()=>new AdminError("unauthenticated",[],0,"","unauthenticated");
  const sent:unknown[]=[];
  const api=apiOf({endSelectedSessions:async value=>{sent.push(value);if(unauthorized)throw rejected();throw new Error("lost response");},previewSessionSelection:async()=>{throw rejected();}});
  const store=intentStore(api,"retired-auth-scope")!;
  await submitSelected(store,api,fingerprint,"others");unauthorized=true;
  await recoverSelected(store,api);
  assert.equal(store.get().intent?.state,"unavailable");assert.equal(intentLocked(store.get()),true);assert.deepEqual(store.get().intent?.fingerprint,fingerprint);
  await replaySelected(store,api);
  assert.equal(store.get().intent?.state,"unavailable");assert.equal(intentLocked(store.get()),true);assert.deepEqual(store.get().intent?.fingerprint,fingerprint);assert.deepEqual(sent,[fingerprint,fingerprint]);
  assert.equal(intentStore(apiOf({}),"new-auth-scope")?.get().intent,null);
});
