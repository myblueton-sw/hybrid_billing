import test from "node:test";
import assert from "node:assert/strict";
import { SetupError } from "./context";
import { termError } from "./terms-copy";
import { approveBindingIntent, approveTermIntent, bindTermIntent, netDaysInput, recordTermIntent, termOffers } from "./term-flow";
import type { PaymentTerm, PaymentTerms, TermBinding } from "./terms";
const hash="a".repeat(64),session={user:"me",workspace:"workspace",csrf:"csrf"},attempt={request:"request-1",key:"key-1"};
const term=(change:Partial<PaymentTerm>={}):PaymentTerm=>({state:"pending",observation:"pending",id:"term-1",digest:hash,revision:"1",netDays:"30",mode:"calendar",author:"other",approver:"",pendingApprover:"",writtenAt:"2026-11-21T01:00:00Z",approvedAt:"",...change});
const approved=term({state:"approved",observation:"approved",approver:"third",approvedAt:"2026-11-21T02:00:00Z"});
const row=(binding:string,change:Partial<TermBinding>={}):TermBinding=>({binding,digest:hash,origin:"configuration",configuration:"configuration-a",term:"term-1",term_revision:"1",net_days:"30",mode:"calendar",zone:"UTC",from:"2026-11-01",to:"",reason:"initial",state:"pending",observation:"pending",author:"other",approver:"",pending_approver:"",written_at:"2026-11-21T03:00:00Z",approved_at:"",...change});
const page=(t:PaymentTerm|null,rows:TermBinding[]=[]):PaymentTerms=>({revision:"r7",at:"2026-11-24T09:00:00Z",scope:{contract:"contract-a",configuration:"configuration-a",digest:hash},zone:"UTC",term:t,snapshot:hash,next:"",count:rows.length,rows,currentEligibility:"unchecked"});
const refused=(run:()=>unknown)=>assert.throws(run,(e:unknown)=>e instanceof SetupError&&e.code==="invalid_setup_request");
test("each step is offered only in the state that admits it, and a recorded own act is a fact, not an offer",()=>{
 assert.deepEqual(termOffers(page(null),"me"),{record:true,approveTerm:"none",apply:false,bindings:{}});
 assert.equal(termOffers(page(term()),"me").approveTerm,"offer");assert.equal(termOffers(page(term({author:"me"})),"me").approveTerm,"own");
 for(const t of [term({state:"registration_unknown",observation:"absent",writtenAt:""}),term({state:"approval_unknown",pendingApprover:"me"}),approved])assert.equal(termOffers(page(t),"me").approveTerm,"none");
 assert.equal(termOffers(page(term()),"me").apply,false);assert.equal(termOffers(page(term({state:"approval_unknown",observation:"approved",approver:"third",approvedAt:"2026-11-21T02:00:00Z"})),"me").apply,false);assert.equal(termOffers(page(approved),"me").apply,true);
 const offers=termOffers(page(approved,[row("a"),row("b",{author:"me"}),row("c",{configuration:"configuration-z"}),row("d",{origin:"profile",configuration:""}),row("e",{state:"approval_unknown",pending_approver:"me"}),row("f",{state:"approved",observation:"approved",approver:"third",approved_at:"2026-11-21T04:00:00Z"}),row("g",{state:"registration_unknown",observation:"absent",written_at:""})]),"me").bindings;
 assert.deepEqual(offers,{a:"offer",b:"own",c:"none",d:"none",e:"none",f:"none",g:"none"});
});
test("net days are a whole number from 0 to 365 in canonical form and nothing is assumed",()=>{
 assert.equal(netDaysInput("30"),"30");assert.equal(netDaysInput(" 030 "),"30");assert.equal(netDaysInput("0"),"0");assert.equal(netDaysInput("365"),"365");
 for(const bad of ["","366","-1","3.5","30d","1e2","٣٠"," "])refused(()=>netDaysInput(bad));
});
test("recording sends the entered days and reason against the read revision and is refused once terms exist",()=>{
 const intent=recordTermIntent(session,page(null),"term-new",{days:"45",reason:" Agreed in the contract "},attempt);
 assert.deepEqual({operation:intent.operation,args:intent.args,rationale:intent.rationale,revision:intent.revision,scope:intent.scope,actor:intent.actor},{operation:"record_term",args:{term:"term-new",net_days:"45"},rationale:"Agreed in the contract",revision:"r7",scope:page(null).scope,actor:"me"});
 refused(()=>recordTermIntent(session,page(null),"term-new",{days:"45",reason:"  "},attempt));refused(()=>recordTermIntent(session,page(null),"term-new",{reason:"Agreed"},attempt));
 refused(()=>recordTermIntent(session,page(term()),"term-new",{days:"45",reason:"Agreed"},attempt));
});
test("approval names the retained record and its digest and is never built for the recorder",()=>{
 assert.deepEqual(approveTermIntent(session,page(term()),attempt).args,{term:"term-1",term_digest:hash});
 refused(()=>approveTermIntent(session,page(term({author:"me"})),attempt));refused(()=>approveTermIntent(session,page(approved),attempt));refused(()=>approveTermIntent(session,page(null),attempt));
 const rows=[row("a"),row("b",{author:"me"})],p=page(approved,rows);
 assert.deepEqual(approveBindingIntent(session,p,rows[0],attempt).args,{binding:"a",binding_digest:hash});
 refused(()=>approveBindingIntent(session,p,rows[1],attempt));refused(()=>approveBindingIntent(session,p,row("a"),attempt));
});
test("applying requires approved terms, an explicit end choice and a closed reason",()=>{
 const p=page(approved);
 assert.deepEqual(bindTermIntent(session,p,"binding-new",{from:"2026-11-01",end:"open",to:"",reason:"initial"},attempt).args,{binding:"binding-new",term:"term-1",term_digest:hash,from:"2026-11-01",reason:"initial"});
 assert.equal(bindTermIntent(session,p,"binding-new",{from:"2026-11-01",end:"date",to:"2027-10-31",reason:"renegotiation"},attempt).args.to,"2027-10-31");
 for(const bad of [{from:"2026-11-01",to:"",reason:"initial"},{from:"2026-11-01",end:"date",to:"",reason:"initial"},{from:"2026-11-01",end:"date",to:"2026-10-31",reason:"initial"},{from:"2026-11-01",end:"open",to:"2027-01-01",reason:"initial"},{from:"2026-13-01",end:"open",to:"",reason:"initial"},{from:"2026-11-01",end:"open",to:"",reason:"other"},{from:"2026-11-01",end:"open",to:""}])refused(()=>bindTermIntent(session,p,"binding-new",bad as Record<string,string>,attempt));
 refused(()=>bindTermIntent(session,page(term()),"binding-new",{from:"2026-11-01",end:"open",to:"",reason:"initial"},attempt));
});
test("every intent carries the read revision, the scope and the session, and an application of other settings or of a billing profile is never approved from here",()=>{
 const rows=[row("a"),row("c",{configuration:"configuration-z"}),row("d",{origin:"profile",configuration:""})],p=page(approved,rows);
 for(const intent of [approveTermIntent(session,page(term()),attempt),bindTermIntent(session,p,"binding-new",{from:"2026-11-01",end:"open",to:"",reason:"initial"},attempt),approveBindingIntent(session,p,rows[0],attempt)]){
  assert.deepEqual({revision:intent.revision,scope:intent.scope,actor:intent.actor,workspace:intent.workspace,csrf:intent.sessionCSRF,rationale:intent.rationale,attempt:intent.attempt},{revision:"r7",scope:p.scope,actor:"me",workspace:"workspace",csrf:"csrf",rationale:"",attempt});
 }
 refused(()=>approveBindingIntent(session,p,rows[1],attempt));refused(()=>approveBindingIntent(session,p,rows[2],attempt));
});
test("a refusal sentence never names a single cause the client cannot know, and each code has its own sentence",()=>{
 const codes=["permission_denied","self_approval_denied","evidence_stale","revision_conflict","prerequisite_failed","invalid_setup_request","invalid_apply","unresolved_operation","reauthentication_required","recovery_read","session_capacity","dependency_unavailable"];
 const en=codes.map(code=>termError("en",code));
 assert.equal(new Set(en).size,codes.length-1);assert.equal(termError("en","evidence_stale"),termError("en","revision_conflict"));
 for(const code of ["permission_denied","evidence_stale","prerequisite_failed"])assert.ok(/may|Possible causes/.test(termError("en",code)),code);
 assert.ok(termError("en","prerequisite_failed").includes("already exist"));assert.ok(termError("en","permission_denied").includes("does not disclose"));
 for(const l of ["ko","ar"] as const)assert.equal(new Set(codes.map(code=>termError(l,code))).size,codes.length-1);
});
