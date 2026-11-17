import { test } from "node:test";
import assert from "node:assert/strict";
import { proxyFor, readerPaths } from "./proxy-config";

test("gated proxy sends every path to the Go API only", () => {
  const gated = proxyFor(true);
  for (const [path, entry] of Object.entries(gated)) {
    assert.equal(entry.target, "http://127.0.0.1:8820", path);
  }
  for (const path of [...readerPaths, "/api/v1", "/api/credit-case-review", "/api/claim-history", "/api/approval-review"]) {
    assert.ok(path in gated, path);
  }
});

test("gated proxy has no Python port", () => {
  const targets = Object.values(proxyFor(true)).map((e) => e.target).join(" ");
  for (const port of ["8793", "8806", "8810"]) assert.equal(targets.includes(port), false, port);
});

test("ungated proxy keeps the development routes", () => {
  const open = proxyFor(false);
  assert.equal(open["/api/v1"].target, "http://127.0.0.1:8820");
  assert.equal(open["/api/focus-overview"].target, "http://127.0.0.1:8806");
  assert.equal(open["/api/focus-receipts"].target, "http://127.0.0.1:8810");
});
