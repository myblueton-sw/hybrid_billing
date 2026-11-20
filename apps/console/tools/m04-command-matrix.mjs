import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";

export async function commandMatrix({ page, navigate, output, outcome }) {
  await mkdir(output, { recursive: true });
  const variants = [];
  const titles =
    outcome === "unknown"
      ? [
          "Command outcome unknown",
          "명령 결과 확인 안 됨",
          "نتيجة الأمر غير معروفة",
        ]
      : ["Command refused", "명령 거절됨", "رُفض الأمر"];
  for (const [index, locale] of ["en", "ko", "ar"].entries()) {
    for (const width of [1440, 1280, 1024, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await navigate("credentials", {}, page, locale);
      const notice = page.locator(".live-held");
      await notice
        .getByRole("heading", { name: titles[index], exact: true })
        .first()
        .waitFor();
      assert.equal(
        await notice
          .getByRole("heading", { name: titles[index], exact: true })
          .count(),
        1,
      );
      assert.equal(
        await notice.getByText("revision_conflict", { exact: true }).count(),
        1,
      );
      assert.equal(
        await notice
          .getByText("PRIVATE_UNRENDERED_MESSAGE", { exact: true })
          .count(),
        0,
      );
      if (outcome === "unknown") {
        assert.equal(
          await notice
            .getByRole("heading", {
              name: ["Command refused", "명령 거절됨", "رُفض الأمر"][index],
              exact: true,
            })
            .count(),
          0,
        );
        assert.equal(await notice.locator("..").getByRole("button").count(), 1);
      }
      await page.evaluate(() => document.fonts.ready);
      const state = await page.evaluate(() => {
        const screen = document.querySelector('[data-security-screen="M04"]');
        const borders = [];
        for (const node of screen.querySelectorAll(
          ".live-held,.service-summary",
        )) {
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
            borders.push(
              values.length >= 3 &&
                Math.max(...values.slice(0, 3)) -
                  Math.min(...values.slice(0, 3)) <=
                  45,
            );
          }
        }
        return {
          locale: screen.lang,
          direction: screen.dir,
          fits: document.documentElement.scrollWidth <= innerWidth,
          font: getComputedStyle(screen).fontFamily,
          neutralBorders: borders.every(Boolean),
        };
      });
      assert.equal(state.locale, locale);
      assert.equal(state.direction, locale === "ar" ? "rtl" : "ltr");
      assert.ok(
        state.fits && state.neutralBorders && state.font.includes("IBM Plex"),
      );
      assert.equal(
        await page
          .locator("#svc-secret,#reauth-password,#signin-password")
          .count(),
        0,
      );
      await page.screenshot({
        path: `${output}/${locale}-${width}-${outcome}.png`,
        fullPage: true,
      });
      variants.push({
        locale,
        width,
        outcome,
        source:
          outcome === "unknown"
            ? "actual accepted original response deliberately dropped; replay409 injected"
            : "actual API first revision409 refusal",
        ...state,
      });
    }
  }
  await page.setViewportSize({ width: 1440, height: 1000 });
  return variants;
}
