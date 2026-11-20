import { test } from "node:test";
import assert from "node:assert/strict";
import { licensePageOf, readLicense, signedEnvelopeOf, licenseSignedFields } from "./licenses";
import { AdminError } from "./admin";
const status = {mode:"evaluation",state:"unverified",reason:"time_unavailable",warning:true,actions:["register","contact"],remaining_seconds:0,last_outcome:"",last_verified:"",last_verified_at:"",entitlement:null,evaluation:null};
const page = {revision:"r1",observed_at:"2026-11-20T00:00:00Z",state:"configured",installation:"synthetic",product:"hybrid-billing",time_authority:"not_connected",status};
test("unverified state retains uncertainty and unconfigured states disclose no injected status",()=>{
 assert.deepEqual(licensePageOf(page),page);
 assert.deepEqual(licensePageOf({...page,state:"not_configured",status:{state:"commercial_active"}}),{revision:page.revision,observed_at:page.observed_at,state:"not_configured"});
 assert.equal(licensePageOf({...page,state:"uninitialized"}).state,"uninitialized");
});
test("malformed enums and incomplete active rights never become valid status",()=>{
 for(const change of [{state:"future"},{reason:"future"},{mode:"future"},{actions:["grant"]},{remaining_seconds:-1},{remaining_seconds:"0"},{state:"evaluation_active",reason:"",remaining_seconds:10},{state:"commercial_active",mode:"commercial",reason:""},{state:"noncommercial",mode:"noncommercial",reason:"",evaluation:{}}])assert.throws(()=>licensePageOf({...page,status:{...status,...change}}));
 for(const change of [{revision:""},{observed_at:"not-a-date"},{product:"other"},{time_authority:"verified"},{status:null}])assert.throws(()=>licensePageOf({...page,...change}));
});
test("read parser rejects malformed success rather than retaining prior data",async()=>{
 let path="";
 await assert.rejects(readLicense({read:async(p)=>{path=p;return {...page,status:{state:"commercial_active"}};}}),e=>e instanceof AdminError && e.failure==="invalid");
 assert.equal(path,"/licenses");
});
test("canonical signed bytes split without changing input and consent terms stay exact",()=>{
 const body=btoa("x".repeat(4096)),signature=btoa("s".repeat(64));
 const parsed=signedEnvelopeOf(JSON.stringify({body,signature,terms:"evaluation-v1"}));
 const fields=licenseSignedFields(parsed,true);
 assert.equal(fields.body.length,4096);assert.equal(fields.body+fields.body_tail,body);assert.equal(fields.signature,signature);assert.equal(fields.terms,"evaluation-v1");
 assert.equal(licenseSignedFields(parsed,false).terms,undefined);
 assert.throws(()=>licenseSignedFields({body,signature},true));
});
test("oversized and noncanonical files are rejected before submission",()=>{
 const base={body:btoa("body"),signature:btoa("s".repeat(64))};
 for(const change of [{body:""},{body:base.body+"\n"},{body:btoa("x".repeat(4097))},{signature:btoa("s".repeat(63))},{terms:""},{terms:"\u0000"},{terms:"가".repeat(67)},{private_key:"submitted-secret"},{extra:true}])assert.throws(()=>signedEnvelopeOf(JSON.stringify({...base,...change})));
 for(const raw of ["{", "null", "[]", " ".repeat(12001)])assert.throws(()=>signedEnvelopeOf(raw));
});
const evaluation={grant:"g1",subject:"subject-1",terms:"evaluation-v1",scope:["core"],start:"2026-11-01T00:00:00Z",end:"2026-11-15T00:00:00Z"};
const entitlement={id:"e1",subject_id:"subject-1",subject:"Synthetic subject",agreement:"a1",scope:["core"],channel:"direct",metric:"installations",start:1,end:2,perpetual:false};
test("subsecond active evaluation stays active even with zero displayed seconds",()=>{
 for(const state of ["evaluation_active","evaluation_expired"]){const result=licensePageOf({...page,status:{...status,state,reason:"",evaluation}});assert.equal(result.state,"configured");}
});
test("producer-valid commercial periods and perpetual records are accepted",()=>{
 for(const state of ["commercial_active","commercial_expired"]){assert.equal(licensePageOf({...page,status:{...status,mode:"commercial",state,reason:"",entitlement}}).state,"configured");}
 assert.equal(licensePageOf({...page,status:{...status,mode:"commercial",state:"commercial_active",reason:"",entitlement:{...entitlement,perpetual:true,end:0}}}).state,"configured");
});
test("impossible dates and malformed rights intervals IDs and scopes are rejected",()=>{
 assert.throws(()=>licensePageOf({...page,observed_at:"2026-02-30T00:00:00Z"}));
 for(const change of [{start:2,end:1},{perpetual:true,end:2},{start:0},{end:253402300800},{id:"has space"},{scope:[]},{scope:["z","a"]},{subject:"\u0000"}])assert.throws(()=>licensePageOf({...page,status:{...status,mode:"commercial",state:"commercial_active",reason:"",entitlement:{...entitlement,...change}}}));
 assert.throws(()=>licensePageOf({...page,status:{...status,state:"evaluation_active",reason:"",evaluation:{...evaluation,end:"2026-11-14T00:00:00Z"}}}));
});

test("issuer terms references preserve exact permitted surrounding spaces",()=>{
 const envelope=signedEnvelopeOf(JSON.stringify({body:btoa("body"),signature:btoa("s".repeat(64)),terms:" v1 "}));
 assert.equal(licenseSignedFields(envelope,true).terms," v1 ");
});
