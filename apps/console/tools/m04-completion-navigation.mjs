import assert from "node:assert/strict";

const terms = {
  en: {
    create: "Create service account",
    audience: "Audience",
    scopes: "Declared scopes",
    search: "Search",
    next: "Next",
    previous: "Previous",
    first: "First",
    incomplete: "No matches on this page. More source records remain to scan.",
    empty: "No matches on this page.",
  },
  ko: {
    create: "서비스 계정 만들기",
    audience: "대상 서비스",
    scopes: "선언된 범위",
    search: "검색",
    next: "다음",
    previous: "이전",
    first: "처음",
    incomplete:
      "이 페이지에는 일치 항목이 없습니다. 조회할 원본 항목이 더 있습니다.",
    empty: "이 페이지에 일치하는 항목이 없습니다.",
  },
  ar: {
    create: "إنشاء حساب خدمة",
    audience: "الخدمة المستهدفة",
    scopes: "النطاقات المُعلَنة",
    search: "بحث",
    next: "التالي",
    previous: "السابق",
    first: "الأول",
    incomplete:
      "لا توجد نتائج مطابقة في هذه الصفحة. ما زالت هناك سجلات مصدر لفحصها.",
    empty: "لا توجد نتائج مطابقة في هذه الصفحة.",
  },
};
export async function navigationCompletion({
  page,
  service,
  credential,
  navigate,
  read,
  capture,
  scenario,
}) {
  const credentialPath = "/api/v1/identity/service-credentials";
  const params = async () =>
    page.evaluate(() =>
      Object.fromEntries(new URLSearchParams(location.hash.split("?")[1])),
    );
  const waitCredentialRows = async (rows) =>
    page.waitForFunction(
      (ids) =>
        JSON.stringify(
          [
            ...document.querySelectorAll(
              'section[aria-labelledby="svc-cred-title"] table tbody tr td:first-child',
            ),
          ].map((n) => n.textContent),
        ) === JSON.stringify(ids),
      rows.map((row) => row.id),
    );
  const waitRendered = async (scope, body) => {
    const tag = await scope.evaluate((node) => node.tagName);
    if (tag === "NAV") return waitCredentialRows(body.rows);
    if (tag === "FIELDSET") {
      const select = scope.locator("select");
      const control = await select.evaluate((node) => ({
        id: node.id,
        value: node.value,
      }));
      const expected = ["", ...body.rows.map((row) => row.id)];
      if (control.value && !expected.includes(control.value))
        expected.push(control.value);
      return page.waitForFunction(
        ({ id, expected }) => {
          const node = document.getElementById(id);
          return (
            node &&
            JSON.stringify(
              [...node.options].map((option) => option.value).sort(),
            ) === JSON.stringify(expected.sort())
          );
        },
        { id: control.id, expected },
      );
    }
    assert.equal(tag, "SECTION", "Unknown paged widget");
    const title = await scope.locator(":scope > h3").innerText();
    return page.waitForFunction(
      ({ title, ids }) => {
        const heading = [...document.querySelectorAll("h3")].find(
          (node) => node.textContent === title,
        );
        if (!heading) return false;
        const rows = [
          ...heading.parentElement.querySelectorAll(":scope > .svc-list-body > ul > li"),
        ];
        return (
          rows.length === ids.length &&
          ids.every((id) => rows.some((row) => row.textContent.includes(id)))
        );
      },
      { title, ids: body.rows.map((row) => row.id) },
    );
  };
  const responseClick = async (scope, name, predicate) => {
    const [response] = await Promise.all([
      page.waitForResponse(
        (response) =>
          response.status() === 200 && predicate(new URL(response.url())),
      ),
      scope.getByRole("button", { name, exact: true }).click(),
    ]);
    const body = await response.json();
    await waitRendered(scope, body);
    return body;
  };
  const tableIDs = async () =>
    page
      .locator(
        'section[aria-labelledby="svc-cred-title"] table tbody tr td:first-child',
      )
      .allTextContents();
  await scenario(
    "credential and global pending cursors replace rows and keep independent query owners",
    async () => {
      await navigate("accounts");
      const [loaded] = await Promise.all([
        page.waitForResponse((response) => {
          const u = new URL(response.url());
          return (
            u.pathname === credentialPath &&
            u.searchParams.get("limit") === "1" &&
            response.status() === 200
          );
        }),
        navigate("credentials", {
          svc_c_service: service,
          svc_c_limit: "1",
          svc_c_selected: credential,
        }),
      ]);
      const first = await loaded.json();
      assert.equal(first.rows.length, 1);
      await page.waitForFunction(
        (id) =>
          document.querySelector(
            'section[aria-labelledby="svc-cred-title"] table tbody tr td',
          )?.textContent === id,
        first.rows[0].id,
      );
      const nav = page
        .locator(
          'section[aria-labelledby="svc-cred-title"] .svc-list-body .list-pager-nav',
        )
        .first();
      const next = await responseClick(
        nav,
        "Next",
        (u) =>
          u.pathname === credentialPath &&
          u.searchParams.get("limit") === "1" &&
          !!u.searchParams.get("after"),
      );
      await page.waitForFunction(
        (id) =>
          document.querySelector(
            'section[aria-labelledby="svc-cred-title"] table tbody tr td',
          )?.textContent === id,
        next.rows[0].id,
      );
      assert.deepEqual(await tableIDs(), [next.rows[0].id]);
      assert.notEqual(next.rows[0].id, first.rows[0].id);
      await page
        .getByRole("heading", { name: credential, exact: true })
        .waitFor();
      const nextURL = await params();
      assert.ok(nextURL.svc_c_after);
      await capture("en", 1440, "credential-next-independent");
      const pending = page
        .getByRole("heading", {
          name: "Credentials awaiting collection",
          exact: true,
        })
        .locator("..");
      await pending.locator("li").nth(24).waitFor();
      assert.equal(await pending.locator("li").count(), 25);
      const pendingNext = await responseClick(
        pending,
        "Next",
        (u) =>
          u.pathname === credentialPath &&
          u.searchParams.get("collection") === "uncollected" &&
          !!u.searchParams.get("after"),
      );
      assert.equal(pendingNext.rows.length, 2);
      await page.waitForFunction(
        (ids) => {
          const heading = [...document.querySelectorAll("h3")].find(
            (n) => n.textContent === "Credentials awaiting collection",
          );
          const rows = [...heading.parentElement.querySelectorAll("li")];
          return (
            rows.length === ids.length &&
            ids.every((id) => rows.some((row) => row.textContent.includes(id)))
          );
        },
        pendingNext.rows.map((row) => row.id),
      );
      assert.deepEqual(await tableIDs(), [next.rows[0].id]);
      assert.deepEqual(await params(), nextURL);
      await capture("en", 1440, "pending-next-independent");
      await responseClick(
        pending,
        "Previous",
        (u) =>
          u.pathname === credentialPath &&
          u.searchParams.get("collection") === "uncollected" &&
          !u.searchParams.get("after"),
      );
      await pending.locator("li").nth(24).waitFor();
      await responseClick(
        pending,
        "Next",
        (u) =>
          u.pathname === credentialPath &&
          u.searchParams.get("collection") === "uncollected" &&
          !!u.searchParams.get("after"),
      );
      await responseClick(
        pending,
        "First",
        (u) =>
          u.pathname === credentialPath &&
          u.searchParams.get("collection") === "uncollected" &&
          !u.searchParams.get("after"),
      );
      await responseClick(
        nav,
        "Previous",
        (u) =>
          u.pathname === credentialPath &&
          u.searchParams.get("limit") === "1" &&
          !u.searchParams.get("after"),
      );
      await responseClick(
        nav,
        "Next",
        (u) =>
          u.pathname === credentialPath &&
          u.searchParams.get("limit") === "1" &&
          !!u.searchParams.get("after"),
      );
      await responseClick(
        nav,
        "First",
        (u) =>
          u.pathname === credentialPath &&
          u.searchParams.get("limit") === "1" &&
          !u.searchParams.get("after"),
      );
      await page.waitForFunction(
        (id) =>
          document.querySelector(
            'section[aria-labelledby="svc-cred-title"] table tbody tr td',
          )?.textContent === id,
        first.rows[0].id,
      );
      assert.deepEqual(await tableIDs(), [first.rows[0].id]);
    },
  );
  await scenario(
    "browser history and view switching preserve owned parameters and reset only invalid credential state",
    async () => {
      await navigate("credentials", {
        svc_c_service: service,
        svc_c_limit: "1",
        svc_a_q: "saved-account",
        marker: "retained-marker",
      });
      await page
        .locator('section[aria-labelledby="svc-cred-title"] table tbody tr')
        .waitFor();
      const first = await params();
      const nav = page
        .locator(
          'section[aria-labelledby="svc-cred-title"] .svc-list-body .list-pager-nav',
        )
        .first();
      const next = await responseClick(
        nav,
        "Next",
        (u) =>
          u.pathname === credentialPath &&
          u.searchParams.get("limit") === "1" &&
          !!u.searchParams.get("after"),
      );
      const second = await params();
      const [back] = await Promise.all([
        page.waitForResponse((r) => {
          const u = new URL(r.url());
          return (
            u.pathname === credentialPath &&
            !u.searchParams.get("after") &&
            u.searchParams.get("limit") === "1" &&
            r.status() === 200
          );
        }),
        page.goBack(),
      ]);
      const backPage = await back.json();
      assert.equal(backPage.rows.length, 1);
      await waitCredentialRows(backPage.rows);
      assert.deepEqual(await params(), first);
      await Promise.all([
        page.waitForResponse((r) => {
          const u = new URL(r.url());
          return (
            u.pathname === credentialPath &&
            !!u.searchParams.get("after") &&
            u.searchParams.get("limit") === "1" &&
            r.status() === 200
          );
        }),
        page.goForward(),
      ]);
      assert.deepEqual(await params(), second);
      await page.waitForFunction(
        (id) =>
          document.querySelector(
            'section[aria-labelledby="svc-cred-title"] table tbody tr td',
          )?.textContent === id,
        next.rows[0].id,
      );
      await page.getByRole("button", { name: "Accounts", exact: true }).click();
      await page.locator("#svc-list-title").waitFor();
      assert.equal((await params()).svc_c_after, second.svc_c_after);
      assert.equal((await params()).svc_a_q, "saved-account");
      await page
        .getByRole("button", { name: "Credentials", exact: true })
        .click();
      await page.locator("#svc-cred-title").waitFor();
      assert.equal((await params()).svc_c_after, second.svc_c_after);
      await navigate("credentials", {
        svc_c_limit: "0",
        svc_a_q: "saved-account",
        marker: "retained-marker",
      });
      await page
        .getByText(
          "This filter or selection is invalid. Reset it to continue.",
          { exact: true },
        )
        .waitFor();
      await page
        .getByRole("button", { name: "Reset filters", exact: true })
        .click();
      await page.locator("#svc-cred-title").waitFor();
      const reset = await params();
      assert.equal(reset.marker, "retained-marker");
      assert.equal(reset.svc_a_q, "saved-account");
      assert.ok(!Object.keys(reset).some((key) => key.startsWith("svc_c_")));
    },
  );
  await scenario(
    "shared audience and scope pickers retain canonical off-page selections without accumulating page rows",
    async () => {
      await navigate("credentials");
      await navigate("accounts");
      await page
        .getByRole("button", { name: "Create service account", exact: true })
        .click();
      const form = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: "Create service account",
          exact: true,
        }),
      });
      const audience = form.getByRole("group", {
        name: "Audience",
        exact: true,
      });
      await audience
        .locator('option[value="test-z-api-09"]')
        .waitFor({ state: "attached" });
      assert.equal(await audience.locator("option").count(), 11);
      await audience.locator("select").selectOption("test-z-api-09");
      await responseClick(
        audience,
        "Next",
        (u) =>
          u.pathname === "/api/v1/identity/service-audiences" &&
          !!u.searchParams.get("after"),
      );
      await audience
        .locator('option[value="test-z-api-11"]')
        .waitFor({ state: "attached" });
      assert.equal(await audience.locator("option").count(), 4);
      assert.equal(
        await audience.locator("select").inputValue(),
        "test-z-api-09",
      );
      const knownAudience = await read(
        "/identity/service-audiences/test-z-api-09",
      );
      await audience
        .locator('option[value="test-z-api-09"]')
        .filter({ hasText: knownAudience.row.resource })
        .waitFor({ state: "attached" });
      assert.ok(
        !(
          await audience.locator('option[value="test-z-api-09"]').innerText()
        ).includes("Unavailable"),
      );
      await responseClick(
        audience,
        "Previous",
        (u) =>
          u.pathname === "/api/v1/identity/service-audiences" &&
          !u.searchParams.get("after"),
      );
      await audience.locator("select").selectOption("test-service-api");
      const scopes = form.getByRole("group", {
        name: "Declared scopes",
        exact: true,
      });
      await scopes
        .locator('option[value="read"]')
        .waitFor({ state: "attached" });
      await scopes.locator("select").selectOption("read");
      await responseClick(
        scopes,
        "Next",
        (u) =>
          u.pathname.endsWith("/test-service-api/scopes") &&
          !!u.searchParams.get("after"),
      );
      await scopes
        .locator('option[value="read-12"]')
        .waitFor({ state: "attached" });
      assert.equal(await scopes.locator("option").count(), 5);
      assert.equal(await scopes.locator("select").inputValue(), "read");
      await responseClick(
        scopes,
        "First",
        (u) =>
          u.pathname.endsWith("/test-service-api/scopes") &&
          !u.searchParams.get("after"),
      );
      await scopes
        .locator('option[value="read-01"]')
        .waitFor({ state: "attached" });
      assert.equal(await scopes.locator("option").count(), 11);
    },
  );
  await scenario(
    "real scan-limited empty scope pages remain distinct from completed empty across locales and widths",
    async () => {
      for (const locale of ["en", "ko", "ar"])
        for (const width of [1440, 1280, 1024, 390]) {
          const t = terms[locale];
          await page.setViewportSize({ width, height: 1000 });
          await navigate("credentials", {}, locale);
          await navigate("accounts", {}, locale);
          await page
            .getByRole("button", { name: t.create, exact: true })
            .click();
          const form = page.locator("form").filter({
            has: page.getByRole("heading", { name: t.create, exact: true }),
          });
          await form
            .getByRole("group", { name: t.audience, exact: true })
            .locator("select")
            .selectOption("test-service-api");
          const scopes = form.getByRole("group", {
            name: t.scopes,
            exact: true,
          });
          await scopes
            .getByRole("textbox", { name: t.search, exact: true })
            .fill("no-scope-matches");
          const first = await responseClick(
            scopes,
            t.search,
            (u) =>
              u.pathname.endsWith("/test-service-api/scopes") &&
              u.searchParams.get("q") === "no-scope-matches" &&
              !u.searchParams.get("after"),
          );
          assert.equal(first.rows.length, 0);
          assert.equal(first.work_units, 96);
          assert.equal(first.scan_stopped, true);
          assert.equal(first.count.certainty, "lower_bound");
          assert.ok(first.next);
          await scopes.getByText(t.incomplete, { exact: true }).waitFor();
          await capture(locale, width, "scope-incomplete-empty");
          const last = await responseClick(
            scopes,
            t.next,
            (u) =>
              u.pathname.endsWith("/test-service-api/scopes") &&
              u.searchParams.get("q") === "no-scope-matches" &&
              !!u.searchParams.get("after"),
          );
          assert.equal(last.rows.length, 0);
          assert.equal(last.next, "");
          assert.equal(last.count.certainty, "exact");
          assert.equal(last.count.value, "0");
          await scopes.getByText(t.empty, { exact: true }).waitFor();
          await capture(locale, width, "scope-complete-empty");
        }
      await page.setViewportSize({ width: 1440, height: 1000 });
      await navigate("credentials");
    },
  );
}
