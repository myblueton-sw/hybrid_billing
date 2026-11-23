import test from "node:test";
import assert from "node:assert/strict";
import { sampleLabel, statementText } from "./statements-copy";
test("a sample label names its use and letter and never shows the code",()=>{
 assert.equal(sampleLabel("en",{code:"synthetic_recipient_a",use:"document_recipient"}),"Sample recipient A");
 assert.equal(sampleLabel("ko",{code:"synthetic_account_b",use:"payment_account"}),"샘플 결제 계좌 B");
 assert.equal(sampleLabel("ar",{code:"synthetic_channel_entity_a",use:"channel_entity"}),"قناة جهة مصدرة تجريبية A");
 assert.equal(sampleLabel("en",{code:"synthetic_channel_document_kind_b",use:"channel_document_kind"}),"Sample document kind channel B");
 assert.equal(sampleLabel("en",{code:"synthetic_unregistered_sender",use:"unregistered_sender"}),"Sample unregistered sender");
 for(const l of ["en","ko","ar"] as const)assert.ok(!sampleLabel(l,{code:"synthetic_template_a",use:"document_template"}).includes("synthetic"));
});
test("an unused or withdrawn statement is worded as supporting no record",()=>{
 for(const key of ["unconsumed","withdrawn"] as const){assert.ok(statementText("en",key).includes("supports no record"));assert.ok(statementText("ko",key).includes("뒷받침하지 않음"));assert.ok(statementText("ar",key).includes("لا تدعم أي سجل"));}
 assert.equal(statementText("en","consumed"),"Used");
});
