import { test } from "node:test";
import assert from "node:assert/strict";
import { licenseSession } from "./live-session";
const session = () => ({user:"synthetic",workspace:"synthetic",csrf:"synthetic",role:"administrator",login:"admin@example.invalid"});
test("navigation shares a pending lock and uncertain outcome, while a new login is isolated",()=>{
 const identity=session(), origin=licenseSession(identity), remounted=licenseSession(identity), replacement=licenseSession(session());
 assert.equal(origin,remounted);
 assert.equal(origin.begin(),true);
 assert.equal(remounted.begin(),false);
 assert.equal(replacement.snapshot().pending,false);
 origin.settle({request:"request-1",operation:"operation-1",outcome:"",unknown:true});
 assert.equal(remounted.snapshot().result?.request,"request-1");
 assert.equal(remounted.begin(),false);
 assert.equal(replacement.snapshot().result,null);
});
test("settlement after the original view unsubscribes notifies the current view and advances its read generation",()=>{
 const controller=licenseSession(session());let old=0,current=0;
 const unsubscribe=controller.subscribe(()=>old++);
 const initial=controller.snapshot();assert.equal(initial,controller.snapshot());
 controller.begin();unsubscribe();controller.subscribe(()=>current++);
 controller.settle({request:"request-2",operation:"",outcome:"mode_selected",unknown:false});
 assert.equal(old,1);assert.equal(current,1);assert.equal(controller.snapshot().settled,1);
 assert.equal(controller.snapshot().pending,false);assert.equal(controller.begin(),true);
 assert.equal(controller.snapshot().result,null);
});
