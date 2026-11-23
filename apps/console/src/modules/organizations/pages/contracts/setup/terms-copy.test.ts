import test from "node:test";
import assert from "node:assert/strict";
import { netDays, termText } from "./terms-copy";
test("net days agree with their number in every language across the allowed range",()=>{
 assert.equal(netDays("en","0"),"0 days");assert.equal(netDays("en","1"),"1 day");assert.equal(netDays("en","30"),"30 days");assert.equal(netDays("en","365"),"365 days");
 assert.equal(netDays("ko","30"),"30일");
 const forms=new Set(["0","1","2","3","11","100","365"].map(n=>netDays("ar",n).replace(/[0-9\u0660-\u0669\s]/g,"")));
 assert.ok(forms.size>=3,[...forms].join("|"));
});
test("the unit word is not appended to the value",()=>{
 for(const l of ["en","ko","ar"] as const)assert.ok(!netDays(l,"1").includes(termText(l,"days")));
});
