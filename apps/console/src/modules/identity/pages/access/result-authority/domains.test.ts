import test from "node:test";
import assert from "node:assert/strict";
import {AuthorityError,capabilities,issuanceDomains,resultDomain,type AuthorityDomain,type Envelope} from "./contract";
import {historyPage,contextPage} from "./reads";
import {inspectionOf} from "./inspection";
import {freezeIntent,mutationTransport,type Intent} from "./mutations";
import {readTransport} from "./transport";
import {AuthorityReads} from "./read-store";
import {AuthorityAttempts,sessionAuthorityAttempts} from "./attempts";
import {reviewDecision} from "./actions";
import {reviewProposal} from "./proposal-model";
import {proposalContext} from "./proposal-context";
const hash="a".repeat(64),now="2026-11-22T09:00:00Z",later="2026-11-22T10:00:00Z";
const session={user:"governor",workspace:"workspace",csrf:"synthetic",role:"admin",login:"synthetic"};
const domains:readonly AuthorityDomain[]=Object.values(issuanceDomains);
const policy=(d:AuthorityDomain)=>JSON.stringify({capabilities:d.capabilities,proposal_lifetime_seconds:"3600",max_grant_validity_seconds:"86400"});
const page=(fields:Record<string,string>):Envelope=>({revision:"r1",at:now,fields:{mode:"synthetic",workspace:"workspace",actor:"governor",current_eligibility:"unchecked",snapshot:hash,next:"",...fields}});
const row=(capability:string)=>({proposal:"p1",digest:hash,kind:"console_ceiling",capability,recipient:"console",state:"pending",proposer:"other",proposed_at:now,expires_at:later,decision_actor:""});
const sessionReply=()=>Response.json({user:session.user,workspace:session.workspace,csrf_token:session.csrf});
test("each issuance domain has its own namespace, grant prefix and closed capabilities",()=>{
 assert.deepEqual(domains.map(d=>d.namespace),["issuance_protected_authority","issuance_term_authority","issuance_draft_authority"]);
 assert.deepEqual(domains.map(d=>d.grantPrefix),["issuance-protected-grant-","issuance-term-grant-","issuance-draft-grant-"]);
 assert.deepEqual(issuanceDomains.protected.capabilities,["register_channel","admit_channel","request_change","verify_change","approve_change","inspect"]);
 assert.deepEqual(issuanceDomains.term.capabilities,["record","approve","bind","approve_binding","inspect"]);
 assert.deepEqual(issuanceDomains.draft.capabilities,["submit","approve","inspect"]);
 assert.deepEqual(resultDomain.capabilities,[...capabilities]);assert.ok(Object.isFrozen(issuanceDomains.term.capabilities));
});
test("reads use the domain's commands, capabilities, policy order and grant prefix",async()=>{
 for(const d of domains){
  const first=d.capabilities[0],commands:string[]=[];
  const history=await historyPage(async command=>{commands.push(command);return page({row_00:JSON.stringify(row(first))});},"workspace","governor",undefined,d);
  assert.equal(history.rows[0].capability,first);
  const ceiling={ceiling:`${d.grantPrefix}${hash}`,capability:first,not_before:now,not_after:later};
  const context=await contextPage(async command=>{commands.push(command);return page({collection:"ceilings",capability:first,policy:policy(d),row_00:JSON.stringify(ceiling)});},"workspace","governor","ceilings",first,undefined,d);
  assert.equal(context.rows.length,1);assert.deepEqual(commands,[`${d.namespace}.history`,`${d.namespace}.context`]);
  await assert.rejects(historyPage(async()=>page({row_00:JSON.stringify(row("apply"))}),"workspace","governor",undefined,d),AuthorityError);
  await assert.rejects(contextPage(async()=>page({}),"workspace","governor","ceilings","apply",undefined,d),e=>e instanceof AuthorityError&&e.code==="invalid_authority_request");
  for(const change of [{policy:policy(resultDomain)},{policy:JSON.stringify({capabilities:[...d.capabilities].reverse(),proposal_lifetime_seconds:"3600",max_grant_validity_seconds:"86400"})},{row_00:JSON.stringify({...ceiling,ceiling:`result-grant-${hash}`})},...domains.filter(o=>o!==d).map(o=>({row_00:JSON.stringify({...ceiling,ceiling:`${o.grantPrefix}${hash}`})}))])
   await assert.rejects(contextPage(async()=>page({collection:"ceilings",capability:first,policy:policy(d),row_00:JSON.stringify(ceiling),...change}),"workspace","governor","ceilings",first,undefined,d),AuthorityError);
 }
 await assert.rejects(historyPage(async()=>page({row_00:JSON.stringify(row("record"))}),"workspace","governor"),AuthorityError);
});
test("read transport sends only the domain's three read commands",async()=>{
 for(const d of domains){
  const urls:string[]=[];
  const read=readTransport(session,async(url,init)=>{
   if(String(url).endsWith("/session"))return sessionReply();
   urls.push(String(url));const body=JSON.parse(String(init?.body));
   return Response.json({request_id:body.request_id,status:"completed",target_id:body.target_id,result_revision:"r1",replay:false,as_of:now,fields:page({}).fields});
  },d);
  await read(`${d.namespace}.history`,"workspace",{});
  assert.deepEqual(urls,[`/api/v1/commands/${d.namespace}.history`]);
  for(const command of ["result_authority.history",`${d.namespace}.approve`,`${d.namespace}.historyx`,...domains.filter(o=>o!==d).map(o=>`${o.namespace}.history`)])await assert.rejects(read(command,"workspace",{}),e=>e instanceof AuthorityError&&e.code==="invalid_authority_request");
  assert.equal(urls.length,1);
 }
 await assert.rejects(readTransport(session,async()=>sessionReply())("issuance_term_authority.history","workspace",{}),AuthorityError);
});
const fieldsOf=(d:AuthorityDomain,capability:string,intent:Intent)=>({mode:"synthetic",receipt:"historical_decision",proposal:"proposal",digest:hash,state:"pending",kind:"console_ceiling",capability,recipient:"console",ceiling:"",proposer:session.user,not_before:intent.fields.not_before,not_after:intent.fields.not_after,proposed_at:"2026-11-22T08:59:00Z",expires_at:"2026-11-22T09:59:00Z",rationale:intent.rationale});
test("mutations post to the domain's namespace and bind the receipt to its grant prefix",async()=>{
 for(const d of domains){
  const first=d.capabilities[0];
  const intent:Intent={action:"propose_console",target:"proposal",fields:{capability:first,not_before:now,not_after:later},rationale:"Synthetic review",revision:"r1",attempt:{request:"request",key:"key"}};
  const reply=(fields:Record<string,string>,state:string)=>({request_id:"request",status:"completed",operation_id:"operation",target_id:"proposal",result_revision:"r2",business_state:state,replay:false,as_of:now,fields});
  const urls:string[]=[];
  const submit=mutationTransport(session,async url=>{if(String(url).endsWith("/session"))return sessionReply();urls.push(String(url));return Response.json(reply(fieldsOf(d,first,intent),"pending"));},d);
  assert.equal((await submit(intent)).inspection.row.capability,first);assert.deepEqual(urls,[`/api/v1/commands/${d.namespace}.propose_console`]);
  assert.throws(()=>freezeIntent({...intent,fields:{...intent.fields,capability:"apply"}},d),AuthorityError);
  assert.throws(()=>freezeIntent(intent),AuthorityError);
  const human={...intent,action:"propose_human" as const,fields:{...intent.fields,recipient:"member",ceiling:`${d.grantPrefix}${hash}`}};
  assert.ok(Object.isFrozen(freezeIntent(human,d)));
  for(const prefix of ["result-grant-",...domains.filter(o=>o!==d).map(o=>o.grantPrefix)])assert.throws(()=>freezeIntent({...human,fields:{...human.fields,ceiling:`${prefix}${hash}`}},d),AuthorityError);
  const decision:Intent={...intent,action:"approve",fields:{digest:hash}};
  const applied=(grant:string)=>({...fieldsOf(d,first,intent),proposer:"other",state:"applied",decided_at:"2026-11-22T09:00:00Z",decided_by:session.user,decision_reason:decision.rationale,grant,recorded_grant_status:"active"});
  const approve=(grant:string)=>mutationTransport(session,async url=>String(url).endsWith("/session")?sessionReply():Response.json(reply(applied(grant),"applied")),d)(decision);
  assert.equal((await approve(`${d.grantPrefix}${hash}`)).inspection.decision?.grant,`${d.grantPrefix}${hash}`);
  await assert.rejects(approve(`result-grant-${hash}`),e=>e instanceof AuthorityError&&e.effect==="unknown"&&e.code==="invalid_authority_receipt");
  const shared={...applied(`${d.grantPrefix}${hash}`),capability:"inspect"};
  assert.equal(inspectionOf({revision:"r2",at:now,fields:shared},"proposal",hash,d).namespace,d.namespace);
  assert.throws(()=>inspectionOf({revision:"r2",at:now,fields:shared},"proposal",hash),AuthorityError);
 }
});
test("stores keep each domain's reads, attempts and reviews apart",async()=>{
 for(const d of domains){
  const first=d.capabilities[0],commands:string[]=[];
  const store=new AuthorityReads(async command=>{commands.push(command);return command.endsWith(".history")?page({row_00:JSON.stringify(row(first))}):{revision:"r1",at:now,fields:{...fieldsOf(d,first,{action:"propose_console",target:"p1",fields:{not_before:now,not_after:later},rationale:"Synthetic review",revision:"r1",attempt:{request:"r",key:"k"}}),proposal:"p1",proposer:"other"}};},"workspace","governor",d);
  await store.history();await store.inspect("p1",hash);await store.context("recipients",first);
  assert.deepEqual(commands,[`${d.namespace}.history`,`${d.namespace}.inspect`,`${d.namespace}.context`]);
  const detail=store.snapshot().detail.value!;assert.equal(detail.row.capability,first);
  const review=reviewDecision(detail,"governor","approve","Synthetic decision",{request:"request-2",key:"key-2"},d);
  const sent:Intent[]=[];
  const attempts=new AuthorityAttempts(async intent=>{sent.push(intent);throw new AuthorityError("permission_denied");},"governor",d);
  await attempts.decide(review);assert.equal(sent.length,1);assert.equal(attempts.snapshot()[0].phase,"refused");
  assert.equal(sent[0].namespace,d.namespace);
  for(const other of [resultDomain,...domains.filter(o=>o!==d)]){
   assert.throws(()=>reviewDecision(detail,"governor","approve","Synthetic decision",{request:"request-4",key:"key-4"},other),(e:unknown)=>e instanceof AuthorityError&&e.code==="decision_review_unavailable");
   assert.throws(()=>new AuthorityAttempts(async()=>{throw new Error();},"governor",other).decide(review),AuthorityError);
   const shared:Intent={action:"propose_console",target:"proposal-2",fields:{capability:"inspect",not_before:now,not_after:later},rationale:"Synthetic review",revision:"r1",attempt:{request:"request-3",key:"key-3"}};
   const bound=freezeIntent(shared,d);assert.equal(bound.namespace,d.namespace);
   assert.throws(()=>freezeIntent(bound,other),(e:unknown)=>e instanceof AuthorityError&&e.code==="invalid_authority_request");
   assert.throws(()=>new AuthorityAttempts(async()=>{throw new Error();},"governor",other).propose(bound),(e:unknown)=>e instanceof AuthorityError&&e.code==="invalid_authority_request");
   await assert.rejects(mutationTransport(session,async()=>{throw new Error("no request may be sent");},other)(bound),(e:unknown)=>e instanceof AuthorityError&&e.code==="invalid_authority_request");
  }
  const proposals:Intent[]=[];
  const accepted=new AuthorityAttempts(async intent=>{proposals.push(intent);throw new AuthorityError("permission_denied");},"governor",d);
  await accepted.propose({action:"propose_console",target:"proposal-5",fields:{capability:"inspect",not_before:now,not_after:later},rationale:"Synthetic review",revision:"r1",attempt:{request:"request-5",key:"key-5"}});
  assert.equal(proposals[0].namespace,d.namespace);
  const ceiling={ceiling:`${d.grantPrefix}${hash}`,capability:"inspect",not_before:now,not_after:"2026-11-22T12:00:00Z"};
  const read=async(_command:string,_target:string,fields:Record<string,string>)=>page({collection:fields.collection,capability:"inspect",policy:policy(d),row_00:JSON.stringify(fields.collection==="ceilings"?ceiling:{user:"member",login:"member@example.invalid"})});
  const context=await proposalContext(read,"workspace","governor","human_grant","inspect",{},d);
  assert.equal(context.ceilings.namespace,d.namespace);
  const draft={recipient:"member",ceiling:ceiling.ceiling,notBefore:now,notAfter:later,rationale:"Synthetic review"};
  const proposal=reviewProposal(context,draft,"proposal-6",{request:"request-6",key:"key-6"},d);
  assert.equal(proposal.intent.namespace,d.namespace);assert.equal(proposal.intent.fields.ceiling,ceiling.ceiling);
  for(const other of [resultDomain,...domains.filter(o=>o!==d)])assert.throws(()=>reviewProposal(context,draft,"proposal-6",{request:"request-6",key:"key-6"},other),AuthorityError);
 }
 const stores=new Set([sessionAuthorityAttempts(session),...domains.map(d=>sessionAuthorityAttempts(session,d))]);
 assert.equal(stores.size,4);assert.equal(sessionAuthorityAttempts(session,issuanceDomains.term),sessionAuthorityAttempts(session,issuanceDomains.term));
});
