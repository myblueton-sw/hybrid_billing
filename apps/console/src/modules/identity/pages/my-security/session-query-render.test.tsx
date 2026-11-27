import { test } from "node:test";
import assert from "node:assert/strict";
import React from "react";
import * as nodeModule from "node:module";
import { renderToStaticMarkup } from "react-dom/server";
import { sq } from "./query-copy";
import type { SessionOperation } from "../../../../service/session-query";

if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url,context,next) => /\.(css|svg)(\?.*)?$/.test(url) ? { format:"module",source:"export default ''",shortCircuit:true } : next(url,context) });
const { SelectionRows } = await import("./SessionSelection");
const { SessionTimeHint } = await import("./LiveSessions");
const { summarySentence, CleanupGroups, operationFailure } = await import("./SessionOperations");
const unknown = { state:"unconfirmed",ended_count:null,cleanup:null } as SessionOperation;
for (const l of ["en","ko","ar"] as const) {
  test(`${l} preview rows retain inactive states and directional session references`, () => {
    const html=renderToStaticMarkup(<SelectionRows l={l} rows={[{handle:"abcdef0123456789",current:false,signed_in:"2026-11-20T00:00:00Z",ends:"2026-11-20T12:00:00Z",state:"unavailable",can_end: false, not_included_operation: ""}]} />);
    assert.ok(html.includes('dir="ltr"'));assert.ok(html.includes(sq.unavailable(l)));assert.ok(html.includes("abcdef0123456789"));
    assert.equal((html.match(/<tr>/g)??[]).length,2);
    assert.ok(html.includes('<time dateTime="2026-11-20T00:00:00Z">') && html.includes('<time dateTime="2026-11-20T12:00:00Z">'));
    assert.ok(!/<bdi[^>]*>[^<]*2026[^<]*<\/bdi>/.test(html));
  });
  test(`${l} unconfirmed summary never asserts zero effects or completed cleanup`, () => { assert.equal(summarySentence(l,unknown),({en:"Result not confirmed",ko:"결과 미확인",ar:"لم تُؤكَّد النتيجة"})[l]); });
}
test("count labels use invariant headings for singular and larger populations", () => {
  for (const n of [0,1,2,3,11,170]) {
    for (const label of [sq.shown("en",n),sq.recorded("en",n),sq.totalOps("en",n),sq.viewCount("en",n)]) assert.doesNotMatch(label,/\b1 (rows|targets|operations)\b/);
    for (const label of [sq.shown("ar",n),sq.recorded("ar",n),sq.totalOps("ar",n)]) assert.match(label,/: /);
  }
  assert.match(sq.fullScope("en"),/other sessions/);
});

for (const l of ["en","ko","ar"] as const) test(`${l} cleanup groups retain outside-scope and already-cleared distinctions`, () => {
  const html = renderToStaticMarkup(<CleanupGroups l={l} row={{ handle:"a",signed_in:"",result:"ended",items:[{item:"C1",state:"done",reason:"access_blocked"},{item:"C3",state:"already_ended",reason:"already_closed"},{item:"C6",state:"outside_scope",reason:"in_flight_requests_finish"},{item:"C8",state:"not_applicable",reason:"no_jobs_or_api_keys"}] }} />);
  assert.equal((html.match(/<section>/g) ?? []).length,3);
  assert.equal((html.match(/<li>/g) ?? []).length,4);
  assert.ok(html.includes(({en:"Already cleared",ko:"이미 정리됨",ar:"أُغلقت مسبقًا"})[l]));
});

for (const l of ["en","ko","ar"] as const) test(`${l} exact ISO timezone example is directionally isolated`, () => {
  const html = renderToStaticMarkup(<SessionTimeHint l={l} />);
  assert.ok(html.includes('<bdi dir="ltr">2026-11-20T09:00:00+09:00</bdi>'));
  assert.equal((html.match(/2026-11-20T09:00:00/g) ?? []).length,1);
});

