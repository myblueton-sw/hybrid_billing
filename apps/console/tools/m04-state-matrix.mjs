import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";

export async function stateMatrix({
  page,
  navigate,
  output,
  service,
  credential,
  prefix,
}) {
  const variants = [];
  await mkdir(output, { recursive: true });
  const settle = () =>
    page.evaluate(async () => {
      await document.fonts.ready;
      await new Promise((done) =>
        requestAnimationFrame(() => requestAnimationFrame(done)),
      );
    });
  const inspect = async (locale, width, state, source) => {
    await settle();
    const actual = await page.evaluate(() => {
      const screen = document.querySelector('[data-security-screen="M04"]');
      const borders = [];
      for (const node of screen.querySelectorAll(
        ".live-note-rule,.live-held,.live-selected,.live-panel,.live-step,.service-summary",
      )) {
        if (!node.getClientRects().length) continue;
        const style = getComputedStyle(node);
        for (const side of ["Top", "Right", "Bottom", "Left"]) {
          if (
            !parseFloat(style["border" + side + "Width"]) ||
            style["border" + side + "Style"] === "none"
          )
            continue;
          const color = style["border" + side + "Color"];
          const values = color.match(/[\d.]+/g)?.map(Number) ?? [];
          if (values.length === 4 && values[3] === 0) continue;
          borders.push({
            element: node.tagName,
            side,
            color,
            neutral:
              values.length >= 3 &&
              Math.max(...values.slice(0, 3)) -
                Math.min(...values.slice(0, 3)) <=
                45,
          });
        }
      }
      const bounds = screen.getBoundingClientRect();
      return {
        locale: screen.lang,
        direction: screen.getAttribute("dir"),
        documentFits: document.documentElement.scrollWidth <= innerWidth,
        screenWidth: bounds.width,
        parentWidth: screen.parentElement.getBoundingClientRect().width,
        font: getComputedStyle(screen).fontFamily,
        borders,
      };
    });
    assert.equal(actual.locale, locale);
    assert.equal(actual.direction, locale === "ar" ? "rtl" : "ltr");
    assert.ok(
      actual.documentFits,
      `document overflow ${locale}/${width}/${state}`,
    );
    assert.ok(
      actual.screenWidth >= actual.parentWidth - 2,
      `screen does not use available width ${locale}/${width}/${state}`,
    );
    assert.ok(actual.font.includes("IBM Plex"), "intended font family missing");
    assert.ok(
      actual.borders.every((border) => border.neutral),
      `decorative colored border ${locale}/${width}/${state}`,
    );
    assert.equal(
      await page.locator("#svc-secret,#signin-password,#link-new").count(),
      0,
    );
    await page.screenshot({
      path: `${output}/${locale}-${width}-${state}.png`,
      fullPage: true,
    });
    variants.push({ locale, width, state, source, ...actual });
  };
  const hold = { en: "On hold", ko: "보류", ar: "غير صالحة حاليًا" };
  const loading = {
    en: "Loading service accounts…",
    ko: "서비스 계정을 불러오는 중…",
    ar: "جارٍ تحميل حسابات الخدمة…",
  };
  for (const locale of ["en", "ko", "ar"]) {
    for (const width of [1440, 1280, 1024, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await navigate(
        "credentials",
        { svc_c_service: service, svc_c_selected: credential },
        page,
        locale,
      );
      await page
        .locator("table tbody tr")
        .filter({ hasText: credential })
        .getByText(hold[locale], { exact: true })
        .waitFor();
      await page
        .getByRole("heading", { name: credential, exact: true })
        .waitFor();
      await inspect(
        locale,
        width,
        "credential-hold",
        "actual suspended-account API metadata",
      );
      const search = `${prefix}-${locale}-${width}`;
      let release, arrived;
      const gate = new Promise((done) => {
        release = done;
      });
      const seen = new Promise((done) => {
        arrived = done;
      });
      const delayed = async (route) => {
        const response = await route.fetch();
        arrived();
        await gate;
        await route.fulfill({ response });
      };
      await page.route("**/api/v1/identity/services?*", delayed);
      await navigate("accounts", { svc_a_q: search }, page, locale);
      await seen;
      try {
        await page
          .getByRole("status")
          .filter({ hasText: loading[locale] })
          .waitFor();
        assert.equal(await page.locator("table tbody tr").count(), 0);
        await inspect(
          locale,
          width,
          "loading",
          "held actual empty-query API response",
        );
      } finally {
        release();
      }
      await page.locator(".live-empty").waitFor();
      assert.equal(await page.locator("table tbody tr").count(), 0);
      await inspect(locale, width, "empty", "actual zero-match API response");
      await page.unroute("**/api/v1/identity/services?*", delayed);
      for (const [state, status, code] of [
        ["unavailable", 503, "dependency_unavailable"],
        ["denied", 403, "permission_denied"],
      ]) {
        const refused = (route) =>
          route.fulfill({
            status,
            contentType: "application/json",
            body: JSON.stringify({ code, request_id: "qa-m04-state-matrix" }),
          });
        await page.route("**/api/v1/identity/services?*", refused);
        const observed = page.waitForResponse(
          (response) =>
            new URL(response.url()).pathname === "/api/v1/identity/services" &&
            response.status() === status,
        );
        await navigate("accounts", { svc_a_q: search + state }, page, locale);
        await observed;
        await page.locator(".service-summary[role=alert]").waitFor();
        assert.equal(await page.locator("table tbody tr").count(), 0);
        await inspect(locale, width, state, "injected HTTP transport response");
        await page.unroute("**/api/v1/identity/services?*", refused);
      }
      console.log(`PASS state matrix ${locale}/${width}`);
    }
  }
  return variants;
}
