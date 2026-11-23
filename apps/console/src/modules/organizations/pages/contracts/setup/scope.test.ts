import test from "node:test";
import assert from "node:assert/strict";
import { configurationOf, type Configuration, type ConfigurationPage } from "./history";
import { SetupError } from "./context";
import { partOf, reconcileScope, scopeEligible, scopeOf, sessionScopeStore, SetupScopeStore } from "./scope";
const hash="a".repeat(64),other="b".repeat(64);
function confirmed(id:string,change:Record<string,string>={}):Configuration{return configurationOf({configuration:id,digest:hash,state:"confirmed",module_observation:"active",revision:"1",contract_revision:"3",billing_profile:"profile",billing_revision:"2",mode:"erp_issuance",profile:"synthetic_erp_document_v1",region:"US",tax:"synthetic-tax",zone:"UTC",languages:"en",proposer:"first",confirmer:"second",setup_on:"2026-11-16",scope_digest:hash,proposal_reason:"Reason",proposed_at:"2026-11-16T01:00:00Z",approval_observed_at:"2026-11-16T02:00:00Z",activated_at:"2026-11-16T02:00:00Z",...change});}
function legacy(id:string):Configuration{const r:Record<string,string>={...confirmed(id)};delete r.setup_on;delete r.scope_digest;delete r.proposal_reason;return configurationOf(r);}
function pending(id:string):Configuration{const r:Record<string,string>={...confirmed(id),state:"proposed",module_observation:"proposed",confirmer:""};delete r.approval_observed_at;delete r.activated_at;return configurationOf(r);}
function page(rows:Configuration[],next=""):ConfigurationPage{return {revision:"r1",at:"2026-11-16T03:00:00Z",contract:"contract",snapshot:hash,next,count:next?20:rows.length,rows};}
test("an unknown or absent part shows issuance settings",()=>{
 assert.equal(partOf(null),"settings");assert.equal(partOf("toString"),"settings");assert.equal(partOf("statements"),"statements");
});
test("only confirmed settings recorded with setup context can become the scope",()=>{
 assert.equal(scopeEligible(confirmed("a")),true);assert.equal(scopeEligible(legacy("a")),false);assert.equal(scopeEligible(pending("a")),false);
 assert.throws(()=>scopeOf(legacy("a")),SetupError);
 assert.deepEqual(scopeOf(confirmed("a")),{configuration:"a",digest:hash,setupOn:"2026-11-16",contractRevision:"3",billingRevision:"2"});
});
test("a single eligible row of a complete history is selected, and nothing is guessed otherwise",()=>{
 assert.equal(reconcileScope(undefined,page([pending("a"),confirmed("b"),legacy("c")]),"")?.configuration,"b");
 assert.equal(reconcileScope(undefined,page([confirmed("a"),confirmed("b")]),""),undefined);
 assert.equal(reconcileScope(undefined,page([confirmed("a")],"a"),""),undefined);
 assert.equal(reconcileScope(undefined,page([confirmed("k")]),"j"),undefined);
 assert.equal(reconcileScope(undefined,page([legacy("a")]),""),undefined);
 assert.equal(reconcileScope(undefined,undefined,""),undefined);
});
test("a chosen scope survives paging and is dropped when its record changed or disappeared",()=>{
 const chosen=scopeOf(confirmed("b"));
 assert.equal(reconcileScope(chosen,page([confirmed("a"),confirmed("b")]),""),chosen);
 assert.equal(reconcileScope(chosen,page([confirmed("k")]),"j"),chosen);
 assert.equal(reconcileScope(chosen,page([confirmed("a")],"a"),""),chosen);
 assert.equal(reconcileScope(chosen,undefined,""),chosen);
 assert.equal(reconcileScope(chosen,page([confirmed("a"),confirmed("c")]),""),undefined);
 assert.equal(reconcileScope(chosen,page([confirmed("a"),confirmed("b",{digest:other})]),""),undefined);
 assert.equal(reconcileScope(chosen,page([confirmed("b",{digest:other})]),"")?.digest,other);
});
test("the store notifies once per change and keeps one store per session, contract and date",()=>{
 const store=new SetupScopeStore();let calls=0;const stop=store.subscribe(()=>{calls++;});
 store.choose(confirmed("a"));store.reconcile(page([confirmed("a"),confirmed("b")]),"");
 assert.equal(calls,1);assert.equal(store.getSnapshot()?.configuration,"a");
 assert.throws(()=>store.choose(pending("b")),SetupError);
 store.reconcile(page([confirmed("b")]),"");assert.equal(store.getSnapshot()?.configuration,"b");assert.equal(calls,2);
 stop();
 const session={user:"u",workspace:"w",csrf:"c"},first=sessionScopeStore(session,"contract","2026-11-16");
 assert.equal(sessionScopeStore(session,"contract","2026-11-16"),first);
 assert.notEqual(sessionScopeStore(session,"contract","2026-11-17"),first);
 assert.notEqual(sessionScopeStore({...session,csrf:"d"},"contract","2026-11-16"),first);
 assert.notEqual(sessionScopeStore(session,"other","2026-11-16"),first);
 for(let i=0;i<40;i++)sessionScopeStore(session,"contract-"+i,"2026-11-16");
 assert.notEqual(sessionScopeStore(session,"contract","2026-11-16"),first);
});
