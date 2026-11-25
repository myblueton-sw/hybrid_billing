import test from "node:test";
import assert from "node:assert/strict";
import {instant,policyOf,historyOf,AuthorityError,capabilities} from "./contract";
import {validityProblems} from "./validity";
import {strictJSON,StrictJSONError} from "../../../../../service/strict-json";
import {strictJSON as documentJSON} from "../../../../issuance/pages/approval-inbox/documents/json";
import {DocumentError} from "../../../../issuance/pages/approval-inbox/documents/error";
const policy={capabilities,proposal_lifetime_seconds:"3600",max_grant_validity_seconds:"86400"};
test("UTC instant preserves nanosecond differences and equal fractional spellings",()=>{
 for(const second of ["0001-01-01T00:00:00","1969-12-31T23:59:59","2000-02-29T23:59:59","2026-11-22T09:00:00","9999-12-31T23:59:59"]){
  const base=instant(second+"Z")!;assert.equal(typeof base,"bigint");
  for(const fraction of [1,9,10,100,1000,999999,1000000,499999999,500000000,999999999]){
   const encoded=String(fraction).padStart(9,"0");assert.equal(instant(`${second}.${encoded}Z`)!-base,BigInt(fraction));
   assert.equal(instant(`${second}.${encoded.replace(/0+$/,"")}Z`),instant(`${second}.${encoded}Z`));
  }
 }
 for(const invalid of ["2026-02-29T00:00:00Z","2026-11-22T24:00:00Z","2026-11-22T00:00:60Z","2026-11-22T00:00:00+09:00","2026-11-22T00:00:00.1234567890Z","0000-01-01T00:00:00Z"])assert.equal(instant(invalid),undefined);
});
test("validity containment rejects a single nanosecond beyond either boundary",()=>{
 for(let n=1;n<=999;n+=7){
  const start=`2026-11-22T09:00:00.${String(n).padStart(9,"0")}Z`,end=`2026-11-23T09:00:00.${String(n).padStart(9,"0")}Z`;
  const ceiling={ceiling:"result-grant-"+"a".repeat(64),capability:"apply" as const,not_before:start,not_after:end};
  assert.deepEqual(validityProblems(start,end,policy,ceiling),[]);
  assert.ok(validityProblems(`2026-11-22T09:00:00.${String(n-1).padStart(9,"0")}Z`,end,policy,ceiling).includes("not_before"));
  assert.ok(validityProblems(start,`2026-11-23T09:00:00.${String(n+1).padStart(9,"0")}Z`,policy,ceiling).includes("not_after"));
 }
});
test("closed authority wire rejects duplicate keys and coerced kinds",()=>{
 assert.throws(()=>policyOf('{"capabilities":[],"capabilities":[]}'),AuthorityError);
 const row={proposal:"p",digest:"a".repeat(64),kind:["console_ceiling"],capability:"apply",recipient:"console",state:"pending",proposer:"governor",proposed_at:"2026-11-22T09:00:00Z",expires_at:"2026-11-22T10:00:00Z",decision_actor:""};
 assert.throws(()=>historyOf(JSON.stringify(row)),AuthorityError);
 assert.deepEqual(policyOf(JSON.stringify(policy)).capabilities,capabilities);
});
test("shared strict JSON extraction preserves the document error boundary",()=>{
 for(const invalid of ['{"a":1,"a":2}','{"a":"\\ud800"}','[1,]','{"x":1e400}']){
  assert.throws(()=>strictJSON(invalid),StrictJSONError);assert.throws(()=>documentJSON(invalid),DocumentError);
 }
 for(const value of [{a:[1,true,null,"مرحبا","한국어"]},[0,-2,"text"],"value"]){const raw=JSON.stringify(value);assert.equal(JSON.stringify(strictJSON(raw)),raw);assert.equal(JSON.stringify(documentJSON(raw)),raw);}
});
