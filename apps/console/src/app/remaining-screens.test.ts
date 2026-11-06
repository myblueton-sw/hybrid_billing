import { test } from "node:test";
import assert from "node:assert/strict";
import {
  remainingGroups,
  remainingIds,
  presentationAudience,
} from "./remaining-screens";
import { resolveRoute } from "../planning/routing";
import { planningScreens } from "../planning/registry";
test("remaining batch preserves canonical identity and excludes held Marketplace", () => {
  assert.equal(remainingIds.size, 32);
  assert.equal(Object.values(remainingGroups).flat().length, 32);
  for (const id of remainingIds)
    assert(planningScreens.some((s) => s.id === id));
  assert(!remainingIds.has("B07"));
});
test("prelogin and portal shell selection does not grant cross-audience access", () => {
  const viewable = new Set(planningScreens.map((s) => s.id));
  for (const hash of [
    "#/login",
    "#/activate?lang=ar",
    "#/recover",
    "#entry-login",
  ]) {
    const audience = presentationAudience(hash);
    assert.equal(audience, "unauthenticated");
    const route = resolveRoute(hash, {
      audience,
      viewable,
      assistant: false,
      designTools: false,
    });
    assert.equal(route.screen?.audience, "prelogin");
  }
  const audience = presentationAudience("#/portal/documents");
  assert.equal(audience, "customer");
  const context = { audience, viewable, assistant: false, designTools: false };
  assert.equal(resolveRoute("#/portal/documents", context).screen?.id, "C02");
  assert.equal(resolveRoute("#/invoices", context).screen?.id, "C01");
  assert.equal(resolveRoute("#/access", context).screen?.id, "C01");
  assert.equal(presentationAudience("#/access?audience=customer"), "admin");
});
