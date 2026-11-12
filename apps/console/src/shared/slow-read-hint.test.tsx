import { test, mock } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import SlowReadHint, {
  SLOW_READ_DELAY_MS,
  scheduleSlowRead,
  slowReadText,
} from "./SlowReadHint";
test("the line is scheduled once and appears only when the delay has passed", () => {
  mock.timers.enable({ apis: ["setTimeout"] });
  try {
    let shown = 0;
    scheduleSlowRead(() => (shown += 1));
    mock.timers.tick(SLOW_READ_DELAY_MS - 1);
    assert.equal(shown, 0);
    mock.timers.tick(1);
    assert.equal(shown, 1);
    mock.timers.tick(SLOW_READ_DELAY_MS * 3);
    assert.equal(shown, 1);
  } finally {
    mock.timers.reset();
  }
});
test("a cancelled schedule never shows the line", () => {
  mock.timers.enable({ apis: ["setTimeout"] });
  try {
    let shown = 0;
    const cancel = scheduleSlowRead(() => (shown += 1));
    mock.timers.tick(SLOW_READ_DELAY_MS - 1);
    cancel();
    mock.timers.tick(SLOW_READ_DELAY_MS);
    assert.equal(shown, 0);
  } finally {
    mock.timers.reset();
  }
});
test("the first render is empty and the wording names no duration or error", () => {
  assert.equal(renderToStaticMarkup(<SlowReadHint locale="en" />), "");
  for (const locale of ["en", "ko", "ar"] as const) {
    const line = slowReadText(locale);
    assert.ok(line.length > 20);
    assert.doesNotMatch(
      line,
      /[\d\u0660-\u0669]|second|minute|초|분 |ثانية|دقيقة|error|fail|busy|오류|실패|장애|خطأ|فشل|تعذ/i,
    );
  }
  assert.match(slowReadText("ko"), /조회/);
  assert.match(slowReadText("en"), /quer/);
  assert.match(slowReadText("ar"), /استعلام/);
});
