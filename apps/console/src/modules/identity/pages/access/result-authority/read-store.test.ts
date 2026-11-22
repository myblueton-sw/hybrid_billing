import test from "node:test";
import assert from "node:assert/strict";
import {AuthorityReads} from "./read-store";
import type {Envelope} from "./contract";
const hash="a".repeat(64),at="2026-11-22T09:00:00Z";
const page=(snapshot=hash):Envelope=>({revision:"r1",at,fields:{mode:"synthetic",workspace:"workspace",actor:"actor",current_eligibility:"unchecked",snapshot,next:""}});
const inspection=(proposal:string):Envelope=>({revision:"r1",at,fields:{mode:"synthetic",receipt:"historical_decision",proposal,digest:hash,state:"pending",kind:"console_ceiling",capability:"apply",recipient:"console",ceiling:"",proposer:"proposer",not_before:at,not_after:"2026-11-22T12:00:00Z",proposed_at:at,expires_at:"2026-11-22T10:00:00Z",rationale:"Proposal"}});
test("late history response cannot replace the newer page",async()=>{
 const resolves:((e:Envelope)=>void)[]=[];
 const store=new AuthorityReads(async()=>new Promise(resolve=>resolves.push(resolve)),"workspace","actor");
 const old=store.history(),current=store.history();resolves[1](page("b".repeat(64)));await current;resolves[0](page());await old;
 assert.equal(store.snapshot().history.value?.snapshot,"b".repeat(64));
});
test("selection and disposal suppress old inspection results",async()=>{
 const resolves:((e:Envelope)=>void)[]=[];
 const store=new AuthorityReads(async()=>new Promise(resolve=>resolves.push(resolve)),"workspace","actor");
 const old=store.inspect("old"),current=store.inspect("current");resolves[1](inspection("current"));await current;resolves[0](inspection("old"));await old;
 assert.equal(store.snapshot().detail.value?.row.proposal,"current");
 const disposed=store.inspect("disposed");store.dispose();resolves[2](inspection("disposed"));await disposed;assert.equal(store.snapshot().detail.value,undefined);
});
test("mutation clears authority immediately and does not steal a later manual selection",async()=>{
 let release!:()=>void;const calls:string[]=[];
 const store=new AuthorityReads(async(command,target)=>{calls.push(target);return command.endsWith("history")?page():inspection(target);},"workspace","actor");
 await store.inspect("old");const work=new Promise<void>(resolve=>{release=resolve;});const pending=store.followMutation("old",work);
 assert.equal(store.snapshot().detail.value,undefined);await store.inspect("new");release();await pending;
 assert.equal(store.snapshot().selected,"new");assert.deepEqual(calls,["old","new"]);
});

test("disposed callbacks cannot dispatch reads but still await an existing mutation",async()=>{
 let calls=0,released=false;const store=new AuthorityReads(async()=>{calls++;return page();},"workspace","actor");store.dispose();
 await store.history();await store.inspect("proposal");await store.context("recipients","apply");
 let release!:()=>void;const pending=store.followMutation("proposal",new Promise<void>(resolve=>{release=()=>{released=true;resolve();};}));
 assert.equal(released,false);release();await pending;assert.equal(calls,0);
});
test("new context navigation suppresses mutation selection before and during automatic refresh",async()=>{
 for(const during of [false,true]){
  let releaseMutation!:()=>void,releaseHistory!:(e:Envelope)=>void;const calls:string[]=[];
  const store=new AuthorityReads(async(command,target)=>{
   calls.push(command);
   if(command.endsWith("history"))return new Promise<Envelope>(resolve=>{releaseHistory=resolve;});
   if(command.endsWith("context"))throw new Error("isolated unavailable context");
   return inspection(target);
  },"workspace","actor");
  const pending=store.followMutation("old",new Promise<void>(resolve=>{releaseMutation=resolve;}));
  if(during){releaseMutation();await Promise.resolve();}
  await store.context("ceilings","apply");
  if(during)releaseHistory(page());else releaseMutation();
  await pending;assert.equal(calls.includes("result_authority.inspect"),false);assert.equal(store.snapshot().collection,"ceilings");
 }
});