const { SessionSettings, PreparationExpiry, limitCopy } = await import("./SessionSettings");
for (const l of ["en","ko","ar"] as const) {
  test(`${l} limits distinguish effective configured value, default, range and unknown source`, () => {
    const html=renderToStaticMarkup(<SessionSettings l={l} settings={[{name:"session_query_page_size",value:7,unit:"rows",default:50,min:1,max:100,source:"start_flag"},{name:"session_query_scan",value:200,unit:"sessions",default:1000,min:100,max:10000,source:"future_source"}]} />);
    assert.ok(html.includes('<strong><bdi dir="ltr">7</bdi></strong>'));
    assert.ok(html.includes('<bdi dir="ltr">50</bdi>'));
    assert.ok(html.includes('<bdi dir="ltr">1–100</bdi>'));
    assert.ok(html.includes(limitCopy.configured(l)) && html.includes(limitCopy.unknownSource(l)));
    assert.ok(!html.includes(limitCopy.source(l)+" "+limitCopy.sourceDefault(l)+"</span>"));
    assert.ok(!html.includes('<input') && !html.includes('<button'));
    assert.equal((html.match(/<dd>/g)??[]).length,3);
    assert.ok(html.includes(limitCopy.unavailable(l)));
  });
  test(`${l} absent effective settings never synthesize defaults or raw unknown keys`, () => {
    const html=renderToStaticMarkup(<SessionSettings l={l} settings={[{name:"private_unknown_key",value:999,unit:"",default:50,min:1,max:100,source:"default"}]} />);
    assert.ok(html.includes(limitCopy.unavailable(l)));
    assert.ok(!html.includes('999') && !html.includes('private_unknown_key'));
  });
  test(`${l} preparation expiry preserves exact server nanoseconds and isolated offset`, () => {
    const at="2026-11-20T09:00:00.123456789+09:00";
    const html=renderToStaticMarkup(<PreparationExpiry l={l} at={at} />);
    assert.ok(html.includes('<bdi dir="ltr">'+at+'</bdi>'));
    assert.ok(html.includes(limitCopy.expiryNote(l)));
  });
}

for (const l of ["en","ko","ar"] as const) test(`${l} preparation names all six limits even when their values are missing`, () => {
  const html=renderToStaticMarkup(<SessionSettings l={l} settings={[]} scope="preparation" />);
  assert.equal((html.match(/<dd>/g)??[]).length,6);
  assert.equal(html.split(limitCopy.unavailable(l)).length-1,6);
  assert.ok(!html.includes('<strong>') && !html.includes(limitCopy.source(l)));
});

const { ExclusionTag } = await import("./LiveSessions");
for (const l of ["en","ko","ar"] as const) test(`${l} session exclusion tag requires an explicit operation reference`, () => {
  const absent=renderToStaticMarkup(<ExclusionTag l={l} reference="" onOperation={()=>{}} />);
  assert.equal(absent,"");
  const present=renderToStaticMarkup(<ExclusionTag l={l} reference="session-end-recorded" onOperation={()=>{}} />);
  assert.ok(present.includes(sq.notIncluded(l)) && present.includes(sq.viewOperation(l)));
  assert.ok(!present.includes('started after') && !present.includes('still active'));
});
test("a referenced operation that is missing reads as unavailable and any other failed read as unread", async () => {
  const { AdminError, failureOf } = await import("../../../../service/admin");
  assert.equal(operationFailure(failureOf(404, { code: "unsupported_command" })), "unavailable");
  assert.equal(operationFailure(new AdminError("invalid", [], 0, "", "operation_mismatch")), "unavailable");
  for (const e of [new AdminError("unavailable"), failureOf(503, { code: "dependency_unavailable" }), failureOf(429, {}), new AdminError("invalid"), new Error("x")]) assert.equal(operationFailure(e), "unread");
});
