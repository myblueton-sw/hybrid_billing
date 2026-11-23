import test from "node:test";
import assert from "node:assert/strict";
import type { SetupReference } from "./attestation";
import { SetupError } from "./context";
import type { ProtectedChange, ProtectedChannel } from "./protected";
import { changeError } from "./changes-copy";
import { decideChangeIntent, decideOffer, effectiveInput, requestChangeIntent, usableChannels, verifyChangeIntent, verifyOffer } from "./change-flow";
const hash="a".repeat(64),other="b".repeat(64),session={user:"me",workspace:"workspace",csrf:"csrf"},attempt={request:"request-1",key:"key-1"};
const read={scope:{contract:"contract-a",configuration:"configuration-a",digest:hash},revision:"r4"};
const change=(c:Partial<ProtectedChange>={}):ProtectedChange=>({change:"change-1",digest:hash,field:"payment_account",phase:"recorded",state:"requested",observation:"requested",provenance:"staff_entry",instruction_kind:"in_person",channel:"channel-1",rules:"current",requester:"other",requested_at:"2026-11-22T04:00:00Z",effective:"2026-12-01T00:00:00Z",approvals_required:"2",verifier:"",verification_outcome:"",verified_at:"",pending_verifier:"",pending_decision_actor:"",ended_by:"",ended_at:"",decisions:[],...c});
const verified={state:"confirmed",observation:"confirmed",verifier:"third",verification_outcome:"confirmed",verified_at:"2026-11-22T05:00:00Z"};
const channel=(id:string,c:Partial<ProtectedChannel>={}):ProtectedChannel=>({channel:id,digest:other,subject_kind:"entity",kind:"portal",state:"admitted",observation:"admitted",registrar:"r",admitter:"a",pending_admitter:"",registered_at:"2026-11-20T01:00:00Z",admitted_at:"2026-11-20T02:00:00Z",retired_by:"",retired_at:"",rules:"current",request_not_before:"2026-11-21T02:00:00Z",...c});
const channels=[channel("channel-1"),channel("channel-2",{kind:"email"}),channel("channel-3",{subject_kind:"customer"}),channel("channel-4",{state:"registered",observation:"registered",admitter:"",admitted_at:"",request_not_before:""}),channel("channel-5",{observation:"retired",retired_by:"x",retired_at:"2026-11-23T01:00:00Z",request_not_before:""})];
const references:SetupReference[]=[{code:"synthetic_account_a",use:"payment_account",applicability:"synthetic_only"},{code:"synthetic_account_b",use:"payment_account",applicability:"synthetic_only"},{code:"synthetic_recipient_a",use:"document_recipient",applicability:"synthetic_only"},{code:"synthetic_channel_entity_a",use:"channel_entity",applicability:"synthetic_only"}];
const refused=(run:()=>unknown)=>assert.throws(run,(e:unknown)=>e instanceof SetupError&&e.code==="invalid_setup_request");
const draft={reference:"synthetic_account_a",channel:"channel-1",provenance:"staff_entry",instruction:"in_person",effective:"2026-12-01T09:30",reason:" Customer asked for a new account "};
test("verification and decision are offered only in the recorded state that admits them, and a recorded own act is a fact",()=>{
 assert.equal(verifyOffer(change(),"me"),"offer");assert.equal(verifyOffer(change({requester:"me"}),"me"),"own");
 for(const row of [change({phase:"request_unknown",state:"",observation:"absent",requested_at:"",approvals_required:""}),change({phase:"verification_unknown",pending_verifier:"me"}),change({observation:"lapsed",ended_at:"2026-12-01T00:00:00Z"}),change(verified)])assert.equal(verifyOffer(row,"me"),"none");
 assert.equal(decideOffer(change(verified),"me"),"offer");assert.equal(decideOffer(change({...verified,requester:"me"}),"me"),"requester");assert.equal(decideOffer(change({...verified,verifier:"me"}),"me"),"verifier");
 assert.equal(decideOffer(change({...verified,decisions:[{approver:"me",verdict:"approve",at:"2026-11-22T06:00:00Z"}]}),"me"),"decided");assert.equal(decideOffer(change({...verified,decisions:[{approver:"fourth",verdict:"approve",at:"2026-11-22T06:00:00Z"}]}),"me"),"offer");
 for(const row of [change(),change({...verified,phase:"decision_unknown",pending_decision_actor:"me"}),change({...verified,state:"scheduled",observation:"scheduled"}),change({...verified,observation:"lapsed",ended_at:"2026-12-01T00:00:00Z"})])assert.equal(decideOffer(row,"me"),"none");
});
test("only accepted, unretired channels of the setting's subject can be chosen, and the effective time is a UTC minute",()=>{
 assert.deepEqual(usableChannels(channels,"payment_account").map(c=>c.channel),["channel-1","channel-2"]);assert.deepEqual(usableChannels(channels,"document_recipient").map(c=>c.channel),["channel-3"]);assert.deepEqual(usableChannels(channels,"document_template"),[]);
 assert.equal(effectiveInput("2026-12-01T09:30"),"2026-12-01T09:30:00Z");assert.equal(effectiveInput(" 2026-12-01T00:00 "),"2026-12-01T00:00:00Z");
 for(const bad of ["","2026-12-01","2026-12-01T09:30:00","2026-02-30T09:30","2026-12-01T24:00","2026-12-01 09:30","2026-12-01T09:30Z"])refused(()=>effectiveInput(bad));
});
test("a change request names the sample of its setting, the chosen channel with its digest, and an instruction form with exactly the channel it needs",()=>{
 const intent=requestChangeIntent(session,read,"r5","change-new","statement-1","payment_account",draft,references,channels,attempt);
 assert.deepEqual({operation:intent.operation,args:intent.args,rationale:intent.rationale,revision:intent.revision,scope:intent.scope,actor:intent.actor},{operation:"request_change",args:{change:"change-new",field:"payment_account",reference:"synthetic_account_a",channel:"channel-1",channel_digest:other,provenance:"staff_entry",instruction:"in_person",effective:"2026-12-01T09:30:00Z",attestation:"statement-1"},rationale:"Customer asked for a new account",revision:"r5",scope:read.scope,actor:"me"});
 assert.equal(requestChangeIntent(session,read,"r5","change-new","statement-1","payment_account",{...draft,instruction:"registered_channel",instructionChannel:"channel-2"},references,channels,attempt).args.instruction_channel,"channel-2");
 assert.equal(requestChangeIntent(session,read,"r5","change-new","statement-1","payment_account",{...draft,instruction:"unregistered_sender"},references,channels,attempt).args.instruction_channel,undefined);
 for(const c of [{reference:"synthetic_recipient_a"},{reference:"synthetic_channel_entity_a"},{reference:""},{channel:"channel-3"},{channel:"channel-4"},{channel:"channel-5"},{channel:""},{provenance:"assistant"},{provenance:""},{instruction:""},{instruction:"registered_channel"},{instruction:"registered_channel",instructionChannel:"channel-1"},{instruction:"registered_channel",instructionChannel:"channel-3"},{instruction:"in_person",instructionChannel:"channel-2"},{effective:"2026-12-01"},{reason:" "}])refused(()=>requestChangeIntent(session,read,"r5","change-new","statement-1","payment_account",{...draft,...c} as Record<string,string>,references,channels,attempt));
 refused(()=>requestChangeIntent(session,read,"r5","change-new","","payment_account",draft,references,channels,attempt));
});
test("verification names the retained change, its digest and a closed outcome and is never built for the requester",()=>{
 const intent=verifyChangeIntent(session,read,"r6",change(),"statement-2","confirmed",attempt);
 assert.deepEqual({operation:intent.operation,args:intent.args,rationale:intent.rationale,revision:intent.revision},{operation:"verify_change",args:{change:"change-1",change_digest:hash,outcome:"confirmed",attestation:"statement-2"},rationale:"",revision:"r6"});
 assert.equal(verifyChangeIntent(session,read,"r6",change(),"statement-2","denied",attempt).args.outcome,"denied");
 refused(()=>verifyChangeIntent(session,read,"r6",change(),"statement-2","",attempt));refused(()=>verifyChangeIntent(session,read,"r6",change(),"statement-2","verified",attempt));
 refused(()=>verifyChangeIntent(session,read,"r6",change({requester:"me"}),"statement-2","confirmed",attempt));refused(()=>verifyChangeIntent(session,read,"r6",change(verified),"statement-2","confirmed",attempt));
});
test("a decision carries a closed verdict, a closed reason only when rejecting, no statement, and is never built for a person the service excludes",()=>{
 const row=change(verified);
 assert.deepEqual(decideChangeIntent(session,read,row,{verdict:"approve"},attempt).args,{change:"change-1",change_digest:hash,verdict:"approve"});
 const reject=decideChangeIntent(session,read,row,{verdict:"reject",rejection:"missing_evidence"},attempt);
 assert.deepEqual({args:reject.args,rationale:reject.rationale,revision:reject.revision},{args:{change:"change-1",change_digest:hash,verdict:"reject",reason:"missing_evidence"},rationale:"",revision:"r4"});
 for(const d of [{},{verdict:"reject"},{verdict:"reject",rejection:"because"},{verdict:"approve",rejection:"other"},{verdict:"maybe"}])refused(()=>decideChangeIntent(session,read,row,d as Record<string,string>,attempt));
 for(const excluded of [change({...verified,requester:"me"}),change({...verified,verifier:"me"}),change({...verified,decisions:[{approver:"me",verdict:"approve",at:"2026-11-22T06:00:00Z"}]}),change()])refused(()=>decideChangeIntent(session,read,excluded,{verdict:"approve"},attempt));
});
test("a channel the session user registered or accepted is not offered, and a request through it is not built",()=>{
 const mine=[channel("channel-1",{registrar:"me"}),channel("channel-2",{admitter:"me"}),channel("channel-6")];
 assert.deepEqual(usableChannels(mine,"payment_account","me").map(c=>c.channel),["channel-6"]);assert.deepEqual(usableChannels(mine,"payment_account").map(c=>c.channel),["channel-1","channel-2","channel-6"]);
 refused(()=>requestChangeIntent(session,read,"r5","change-new","statement-1","payment_account",draft,references,mine,attempt));
 assert.equal(requestChangeIntent(session,read,"r5","change-new","statement-1","payment_account",{...draft,channel:"channel-6"},references,mine,attempt).args.channel,"channel-6");
 assert.equal(requestChangeIntent(session,read,"r5","change-new","statement-1","payment_account",{...draft,channel:"channel-6",instruction:"registered_channel",instructionChannel:"channel-2"},references,mine,attempt).args.instruction_channel,"channel-2","an instruction that arrived through the user's own channel must be recordable");
 refused(()=>requestChangeIntent(session,read,"r5","change-new","statement-1","payment_account",{...draft,channel:"channel-2",instruction:"registered_channel",instructionChannel:"channel-6"},references,mine,attempt));
 refused(()=>requestChangeIntent(session,read,"r5","change-new","statement-1","payment_account",{...draft,channel:"channel-6",instruction:"registered_channel",instructionChannel:"channel-6"},references,mine,attempt));
});
test("every change intent carries the session, and each refusal code has its own open-ended sentence",()=>{
 const row=change(verified);
 for(const intent of [requestChangeIntent(session,read,"r5","change-new","statement-1","payment_account",draft,references,channels,attempt),verifyChangeIntent(session,read,"r6",change(),"statement-2","confirmed",attempt),decideChangeIntent(session,read,row,{verdict:"approve"},attempt)])assert.deepEqual({workspace:intent.workspace,csrf:intent.sessionCSRF,actor:intent.actor,scope:intent.scope,attempt:intent.attempt},{workspace:"workspace",csrf:"csrf",actor:"me",scope:read.scope,attempt});
 const codes=["permission_denied","self_approval_denied","evidence_stale","prerequisite_failed","invalid_request","invalid_setup_request","invalid_verify","invalid_decide","unresolved_operation","reauthentication_required","recovery_read","dependency_unavailable"];
 for(const l of ["en","ko","ar"] as const)assert.equal(new Set(codes.map(code=>changeError(l,code))).size,codes.length);
 for(const code of ["permission_denied","evidence_stale","prerequisite_failed"])assert.ok(/may|Possible causes/.test(changeError("en",code)),code);
 assert.ok(changeError("en","self_approval_denied").includes("registered or accepted the verification channel"));assert.equal(changeError("en","revision_conflict"),changeError("en","evidence_stale"));
});
