import test from "node:test";
import assert from "node:assert/strict";
import { duration, rulesError, rulesText } from "./rules-copy";
import { channelError } from "./channels-copy";
import { changeError } from "./changes-copy";
import { termError } from "./terms-copy";
import { stepError, stepText } from "./step-copy";
import { writeError } from "./write-copy";
test("a duration is shown in the largest whole unit and never rounded",()=>{
 assert.equal(duration("en","86400"),"1 day");assert.equal(duration("en","172800"),"2 days");assert.equal(duration("en","90000"),"25 hours");
 assert.equal(duration("en","5400"),"90 minutes");assert.equal(duration("en","3601"),"3,601 seconds");
 assert.equal(duration("ko","604800"),"7일");assert.ok(duration("ar","3600").length>0);
});
test("every rules label exists in three languages",()=>{
 for(const l of ["en","ko","ar"] as const)for(const key of ["purpose","installation","none","unowned","denied","prerequisite","observed","pinned"] as const)assert.ok(rulesText(l,key).length>0);
});
test("the request limit of the page has its own sentence for every kind of setup write",()=>{
 for(const l of ["en","ko","ar"] as const){
  const limit=stepText(l,"attemptCapacity");
  assert.notEqual(limit,stepError(l,"some_other_code"));
  for(const explain of [rulesError,channelError,changeError,termError,stepError])assert.equal(explain(l,"attempt_capacity"),limit);
  assert.equal(writeError(l,"attempt_capacity"),limit);assert.notEqual(writeError(l,"attempt_capacity"),writeError(l,"some_other_code"));
  assert.ok(limit.includes("100"));
 }
});
