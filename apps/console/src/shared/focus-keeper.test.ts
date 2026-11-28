import { test } from "node:test";
import assert from "node:assert/strict";
import { keeperTarget } from "./focus-keeper";

const heading = (name: string, shown = true, hidden = false) => ({ name, getClientRects: () => ({ length: shown ? 1 : 0 }), closest: () => (hidden ? {} : null) });
const node = (tagName: string, found: ReturnType<typeof heading>[], isConnected = true) => ({ tagName, isConnected, querySelectorAll: () => found });

test("the keeper picks the first shown heading in the nearest connected container inside main or a dialog", () => {
  const form = heading("form");
  assert.equal(keeperTarget([node("DIV", []), node("FORM", [heading("gone", false), form]), node("MAIN", [heading("page")])]), form);
  assert.equal(keeperTarget([node("FORM", [heading("removed")], false), node("SECTION", [form]), node("MAIN", [])]), form);
  assert.equal(keeperTarget([node("DIV", [heading("aria-hidden", true, true)]), node("DIALOG", [heading("title")])])?.name, "title");
});

test("the keeper stops at main or a dialog and never acts outside them", () => {
  assert.equal(keeperTarget([node("DIV", []), node("MAIN", []), node("DIV", [heading("outer")])]), null);
  assert.equal(keeperTarget([node("HEADER", [heading("brand")]), node("DIV", [heading("app")])]), null);
  assert.equal(keeperTarget([node("FORM", [heading("x")]), node("MAIN", [], false)]), null);
});
