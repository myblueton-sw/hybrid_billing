import { test } from "node:test";
import assert from "node:assert/strict";
import { planningScreens, legacyRedirects, legacyReviewMap } from "./registry";
import { resolveRoute, type RouteContext } from "./routing";
import { viewPermissions, canHold, canDelegate } from "./permissions";
const context = (audience: RouteContext["audience"]): RouteContext => ({
  audience,
  viewable: new Set(planningScreens.map((s) => s.id)),
  assistant: true,
  designTools: true,
});
test("49 unique planning screens and all 112 legacy redirects preserve query context", () => {
  assert.equal(planningScreens.length, 49);
  assert.equal(new Set(planningScreens.map((s) => s.path)).size, 49);
  assert.equal(Object.keys(legacyRedirects).length, 112);
  for (const [id, target] of Object.entries(legacyRedirects)) {
    const s = planningScreens.find((s) => s.id === target.screen);
    const audience =
      s?.audience === "portal"
        ? "customer"
        : s?.audience === "prelogin"
          ? "unauthenticated"
          : "admin";
    const route = resolveRoute(
      `#${id}?lang=ar&mode=unconnected&scope=sample-b`,
      context(audience),
    );
    assert.equal(route.path, target.target, id);
    assert.equal(route.query.get("lang"), "ar");
    assert.equal(route.query.get("mode"), "unconnected");
    assert.equal(route.query.get("scope"), "sample-b");
    assert.equal(legacyReviewMap[id], target.screen);
  }
});
test("audience precedes route and shared paths select separate projection", () => {
  for (const s of planningScreens.filter((s) => s.audience === "admin")) {
    assert.equal(
      resolveRoute("#" + s.path, context("customer")).screen?.id,
      "C01",
    );
    for (const [id, target] of Object.entries(legacyRedirects).filter(
      ([, v]) => v.screen === s.id,
    ))
      assert.equal(
        resolveRoute("#" + id, context("customer")).screen?.id,
        "C01",
        target.target,
      );
  }
  for (const path of ["/tasks", "/my-security"])
    assert.equal(
      resolveRoute("#" + path, context("customer")).projection,
      "customer",
    );
  assert.equal(
    resolveRoute("#/portal/documents", context("admin")).screen?.id,
    "U01",
  );
  assert.equal(
    resolveRoute("#/users", context("unauthenticated")).screen?.id,
    "L01",
  );
  assert.equal(
    resolveRoute("#/my-security/OTHER-USER", context("customer")).screen?.id,
    "C01",
  );
});
test("unknown, malformed and unpermitted routes fail closed including home", () => {
  for (const path of [
    "#no-such",
    "#/contracts/a/unknown",
    "#/contracts/a/summary/extra",
    "#/contracts/a%2fb",
    "#/contracts/%",
    "#/contracts/..",
  ])
    assert.equal(resolveRoute(path, context("admin")).screen?.id, "U01");
  const c = { ...context("admin"), viewable: new Set<string>() };
  assert.equal(resolveRoute("#/users", c).screen, null);
  assert.equal(resolveRoute("#/users", c).denied, true);
  assert.equal(
    resolveRoute("#/assistant", { ...context("admin"), assistant: false })
      .screen?.id,
    "U01",
  );
});
test("object tabs and interim wizard steps resolve deterministically", () => {
  const r = resolveRoute(
    "#/billing-runs/RUN-0001/comparison?period=2026-10",
    context("admin"),
  );
  assert.equal(r.objectId, "RUN-0001");
  assert.equal(r.tab, "comparison");
  assert.equal(r.query.get("period"), "2026-10");
  assert.equal(
    resolveRoute("#/onboarding/first-receipt", context("admin")).step,
    "first-receipt",
  );
});
test("unbound actions confer no permissions; view catalog refuses wrong classes and grants", () => {
  for (const p of viewPermissions) {
    assert.equal(p.executable, false);
    assert.equal(canDelegate(p, "admin", new Set([p.key]), true), false);
    if (p.allowedClasses.length === 1 && p.allowedClasses[0] === "admin")
      assert.equal(canHold(p, "customer"), false);
  }
});

test("prototype property names never become legacy redirects", () => {
  for (const audience of ["admin", "customer", "unauthenticated"] as const)
    for (const key of ["constructor", "__proto__", "toString"]) {
      const result = resolveRoute("#" + key, context(audience));
      assert.equal(
        result.screen?.id,
        audience === "admin" ? "U01" : audience === "customer" ? "C01" : "L01",
      );
      if (audience !== "unauthenticated")
        assert.equal(
          resolveRoute("#" + key, {
            ...context(audience),
            viewable: new Set<string>(),
          }).screen,
          null,
        );
    }
});

test("developer directory is gated independently of public help", () => {
  const c = { ...context("admin"), designTools: false };
  assert.equal(resolveRoute("#/design/screens", c).screen?.id, "U01");
  assert.equal(resolveRoute("#/design/overview", c).screen?.id, "U01");
  assert.equal(resolveRoute("#/help/contribute", c).tool, "/help/contribute");
  assert.equal(
    resolveRoute("#/help/contribute", { ...c, audience: "customer" }).screen
      ?.id,
    "C01",
  );
});
