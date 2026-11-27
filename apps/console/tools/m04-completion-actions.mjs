import assert from "node:assert/strict";

const copy = {
  en: {
    edit: "Edit declaration",
    revoke: "Revoke",
    revokeTitle: "Revoke credential",
    discard: "Discard previous draft",
    cancel: "Cancel",
  },
  ko: {
    edit: "선언 수정",
    revoke: "폐기",
    revokeTitle: "자격 증명 폐기",
    discard: "이전 초안 버리기",
    cancel: "취소",
  },
  ar: {
    edit: "تعديل الإقرار",
    revoke: "إبطال",
    revokeTitle: "إبطال بيانات الاعتماد",
    discard: "تجاهل المسودة السابقة",
    cancel: "إلغاء",
  },
};
export async function actionCompletion({
  page,
  context,
  base,
  password,
  service,
  credential,
  navigate,
  read,
  capture,
  scenario,
}) {
  let commands = 0;
  const count = (request) => {
    if (new URL(request.url()).pathname.startsWith("/api/v1/commands/"))
      commands++;
  };
  const formWith = (title) =>
    page
      .locator("form")
      .filter({ has: page.getByRole("heading", { name: title, exact: true }) });
  const openAccount = async (locale) => {
    await navigate("credentials", {}, locale);
    await navigate("accounts", { svc_a_selected: service }, locale);
    await page.getByRole("heading", { name: service, exact: true }).waitFor();
  };
  const openRevoke = async (locale) => {
    await navigate("accounts", {}, locale);
    await navigate(
      "credentials",
      { svc_c_service: service, svc_c_selected: credential },
      locale,
    );
    const detail = page.locator("section.live-step").filter({
      has: page.getByRole("heading", { name: credential, exact: true }),
    });
    await detail
      .getByRole("button", { name: copy[locale].revoke, exact: true })
      .click();
    const form = formWith(copy[locale].revokeTitle);
    await form.locator("#live-reason").waitFor();
    return form;
  };
  const fieldError = async (form, field) => {
    const control = form.locator("#" + field);
    assert.equal(await control.getAttribute("aria-invalid"), "true");
    assert.ok(
      (await control.getAttribute("aria-describedby"))
        .split(" ")
        .includes(field + "-error"),
    );
    assert.equal(await form.locator("#" + field + "-error").count(), 1);
    const before = page.url();
    const link = form.locator(`#svc-form-errors a[href="#${field}"]`);
    await link.focus();
    await link.press("Enter");
    assert.equal(page.url(), before);
    assert.equal(await page.evaluate(() => document.activeElement?.id), field);
  };
  page.on("request", count);
  try {
    await scenario(
      "distinct declaration and revoke field errors preserve scope and dispatch zero commands",
      async () => {
        for (const locale of ["en", "ko", "ar"]) {
          for (const width of [1440, 1280, 1024, 390]) {
            await page.setViewportSize({ width, height: 1000 });
            await openAccount(locale);
            await page
              .getByRole("button", { name: copy[locale].edit, exact: true })
              .click();
            const form = formWith(copy[locale].edit);
            assert.equal(
              await form.locator("#svc-owner-select,#svc-name").count(),
              0,
            );
            await form.locator("#svc-purpose").fill("");
            await form.locator("#live-reason").fill("");
            await form.locator("button[type=submit]").click();
            assert.equal(
              await page.evaluate(() => document.activeElement?.id),
              "svc-form-errors",
            );
            await capture(
              locale,
              width,
              "declaration-errors",
              "svc-form-errors",
            );
            await fieldError(form, "svc-purpose");
            await fieldError(form, "live-reason");
            await form
              .locator("#svc-purpose")
              .fill("Synthetic corrected declaration purpose");
            assert.equal(
              await form.locator("#svc-purpose").getAttribute("aria-invalid"),
              null,
            );
            assert.equal(
              await form.locator("#live-reason").getAttribute("aria-invalid"),
              "true",
            );
            await form
              .getByRole("button", { name: copy[locale].discard, exact: true })
              .click();
            assert.equal(await form.locator("#svc-form-errors").count(), 0);
            const revoke = await openRevoke(locale);
            await revoke.locator("#live-reason").fill("");
            await revoke.locator("button[type=submit]").click();
            assert.equal(
              await page.evaluate(() => document.activeElement?.id),
              "svc-form-errors",
            );
            await capture(locale, width, "revoke-errors", "svc-form-errors");
            await fieldError(revoke, "live-reason");
            await revoke
              .locator("#live-reason")
              .fill("Synthetic corrected revocation rationale");
            assert.equal(await revoke.locator("#svc-form-errors").count(), 0);
            assert.equal(
              await revoke.locator("#live-reason").getAttribute("aria-invalid"),
              null,
            );
            await revoke
              .getByRole("button", { name: copy[locale].cancel, exact: true })
              .click();
            assert.equal(commands, 0);
          }
        }
      },
    );
    await page.setViewportSize({ width: 1440, height: 1000 });
    await scenario(
      "actual password confirmation refresh retains unsent revoke rationale before explicit submission",
      async () => {
        const form = await openRevoke("en");
        const reason =
          "Synthetic exact revoke draft retained through real password confirmation";
        await form.locator("#live-reason").fill(reason);
        const before = await read(
          "/identity/service-credentials/" + credential,
        );
        assert.equal(before.eligibility.step_up_state, "ready");
        assert.equal(before.eligibility.can_revoke, true);
        const detailPath = "/api/v1/identity/service-credentials/" + credential;
        let staleServed = 0;
        const stale = async (route) => {
          const real = await (await route.fetch()).json();
          staleServed++;
          await route.fulfill({
            status: 200,
            json: { ...real, eligibility: { ...real.eligibility, step_up_state: "reauthentication_required", can_revoke: false, rotate_step_up_state: "reauthentication_required", can_rotate: false } },
          });
        };
        const isDetail = (u) => new URL(u.toString()).pathname === detailPath;
        await page.route(isDetail, stale);
        const [refreshed] = await Promise.all([
          page.waitForResponse(
            (response) =>
              new URL(response.url()).pathname ===
                "/api/v1/identity/service-credentials/" + credential &&
              response.status() === 200,
          ),
          (async () => {
            await form
              .getByRole("button", { name: "Cancel", exact: true })
              .click();
            await navigate("accounts");
            await navigate("credentials", {
              svc_c_service: service,
              svc_c_selected: credential,
            });
          })(),
        ]);
        assert.equal(
          (await refreshed.json()).eligibility.step_up_state,
          "reauthentication_required",
        );
        const canonical = page
          .locator("section.live-step")
          .filter({
            has: page.getByRole("heading", { name: credential, exact: true }),
          });
        await canonical
          .getByRole("button", { name: "Reauthenticate", exact: true })
          .waitFor();
        assert.equal(await form.count(), 0);
        assert.equal(commands, 0);
        await capture("en", 1440, "revoke-actual-stale");
        await canonical
          .getByRole("button", { name: "Reauthenticate", exact: true })
          .click();
        await page.locator("dialog.service-reauth[open]").waitFor();
        assert.equal(await page.locator("#reauth-password").inputValue(), "");
        const confirmed = page.waitForResponse(
          (response) =>
            new URL(response.url()).pathname === "/api/v1/me/reauthenticate",
        );
        assert.ok(staleServed > 0);
        await page.unroute(isDetail, stale);
        await page.locator("#reauth-password").fill(password);
        await page.locator("dialog.service-reauth button[type=submit]").click();
        assert.equal((await confirmed).status(), 204);
        await page
          .locator("dialog.service-reauth")
          .waitFor({ state: "detached" });
        await canonical
          .getByRole("button", { name: "Revoke", exact: true })
          .click();
        await form.locator("#live-reason").waitFor();
        assert.equal(await form.locator("#live-reason").inputValue(), reason);
        assert.equal(commands, 0);
        const current = await read(
          "/identity/service-credentials/" + credential,
        );
        assert.equal(current.eligibility.step_up_state, "ready");
        assert.equal(current.eligibility.can_revoke, true);
        await capture("en", 1440, "revoke-actual-confirmed");
        const accepted = page.waitForResponse(
          (response) =>
            new URL(response.url()).pathname ===
            "/api/v1/commands/service.revoke_credential",
        );
        await form.locator("button[type=submit]").click();
        const response = await accepted;
        assert.equal(response.status(), 200);
        assert.equal(
          (await response.json()).business_state,
          "credential_revoked",
        );
        await form.waitFor({ state: "detached" });
        assert.equal(commands, 1);
        assert.equal(
          (await read("/identity/service-credentials/" + credential)).row
            .current_state,
          "revoked",
        );
      },
    );
    await scenario(
      "actual Suspend Reinstate and Retire preserve canonical lifecycle meaning",
      async () => {
        for (const [action, title, state, business] of [
          [
            "suspend",
            "Suspend service account",
            "suspended",
            "service_suspended",
          ],
          [
            "reinstate",
            "Reinstate service account",
            "active",
            "service_active",
          ],
          ["retire", "Retire service account", "retired", "service_retired"],
        ]) {
          await openAccount("en");
          await page.getByRole("button", { name: title, exact: true }).click();
          const form = formWith(title);
          await form
            .locator("#live-reason")
            .fill("Synthetic explicit " + action + " lifecycle verification");
          const accepted = page.waitForResponse(
            (response) =>
              new URL(response.url()).pathname ===
              "/api/v1/commands/service." + action,
          );
          await form.locator("button[type=submit]").click();
          const response = await accepted;
          assert.equal(response.status(), 200);
          const result = await response.json();
          assert.equal(result.status, "completed");
          assert.equal(result.business_state, business);
          await form.waitFor({ state: "detached" });
          const detail = await read("/identity/services/" + service);
          assert.equal(detail.row.state, state);
          if (action === "retire")
            assert.equal(detail.eligibility.can_reinstate, false);
          await capture("en", 1440, "actual-" + action);
        }
        assert.equal(commands, 4);
      },
    );
  } finally {
    page.off("request", count);
  }
}
