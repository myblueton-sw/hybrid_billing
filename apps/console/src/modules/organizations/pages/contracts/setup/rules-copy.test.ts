import test from "node:test";
import assert from "node:assert/strict";
import { duration, rulesText } from "./rules-copy";
test("a duration is shown in the largest whole unit and never rounded",()=>{
 assert.equal(duration("en","86400"),"1 day");assert.equal(duration("en","172800"),"2 days");assert.equal(duration("en","90000"),"25 hours");
 assert.equal(duration("en","5400"),"90 minutes");assert.equal(duration("en","3601"),"3,601 seconds");
 assert.equal(duration("ko","604800"),"7일");assert.ok(duration("ar","3600").length>0);
});
test("every rules label exists in three languages",()=>{
 for(const l of ["en","ko","ar"] as const)for(const key of ["purpose","installation","none","unowned","denied","prerequisite","observed","pinned"] as const)assert.ok(rulesText(l,key).length>0);
});
