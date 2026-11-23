import assert from "node:assert/strict";
import test from "node:test";
import {accessPhrase,actionLabel,fieldLabel,kindLabel} from "./live-copy";
import {capabilityLabel,issuanceAreas} from "./result-authority/areas";

const actions:Record<string,Record<string,string>>={
 "issuance-protected":{register_channel:"register_issuance_channel",admit_channel:"admit_issuance_channel",request_change:"request_issuance_protected_change",verify_change:"verify_issuance_protected_change",approve_change:"approve_issuance_protected_change"},
 "issuance-terms":{record:"record_issuance_term",approve:"approve_issuance_term",bind:"bind_issuance_term",approve_binding:"approve_issuance_term_binding"},
 "issuance-drafts":{submit:"submit_issuance_draft",approve:"approve_issuance_draft"},
};
const locales=["en","ko","ar"] as const;

test("every issuance action is labelled with the capability name of its permission area",()=>{
 let count=0;
 for(const area of issuanceAreas)for(const [capability,action] of Object.entries(actions[area.param]))for(const l of locales){
  assert.ok(actionLabel(l,action));
  assert.equal(actionLabel(l,action),capabilityLabel(l,area,capability));
  count++;
 }
 assert.equal(count,33);
 for(const area of issuanceAreas)assert.deepEqual(Object.keys(area.capabilities).filter(c=>c!=="inspect").sort(),Object.keys(actions[area.param]).sort());
});

test("issuance resource kinds and evidence fields are labelled in every language",()=>{
 for(const l of locales){
  const seen=new Set<string>();
  for(const kind of ["synthetic_issuance_protected","synthetic_issuance_terms","synthetic_issuance_draft"]){const label=kindLabel(l,kind);assert.ok(label&&label!==kind);seen.add(label);}
  for(const field of ["issuance_protected","issuance_terms","issuance_draft"]){const label=fieldLabel(l,field);assert.ok(label&&label!==field);seen.add(label);}
  assert.equal(seen.size,6);
 }
});

test("an issuance inspect grant stays out of the generic request form",()=>{
 for(const [kind,field] of [["synthetic_issuance_protected","issuance_protected"],["synthetic_issuance_terms","issuance_terms"],["synthetic_issuance_draft","issuance_draft"]])for(const l of locales)assert.equal(accessPhrase(l,"view_cost",kind,[field]),null);
});
