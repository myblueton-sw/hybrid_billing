import test from "node:test";
import assert from "node:assert/strict";
import {AuthorityError} from "./contract";
import {reviewActions,reviewDecision} from "./actions";
import type {Inspection} from "./inspection";
const hash="a".repeat(64),at="2026-11-22T09:00:00Z";
const detail:Inspection={row:{proposal:"proposal",digest:hash,kind:"human_grant",capability:"apply",recipient:"member",state:"pending",proposer:"proposer",proposed_at:at,expires_at:"2026-11-22T10:00:00Z",decision_actor:""},ceiling:`result-grant-${hash}`,notBefore:at,notAfter:"2026-11-22T12:00:00Z",rationale:"Proposal review",revision:"r1",at};
const attempt={request:"request",key:"key"};
test("historical actor rules expose review controls without asserting current eligibility",()=>{
 assert.deepEqual(reviewActions(detail,"proposer"),["withdraw"]);
 assert.deepEqual(reviewActions(detail,"member"),[]);
 assert.deepEqual(reviewActions(detail,"reviewer"),["approve","reject"]);
 for(const state of ["applying","applied","rejected","withdrawn","expired","superseded"] as const)assert.deepEqual(reviewActions({...detail,row:{...detail.row,state}},"reviewer"),[]);
 assert.throws(()=>reviewDecision(detail,"proposer","approve","Review",attempt),AuthorityError);
 const reviewed=reviewDecision(detail,"reviewer","approve","Review",attempt);assert.equal(reviewed.intent.revision,"r1");assert.equal(reviewed.intent.fields.digest,hash);assert.ok(Object.isFrozen(reviewed.detail.row));
});
test("unknown approval recovery belongs only to the original decision actor and rationale",()=>{
 const unknown:Inspection={...detail,row:{...detail.row,state:"decision_unknown",decision_actor:"reviewer"},decision:{at,actor:"reviewer",rationale:"Original decision",grant:`result-grant-${hash}`,recordedGrantStatus:""}};
 assert.deepEqual(reviewActions(unknown,"reviewer"),["recover"]);
 for(const actor of ["proposer","member","other"])assert.deepEqual(reviewActions(unknown,actor),[]);
 for(const action of ["approve","reject","withdraw"] as const)assert.throws(()=>reviewDecision(unknown,"reviewer",action,"Original decision",attempt),AuthorityError);
 assert.throws(()=>reviewDecision(unknown,"reviewer","recover","Changed decision",attempt),AuthorityError);
 const recovered=reviewDecision(unknown,"reviewer","recover","Original decision",attempt);assert.equal(recovered.intent.action,"approve");assert.equal(recovered.intent.target,detail.row.proposal);assert.equal(recovered.intent.fields.digest,hash);
});
