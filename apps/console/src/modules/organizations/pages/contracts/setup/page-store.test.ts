import test from "node:test";
import assert from "node:assert/strict";
import { SetupError } from "./context";
import { PageReadStore } from "./page-store";
type Page={next:string;snapshot:string;rows:string[]};
function deferred<T>(){let resolve!:(value:T)=>void;const promise=new Promise<T>(done=>{resolve=done;});return {promise,resolve};}
test("the next page continues the same snapshot and a first-page load starts a new one",async()=>{
 const calls:string[][]=[];
 const store=new PageReadStore<Page>(async(after,snapshot)=>{calls.push([after,snapshot]);return after?{next:"",snapshot,rows:["k"]}:{next:"j",snapshot:"s1",rows:["a"]};});
 await store.load();await store.load(true);
 assert.deepEqual(calls,[["",""],["j","s1"]]);assert.equal(store.getSnapshot().after,"j");assert.deepEqual(store.getSnapshot().value?.rows,["k"]);
 await store.load(true);assert.equal(calls.length,2);
 await store.load();assert.deepEqual(calls[2],["",""]);assert.equal(store.getSnapshot().after,"");
});
test("a refused next page shows its code and no rows of the previous page",async()=>{
 let refuse=false;
 const store=new PageReadStore<Page>(async()=>{if(refuse)throw new SetupError("evidence_stale");return {next:"j",snapshot:"s1",rows:["a"]};});
 await store.load();refuse=true;await store.load(true);
 assert.deepEqual(store.getSnapshot(),{loading:false,after:"j",error:"evidence_stale"});
 await store.load(true);assert.equal(store.getSnapshot().error,"evidence_stale");
});
test("a late reply never replaces a newer read and a disposed store stays empty",async()=>{
 const old=deferred<Page>();let calls=0;
 const store=new PageReadStore<Page>(()=>++calls===1?old.promise:Promise.resolve({next:"",snapshot:"s2",rows:["new"]}));
 const first=store.load();await store.load();old.resolve({next:"",snapshot:"s1",rows:["old"]});await first;
 assert.deepEqual(store.getSnapshot().value?.rows,["new"]);
 const pending=deferred<Page>(),other=new PageReadStore<Page>(()=>pending.promise),read=other.load();other.dispose();pending.resolve({next:"",snapshot:"s",rows:[]});await read;
 assert.deepEqual(other.getSnapshot(),{loading:false,after:""});
});
