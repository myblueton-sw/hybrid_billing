import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {areaOf,authorityAreas,capabilityLabel,issuanceAreas,resultArea,words} from "./areas";
import {issuanceDomains,resultDomain} from "./contract";
import {label} from "./copy";
const locales=["en","ko","ar"] as const;
test("the area parameter selects one area and anything else selects none",()=>{
 assert.deepEqual(authorityAreas.map(area=>area.param),["result","issuance-protected","issuance-terms","issuance-drafts"]);
 for(const area of authorityAreas)assert.equal(areaOf(area.param),area);
 for(const value of [null,"","Result","issuance","issuance-term","issuance_protected","__proto__","constructor"])assert.equal(areaOf(value),undefined);
 assert.deepEqual(issuanceAreas.map(area=>area.domain),[issuanceDomains.protected,issuanceDomains.term,issuanceDomains.draft]);
 assert.equal(resultArea.domain,resultDomain);assert.ok(Object.isFrozen(authorityAreas[1].capabilities));
});
test("every capability of an area has its own label in three languages",()=>{
 for(const area of authorityAreas){
  assert.deepEqual(Object.keys(area.capabilities),[...area.domain.capabilities]);
  for(const [index,l] of locales.entries()){
   const labels=area.domain.capabilities.map(value=>capabilityLabel(l,area,value));
   assert.equal(new Set(labels).size,labels.length);
   for(const [i,value] of area.domain.capabilities.entries()){assert.equal(labels[i],area.capabilities[value][index]);assert.notEqual(labels[i],value);assert.ok(labels[i].trim());}
   for(const text of [words(l,area.title),words(l,area.intro),words(l,area.refusal)])assert.ok(text.trim()&&!text.includes("${")&&!text.includes("undefined"));
  }
  assert.equal(capabilityLabel("en",area,"unknown_capability"),"unknown_capability");assert.equal(capabilityLabel("en",area,"constructor"),"constructor");
 }
});
test("shared capability names are labelled by their area and not as actions",()=>{
 const [protectedArea,terms,drafts]=issuanceAreas;
 for(const l of locales){
  const inspect=[resultArea,protectedArea,terms,drafts].map(area=>capabilityLabel(l,area,"inspect"));
  assert.equal(new Set(inspect).size,4);
  assert.notEqual(capabilityLabel(l,terms,"approve"),capabilityLabel(l,drafts,"approve"));
  for(const area of [terms,drafts])assert.notEqual(capabilityLabel(l,area,"approve"),label(l,"approve"));
 }
});
test("result copy and identifiers are unchanged and issuance areas have their own",()=>{
 assert.equal(words("en",resultArea.title),"ERP result permissions");assert.equal(words("ko",resultArea.title),"ERP 결과 권한");
 assert.equal(resultArea.item,"result-authority");
 for(const value of resultDomain.capabilities)for(const l of locales)assert.equal(capabilityLabel(l,resultArea,value),label(l,value));
 assert.deepEqual(issuanceAreas.map(area=>area.item),["issuance-protected-authority","issuance-terms-authority","issuance-drafts-authority"]);
 for(const area of issuanceAreas){assert.equal(area.item,`${area.param}-authority`);for(const l of locales)assert.equal(words(l,area.intro).includes("24"),false);}
 assert.equal(new Set(authorityAreas.flatMap(area=>locales.map(l=>words(l,area.title)))).size,12);
});
test("the panel and the navigation take the area instead of result literals",()=>{
 const read=(name:string)=>readFileSync(new URL(name,import.meta.url),"utf8");
 const screen=read("./Screen.tsx"),proposal=read("./Proposal.tsx"),detail=read("./Detail.tsx"),area=read("../Area.tsx"),css=read("./areas.css")+read("./screen.css");
 for(const source of [screen,proposal,detail]){assert.equal(/result_authority|result-grant-/.test(source),false);assert.equal(/label\(l,[a-z.]*capability\)/.test(source),false);}
 assert.ok(screen.includes("useAuthorityReads(session,area.domain)")&&screen.includes("useAuthorityAttempts(session,area.domain)"));
 assert.ok(proposal.includes("readTransport(session,undefined,area.domain)")&&proposal.includes("`${area.item}-${a.request}`")&&proposal.includes(",a,area.domain)")&&proposal.includes("cursors,area.domain)"));
 assert.equal(proposal.includes("result-authority-"),false);assert.ok(detail.includes("attempt(),area.domain)"));
 assert.ok(area.includes("area.param])")&&area.includes('aria-current={current?"page":undefined}')&&area.includes("authorityAreas.map"));
 assert.equal(/border[^;}]*:(?![^;}]*#d5d7da)[^;}]*#[0-9a-f]{3,6}/i.test(read("./areas.css")),false);
 assert.equal(css.includes("currentColor"),false);
});
