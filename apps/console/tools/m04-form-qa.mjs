import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";

const titles = {
  en: {
    create: "Create service account",
    suspend: "Suspend service account",
    retire: "Retire service account",
    discard: "Discard previous draft",
    cancel: "Cancel",
  },
  ko: {
    create: "서비스 계정 만들기",
    suspend: "서비스 계정 정지",
    retire: "서비스 계정 사용 종료",
    discard: "이전 초안 버리기",
    cancel: "취소",
  },
  ar: {
    create: "إنشاء حساب خدمة",
    suspend: "تعليق حساب الخدمة",
    retire: "إيقاف حساب الخدمة نهائيًا",
    discard: "تجاهل المسودة السابقة",
    cancel: "إلغاء",
  },
};
export async function formsMatrix({
  page,
  navigate,
  output,
  service,
  otherService,
}) {
  await mkdir(output, { recursive: true });
  const variants = [];
  let commands = 0;
  const count = (request) => {
    if (new URL(request.url()).pathname.startsWith("/api/v1/commands/"))
      commands++;
  };
  page.on("request", count);
  const summary = () => page.locator("#svc-form-errors");
  const capture = async (
    locale,
    width,
    state,
    expectedFocus = "svc-form-errors",
  ) => {
    await page.evaluate(() => document.fonts.ready);
    if (state === "issue-errors" || state === "rejected-edit") {
      const ratio = await page.locator("form").filter({ has: page.locator("#svc-issue-title") }).evaluate(form => {
        for (const value of form.querySelectorAll(".live-facts dd")) {
          const walk = document.createTreeWalker(value, NodeFilter.SHOW_TEXT);
          let text;
          while (text = walk.nextNode()) {
            if (!/^\d+ \/ \d+$/.test(text.textContent)) continue;
            const used = document.createRange(), limit = document.createRange();
            used.setStart(text, 0); used.setEnd(text, 1);
            const index = text.textContent.indexOf("/") + 2;
            limit.setStart(text, index); limit.setEnd(text, index + 1);
            return { used: used.getBoundingClientRect().left, limit: limit.getBoundingClientRect().left };
          }
        }
        return null;
      });
      assert.ok(ratio && ratio.used < ratio.limit, "credential used/limit ratio reverses its meaning in RTL");
    }
    const actual = await page.evaluate(() => {
      const screen = document.querySelector('[data-security-screen="M04"]');
      const borders = [];
      for (const node of screen.querySelectorAll(
        ".live-step,.service-summary",
      )) {
        if (!node.getClientRects().length) continue;
        const style = getComputedStyle(node);
        for (const side of ["Top", "Right", "Bottom", "Left"]) {
          if (
            !parseFloat(style["border" + side + "Width"]) ||
            style["border" + side + "Style"] === "none"
          )
            continue;
          const values =
            style["border" + side + "Color"].match(/[\d.]+/g)?.map(Number) ??
            [];
          if (values.length === 4 && values[3] === 0) continue;
          borders.push(
            values.length >= 3 &&
              Math.max(...values.slice(0, 3)) -
                Math.min(...values.slice(0, 3)) <=
                45,
          );
        }
      }
      return {
        direction: screen.dir,
        fits: document.documentElement.scrollWidth <= innerWidth,
        font: getComputedStyle(screen).fontFamily,
        focus: document.activeElement?.id,
        neutralBorders: borders.every(Boolean),
      };
    });
    assert.equal(actual.direction, locale === "ar" ? "rtl" : "ltr");
    assert.ok(
      actual.fits && actual.neutralBorders && actual.font.includes("IBM Plex"),
    );
    assert.equal(actual.focus, expectedFocus);
    assert.equal(
      await page
        .locator("#svc-secret,#reauth-password,#signin-password")
        .count(),
      0,
    );
    await page.screenshot({
      path: `${output}/${locale}-${width}-${state}.png`,
      fullPage: true,
    });
    variants.push({
      locale,
      width,
      state,
      source: state.includes("picker")
        ? "held owner request and injected503; stable keyboard target and actual Reload recovery"
        : "actual mounted form and current metadata; invalid input dispatches no command",
      ...actual,
    });
  };
  const activate = async (field) => {
    const url = page.url();
    const link = summary().locator('a[href="#' + field + '"]');
    await link.focus();
    await link.press("Enter");
    assert.equal(
      page.url(),
      url,
      "field-error navigation replaced the app route",
    );
  };
  try {
    let release;
    const held = new Promise((done) => {
      release = done;
    });
    const handler = async (route) => {
      await held;
      return route.fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({
          code: "dependency_unavailable",
          request_id: "qa-form-owner",
        }),
      });
    };
    await page.route("**/api/v1/identity/service-owners?*", handler);
    try {
      await navigate("accounts");
      const requested = page.waitForRequest(
        (request) =>
          new URL(request.url()).pathname === "/api/v1/identity/service-owners",
      );
      await page
        .getByRole("button", { name: titles.en.create, exact: true })
        .click();
      await requested;
      const form = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: titles.en.create,
          exact: true,
        }),
      });
      await form.locator("input").first().fill("INVALID");
      await form.locator("button[type=submit]").click();
      await activate("svc-owner");
      assert.equal(
        await page.evaluate(() => document.activeElement?.id),
        "svc-owner",
      );
      await capture("en", 1440, "pending-picker", "svc-owner");
      release();
      await form
        .locator("#svc-owner")
        .getByText("dependency_unavailable", { exact: true })
        .waitFor();
      await activate("svc-owner");
      await capture("en", 1440, "failed-picker", "svc-owner");
      await page.unroute("**/api/v1/identity/service-owners?*", handler);
      await form
        .locator("#svc-owner")
        .getByRole("button", { name: "Reload", exact: true })
        .click();
      await form.locator("#svc-owner-select").waitFor();
      await activate("svc-owner");
      assert.equal(
        await page.evaluate(() => document.activeElement?.id),
        "svc-owner-select",
      );
      await form
        .getByRole("button", { name: titles.en.discard, exact: true })
        .click();
      assert.equal(await summary().count(), 0);
    } finally {
      release();
      await page.unroute("**/api/v1/identity/service-owners?*", handler);
    }
    for (const locale of ["en", "ko", "ar"]) {
      for (const width of [1440, 1280, 1024, 390]) {
        await page.setViewportSize({ width, height: 1000 });
        await navigate("credentials", {}, page, locale);
        await navigate("accounts", {}, page, locale);
        await page
          .getByRole("button", { name: titles[locale].create, exact: true })
          .click();
        const form = page.locator("form").filter({
          has: page.getByRole("heading", {
            name: titles[locale].create,
            exact: true,
          }),
        });
        await form.locator("input").first().fill("INVALID");
        await form.locator("button[type=submit]").click();
        assert.equal(
          await form.locator("#svc-purpose").getAttribute("aria-invalid"),
          "true",
          "missing purpose error is not associated with its input",
        );
        for (const id of [
          "svc-name",
          "svc-owner",
          "svc-organization",
          "svc-scopes",
          "live-reason",
        ])
          assert.equal(
            await form.locator("#" + id).getAttribute("aria-invalid"),
            "true",
            id,
          );
        await summary().waitFor();
        await capture(locale, width, "create-errors");
        await activate("svc-scopes");
        assert.equal(
          await page.evaluate(() => document.activeElement?.id),
          "svc-scopes",
        );
        await activate("svc-purpose");
        assert.equal(
          await page.evaluate(() => document.activeElement?.id),
          "svc-purpose",
        );
        await form.locator("#svc-purpose").fill("Synthetic corrected purpose");
        assert.equal(
          await form.locator("#svc-purpose").getAttribute("aria-invalid"),
          null,
        );
        assert.equal(
          await form.locator("#svc-name").getAttribute("aria-invalid"),
          "true",
        );
        await form.locator("button[type=submit]").click();
        assert.equal(
          await page.evaluate(() => document.activeElement?.id),
          "svc-form-errors",
        );
        await form
          .getByRole("button", { name: titles[locale].discard, exact: true })
          .click();
        assert.equal(await summary().count(), 0);
        assert.equal(commands, 0);
      }
    }
    for (const locale of ["en", "ko", "ar"]) {
      for (const width of [1440, 1280, 1024, 390]) {
        await page.setViewportSize({ width, height: 1000 });
        await navigate(
          "credentials",
          { svc_c_service: service, svc_c_issue: service },
          page,
          locale,
        );
        const form = page
          .locator("form")
          .filter({ has: page.locator("#svc-issue-title") });
        await form.locator("#svc-days").fill("0");
        await form.locator("#live-reason").fill("");
        await form.locator("button[type=submit]").click();
        for (const id of ["svc-days", "live-reason"]) {
          assert.equal(
            await form.locator("#" + id).getAttribute("aria-invalid"),
            "true",
          );
          assert.ok(
            (await form.locator("#" + id).getAttribute("aria-describedby"))
              .split(" ")
              .includes(id + "-error"),
          );
          assert.equal(await form.locator("#" + id + "-error").count(), 1);
        }
        await capture(locale, width, "issue-errors");
        await activate("svc-days");
        assert.equal(
          await page.evaluate(() => document.activeElement?.id),
          "svc-days",
        );
        await form.locator("#svc-days").fill("1");
        assert.equal(
          await form.locator("#svc-days").getAttribute("aria-invalid"),
          null,
        );
        assert.equal(
          await form.locator("#live-reason").getAttribute("aria-invalid"),
          "true",
        );
        await form
          .locator("#live-reason")
          .fill("Synthetic corrected issuance rationale");
        assert.equal(await summary().count(), 0);
        const previous = await form.locator("#live-reason").inputValue();
        const counter = await form.locator(".live-counter bdi").innerText();
        const limit = Number(counter.split("/")[1].trim());
        assert.ok(Number.isSafeInteger(limit) && limit > 0);
        await form.locator("#live-reason").fill("A".repeat(limit + 1));
        assert.equal(await form.locator("#live-reason").inputValue(), previous);
        assert.equal(
          await form.locator("#live-reason").getAttribute("aria-invalid"),
          null,
        );
        assert.equal(
          await page.evaluate(() => document.activeElement?.id),
          "live-reason",
        );
        if (width === 390)
          await capture(locale, width, "rejected-edit", "live-reason");
        assert.equal(commands, 0);
      }
    }
    await navigate("credentials", {
      svc_c_service: otherService,
      svc_c_issue: otherService,
    });
    await page
      .getByRole("button", { name: titles.en.discard, exact: true })
      .click();
    for (const locale of ["en", "ko", "ar"]) {
      for (const width of [1440, 1280, 1024, 390]) {
        await page.setViewportSize({ width, height: 1000 });
        await navigate("credentials", {}, page, locale);
        await navigate(
          "accounts",
          { svc_a_selected: otherService },
          page,
          locale,
        );
        await page
          .getByRole("button", { name: titles[locale].suspend, exact: true })
          .click();
        const form = page.locator("form").filter({
          has: page.getByRole("heading", {
            name: titles[locale].suspend,
            exact: true,
          }),
        });
        await form.locator("#live-reason").fill("");
        await form.locator("button[type=submit]").click();
        assert.equal(
          await form.locator("#live-reason").getAttribute("aria-invalid"),
          "true",
        );
        await capture(locale, width, "state-errors");
        await activate("live-reason");
        assert.equal(
          await page.evaluate(() => document.activeElement?.id),
          "live-reason",
        );
        await form
          .locator("#live-reason")
          .fill("Synthetic corrected state rationale");
        assert.equal(
          await form.locator("#live-reason").getAttribute("aria-invalid"),
          null,
        );
        assert.equal(await summary().count(), 0);
        await form
          .getByRole("button", { name: titles[locale].cancel, exact: true })
          .click();
        assert.equal(commands, 0);
      }
    }
    await navigate("accounts", { svc_a_selected: otherService });
    await page
      .getByRole("button", { name: titles.en.retire, exact: true })
      .click();
    await page
      .getByRole("button", { name: titles.en.discard, exact: true })
      .click();
    await page
      .locator("form")
      .filter({
        has: page.getByRole("heading", { name: titles.en.retire, exact: true }),
      })
      .getByRole("button", { name: titles.en.cancel, exact: true })
      .click();
    assert.equal(variants.length, 41);
    return variants;
  } finally {
    page.off("request", count);
    await page.setViewportSize({ width: 1440, height: 1000 });
    await navigate("credentials");
  }
}
