import test from "node:test";
import assert from "node:assert/strict";
import {AuthorityAttempts} from "./attempts";
import {AuthorityError} from "./contract";
import {reviewDecision} from "./actions";
import type {Inspection} from "./inspection";
import type {Intent,Receipt} from "./mutations";
const hash="a".repeat(64),at="2026-11-22T09:00:00Z";
const detail:Inspection={row:{proposal:"proposal",digest:hash,kind:"console_ceiling",capability:"apply",recipient:"console",state:"decision_unknown",proposer:"proposer",proposed_at:at,expires_at:"2026-11-22T10:00:00Z",decision_actor:"reviewer"},ceiling:"",notBefore:at,notAfter:"2026-11-22T12:00:00Z",rationale:"Proposal",revision:"r2",at,decision:{at,actor:"reviewer",rationale:"Original decision",grant:`result-grant-${hash}`,recordedGrantStatus:""}};
const intent:Intent={action:"propose_console",target:"proposal",fields:{capability:"apply",not_before:at,not_after:detail.notAfter},rationale:"Proposal",revision:"r1",attempt:{request:"request",key:"key"}};
const receipt:Receipt={inspection:{...detail,row:{...detail.row,state:"applied"}},operation:"operation",replay:false};
test("unknown attempts survive clean retry refusal and block replacement proposals",async()=>{
 let calls=0;
 const store=new AuthorityAttempts(async()=>{calls++;throw new AuthorityError(calls===1?"service_unavailable":"permission_denied",calls===1?"unknown":"none");},"reviewer");
 await store.propose(intent);assert.equal(store.snapshot()[0].phase,"unknown");
 await store.retry("key");assert.equal(store.snapshot()[0].phase,"unknown");
 assert.throws(()=>store.propose({...intent,target:"new-proposal",attempt:{request:"new",key:"new"}}),AuthorityError);assert.equal(calls,2);
});
test("duplicate submission while pending dispatches only once",async()=>{
 let release!:(receipt:Receipt)=>void,calls=0;
 const store=new AuthorityAttempts(async()=>{calls++;return new Promise(resolve=>{release=resolve;});},"reviewer");
 const pending=store.propose(intent);
 assert.throws(()=>store.propose(intent),AuthorityError);
 await store.retry("key");assert.equal(calls,1);release(receipt);await pending;assert.equal(store.snapshot()[0].phase,"recorded");
});
test("fresh-key recovery settles the matching unknown decision without rewriting the old attempt",async()=>{
 let calls=0;
 const store=new AuthorityAttempts(async()=>{if(++calls===1)throw new AuthorityError("service_unavailable","unknown");return receipt;},"reviewer");
 const pending={...detail,row:{...detail.row,state:"pending" as const,decision_actor:""},decision:undefined};
 await store.decide(reviewDecision(pending,"reviewer","approve","Original decision",{request:"request",key:"key"}));
 await store.decide(reviewDecision(detail,"reviewer","recover","Original decision",{request:"recover",key:"recover"}));
 const old=store.snapshot().find(e=>e.intent.attempt.key==="key")!;
 assert.equal(old.phase,"unknown");assert.equal(store.settled(old),true);await store.retry("key");assert.equal(calls,2);
 assert.throws(()=>new AuthorityAttempts(async()=>receipt,"other").decide(reviewDecision(detail,"reviewer","recover","Original decision",{request:"foreign",key:"foreign"})),AuthorityError);
});
