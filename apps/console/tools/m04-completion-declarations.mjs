import assert from "node:assert/strict";
import { isDeepStrictEqual } from "node:util";

const terms = {
  en: {
    history: "Declaration history",
    version: "Version",
    search: "Search",
    unavailable: "Version unavailable",
    unknown: "Command outcome unknown",
  },
  ko: {
    history: "선언 이력",
    version: "버전",
    search: "검색",
    unavailable: "버전 조회 불가",
    unknown: "명령 결과 확인 안 됨",
  },
  ar: {
    history: "سجل الإقرارات",
    version: "الإصدار",
    search: "بحث",
    unavailable: "الإصدار غير متاح",
    unknown: "نتيجة الأمر غير معروفة",
  },
};
export async function declarationCompletion({
  page,
  service,
  navigate,
  read,
  capture,
  scenario,
}) {
  const path = "/api/v1/identity/services/" + service + "/declarations";
  const open = async (locale = "en") => {
    await navigate("credentials", {}, locale);
    await navigate("accounts", { svc_a_selected: service }, locale);
    await page.getByRole("heading", { name: service, exact: true }).waitFor();
    return page
      .getByRole("heading", { name: terms[locale].history, exact: true })
      .locator("..");
  };
  const historyRead = async (scope, button, predicate) => {
    const [response] = await Promise.all([
      page.waitForResponse((response) => {
        const url = new URL(response.url());
        return (
          url.pathname === path && response.status() === 200 && predicate(url)
        );
      }),
      scope.getByRole("button", { name: button, exact: true }).click(),
    ]);
    const body = await response.json();
    const title = await scope.locator(":scope > h3").innerText();
    await page.waitForFunction(
      ({ title, versions }) => {
        const heading = [...document.querySelectorAll("h3")].find(
          (node) => node.textContent === title,
        );
        return (
          heading &&
          JSON.stringify(
            [
              ...heading.parentElement.querySelectorAll(
                ":scope > dl.live-facts > div:first-child dd",
              ),
            ].map((node) => node.textContent),
          ) === JSON.stringify(versions)
        );
      },
      { title, versions: body.rows.map((row) => row.version) },
    );
    return body;
  };
  const waitVersions = async (versions) => {
    await page.waitForFunction((versions) => {
      const heading = [...document.querySelectorAll("h3")].find(
        (n) => n.textContent === "Declaration history",
      );
      return (
        heading &&
        JSON.stringify(
          [
            ...heading.parentElement.querySelectorAll(
              ":scope > dl.live-facts > div:first-child dd",
            ),
          ].map((n) => n.textContent),
        ) === JSON.stringify(versions)
      );
    }, versions);
  };
  await scenario(
    "retained declaration pages replace rows and historical observation cannot become the edit version",
    async () => {
      const history = await open();
      const first = Array.from({ length: 10 }, (_, i) => String(11 - i));
      await waitVersions(first);
      const last = await historyRead(
        history,
        "Next",
        (u) => !!u.searchParams.get("after"),
      );
      assert.deepEqual(
        last.rows.map((row) => row.version),
        ["1"],
      );
      await waitVersions(["1"]);
      await capture("en", 1440, "history-next-replaces-rows");
      await historyRead(
        history,
        "Previous",
        (u) => !u.searchParams.get("after"),
      );
      await waitVersions(first);
      await historyRead(history, "Next", (u) => !!u.searchParams.get("after"));
      await historyRead(history, "First", (u) => !u.searchParams.get("after"));
      await waitVersions(first);
      await history
        .getByRole("textbox", { name: "Version", exact: true })
        .fill("1");
      const original = await historyRead(
        history,
        "Search",
        (u) => u.searchParams.get("version") === "1",
      );
      assert.equal(original.availability, "recorded");
      assert.deepEqual(
        original.rows.map((row) => row.version),
        ["1"],
      );
      await waitVersions(["1"]);
      await page
        .getByRole("button", { name: "Edit declaration", exact: true })
        .click();
      const form = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: "Edit declaration",
          exact: true,
        }),
      });
      assert.equal(
        await form.locator("#svc-purpose").inputValue(),
        "Synthetic retained declaration 11",
      );
      assert.equal(
        await form
          .locator("dt")
          .filter({ hasText: /^Version$/ })
          .locator("..")
          .locator("dd")
          .innerText(),
        "11",
      );
      await capture("en", 1440, "history-edit-current-not-observed");
      assert.equal(
        (await read("/identity/services/" + service)).row.declaration.version,
        "11",
      );
    },
  );
  await scenario(
    "unavailable declaration versions remain explicit in all locale and viewport variants",
    async () => {
      for (const locale of ["en", "ko", "ar"])
        for (const width of [1440, 1280, 1024, 390]) {
          await page.setViewportSize({ width, height: 1000 });
          const history = await open(locale);
          await history
            .getByRole("textbox", { name: terms[locale].version, exact: true })
            .fill("99");
          const unavailable = await historyRead(
            history,
            terms[locale].search,
            (u) => u.searchParams.get("version") === "99",
          );
          assert.equal(unavailable.availability, "version_unavailable");
          assert.deepEqual(unavailable.rows, []);
          await history
            .getByText(terms[locale].unavailable, { exact: true })
            .waitFor();
          assert.equal(
            await history.locator(":scope > dl.live-facts").count(),
            0,
          );
          await capture(locale, width, "history-version-unavailable");
        }
      await page.setViewportSize({ width: 1440, height: 1000 });
    },
  );
  await scenario(
    "accepted lost declaration response retries the exact original envelope after canonical refresh",
    async () => {
      await open();
      await page
        .getByRole("button", { name: "Edit declaration", exact: true })
        .click();
      const form = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: "Edit declaration",
          exact: true,
        }),
      });
      const purpose =
        "Synthetic original accepted declaration lost in delivery";
      await form.locator("#svc-purpose").fill(purpose);
      await form
        .locator("#live-reason")
        .fill("Synthetic exact original declaration replay verification");
      const envelopes = [],
        outcomes = [];
      const handler = async (route) => {
        envelopes.push(route.request().postDataJSON());
        const response = await route.fetch();
        assert.equal(response.status(), 200);
        const body = await response.json();
        assert.equal(body.status, "completed");
        assert.equal(body.business_state, "service_declared");
        outcomes.push(body);
        if (outcomes.length === 1) await route.abort("failed");
        else await route.fulfill({ response });
      };
      await page.route("**/api/v1/commands/service.declare", handler);
      try {
        await form.locator("button[type=submit]").click();
        await page
          .getByRole("heading", { name: terms.en.unknown, exact: true })
          .first()
          .waitFor();
        assert.equal(outcomes.length, 1);
        assert.equal(outcomes[0].replay, false);
        assert.equal(outcomes[0].request_id, envelopes[0].request_id);
        for (const key of ["operation_id", "result_revision", "as_of"])
          assert.ok(
            typeof outcomes[0][key] === "string" && outcomes[0][key].length > 0,
            "Original accepted result identity missing: " + key,
          );
        assert.equal(outcomes[0].fields.declaration_version, "12");
        assert.equal(envelopes[0].body.declaration_version, "11");
        const current = await read("/identity/services/" + service);
        assert.equal(current.row.declaration.version, "12");
        assert.equal(current.row.declaration.purpose, purpose);
        for (const locale of ["en", "ko", "ar"])
          for (const width of [1440, 1280, 1024, 390]) {
            await page.setViewportSize({ width, height: 1000 });
            await navigate("credentials", {}, locale);
            await navigate("accounts", { svc_a_selected: service }, locale);
            await page
              .getByRole("heading", {
                name: terms[locale].unknown,
                exact: true,
              })
              .first()
              .waitFor();
            await page
              .getByRole("heading", { name: service, exact: true })
              .waitFor();
            assert.equal(outcomes.length, 1);
            await capture(
              locale,
              width,
              "declaration-original-unknown-after-readback",
            );
          }
        await page.setViewportSize({ width: 1440, height: 1000 });
        await navigate("credentials");
        await navigate("accounts", { svc_a_selected: service });
        const [retry] = await Promise.all([
          page.waitForResponse(
            (response) =>
              new URL(response.url()).pathname ===
                "/api/v1/commands/service.declare" && response.status() === 200,
          ),
          page
            .getByRole("button", { name: "Try again", exact: true })
            .first()
            .click(),
        ]);
        const replay = await retry.json();
        assert.equal(replay.replay, true);
        assert.equal(envelopes.length, 2);
        assert.ok(
          isDeepStrictEqual(envelopes[1], envelopes[0]),
          "Original declaration envelope changed",
        );
        for (const key of [
          "request_id",
          "operation_id",
          "result_revision",
          "business_state",
          "fields",
          "as_of",
        ])
          assert.ok(
            isDeepStrictEqual(replay[key], outcomes[0][key]),
            "Original accepted result identity changed: " + key,
          );
        assert.equal(
          (await read("/identity/services/" + service)).row.declaration.version,
          "12",
        );
        await page
          .getByRole("heading", { name: "Command completed", exact: true })
          .first()
          .waitFor();
      } finally {
        await page.unroute("**/api/v1/commands/service.declare", handler);
      }
    },
  );
}
