import assert from "node:assert/strict";
import { createElement, type ComponentType } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import test from "node:test";
import {
  viewFields,
  fieldsFor,
  allowedView,
  contextKey,
  validateDraft,
  reviewState,
  type Rule,
} from "./model";
import type { Props } from "./types";
import Entry from "./Entry";
import Users from "./Users";
import Authentication from "./Authentication";
import Access from "./Access";
import ServiceAccounts from "./ServiceAccounts";
import MySecurity from "./MySecurity";
const bodies: Record<string, ComponentType<Props>> = {
  L01: Entry,
  L02: Entry,
  L03: Entry,
  M01: Users,
  M02: Authentication,
  M03: Access,
  M04: ServiceAccounts,
  M05: MySecurity,
};
const data = (entries: Record<string, string>) => {
  const f = new FormData();
  for (const [k, v] of Object.entries(entries)) f.set(k, v);
  return f;
};
function render(
  id: string,
  view: string,
  locale: Props["locale"] = "en",
  extraParams = "",
) {
  return renderToStaticMarkup(
    createElement(bodies[id], {
      screenId: id,
      view,
      locale,
      params: new URLSearchParams(extraParams),
      onState: () => {},
      onDirty: () => {},
    }),
  );
}
test("freeze eight screens and 32 canonical views independently", () => {
  assert.deepEqual(Object.keys(viewFields), [
    "L01",
    "L02",
    "L03",
    "M01",
    "M02",
    "M03",
    "M04",
    "M05",
  ]);
  assert.equal(
    Object.values(viewFields).reduce((n, v) => n + Object.keys(v).length, 0),
    32,
  );
  assert.equal(allowedView("M05", "support"), false);
  assert.equal(allowedView("constructor", "name"), false);
  assert.deepEqual(fieldsFor("__proto__", "toString"), []);
  assert.equal(allowedView("L02", "bootstrap"), false);
  assert.deepEqual(fieldsFor("L02", "invitation"), ["email"]);
});
test("every view renders meaningful distinct domain content in all three locales", () => {
  const all = new Set<string>();
  for (const [id, views] of Object.entries(viewFields))
    for (const view of Object.keys(views))
      for (const locale of ["en", "ko", "ar"] as const) {
        const html = render(id, view, locale);
        assert.match(html, /<h2>/, `${id}/${view}/${locale}`);
        assert.match(html, /<section/, `${id}/${view}/${locale}`);
        assert.ok(!html.includes("undefined"));
        assert.ok(!/<input[^>]+type="password"/.test(html));
        assert.ok(!/name="(?:password|token|secret|code|proof)"/.test(html));
        all.add(html);
      }
  assert.equal(all.size, 96);
});
test("login identity fields follow the spec and never disclose account existence", () => {
  const html = render("L01", "method");
  assert.match(html, /name="workspace"/);
  assert.match(html, /name="loginId"/);
  assert.doesNotMatch(html, /name="email"/);
  assert.doesNotMatch(
    html,
    /sample-a|sample-b|DEMO Workspace|Account found|Account does not exist/,
  );
  assert.match(html, /No account lookup/);
  assert.doesNotMatch(
    render("L02", "invitation"),
    /name="(?:roleRef|token|workspace)"/,
  );
});
test("all fields are frozen per view; provider conditional fields are a subset", () => {
  for (const [id, views] of Object.entries(viewFields))
    for (const view of Object.keys(views)) {
      const html = render(id, view);
      for (const name of [...html.matchAll(/name="([^"]+)"/g)].map((m) => m[1]))
        assert.ok(fieldsFor(id, view).includes(name), `${id}/${view}/${name}`);
    }
  const html = render("M02", "providers");
  assert.match(html, /name="redirect"/);
  assert.doesNotMatch(html, /name="endpoint"/);
});
test("personal screen structurally excludes internal controls regardless of URL flags", () => {
  for (const view of Object.keys(viewFields.M05)) {
    const html = render(
      "M05",
      view,
      "en",
      "audience=admin&mode=synthetic&role=IA&support=1",
    );
    assert.doesNotMatch(
      html,
      /name="(?:ownerRef|supportRef|roleRef|scope|parentRef|customerRef|substitute|actor)"/,
    );
    assert.doesNotMatch(html, /name="(?:internal_note|margin|cost)"/);
  }
  assert.match(render("M05", "notifications"), /not saved or applied/);
});
test("method and source declarations never become policy or permission decisions", () => {
  assert.match(render("M03", "effects"), /Not evaluated/);
  assert.doesNotMatch(render("M03", "effects"), />Allow<|>Deny</);
  assert.match(
    render("M02", "policies"),
    /Local checks do not test a connection/,
  );
  assert.match(
    render("M01", "revocation"),
    /confirmed security denial and later cleanup are separate/,
  );
  assert.doesNotMatch(render("M01", "users"), />0</);
});
test("canonical references, bounded identifiers and required fields validate without lookup", () => {
  const rules: Record<string, Rule> = {
    workspace: { required: true, kind: "reference" },
    email: { required: true, kind: "email" },
  };
  assert.deepEqual(
    validateDraft(
      data({ workspace: "QA-WORKSPACE-1", email: "qa@example.test" }),
      rules,
    ),
    [],
  );
  assert.deepEqual(
    validateDraft(data({ workspace: "../other", email: "not-email" }), rules),
    ["workspace", "email"],
  );
  assert.deepEqual(
    validateDraft(
      data({ workspace: "x".repeat(129), email: "qa@example.test" }),
      rules,
    ),
    ["workspace"],
  );
  assert.deepEqual(validateDraft(data({ workspace: "", email: "" }), rules), [
    "workspace",
    "email",
  ]);
});
test("URL shape checks never permit plaintext LDAP, credentials or URL secrets", () => {
  for (const endpoint of [
    "ldap://example.test",
    "ldaps://user:password@example.test",
    "ldaps://example.test?token=QA",
    "ldaps://example.test#QA",
    "ldaps:///path",
  ]) {
    assert.deepEqual(
      validateDraft(data({ endpoint }), {
        endpoint: { required: true, kind: "ldaps" },
      }),
      ["endpoint"],
    );
  }
  assert.deepEqual(
    validateDraft(data({ endpoint: "ldaps://example.test" }), {
      endpoint: { required: true, kind: "ldaps" },
    }),
    [],
  );
  assert.deepEqual(
    validateDraft(data({ redirect: "http://example.test/callback" }), {
      redirect: { required: true, kind: "url" },
    }),
    ["redirect"],
  );
});
test("date intervals are real, ordered and independent of current clock", () => {
  const rules: Record<string, Rule> = {
    starts: { required: true, kind: "date" },
    expires: { required: true, kind: "date" },
  };
  for (const pair of [
    { starts: "2026-02-29", expires: "2026-03-01" },
    { starts: "2026-03-01", expires: "2026-03-01" },
    { starts: "2026-03-02", expires: "2026-03-01" },
  ])
    assert.ok(validateDraft(data(pair), rules).length > 0);
  assert.deepEqual(
    validateDraft(data({ starts: "2000-02-28", expires: "2000-02-29" }), rules),
    [],
  );
});
test("quiet hours require real times, explicit zone and nonempty range without DST arithmetic", () => {
  const rules: Record<string, Rule> = {
    quietStart: { required: true, kind: "time" },
    quietEnd: { required: true, kind: "time" },
    timeZone: { required: true, kind: "zone" },
  };
  assert.deepEqual(
    validateDraft(
      data({ quietStart: "22:00", quietEnd: "07:00", timeZone: "Asia/Seoul" }),
      rules,
    ),
    [],
  );
  assert.ok(
    validateDraft(
      data({
        quietStart: "22:00",
        quietEnd: "22:00",
        timeZone: "Mars/Unknown",
      }),
      rules,
    ).includes("quietEnd"),
  );
  assert.ok(
    validateDraft(
      data({ quietStart: "24:01", quietEnd: "07:00", timeZone: "UTC" }),
      rules,
    ).includes("quietStart"),
  );
});
test("service scope checks do not turn execution into human approval", () => {
  for (const scope of ["approve", "billing.approve", "human approval"]) {
    assert.deepEqual(
      validateDraft(data({ scope }), {
        scope: { required: true, kind: "service-scope" },
      }),
      ["scope"],
    );
  }
  assert.deepEqual(
    validateDraft(data({ scope: "billing.execute" }), {
      scope: { required: true, kind: "service-scope" },
    }),
    [],
  );
  assert.match(
    render("M04", "accounts"),
    /all proposed scopes still require server evaluation/,
  );
});
test("scope changes reset ephemeral draft identity, unrelated and secret URL params are ignored", () => {
  const initial = contextKey(
    "L01",
    "method",
    new URLSearchParams("scope=QA-A"),
  );
  assert.notEqual(
    initial,
    contextKey("L01", "method", new URLSearchParams("scope=QA-B")),
  );
  assert.notEqual(
    initial,
    contextKey("L01", "mfa", new URLSearchParams("scope=QA-A")),
  );
  assert.notEqual(
    initial,
    contextKey(
      "L01",
      "method",
      new URLSearchParams("scope=QA-A&permission_revision=new"),
    ),
  );
  assert.equal(
    initial,
    contextKey(
      "L01",
      "method",
      new URLSearchParams(
        "scope=QA-A&token=QA-DO-NOT-RETAIN&password=QA-SECRET",
      ),
    ),
  );
  assert.doesNotMatch(initial, /SECRET|token|password/);
});
test("review state includes structural change identity without input values or success claims", () => {
  const state = JSON.parse(
    reviewState("providers", 3, true, "checked", [], "QA-instance", "oidc"),
  );
  assert.deepEqual(state, {
    phase: "unconnected",
    view: "providers",
    revision: 3,
    dirty: true,
    stage: "checked",
    errors: [],
    instance: "QA-instance",
    variant: "oidc",
  });
  assert.equal(Object.hasOwn(state, "values"), false);
  assert.equal(Object.hasOwn(state, "result"), false);
});
