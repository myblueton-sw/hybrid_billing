import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import {
  Missing,
  SourceAmount,
  SourceInstant,
  SourcePeriod,
  SourceQuantity,
  SourceText,
} from "./SourceValues";
const text = (markup: string) =>
  markup
    .replace(/<[^>]+>/g, "\n")
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
const html = (node: Parameters<typeof renderToStaticMarkup>[0]) =>
  renderToStaticMarkup(node);
test("a missing value reads Missing alone in each language", () => {
  assert.deepEqual(text(html(<Missing locale="en" />)), ["Missing"]);
  assert.deepEqual(text(html(<SourceText value={null} locale="ko" />)), [
    "누락",
  ]);
  for (const locale of ["en", "ko", "ar"] as const) {
    const markup = html(
      <SourceAmount
        value={null}
        currency="USD"
        unit={null}
        note
        locale={locale}
      />,
    );
    assert.equal(text(markup).length, 1);
    assert.ok(!/USD|—|\d/.test(text(markup)[0]));
    assert.ok(markup.includes('data-source-value="missing"'));
  }
});
test("an amount keeps its sign, currency and exact digits", () => {
  assert.deepEqual(
    text(
      html(
        <SourceAmount
          value="-2613700000000000000"
          currency="USD"
          locale="en"
        />,
      ),
    ),
    ["USD -2.6137"],
  );
  assert.deepEqual(
    text(
      html(
        <SourceAmount
          value="1000000000000000001"
          currency="CNY"
          note
          locale="en"
        />,
      ),
    ),
    ["CNY 1.000000000000000001"],
  );
});
test("exact zero is labeled only when a note is requested", () => {
  assert.deepEqual(
    text(html(<SourceAmount value="0" currency="USD" note locale="en" />)),
    ["USD 0.00", "Exact zero"],
  );
  assert.deepEqual(
    text(html(<SourceAmount value="0" currency="USD" locale="en" />)),
    ["USD 0.00"],
  );
  assert.deepEqual(
    text(html(<SourceAmount value="0" currency="USD" note locale="ko" />)),
    ["USD 0.00", "정확한 0"],
  );
});
test("a unit price shows its unit or states that the unit is missing", () => {
  assert.deepEqual(
    text(
      html(
        <SourceAmount
          value="750000000000000000"
          currency="USD"
          unit="Hours"
          locale="en"
        />,
      ),
    ),
    ["USD 0.75 / Hours"],
  );
  for (const unit of [null, "", "   "])
    assert.deepEqual(
      text(
        html(
          <SourceAmount
            value="750000000000000000"
            currency="USD"
            unit={unit}
            locale="en"
          />,
        ),
      ),
      ["USD 0.75", "Unit missing"],
    );
});
test("a quantity states missing value, missing unit and exact zero separately", () => {
  assert.deepEqual(
    text(
      html(
        <SourceQuantity
          value="8000000000000000000"
          unit="OCPU Hours"
          locale="en"
        />,
      ),
    ),
    ["8.00 OCPU Hours"],
  );
  assert.deepEqual(
    text(html(<SourceQuantity value={null} unit="GB" locale="en" />)),
    ["Missing"],
  );
  assert.deepEqual(
    text(html(<SourceQuantity value={null} unit={null} locale="en" />)),
    ["Missing", "Unit missing"],
  );
  assert.deepEqual(
    text(html(<SourceQuantity value="0" unit={null} note locale="en" />)),
    ["0.00", "Unit missing", "Exact zero"],
  );
});
test("a UTC instant is shortened and anything else is shown as collected", () => {
  assert.deepEqual(
    text(html(<SourceInstant value="2024-09-30T22:00:00+00:00" locale="en" />)),
    ["2024-09-30 22:00:00", "UTC"],
  );
  assert.deepEqual(
    text(html(<SourceInstant value="2024-09-30T22:00:00Z" locale="en" />)),
    ["2024-09-30T22:00:00Z", "Shown as collected; not converted to UTC"],
  );
  assert.deepEqual(
    text(
      html(
        <SourcePeriod
          start="2024-09-01T00:00:00+00:00"
          end="2024-09-02T00:00:00+00:00"
          locale="en"
        />,
      ),
    ),
    [
      "Start",
      "2024-09-01 00:00:00",
      "UTC",
      "End",
      "2024-09-02 00:00:00",
      "UTC",
    ],
  );
});
