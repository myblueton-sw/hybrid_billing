import test from "node:test";
import assert from "node:assert/strict";
import { SetupEditor, sessionSetupEditor } from "./editor";
import { SetupWriteStore } from "./write-store";
import { SetupError, type SetupContext } from "./context";
import type { SetupOptions } from "./client";
import { testAction, testDigest, testRow, testSession } from "./write-fixture";
const context:SetupContext={revision:"r4",at:"2026-11-16T00:00:00Z",contract:"contract",contractName:"Sample",on:"2026-11-16",lifecycleSequence:"1",contractRevision:"2",billingProfile:"profile",billingProfileName:"Sample",billingRevision:"3",buyer:"buyer",payer:"payer",issuer:"issuer",currency:"USD",scopeDigest:testDigest,profileBasis:"current",currentEligibility:"unchecked"};
const options:SetupOptions={revision:"r4",at:context.at,contract:context.contract,on:context.on,scopeDigest:testDigest,option:"synthetic_erp_sample_v1",applicability:"synthetic_only",mode:"erp_issuance",profile:"synthetic_erp_document_v1",region:"US",tax:"synthetic-tax",zone:"UTC",languages:["ar","en","ko"],currentEligibility:"unchecked"};
const reads={context:async()=>context,options:async()=>options,action:async(request:Parameters<typeof testAction>[0])=>testAction(request),configurations:async()=>({revision:"r4",at:context.at,contract:"contract",snapshot:testDigest,next:"",count:1,rows:[testRow(true)]})};
function fixture(client=reads){
 const sent:unknown[]=[];
 const writes=new SetupWriteStore(testSession,client,async intent=>{sent.push(intent);throw new SetupError("service_unavailable","unknown");});
 const editor=new SetupEditor(testSession,"contract",context.on,writes,client);
 editor.edit({preset:true,languages:["en","ko"],reason:"Synthetic setup"});
 return {editor,writes,sent};
}
test("fresh review freezes inputs and duplicate submission cannot dispatch twice",async()=>{
 const {editor,writes,sent}=fixture();await editor.check();assert.ok(editor.snapshot().prepared);
 await editor.submit();await assert.rejects(editor.submit(),SetupError);
 assert.equal(sent.length,1);assert.equal(writes.snapshot().entries[0].phase,"unknown");
 const saved=editor.snapshot().draft;editor.invalidate();assert.deepEqual(editor.snapshot().draft,saved);
 await writes.retry(writes.snapshot().entries[0].intent.attempt.key);assert.deepEqual(sent[0],sent[1]);
});
test("input changes suppress delayed source checks before action inspection",async()=>{
 let release!:(value:SetupContext)=>void,actions=0;
 const {editor}=fixture({...reads,context:()=>new Promise(r=>{release=r;}),action:async request=>{actions++;return testAction(request);}});
 const checking=editor.check();editor.edit({reason:"Changed reason"});release(context);await checking;
 assert.equal(actions,0);assert.equal(editor.snapshot().prepared,undefined);assert.equal(editor.snapshot().draft.reason,"Changed reason");
});
test("mixed revisions and explicit denied action never prepare submission",async()=>{
 const mixed=fixture({...reads,options:async()=>({...options,revision:"r3"})});await mixed.editor.check();
 assert.equal(mixed.editor.snapshot().error,"evidence_stale");assert.equal(mixed.editor.snapshot().prepared,undefined);
 const denied=fixture({...reads,action:async request=>testAction(request,"r4",{permission:"denied",step_up:"unchecked",action:"blocked",code:"permission_denied"})});await denied.editor.check();assert.equal(denied.editor.snapshot().error,"permission_denied");assert.equal(denied.sent.length,0);
});
test("retained recovery uses fresh history and original proposal reason",async()=>{
 const {editor}=fixture();editor.edit({selected:"config",reason:"Must not replace original"});
 await editor.check(testRow(true),{after:"",snapshot:testDigest});
 assert.equal(editor.snapshot().recovery,true);assert.equal(editor.snapshot().prepared?.rationale,"Synthetic setup");
 assert.equal(editor.snapshot().prepared?.expected.digest,testDigest);
});
test("session, contract and date isolate retained inputs; reentry reuses draft",()=>{
 const a=sessionSetupEditor(testSession,"contract",context.on);a.edit({reason:"Retained input"});a.invalidate();
 assert.equal(sessionSetupEditor({...testSession},"contract",context.on),a);
 for(const b of [sessionSetupEditor({...testSession,csrf:"different"},"contract",context.on),sessionSetupEditor(testSession,"other",context.on),sessionSetupEditor(testSession,"contract","2026-11-17")])assert.equal(b.snapshot().draft.reason,"");
});
test("leaving during options or action lookup cannot restore reviewed guidance",async()=>{
 for(const stage of ["options","action"] as const){
  let release!:()=>void,started!:()=>void;
  const ready=new Promise<void>(resolve=>{started=resolve;});
  const hold=async()=>{started();await new Promise<void>(resolve=>{release=resolve;});};
  const client={...reads,options:async()=>{if(stage==="options")await hold();return options;},action:async(request:Parameters<typeof testAction>[0])=>{if(stage==="action")await hold();return testAction(request);}};
  const {editor,writes}=fixture(client);const checking=editor.check();await ready;editor.invalidate();release();await checking;
  assert.equal(editor.snapshot().prepared,undefined);assert.equal(writes.snapshot().inspection,undefined);
 }
});
test("fresh recovery requires the selected retained identity in refreshed history",async()=>{
 const {editor}=fixture({...reads,configurations:async()=>({...await reads.configurations(),rows:[],count:0})});
 editor.edit({selected:"config"});await editor.check(testRow(true),{after:"",snapshot:testDigest});
 assert.equal(editor.snapshot().prepared,undefined);assert.equal(editor.snapshot().error,"inspection_required");
});
test("failed post-write refresh cannot erase a recorded result",async()=>{
 const {SetupReadStore}=await import("./read-store");
 const writes=new SetupWriteStore(testSession,reads,async()=>({configuration:testRow(),revision:"r5",operation:"op-1",replay:false}));
 const editor=new SetupEditor(testSession,"contract",context.on,writes,reads);
 editor.edit({preset:true,languages:["en"],reason:"Synthetic setup"});await editor.check();await editor.submit();
 const broken=new SetupReadStore({...reads,context:async()=>{throw new SetupError("service_unavailable");},configurations:async()=>{throw new SetupError("service_unavailable");}},"contract",context.on);
 broken.invalidate();await broken.load();editor.invalidate();
 assert.equal(broken.getSnapshot().history.error,"service_unavailable");assert.equal(writes.snapshot().entries[0].phase,"recorded");assert.ok(writes.snapshot().entries[0].receipt);
});
test("a recorded write clears the draft and gives the next proposal a new identifier, while an unsettled or refused one keeps it",async()=>{
 let mode="record";
 const writes=new SetupWriteStore(testSession,reads,async intent=>{if(mode==="refuse")throw new SetupError("prerequisite_failed","none");if(mode==="lose")throw new SetupError("service_unavailable","unknown");return {configuration:testRow(),revision:"r5",operation:"operation-"+intent.attempt.key,replay:false};});
 const editor=new SetupEditor(testSession,"contract",context.on,writes,reads),filled={preset:true,languages:["en","ko"],reason:"Synthetic setup"};
 editor.edit(filled);const first=editor.snapshot().draft.target;
 await editor.check();await editor.submit();
 assert.equal(writes.snapshot().entries[0].phase,"recorded");
 const cleared=editor.snapshot().draft;
 assert.deepEqual([cleared.preset,cleared.languages,cleared.reason,cleared.selected],[false,[],"",""]);assert.notEqual(cleared.target,first);
 mode="refuse";editor.edit(filled);const second=editor.snapshot().draft;
 await editor.check();await editor.submit();
 assert.equal(writes.snapshot().entries[1].phase,"refused");assert.deepEqual(editor.snapshot().draft,second,"a refused write keeps the draft");
 mode="lose";await editor.check();await editor.submit();
 assert.equal(writes.snapshot().entries[2].phase,"unknown");assert.deepEqual(editor.snapshot().draft,second,"an unknown outcome keeps the draft");
});
