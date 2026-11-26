import test from "node:test";
import assert from "node:assert/strict";
import type { AttestationReceipt } from "./attestation";
import { statementIntent, type StatementStep } from "./channel-flow";
import { SetupError } from "./context";
import type { ProtectedWriteIntent, ProtectedWriteReceipt } from "./protected-writes";
import { sendChained } from "./statement-chain";
import { StepWriteStore } from "./step-store";
const hash="a".repeat(64),owner={user:"me",workspace:"workspace",csrf:"csrf"},read={scope:{contract:"contract-a",configuration:"configuration-a",digest:hash},revision:"r4"};
const claim=()=>statementIntent(owner,read,"channel_admission","statement-1","channel-1",hash,{statement:"Text"},{request:"request-s",key:"key-s"});
const action=(attestation:string,revision:string)=>({operation:"admit_channel",actor:"me",workspace:"workspace",sessionCSRF:"csrf",revision,attempt:{request:"request-a",key:"key-a"},scope:read.scope,args:{channel:"channel-1",channel_digest:hash,attestation},rationale:""}) as ProtectedWriteIntent;
function stores(statement:"recorded"|"consumed"|"refused"|"unknown"){
 const sent:ProtectedWriteIntent[]=[];
 const statements=new StepWriteStore<StatementStep,AttestationReceipt>(owner,async i=>{if(statement==="refused")throw new SetupError("prerequisite_failed","none");if(statement==="unknown")throw new SetupError("service_unavailable","unknown");return {attestation:i.attestation,revision:"r5",state:statement==="consumed"?"consumed":"unconsumed"} as AttestationReceipt;});
 const actions=new StepWriteStore<ProtectedWriteIntent,ProtectedWriteReceipt>(owner,async i=>{sent.push(i);return {state:"admitted"} as ProtectedWriteReceipt;});
 return {statements,actions,sent};
}
test("the step follows a statement that ended as recorded and unused, with the statement's identifier and receipt revision",async()=>{
 const {statements,actions,sent}=stores("recorded");
 assert.equal(await sendChained(statements,actions,{subject:"channel-1",form:"accept",statement:claim(),build:action},"r4"),"sent");
 assert.deepEqual(sent.map(i=>[i.args.attestation,i.revision]),[["statement-1","r5"]]);assert.equal(actions.snapshot().entries[0].form,"accept");
});
test("no step is sent when the statement request is refused, unknown or returns a statement that is not unused",async()=>{
 for(const mode of ["refused","unknown","consumed"] as const){
  const {statements,actions,sent}=stores(mode);
  assert.equal(await sendChained(statements,actions,{subject:"channel-1",form:"accept",statement:claim(),build:action},"r4"),"statement_unsettled",mode);
  assert.equal(sent.length,0,mode);assert.equal(actions.snapshot().entries.length,0,mode);assert.equal(statements.snapshot().entries.length,1,mode);
 }
});
test("a reused statement sends one request with the revision of the current read, and a blocked store sends nothing",async()=>{
 const {statements,actions,sent}=stores("recorded");
 assert.equal(await sendChained(statements,actions,{subject:"channel-1",form:"accept",reuse:{id:"statement-kept",text:"Text",locator:""},build:action},"r9"),"sent");
 assert.deepEqual(sent.map(i=>[i.args.attestation,i.revision]),[["statement-kept","r9"]]);assert.equal(statements.snapshot().entries.length,0);
 const blocked=stores("unknown");await blocked.statements.submit(claim(),"statement-1");
 await assert.rejects(()=>sendChained(blocked.statements,blocked.actions,{subject:"channel-1",form:"accept",statement:{...claim(),attestation:"statement-2",attempt:{request:"request-t",key:"key-t"}},build:action},"r4"),(e:unknown)=>e instanceof SetupError&&e.code==="unresolved_operation");
 assert.equal(blocked.sent.length,0);
});
