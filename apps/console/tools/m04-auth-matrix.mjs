import assert from "node:assert/strict";
import { mkdir } from "node:fs/promises";

const copy = {
  en: {
    prerequisite:
      "The authentication policy requires multi-factor authentication, which this service does not support. Confirming your password cannot satisfy this requirement.",
    unavailable:
      "Current authentication requirements could not be verified. Reload this view before trying again.",
    reauth: "Reauthenticate",
    other:
      "A credential issuance draft for another service account is retained.",
    keep: "Keep previous draft",
    discard: "Discard previous draft",
  },
  ko: {
    prerequisite:
      "인증 정책에 따라 다중 요소 인증이 필요하지만 이 서비스에서는 지원하지 않습니다. 비밀번호 재확인만으로는 이 요건을 충족할 수 없습니다.",
    unavailable:
      "현재 인증 요건을 확인할 수 없습니다. 이 보기를 새로 조회한 후 다시 시도하세요.",
    reauth: "다시 인증",
    other: "다른 서비스 계정의 자격 증명 발급 초안이 보존되어 있습니다.",
    keep: "이전 초안 유지",
    discard: "이전 초안 버리기",
  },
  ar: {
    prerequisite:
      "تتطلب سياسة المصادقة مصادقة متعددة العوامل لا تدعمها هذه الخدمة. لا يكفي تأكيد كلمة المرور لاستيفاء هذا المتطلب.",
    unavailable:
      "تعذّر التحقق من متطلبات المصادقة الحالية. أعد تحميل هذا العرض قبل المحاولة مجدداً.",
    reauth: "إعادة المصادقة",
    other: "توجد مسودة محفوظة لإصدار بيانات اعتماد لحساب خدمة آخر.",
    keep: "الاحتفاظ بالمسودة السابقة",
    discard: "تجاهل المسودة السابقة",
  },
};

