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
const { summarySentence, CleanupGroups } = await import("./SessionOperations");
const unknown = { state:"unconfirmed",ended_count:null,cleanup:null } as SessionOperation;
for (const l of ["en","ko","ar"] as const) {
  test(`${l} preview rows retain inactive states and directional session references`, () => {
    const html=renderToStaticMarkup(<SelectionRows l={l} rows={[{handle:"abcdef0123456789",current:false,signed_in:"2026-11-20T00:00:00Z",ends:"2026-11-20T12:00:00Z",state:"unavailable",can_end:false}]} />);
    assert.ok(html.includes('dir="ltr"'));assert.ok(html.includes(sq.unavailable(l)));assert.ok(html.includes("abcdef0123456789"));
    assert.equal((html.match(/<tr>/g)??[]).length,2);
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
