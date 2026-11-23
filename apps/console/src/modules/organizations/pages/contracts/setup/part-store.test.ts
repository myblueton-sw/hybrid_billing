import test from "node:test";
import assert from "node:assert/strict";
import { SetupError } from "./context";
import { PartReadStore } from "./part-store";
function deferred<T>(){let resolve!:(value:T)=>void,reject!:(error:unknown)=>void;const promise=new Promise<T>((a,b)=>{resolve=a;reject=b;});return {promise,resolve,reject};}
test("a reload clears the shown value and a late reply never replaces a newer read",async()=>{
 const first=deferred<string>();let calls=0;
 const store=new PartReadStore(()=>++calls===1?first.promise:Promise.resolve("new"));
 const old=store.load();assert.deepEqual(store.getSnapshot(),{loading:true});
 await store.load();first.resolve("old");await old;
 assert.deepEqual(store.getSnapshot(),{loading:false,value:"new"});
});
test("a refusal keeps its code, an unexpected failure is unavailable, and a failed reload leaves no old value",async()=>{
 let step=0;
 const store=new PartReadStore<string>(async()=>{step++;if(step===1)return "value";if(step===2)throw new SetupError("permission_denied");throw new Error("boom");});
 await store.load();assert.equal(store.getSnapshot().value,"value");
 await store.load();assert.deepEqual(store.getSnapshot(),{loading:false,error:"permission_denied"});
 await store.load();assert.deepEqual(store.getSnapshot(),{loading:false,error:"service_unavailable"});
});
test("a disposed store ignores the reply of a read in flight",async()=>{
 const pending=deferred<string>(),store=new PartReadStore(()=>pending.promise);let calls=0;store.subscribe(()=>{calls++;});
 const read=store.load();store.dispose();pending.resolve("late");await read;
 assert.deepEqual(store.getSnapshot(),{loading:false});assert.equal(calls,1);
});