export async function authMatrix({
  page,
  navigate,
  output,
  service,
  password,
  otherService,
  login,
  workspace,
}) {
  await mkdir(output, { recursive: true });
  const variants = [];
  let mode = "ready",
    commands = 0,
    reauthentications = 0;
  let previousReason;
  const target = "/api/v1/identity/services/" + service;
  const observeRequest = (request) => {
    if (new URL(request.url()).pathname.startsWith("/api/v1/commands/"))
      commands++;
  };
  const observeResponse = (response) => {
    if (
      new URL(response.url()).pathname === "/api/v1/me/reauthenticate" &&
      response.status() === 204
    ) {
      reauthentications++;
      mode = "ready";
    }
  };
  const handler = async (route) => {
    const response = await route.fetch();
    if (response.status() !== 200 || mode === "ready")
      return route.fulfill({ response });
    const body = await response.json();
    body.eligibility.step_up_state = mode;
    for (const key of [
      "can_issue",
      "can_reinstate",
      "can_declare",
      "can_suspend",
      "can_retire",
    ])
      body.eligibility[key] = false;
    return route.fulfill({ response, json: body });
  };
  const form = () =>
    page.locator("form").filter({ has: page.locator("#svc-issue-title") });
  const refresh = async () => {
    const read = page.waitForResponse(
      (response) =>
        new URL(response.url()).pathname === target &&
        response.status() === 200,
    );
    await page
      .locator(
        'section[aria-labelledby="svc-cred-title"] > .live-toolbar button',
      )
      .click();
    await read;
  };
  const capture = async (locale, width, state) => {
    await page.evaluate(async () => {
      await document.fonts.ready;
      await new Promise((done) =>
        requestAnimationFrame(() => requestAnimationFrame(done)),
      );
    });
    const actual = await page.evaluate(() => {
      const screen = document.querySelector('[data-security-screen="M04"]');
      const dialog = document.querySelector("dialog.service-reauth[open]");
      const rectangle = dialog?.getBoundingClientRect();
      const borders = [];
      for (const node of [
        ...screen.querySelectorAll(".live-note-rule,.live-step"),
        ...(dialog ? [dialog] : []),
      ]) {
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
      return {
        locale: screen.lang,
        direction: screen.getAttribute("dir"),
        documentFits: document.documentElement.scrollWidth <= innerWidth,
        dialogLocale: dialog?.lang ?? null,
        dialogFits:
          !rectangle || (rectangle.left >= 0 && rectangle.right <= innerWidth),
        font: getComputedStyle(screen).fontFamily,
        borders,
      };
    });
    assert.equal(actual.locale, locale);
    assert.equal(actual.direction, locale === "ar" ? "rtl" : "ltr");
    assert.ok(
      actual.documentFits && actual.dialogFits,
      `authentication layout overflow ${locale}/${width}/${state}`,
    );
    assert.ok(
      actual.borders.every((border) => border.neutral),
      "decorative authentication border",
    );
    assert.ok(actual.font.includes("IBM Plex"));
    if (state === "password-dialog") {
      assert.equal(actual.dialogLocale, locale);
      assert.equal(await page.locator("#reauth-password").inputValue(), "");
    }
    assert.equal(await page.locator("#svc-secret").count(), 0);
    assert.equal(await page.locator("#signin-password,#link-new").count(), 0);
    await page.screenshot({
      path: `${output}/${locale}-${width}-${state}.png`,
      fullPage: true,
    });
    variants.push({
      locale,
      width,
      state,
      source: [
        "refreshed-draft",
        "other-account-draft",
        "new-session-empty-draft",
      ].includes(state)
        ? "actual API and authenticated draft lifecycle"
        : "injected eligibility metadata over actual API",
      ...actual,
    });
  };
  page.on("request", observeRequest);
  page.on("response", observeResponse);
  await page.route("**" + target + "?*", handler);
  try {
    for (const locale of ["en", "ko", "ar"]) {
      for (const width of [1440, 1280, 1024, 390]) {
        await page.setViewportSize({ width, height: 1000 });
        await navigate(
          "credentials",
          { svc_c_service: service, svc_c_issue: service },
          page,
          locale,
        );
        await form().locator("#svc-days").waitFor();
        if (previousReason) {
          assert.equal(await form().locator("#svc-days").inputValue(), "1");
          assert.equal(
            await form().locator("#live-reason").inputValue(),
            previousReason,
            "locale/navigation replaced an unsent draft",
          );
        }
        for (const [state, key] of [
          ["prerequisite_failed", "prerequisite"],
          ["unavailable", "unavailable"],
        ]) {
          mode = state;
          await refresh();
          const notice = form().locator(".live-note-rule");
          await notice.getByText(copy[locale][key], { exact: true }).waitFor();
          assert.equal(
            await notice
              .getByRole("button", { name: copy[locale].reauth, exact: true })
              .count(),
            0,
          );
          assert.equal(await notice.getByRole("button").count(), 1);
          assert.equal(
            await form().locator("button[type=submit]").isDisabled(),
            true,
          );
          assert.equal(commands, 0);
          await capture(locale, width, state);
        }
        mode = "reauthentication_required";
        await refresh();
        const reason = `Synthetic authentication draft ${locale}/${width}`;
        await form().locator("#svc-days").fill("1");
        await form().locator("#live-reason").fill(reason);
        assert.equal(
          await form().locator("button[type=submit]").isDisabled(),
          true,
        );
        await capture(locale, width, "reauthentication-required");
        await form()
          .getByRole("button", { name: copy[locale].reauth, exact: true })
          .click();
        await page.locator("dialog.service-reauth[open]").waitFor();
        await capture(locale, width, "password-dialog");
        const confirmed = page.waitForResponse(
          (response) =>
            new URL(response.url()).pathname === "/api/v1/me/reauthenticate" &&
            response.status() === 204,
        );
        await page.locator("#reauth-password").fill(password);
        await page.locator("dialog.service-reauth button[type=submit]").click();
        await confirmed;
        await page
          .locator("dialog.service-reauth")
          .waitFor({ state: "detached" });
        await page.waitForFunction(() => {
          const title = document.querySelector("#svc-issue-title");
          return (
            title?.closest("form")?.querySelector("button[type=submit]")
              ?.disabled === false
          );
        });
        assert.equal(
          await form().locator("#svc-days").inputValue(),
          "1",
          "password confirmation replaced the untouched issuance draft",
        );
        assert.equal(
          await form().locator("#live-reason").inputValue(),
          reason,
          "password confirmation discarded the untouched rationale",
        );
        assert.equal(
          commands,
          0,
          "reauthentication automatically submitted a command",
        );
        await capture(locale, width, "refreshed-draft");
        await navigate("accounts", { svc_a_selected: service }, page, locale);
        await page
          .getByRole("heading", { name: service, exact: true })
          .waitFor();
        await navigate(
          "credentials",
          { svc_c_service: service, svc_c_issue: service },
          page,
          locale,
        );
        await form().locator("#svc-days").waitFor();
        assert.equal(await form().locator("#svc-days").inputValue(), "1");
        assert.equal(
          await form().locator("#live-reason").inputValue(),
          reason,
          "view remount discarded an unsent draft",
        );
        previousReason = reason;
        console.log(`PASS authentication matrix ${locale}/${width}`);
      }
    }
    assert.equal(reauthentications, 12);
    for (const locale of ["en", "ko", "ar"]) {
      await navigate(
        "credentials",
        { svc_c_service: otherService, svc_c_issue: otherService },
        page,
        locale,
      );
      await page.getByText(copy[locale].other, { exact: true }).waitFor();
      assert.equal(await page.locator("#svc-days").count(), 0);
      await capture(locale, 390, "other-account-draft");
      await page
        .getByRole("button", { name: copy[locale].keep, exact: true })
        .click();
      await form().locator("#svc-days").waitFor();
      assert.equal(await form().locator("#svc-days").inputValue(), "1");
      assert.equal(
        await form().locator("#live-reason").inputValue(),
        previousReason,
      );
    }
    await navigate(
      "credentials",
      { svc_c_service: otherService, svc_c_issue: otherService },
      page,
      "ar",
    );
    await page.getByText(copy.ar.other, { exact: true }).waitFor();
    await page
      .getByRole("button", { name: copy.ar.discard, exact: true })
      .click();
    await form().locator("#live-reason").waitFor();
    assert.equal(await form().locator("#live-reason").inputValue(), "");
    await form().locator("#svc-days").fill("1");
    await form()
      .locator("#live-reason")
      .fill("Synthetic draft cleared on explicit sign-out");
    let unloads = 0;
    const observeDialog = (dialog) => {
      if (dialog.type() === "beforeunload") unloads++;
    };
    page.on("dialog", observeDialog);
    await page.locator(".service-session button").click();
    await page.locator("#signin-password").waitFor();
    const origin = new URL(page.url()).origin;
    await page.goto(origin + "/?service=api&lang=en");
    await page.locator("#signin-workspace").fill(workspace);
    await page.locator("#signin-email").fill(login);
    await page.locator("#signin-password").fill(password);
    await page.locator("form button[type=submit]").click();
    await page.locator(".service-session").waitFor();
    await navigate(
      "credentials",
      { svc_c_service: otherService, svc_c_issue: otherService },
      page,
      "en",
    );
    await form().locator("#live-reason").waitFor();
    assert.equal(
      await form().locator("#live-reason").inputValue(),
      "",
      "new authentication lease inherited an old draft",
    );
    assert.equal(
      await page.locator(".live-attempts").count(),
      0,
      "new lease inherited original-session pickup recovery",
    );
    assert.equal(
      unloads,
      0,
      "explicit sign-out retained the draft unload guard",
    );
    page.off("dialog", observeDialog);
    assert.equal(
      commands,
      0,
      "draft recovery dispatched an unrequested command",
    );
    await capture("en", 390, "new-session-empty-draft");
    return variants;
  } finally {
    page.off("request", observeRequest);
    page.off("response", observeResponse);
    await page.unroute("**" + target + "?*", handler);
  }
}
