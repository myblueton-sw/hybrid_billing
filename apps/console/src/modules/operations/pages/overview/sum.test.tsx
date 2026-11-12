import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import Sum, { MissingCount } from "./Sum";
test("a sum without present values reads as a phrase without currency, digit or dash", () => {
  for (const [locale, phrase] of [
    ["en", "No values present"],
    ["ko", "값 없음"],
    ["ar", "لا قيم موجودة"],
  ] as const) {
    const html = renderToStaticMarkup(
      <Sum value={null} currency="USD" locale={locale} />,
    );
    assert.ok(html.includes(phrase));
    assert.ok(html.includes('data-u01-sum="none"'));
    assert.equal(html.replace(/<[^>]+>/g, ""), phrase);
    assert.doesNotMatch(html, /bdi|data-source-value|source-state/);
  }
});
test("a present sum keeps the shared signed format and is never a row state", () => {
  const html = renderToStaticMarkup(
    <Sum value="-2000000000000000000" currency="USD" locale="en" />,
  );
  assert.equal(html.replace(/<[^>]+>/g, ""), "USD -2.00");
  assert.doesNotMatch(html, /data-source-value|source-state/);
  const zero = renderToStaticMarkup(
    <Sum value="0" currency="USD" locale="en" />,
  );
  assert.equal(zero.replace(/<[^>]+>/g, ""), "USD 0.00");
  assert.doesNotMatch(zero, /data-source-value|Exact zero/);
});
test("the missing count appears only when a value is missing", () => {
  assert.equal(
    renderToStaticMarkup(<MissingCount count={0} locale="en" />),
    "",
  );
  assert.ok(
    renderToStaticMarkup(<MissingCount count={3} locale="ko" />).includes(
      "누락 3개",
    ),
  );
});
