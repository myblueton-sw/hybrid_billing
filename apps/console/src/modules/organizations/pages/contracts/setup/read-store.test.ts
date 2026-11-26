import test from "node:test";
import assert from "node:assert/strict";
import { SetupReadStore } from "./read-store";
import { setupContextOf, SetupError, type SetupContext } from "./context";
import type { SetupOptions } from "./client";
import type { ConfigurationPage } from "./history";
const hash="a".repeat(64);
const scope=setupContextOf({revision:"r1",at:"2026-11-16T00:00:00Z",fields:{schema:"hb_issuance_setup_context_v1",contract:"contract",contract_name:"Example",on:"2026-11-16",lifecycle_sequence:"1",contract_revision:"1",billing_profile:"profile",billing_profile_name:"Example",billing_revision:"1",buyer:"buyer",payer:"payer",issuer:"issuer",currency:"USD",scope_digest:hash,profile_basis:"current",current_eligibility:"unchecked"}},"contract","2026-11-16");
const option:SetupOptions={revision:"r1",at:scope.at,contract:"contract",on:scope.on,scopeDigest:hash,option:"synthetic_erp_sample_v1",applicability:"synthetic_only",mode:"erp_issuance",profile:"synthetic_erp_document_v1",region:"US",tax:"synthetic-tax",zone:"UTC",languages:["ar","en","ko"],currentEligibility:"unchecked"};
const page:ConfigurationPage={revision:"r1",at:scope.at,contract:"contract",snapshot:hash,next:"",count:0,rows:[]};
function deferred<T>(){let resolve!:(value:T)=>void;const promise=new Promise<T>(done=>{resolve=done;});return {promise,resolve};}
test("late source responses never replace a newer scope read",async()=>{
 const old=deferred<SetupContext>();let calls=0;
 const store=new SetupReadStore({context:async()=>++calls===1?old.promise:{...scope,billingRevision:"2"},options:async()=>option,configurations:async()=>page},"contract",scope.on);
 const first=store.source();await store.source();old.resolve(scope);await first;
 assert.equal(store.getSnapshot().context.value?.billingRevision,"2");
});
test("history remains available when current source fails and failed reload clears old data",async()=>{
 let fail=false;
 const store=new SetupReadStore({context:async()=>{throw new SetupError("evidence_stale");},options:async()=>option,configurations:async()=>{if(fail)throw new SetupError("permission_denied");return page;}},"contract",scope.on);
 await store.load();assert.equal(store.getSnapshot().context.error,"evidence_stale");assert.equal(store.getSnapshot().history.value,page);
 fail=true;await store.history();assert.equal(store.getSnapshot().history.value,undefined);assert.equal(store.getSnapshot().history.error,"permission_denied");
});
test("dispose suppresses late session results and pagination uses matching snapshot",async()=>{
 const pending=deferred<ConfigurationPage>();const args:string[][]=[];let calls=0;
 const store=new SetupReadStore({context:async()=>scope,options:async()=>option,configurations:async(...input)=>{args.push(input.map(v=>v??""));return ++calls===1?{...page,next:"config-10",count:11}:pending.promise;}},"contract",scope.on);
 await store.history();const next=store.history(true);
 assert.deepEqual(args[1],["contract","config-10",hash]);assert.equal(store.getSnapshot().history.value,undefined);
 store.dispose();pending.resolve(page);await next;assert.equal(store.getSnapshot().history.value,undefined);
});
