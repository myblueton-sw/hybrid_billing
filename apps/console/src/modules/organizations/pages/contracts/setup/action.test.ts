import test from "node:test";
import assert from "node:assert/strict";
import { SetupError } from "./context";
import { testAction } from "./write-fixture";
const cases=(values:Record<string,string>[])=>values;
test("action decoder accepts only the closed permission/step-up/outcome combinations",()=>{
 for(const fields of cases([{permission:"denied",step_up:"unchecked",action:"blocked",code:"permission_denied"},{permission:"unavailable",step_up:"unchecked",action:"blocked",code:"authority_unavailable"},{step_up:"unchecked",action:"blocked",code:"installation_held"},{step_up:"reauthentication_required",action:"blocked",code:"reauthentication_required"},{action:"recorded"},{action:"blocked",code:"configuration_exists"}]))assert.ok(testAction(undefined,"r4",fields));
 for(const fields of cases([{permission:"denied"},{step_up:"unchecked"},{action:"ready"},{current_eligibility:"eligible"},{action:"recorded",code:"permission_denied"},{action:"blocked",code:"invented"},{configuration:"other"},{contract:"other"}]))assert.throws(()=>testAction(undefined,"r4",fields),SetupError);
});
