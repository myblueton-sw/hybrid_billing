import { createRequire } from "node:module";
import { randomBytes } from "node:crypto";
import { test, after } from "node:test";
import assert from "node:assert/strict";
import { measurePresentation } from "./presentation-measure.mjs";
const args = Object.fromEntries(process.argv.slice(2).flatMap((v, i, all) => v.startsWith("--") ? [[v.slice(2), all[i + 1]]] : []));
const { chromium } = createRequire(import.meta.url)(args["playwright-module"] || "playwright");
const browser = await chromium.launch({ headless: true, ...(args.chromium ? { executablePath: args.chromium } : {}) });
after(() => browser.close());
async function fixture(html, check) {
  const page = await browser.newPage();
  try { await page.setContent(html); await check(page); } finally { await page.close(); }
}
test("translucent borders remain unresolved rather than asserting two painted colors", async () => {
  await fixture('<main><input id="alpha" style="appearance:none;background:white;border:2px solid rgba(0,0,0,.5)" value="sample"></main>', async page => {
    const result = await measurePresentation(page), borders = result.results.filter(r => r.kind.startsWith("field-boundary"));
    assert.equal(borders.length, 8);
    assert.ok(borders.every(r => r.uncertainty.includes("border-underlay-unverified") && !r.candidate));
  });
});
test("focused-element opacity is retained for outer outline measurements", async () => {
  await fixture('<style>input{appearance:none;opacity:.5}input:focus-visible{outline:2px solid #777;outline-offset:2px}</style><main><input id="faded"></main>', async page => {
    await page.keyboard.press("Tab");
    const result = await measurePresentation(page), outline = result.results.find(r => r.kind === "focus-outline-outside");
    assert.equal(result.focus.focusVisible, true);
    assert.ok(outline.uncertainty.includes("group-opacity"));
    assert.equal(outline.candidate, false);
  });
});
test("native appearance and paint internals are not certified from CSS values", async () => {
  await fixture('<main><input id="native" style="appearance:auto" value="sample"><input id="check" type="checkbox"></main>', async page => {
    const result = await measurePresentation(page);
    assert.ok(result.results.filter(r => r.element.id === "native").every(r => r.uncertainty.includes("native-appearance-unverified")));
    assert.ok(result.excluded.some(r => r.element.id === "check" && r.reason === "native-painted-internals-unmeasured"));
  });
});
test("horizontal-only clipping does not classify vertical overflow as clipped", async () => {
  await fixture('<style>#clip{position:relative;width:300px;height:24px;overflow-x:clip;overflow-y:visible}input{appearance:none;position:absolute;left:30px;top:40px;width:100px}input:focus-visible{outline:2px solid black;outline-offset:2px}</style><main><div id="clip"><input></div></main>', async page => {
    await page.keyboard.press("Tab");
    const result = await measurePresentation(page);
    assert.equal(result.focus.geometryOnly, true);
    assert.ok(!result.focus.clippedBy.some(e => e.id === "clip"));
  });
});
test("placeholder uses its own typography and observations omit entered values", async () => {
  await fixture('<style>input{appearance:none;background:white;color:black;font-size:12px}input::placeholder{font-size:30px;font-weight:700;color:#777;opacity:1}</style><main><input id="hint" placeholder="Example"><input id="value"></main>', async page => {
    const privateValue = randomBytes(24).toString("base64url");
    await page.locator("#value").fill(privateValue);
    const result = await measurePresentation(page), placeholder = result.results.find(r => r.kind === "visible-placeholder"), value = result.results.find(r => r.kind === "control-value");
    assert.equal(placeholder.threshold, 3);
    assert.equal(placeholder.candidate, false);
    assert.equal(value.ratio, 21);
    assert.ok(!JSON.stringify(result).includes(privateValue));
  });
});
test("native modal underlay controls are excluded from painted contrast observations", async () => {
  await fixture('<main><input id="underlay" value="sample"></main><dialog><input id="inside" value="sample"></dialog>', async page => {
    await page.evaluate(() => document.querySelector("dialog").showModal());
    const result = await measurePresentation(page);
    assert.ok(result.excluded.some(r => r.element.id === "underlay" && r.reason === "modal-underlay"));
    assert.ok(!result.results.some(r => r.element.id === "underlay"));
    assert.ok(result.results.some(r => r.element.id === "inside"));
  });
});
test("opaque border outside measurements preserve field opacity uncertainty", async () => {
  await fixture('<main><input id="faded-border" style="appearance:none;opacity:.5;border:2px solid black" value="sample"></main>', async page => {
    const result = await measurePresentation(page), outside = result.results.filter(r => r.kind.endsWith("-outside"));
    assert.equal(outside.length, 4);
    assert.ok(outside.every(r => r.uncertainty.includes("group-opacity") && !r.candidate));
  });
});
test("nonmodal dialogs do not hide the surrounding controls from measurement", async () => {
  await fixture('<main><input id="outside" value="sample"></main><dialog><input value="sample"></dialog>', async page => {
    await page.evaluate(() => document.querySelector("dialog").show());
    const result = await measurePresentation(page);
    assert.ok(!result.excluded.some(r => r.element.id === "outside" && r.reason === "modal-underlay"));
    assert.ok(result.results.some(r => r.element.id === "outside"));
  });
});
