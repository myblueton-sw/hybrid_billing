import test from "node:test";
import assert from "node:assert/strict";
import type { Attestation, AttestationReceipt, SetupReference } from "./attestation";
import { SetupError } from "./context";
import type { ProtectedChannel } from "./protected";
import type { ProtectedWriteIntent } from "./protected-writes";
import { channelError } from "./channels-copy";
import { statementError } from "./statements-copy";
import { actionStatementNote, recordedStatement, statementRequestNote, admitChannelIntent, channelOffer, channelReferences, registerChannelIntent, reusableStatement, statementIntent, withdrawIntent, type ProtectedEntry, type StatementEntry } from "./channel-flow";
const hash="a".repeat(64),other="b".repeat(64),session={user:"me",workspace:"workspace",csrf:"csrf"},attempt={request:"request-1",key:"key-1"};
const read={scope:{contract:"contract-a",configuration:"configuration-a",digest:hash},revision:"r4"};
const channel=(change:Partial<ProtectedChannel>={}):ProtectedChannel=>({channel:"channel-1",digest:hash,subject_kind:"entity",kind:"portal",state:"registered",observation:"registered",registrar:"other",admitter:"",pending_admitter:"",registered_at:"2026-11-21T01:00:00Z",admitted_at:"",retired_by:"",retired_at:"",rules:"current",request_not_before:"",...change});
const references:SetupReference[]=[{code:"synthetic_channel_entity_a",use:"channel_entity",applicability:"synthetic_only"},{code:"synthetic_channel_entity_b",use:"channel_entity",applicability:"synthetic_only"},{code:"synthetic_channel_customer_a",use:"channel_customer",applicability:"synthetic_only"},{code:"synthetic_account_a",use:"payment_account",applicability:"synthetic_only"}];
const refused=(run:()=>unknown)=>assert.throws(run,(e:unknown)=>e instanceof SetupError&&e.code==="invalid_setup_request");
const draft={subject:"entity",kind:"portal",reference:"synthetic_channel_entity_a",reason:" Registered after the signed form ",statement:"I registered the portal channel.\nSecond line.",locator:" Folder 12 "};
test("acceptance is offered only for a registered channel observed as registered, and never to its registrar",()=>{
 assert.equal(channelOffer(channel(),"me"),"offer");assert.equal(channelOffer(channel({registrar:"me"}),"me"),"own");
 for(const row of [channel({state:"registration_unknown",observation:"absent",registered_at:""}),channel({state:"registration_unknown"}),channel({state:"admission_unknown",pending_admitter:"me"}),channel({state:"admitted",observation:"admitted",admitter:"third",admitted_at:"2026-11-21T02:00:00Z"}),channel({observation:"retired",retired_by:"third",retired_at:"2026-11-22T02:00:00Z"})])assert.equal(channelOffer(row,"me"),"none");
});
test("a statement names its step, its target and, where bound, the target digest, and keeps the text as written",()=>{
 const registration=statementIntent(session,read,"channel_registration","statement-1","channel-new","",draft,attempt);
 assert.deepEqual({operation:registration.operation,action:registration.action,purpose:registration.purpose,attestation:registration.attestation,target:registration.target,digest:registration.targetDigest,statement:registration.statement,locator:registration.locator,revision:registration.revision,scope:registration.scope,actor:registration.actor},{operation:"attest_channel_registration",action:"attest",purpose:"channel_registration",attestation:"statement-1",target:"channel-new",digest:"",statement:"I registered the portal channel.\nSecond line.",locator:"Folder 12",revision:"r4",scope:read.scope,actor:"me"});
 assert.equal(statementIntent(session,read,"channel_admission","statement-2","channel-1",hash,draft,attempt).targetDigest,hash);
 refused(()=>statementIntent(session,read,"channel_admission","statement-2","channel-1","",draft,attempt));refused(()=>statementIntent(session,read,"channel_registration","statement-1","channel-new",hash,draft,attempt));
 refused(()=>statementIntent(session,read,"channel_registration","statement-1","channel-new","",{...draft,statement:"  \n "},attempt));
});
test("only the author withdraws, and only a statement the read shows as unused",()=>{
 const row=(change:Partial<Attestation>={}):Attestation=>({attestation:"statement-1",digest:hash,purpose:"channel_registration",target:"channel-1",actor:"me",recorded_at:"2026-11-21T01:00:00Z",statement:"Text",locator:"",state:"unconsumed",assurance:"human_attestation",...change});
 const intent=withdrawIntent(session,read,row(),attempt);
 assert.deepEqual({operation:intent.operation,action:intent.action,attestation:intent.attestation,target:intent.target,statement:intent.statement,revision:intent.revision},{operation:"withdraw_channel_registration",action:"withdraw",attestation:"statement-1",target:"",statement:"",revision:"r4"});
 refused(()=>withdrawIntent(session,read,row({actor:"other"}),attempt));refused(()=>withdrawIntent(session,read,row({state:"consumed"}),attempt));refused(()=>withdrawIntent(session,read,row({state:"withdrawn"}),attempt));
});
test("registering needs a sample of the chosen subject, a closed type, a reason and a statement identifier",()=>{
 const intent=registerChannelIntent(session,read,"r5","channel-new","statement-1",draft,references,attempt);
 assert.deepEqual({operation:intent.operation,args:intent.args,rationale:intent.rationale,revision:intent.revision,scope:intent.scope},{operation:"register_channel",args:{channel:"channel-new",subject_kind:"entity",kind:"portal",reference:"synthetic_channel_entity_a",attestation:"statement-1"},rationale:"Registered after the signed form",revision:"r5",scope:read.scope});
 assert.deepEqual(channelReferences(references,"entity").map(e=>e.code),["synthetic_channel_entity_a","synthetic_channel_entity_b"]);assert.deepEqual(channelReferences(references,""),[]);
 for(const change of [{reference:"synthetic_channel_customer_a"},{reference:"synthetic_account_a"},{reference:""},{subject:""},{kind:"fax"},{kind:""},{reason:" "}])refused(()=>registerChannelIntent(session,read,"r5","channel-new","statement-1",{...draft,...change},references,attempt));
 refused(()=>registerChannelIntent(session,read,"r5","channel-new","",draft,references,attempt));
});
test("accepting names the retained channel and digest, takes no reason and is never built for the registrar",()=>{
 const intent=admitChannelIntent(session,read,"r6",channel(),"statement-2",attempt);
 assert.deepEqual({operation:intent.operation,args:intent.args,rationale:intent.rationale,revision:intent.revision},{operation:"admit_channel",args:{channel:"channel-1",channel_digest:hash,attestation:"statement-2"},rationale:"",revision:"r6"});
 refused(()=>admitChannelIntent(session,read,"r6",channel({registrar:"me"}),"statement-2",attempt));refused(()=>admitChannelIntent(session,read,"r6",channel({state:"admitted",observation:"admitted",admitter:"third",admitted_at:"2026-11-21T02:00:00Z"}),"statement-2",attempt));
});
test("a statement of this session is reused only while nothing of this session may have used or withdrawn it, and only for the same step, target and digest",()=>{
 const statement=(id:string,phase:StatementEntry["phase"],change:Partial<StatementEntry["intent"]>={}):StatementEntry=>({intent:{...statementIntent(session,read,"channel_admission",id,"channel-1",hash,draft,{request:"r-"+id,key:"k-"+id}),...change},subject:id,phase,at:"2026-11-24T09:00:00Z",receipt:phase==="recorded"||phase==="reconciled"?{revision:"r5",attestation:id,state:"unconsumed"} as AttestationReceipt:undefined});
 const action=(id:string,phase:ProtectedEntry["phase"]):ProtectedEntry=>({intent:{operation:"admit_channel",args:{channel:"channel-1",channel_digest:hash,attestation:id},scope:read.scope} as unknown as ProtectedWriteIntent,subject:"channel-1",phase,at:"2026-11-24T09:01:00Z"});
 const find=(s:StatementEntry[],a:ProtectedEntry[]=[],digest=hash,target="channel-1")=>reusableStatement(s,a,read.scope,"me","channel_admission",target,digest)?.intent.attestation;
 assert.equal(find([statement("s1","recorded")]),"s1");assert.equal(find([statement("s1","reconciled")]),"s1");
 for(const phase of ["pending","unknown","refused"] as const)assert.equal(find([statement("s1",phase)]),undefined);
 assert.equal(find([statement("s1","recorded")],[action("s1","refused")]),"s1");
 for(const phase of ["pending","unknown","recorded","reconciled"] as const)assert.equal(find([statement("s1","recorded")],[action("s1",phase)]),undefined,phase);
 assert.equal(find([statement("s1","recorded")],[],other),undefined);assert.equal(find([statement("s1","recorded")],[],hash,"channel-2"),undefined);
 assert.equal(find([statement("s1","recorded",{purpose:"channel_registration"})]),undefined);assert.equal(find([statement("s1","recorded",{actor:"other"})]),undefined);
 assert.equal(find([statement("s1","recorded",{scope:{...read.scope,configuration:"configuration-z"}})]),undefined);
 const withdrawn:StatementEntry={...statement("w1","unknown"),intent:{...statement("w1","unknown").intent,action:"withdraw",attestation:"s1"}};
 assert.equal(find([statement("s1","recorded"),withdrawn]),undefined);assert.equal(find([statement("s1","recorded"),{...withdrawn,phase:"refused"}]),"s1");
 assert.equal(reusableStatement([statement("s1","recorded",{purpose:"channel_registration",target:"channel-new",targetDigest:""})],[],read.scope,"me","channel_registration")?.intent.target,"channel-new");
});
test("an action is sent only after its statement request ended as recorded and unused, and the notes after sending follow what may have happened",()=>{
 const entry=(phase:StatementEntry["phase"],state="unconsumed"):StatementEntry=>({intent:statementIntent(session,read,"channel_admission","s1","channel-1",hash,draft,attempt),subject:"s1",phase,at:"2026-11-24T09:00:00Z",receipt:phase==="recorded"||phase==="reconciled"?{revision:"r5",attestation:"s1",state} as AttestationReceipt:undefined});
 assert.deepEqual(recordedStatement(entry("recorded")),{attestation:"s1",revision:"r5"});
 for(const e of [undefined,entry("pending"),entry("unknown"),entry("refused"),entry("reconciled"),entry("recorded","consumed"),entry("recorded","withdrawn")])assert.equal(recordedStatement(e),undefined);
 assert.equal(reusableStatement([entry("recorded","consumed")],[],read.scope,"me","channel_admission","channel-1",hash),undefined);assert.equal(reusableStatement([entry("reconciled","withdrawn")],[],read.scope,"me","channel_admission","channel-1",hash),undefined);
 assert.equal(actionStatementNote({phase:"refused"}),"notUsedNote");assert.equal(actionStatementNote({phase:"refused",recovers:"key-a"}),"unknownUseNote");
 assert.equal(actionStatementNote({phase:"unknown"}),"unknownUseNote");assert.equal(actionStatementNote({phase:"pending"}),"unknownUseNote");assert.equal(actionStatementNote({phase:"recorded"}),undefined);assert.equal(actionStatementNote({phase:"reconciled"}),undefined);
 const action=(phase:ProtectedEntry["phase"],id="s1"):ProtectedEntry=>({intent:{operation:"admit_channel",args:{attestation:id},scope:read.scope} as unknown as ProtectedWriteIntent,subject:"channel-1",phase,at:"2026-11-24T09:01:00Z"});
 assert.equal(statementRequestNote(entry("unknown"),[]),"stepNotSent");assert.equal(statementRequestNote(entry("pending"),[]),"stepNotSent");assert.equal(statementRequestNote(entry("refused"),[]),"stepNotSentRefused");
 assert.equal(statementRequestNote(entry("recorded"),[]),undefined);assert.equal(statementRequestNote(entry("recorded"),[action("recorded")]),"usedInSession");assert.equal(statementRequestNote(entry("recorded"),[action("reconciled")]),"usedInSession");
 for(const phase of ["refused","unknown","pending"] as const)assert.equal(statementRequestNote(entry("recorded"),[action(phase)]),undefined,phase);
 assert.equal(statementRequestNote(entry("recorded"),[action("recorded","other")]),undefined);
 const withdrawal:StatementEntry={...entry("unknown"),intent:{...entry("unknown").intent,action:"withdraw"}};assert.equal(statementRequestNote(withdrawal,[]),undefined);
});
test("each refusal code of the channel and statement steps has its own open-ended sentence",()=>{
 const codes=["permission_denied","self_approval_denied","evidence_stale","prerequisite_failed","invalid_request","unresolved_operation","reauthentication_required","recovery_read","dependency_unavailable"];
 for(const l of ["en","ko","ar"] as const){assert.equal(new Set(codes.map(code=>statementError(l,code))).size,codes.length);assert.equal(new Set(codes.map(code=>channelError(l,code))).size,codes.length);}
 assert.notEqual(channelError("en","invalid_setup_request"),channelError("en","invalid_accept"));assert.notEqual(statementError("en","invalid_text"),statementError("en","invalid_request"));
 for(const explain of [statementError,channelError])for(const code of ["permission_denied","evidence_stale","prerequisite_failed"])assert.ok(/may|Possible causes/.test(explain("en",code)),code);
 assert.ok(statementError("en","prerequisite_failed").includes("ten unused statements"));assert.ok(statementError("en","recovery_read").includes("unchanged retry does not need it"));
});
