import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import assert from "node:assert/strict";
const args = Object.fromEntries(
  process.argv
    .slice(2)
    .flatMap((v, i, a) => (v.startsWith("--") ? [[v.slice(2), a[i + 1]]] : [])),
);
const { chromium } = createRequire(import.meta.url)(
  args["playwright-module"] || "playwright",
);
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } }),
  faults = [];
page.on("pageerror", (e) => faults.push(e.message));
const url = args.url || "http://127.0.0.1:8805/",
  out = args.output || "qa-output/marketplace",
  endpoint = "**/api/focus-marketplace?*";
await mkdir(out, { recursive: true });
const loaded = () =>
  page
    .locator(".b07-screen[data-b07-state=loaded]")
    .waitFor({ timeout: 60000 });
const selected = (id) =>
  page.waitForFunction(
    (id) =>
      document
        .querySelector(".b07-screen")
        ?.getAttribute("data-b07-selected") === id,
    id,
  );
const tab = async (i) => {
  await page.locator(".planning-tabs button").nth(i).click();
  await page
    .locator(
      `.b07-screen[data-b07-view=${["buyer", "seller", "credits", "mapping"][i]}]`,
    )
    .waitFor();
};
const row = () => page.locator("button[data-b07-row]:visible").first();
const response = () =>
  page.waitForResponse((r) => r.url().includes("/api/focus-marketplace?"));
let passed = false,
  captures = [];
try {
  const first = response();
  await page.goto(url + "#/marketplace?lang=en");
  const initial = await (await first).json();
  await loaded();
  assert.equal(initial.authorized_rows, 1121);
  assert.deepEqual(initial.counts, {
    candidate: 1,
    no_difference: 1005,
    unknown: 115,
  });
  assert.equal(
    initial.candidate_amounts[0].billed_cost.coefficient,
    "342000000000000000",
  );
  assert.equal(initial.candidate_amounts[0].effective_cost.coefficient, "0");
  assert.equal(
    await page.locator(".b07-screen input:not(#b07-search)").count(),
    0,
  );
  const id = initial.rows[0].id;
  await row().click();
  await selected(id);
  await page.locator("[data-b07-detail-heading]:visible").waitFor();
  assert.match(
    await page.locator(".b07-desktop-detail").innerText(),
    /Red Hat Inc\.|Amazon Web Services/,
  );
  await page.locator(".b07-desktop-detail button").click();
  await selected("");
  await page.waitForFunction(
    (id) => document.activeElement?.getAttribute("data-b07-row") === id,
    id,
  );
  await row().click();
  await selected(id);
  await page.locator(".b07-desktop-detail button").focus();
  await page.keyboard.press("Escape");
  await selected("");
  await page.waitForFunction(
    (id) => document.activeElement?.getAttribute("data-b07-row") === id,
    id,
  );
  await tab(1);
  assert.equal(await page.locator(".b07-settlement strong").count(), 7);
  assert.equal(await page.locator(".b07-seller input").count(), 0);
  assert.match(await page.locator(".b07-seller").innerText(), /Refunds|Taxes/);
  await tab(2);
  await row().click();
  await selected(id);
  assert.match(
    await page.locator(".b07-desktop-detail").innerText(),
    /Unverified|Not connected/,
  );
  await tab(3);
  assert.match(
    await page.locator(".b07-desktop-detail").innerText(),
    /Offer \/ agreement/,
  );
  assert.equal(await page.locator(".b07-detail input").count(), 0);
  await tab(0);
  for (const text of ["AWS", " AWS "]) {
    await page.locator("#b07-search").fill(text);
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await loaded();
    assert.equal(await page.locator("button[data-b07-row]:visible").count(), 1);
  }
  await page.locator("#b07-search").fill("HB107_NO_MATCH");
  const noMatch = response();
  await page.getByRole("button", { name: "Search", exact: true }).click();
  const emptyAccounts = await (await noMatch).json();
  await loaded();
  assert.equal(emptyAccounts.account_matches, 0);
  assert.equal(emptyAccounts.counts.candidate, 1);
  assert.equal(await page.locator("button[data-b07-account]").count(), 0);
  await page.locator("#b07-search").fill("");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await loaded();
  await page.locator(".b07-account-scope summary").click();
  await page.locator("#b07-account-next").click();
  await loaded();
  assert.match(await page.locator(".b07-account-scope").innerText(), /26–50/);
  await page.locator("button[data-b07-account]").first().click();
  await loaded();
  assert.match(
    await page.locator(".b07-account-scope summary").innerText(),
    /One account scope selected/,
  );
  assert.match(
    await page.locator("[data-b07-selected-account]").textContent(),
    /^[0-9a-f]{64}$/,
  );
  await page.locator(".b07-account-scope summary").click();
  await page.locator("#b07-all-accounts").click();
  await loaded();
  await page.route(
    endpoint,
    (r) =>
      r.fulfill({
        status: 403,
        contentType: "application/json",
        body: JSON.stringify({
          error: "scope_denied",
          request_id: "qa-denied",
        }),
      }),
    { times: 1 },
  );
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await page.locator(".b07-screen[data-b07-state=error]").waitFor();
  assert.equal(await page.locator("button[data-b07-row]").count(), 0);
  assert.equal(
    await page
      .locator(
        ".b07-screen :is([data-source-value],[data-b07-field],[data-b07-account],[data-b07-sum])",
      )
      .count(),
    0,
  );
  assert.equal(
    await page.getByRole("button", { name: "Retry", exact: true }).count(),
    0,
  );
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await loaded();
  await page.route(
    endpoint,
    (r) =>
      r.fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({ error: "source_busy", request_id: "qa-busy" }),
      }),
    { times: 1 },
  );
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await page.locator(".b07-screen[data-b07-state=error]").waitFor();
  await page.getByRole("button", { name: "Retry", exact: true }).click();
  await loaded();
  await page.route(
    endpoint,
    (r) =>
      r.fulfill({
        status: 409,
        contentType: "application/json",
        body: JSON.stringify({
          error: "query_snapshot_changed",
          request_id: "qa-stale",
        }),
      }),
    { times: 1 },
  );
  await row().click();
  await page.locator(".b07-screen[data-b07-state=error]").waitFor();
  assert.equal(await page.locator(".b07-detail").count(), 0);
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await loaded();
  assert.equal(
    new URLSearchParams(new URL(page.url()).hash.split("?")[1]).has(
      "marketplace-row",
    ),
    false,
  );
  let release, started;
  const start = new Promise((r) => (started = r)),
    gate = new Promise((r) => (release = r));
  await page.route(endpoint, async (r) => {
    if (new URL(r.request().url()).searchParams.get("provider") === "AWS") {
      started();
      await gate;
      await r
        .fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            ...initial,
            selection: { ...initial.selection, provider: "AWS" },
          }),
        })
        .catch(() => {});
    } else await r.continue();
  });
  await page.locator("#b07-provider").selectOption("AWS");
  await start;
  await page.goto(url + "#/marketplace?lang=en&focus-provider=Google+Cloud");
  await loaded();
  release();
  await page.unroute(endpoint);
  assert.match(await page.locator(".b07-summary").innerText(), /115/);
  assert.equal(await row().count(), 0);
  await page.locator("#b07-provider").selectOption("VMware");
  await loaded();
  assert.match(await page.locator(".b07-summary").innerText(), /3/);
  assert.equal(await row().count(), 0);
  assert.equal(
    await page.evaluate(() => localStorage.getItem("hb-console-review-v1")),
    null,
  );
  for (const lang of ["ko", "en", "ar"]) {
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + `#/marketplace?lang=${lang}`);
      await loaded();
      await row().click();
      await selected(id);
      for (const [i, view] of [
        "buyer",
        "seller",
        "credits",
        "mapping",
      ].entries()) {
        await tab(i);
        await page.locator(`.b07-screen[data-b07-view=${view}]`).waitFor();
        await page.evaluate(() => document.fonts.ready);
        const metrics = await page.locator(".b07-screen").evaluate((el) => {
          const visible = (x) => x.getClientRects().length > 0,
            style = getComputedStyle(el);
          return {
            font: style.fontSize,
            family: style.fontFamily,
            direction: style.direction,
            overflow: document.documentElement.scrollWidth > innerWidth + 1,
            bad: [...el.querySelectorAll("button,input,select,td,th")]
              .filter(visible)
              .filter((x) => getComputedStyle(x).fontSize !== "16px")
              .map((x) => x.tagName),
            rounded: [
              ...el.querySelectorAll("button,input,select,table,section"),
            ]
              .filter(visible)
              .some(
                (x) =>
                  getComputedStyle(x).borderRadius !== "0px" ||
                  getComputedStyle(x).boxShadow !== "none",
              ),
            headings: [...el.querySelectorAll("h2")]
              .filter(visible)
              .every((x) => getComputedStyle(x).fontSize === "20px"),
            borders: [
              ...el.querySelectorAll("button,input,select,section,td,th"),
            ]
              .filter(visible)
              .flatMap((x) => {
                const s = getComputedStyle(x);
                return ["Top", "Right", "Bottom", "Left"]
                  .filter((k) => parseFloat(s[`border${k}Width`]) > 0)
                  .map((k) => s[`border${k}Color`]);
              }),
          };
        });
        const regression = await page.locator(".b07-screen").evaluate((el) => ({
          detailWeights: [
            ...el.querySelectorAll(".b07-detail p,.b07-detail dt"),
          ]
            .filter((x) => x.getClientRects().length)
            .map((x) => getComputedStyle(x).fontWeight),
          isolatedVersion: [
            ...el.querySelectorAll("p[role=status] bdi[dir=ltr]"),
          ].some((x) => x.textContent?.startsWith("FOCUS ")),
        }));
        assert.ok(regression.detailWeights.every((w) => w === "400"));
        assert.equal(regression.isolatedVersion, true);
        assert.equal(metrics.font, "16px");
        assert.equal(metrics.direction, lang === "ar" ? "rtl" : "ltr");
        assert.equal(metrics.overflow, false);
        assert.deepEqual(metrics.bad, []);
        assert.equal(metrics.rounded, false);
        assert.equal(metrics.headings, true);
        assert.ok(metrics.family.includes("IBM Plex"));
        const neutral = await page.locator(".b07-screen").evaluate((el) => {
          const s = getComputedStyle(el);
          return ["--border-control", "--border-section", "--border-row"].map(
            (v) => {
              const d = document.createElement("span");
              d.style.color = s.getPropertyValue(v);
              el.append(d);
              const c = getComputedStyle(d).color;
              d.remove();
              return c;
            },
          );
        });
        assert.ok(
          metrics.borders.every(
            (c) => neutral.includes(c) || c === "rgba(0, 0, 0, 0)",
          ),
        );
        if (width === 390 && view !== "seller") {
          const detail = page.locator(".b07-mobile-list .b07-detail:visible");
          assert.equal(await detail.count(), 1);
          assert.equal(
            await detail.evaluate(
              (el) =>
                el.closest("article")?.querySelector("button[data-b07-row]") !==
                null,
            ),
            true,
          );
        }
        await page.screenshot({
          path: `${out}/${lang}-${width}-${view}.png`,
          fullPage: true,
        });
        captures.push({ lang, width, view, metrics });
      }
    }
  }
  const words = {
    en: {
      missing: "Missing",
      zero: "Exact zero",
      none: "No values present",
      sourceNull: "Source null",
      service: "Service",
      header: "Billing account IDCharge period start",
    },
    ko: {
      missing: "누락",
      zero: "정확한 0",
      none: "값 없음",
      sourceNull: "원천 null",
      service: "서비스",
      header: "결제 계정 ID과금 기간 시작",
    },
    ar: {
      missing: "مفقود",
      zero: "صفر دقيق",
      none: "لا قيم موجودة",
      sourceNull: "قيمة مصدر فارغة",
      service: "الخدمة",
      header: "معرّف حساب الفوترةبداية فترة الرسوم",
    },
  };
  const superseded =
    /No observed amount|관측 금액 없음|لا مبلغ مرصود|None in source|원천 값 없음|Service unavailable|서비스 값 없음|[Cc]anonical account|원천 계정 참조|공급자|원천 배치|Source batch|Charge interval|Billing interval|Source provenance class|원천 출처 분류|—|−/;
  const fresh = async (lang, width) => {
    await page.setViewportSize({ width, height: 1000 });
    await page.goto("about:blank");
    await page.goto(url + `#/marketplace?lang=${lang}`);
    await loaded();
    await row().click();
    await selected(id);
    await page
      .locator(
        ".b07-unclassified, .b07-account-scope, .b07-source-context, .b07-full-table",
      )
      .evaluateAll((els) => els.forEach((el) => (el.open = true)));
    await page.evaluate(() => document.fonts.ready);
  };
  const fit = () =>
    page.locator(".b07-screen").evaluate((el) => ({
      overflow: document.documentElement.scrollWidth > innerWidth + 1,
      wide: [
        ...el.querySelectorAll(
          ".b07-detail dd,.b07-detail dt,td,th,.b07-account-list button > span,.b07-mobile-list article,.b07-currency-totals dd",
        ),
      ]
        .filter((x) => x.getClientRects().length)
        .filter((x) => {
          const box = x.getBoundingClientRect();
          return (
            x.scrollWidth > x.clientWidth + 1 ||
            [...x.querySelectorAll("*")]
              .filter((c) => c.getClientRects().length)
              .some((c) => {
                const r = c.getBoundingClientRect();
                return r.left < box.left - 1 || r.right > box.right + 1;
              })
          );
        })
        .map((x) => `${x.tagName}:${x.textContent?.slice(0, 40)}`),
      broken: [
        ...el.querySelectorAll(
          ".b07-account-value bdi,td [data-source-instant=raw],.b07-code > span,.b07-detail .source-instant",
        ),
      ]
        .filter((x) => x.getClientRects().length > 1)
        .map((x) => x.textContent),
      isolatedState: [...el.querySelectorAll(".source-state")].filter((x) =>
        x.closest("bdi[dir=ltr]"),
      ).length,
      values: [...el.querySelectorAll("[data-source-value]")].filter(
        (x) => x.getClientRects().length,
      ).length,
    }));
  const text = (selector) => page.locator(selector).first().textContent();
  const visibleDetail = ".b07-detail:visible";
  const instant = (v) => `${v.slice(0, 10)} ${v.slice(11, 19)} UTC`;
  await fresh("en", 1440);
  const real = initial.rows[0];
  for (const k of [
    "charge_start",
    "charge_end",
    "billing_start",
    "billing_end",
  ])
    assert.equal(
      await text(`${visibleDetail} [data-b07-field=${k}] dd`),
      instant(real[k]),
    );
  for (const k of ["billing_account", "account", "batch", "ordinal", "tenant"])
    assert.equal(
      await text(`${visibleDetail} [data-b07-field=${k}] dd`),
      real[k],
    );
  assert.equal(
    await text(`${visibleDetail} [data-b07-field=id] .b07-code`),
    real.id,
  );
  assert.equal(
    await page.locator(`${visibleDetail} [data-b07-field=id]`).isVisible(),
    true,
  );
  assert.match(
    await text(`${visibleDetail} [data-b07-field=id] small`),
    /not a transaction ID/,
  );
  const provenance = await text(
    `${visibleDetail} [data-b07-field=evidence_class] dd`,
  );
  assert.equal(
    provenance,
    {
      "official-public": "Public sample",
      "official-public-subset": "Public sample subset",
      "synthetic-contract-fixture": "Synthetic test data",
    }[real.evidence_class],
  );
  assert.equal(
    await text(
      `${visibleDetail} [data-b07-field=effective_cost] .source-state-note`,
    ),
    words.en.zero,
  );
  assert.equal(await page.locator("td .source-state-note").count(), 0);
  assert.equal(await text("thead th:nth-child(2)"), words.en.header);
  assert.equal(
    await text("td [data-b07-field=charge_start]"),
    instant(real.charge_start),
  );
  assert.match(
    await page.locator(".b07-account-list button").first().innerText(),
    /Billing account ID[\s\S]*Provider[\s\S]*Subaccount ID/,
  );
  const patch = (d, kind) => {
    if (d.rows?.length !== 1 || d.counts.candidate !== 1) return d;
    const amount = d.candidate_amounts.find(
      (a) => a.currency === d.rows[0].currency,
    );
    const each = (change) => {
      change(d.rows[0]);
      if (d.selected?.id === d.rows[0].id) change(d.selected);
    };
    if (kind === "missing") {
      each((r) => {
        r.billed_cost = null;
        r.service = null;
        r.sub_account = null;
        r.charge_start = r.charge_start.replace("+00:00", "Z");
        r.fields.description = { state: "source_null", value: null };
      });
      amount.billed_cost = { coefficient: null, present: 0, missing: 1 };
      if (d.accounts[0]) d.accounts[0].sub_account = null;
      if (d.unknown_amounts[0])
        d.unknown_amounts[0].billed_cost = {
          coefficient: null,
          present: 0,
          missing: d.unknown_amounts[0].rows,
        };
    }
    if (kind === "partial") {
      const extra = structuredClone(d.rows[0]);
      extra.id = (extra.id[0] === "a" ? "b" : "a") + extra.id.slice(1);
      extra.billed_cost = null;
      d.rows.push(extra);
      d.counts.candidate += 1;
      d.counts.no_difference -= 1;
      amount.rows += 1;
      amount.billed_cost.missing += 1;
      amount.effective_cost.present += 1;
    }
    if (kind === "negative") {
      each((r) => (r.billed_cost = "-2000000000000000000"));
      amount.billed_cost.coefficient = "-2000000000000000000";
    }
    return d;
  };
  const degrade = (kind) =>
    page.route(endpoint, async (r) => {
      const live = await r.fetch();
      if (!live.ok()) return r.fulfill({ response: live });
      await r.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(patch(await live.json(), kind)),
      });
    });
  assert.equal(initial.counts.candidate, 1);
  assert.equal(initial.rows.length, 1);
  await degrade("missing");
  for (const lang of ["en", "ko", "ar"]) {
    const w = words[lang];
    await fresh(lang, 1440);
    assert.equal(
      await page.locator(".b07-screen[data-b07-state=loaded]").count(),
      1,
    );
    assert.equal(await text("td [data-b07-field=service]"), w.missing);
    assert.equal(await text("td.money"), w.missing);
    assert.equal(
      await page
        .locator("td.money .source-value[data-source-value=zero]")
        .count(),
      1,
    );
    assert.equal(await page.locator("td .source-state-note").count(), 0);
    assert.equal(
      await text("td [data-b07-field=charge_start] [data-source-instant=raw]"),
      real.charge_start.replace("+00:00", "Z"),
    );
    assert.equal(
      await page.locator("td [data-b07-field=charge_start] small").count(),
      1,
    );
    assert.equal(
      await text(`${visibleDetail} [data-b07-field=billed_cost] dd`),
      w.missing,
    );
    assert.equal(
      await text(
        `${visibleDetail} [data-b07-field=effective_cost] .source-state-note`,
      ),
      w.zero,
    );
    assert.equal(
      await text(`${visibleDetail} [data-b07-field=sub_account] dd`),
      w.missing,
    );
    assert.equal(
      await page
        .locator(
          `${visibleDetail} [data-b07-field=charge_start] [data-source-instant=raw]`,
        )
        .count(),
      1,
    );
    assert.equal(
      await text(".b07-currency-totals [data-b07-sum=none]"),
      w.none,
    );
    assert.match(
      await page.locator(".b07-currency-totals").textContent(),
      /0 \/ 1/,
    );
    assert.equal(await text(".b07-unclassified [data-b07-sum=none]"), w.none);
    assert.ok(
      await page.locator(".b07-unclassified [data-b07-sum=none]").isVisible(),
    );
    assert.equal(
      await text(".b07-account-list button .source-state"),
      w.missing,
    );
    assert.doesNotMatch(
      await page.locator(".b07-screen").textContent(),
      superseded,
    );
    const shape = await fit();
    assert.equal(shape.overflow, false);
    assert.deepEqual(shape.wide, []);
    assert.deepEqual(shape.broken, []);
    assert.equal(shape.isolatedState, 0);
    assert.ok(shape.values > 0);
    await page.screenshot({
      path: `${out}/${lang}-1440-missing.png`,
      fullPage: true,
    });
  }
  for (const width of [1251, 1024, 768]) {
    await fresh("ar", width);
    const shape = await fit();
    assert.equal(shape.overflow, false, `missing ${width}`);
    assert.deepEqual(shape.wide, [], `missing ${width}`);
    assert.deepEqual(shape.broken, [], `missing ${width}`);
    await page.screenshot({
      path: `${out}/ar-${width}-missing.png`,
      fullPage: true,
    });
  }
  for (const lang of ["ko", "ar"]) {
    const w = words[lang];
    await fresh(lang, 390);
    const card = ".b07-mobile-list article:visible";
    assert.equal(
      await text(`${card} h3[data-b07-field=service]`),
      `${w.service}: ${w.missing}`,
    );
    assert.equal(
      await text(`${card} p[data-b07-field=description]`),
      w.sourceNull,
    );
    assert.equal(await text(`${card} > .b07-row-costs dd`), w.missing);
    assert.equal(
      await page.locator(`${card} > .b07-row-costs .source-state-note`).count(),
      0,
    );
    const shape = await fit();
    assert.equal(shape.overflow, false);
    assert.deepEqual(shape.wide, []);
    assert.deepEqual(shape.broken, []);
    assert.equal(shape.isolatedState, 0);
    await page.screenshot({
      path: `${out}/${lang}-390-missing.png`,
      fullPage: true,
    });
  }
  await page.unroute(endpoint);
  await degrade("partial");
  await fresh("en", 1440);
  assert.equal(await page.locator("button[data-b07-row]:visible").count(), 2);
  assert.equal(
    await page.locator("td.money [data-source-value=missing]").count(),
    1,
  );
  assert.equal(
    await text(".b07-currency-totals [data-b07-sum=value]"),
    "USD 0.342",
  );
  assert.match(
    await page.locator(".b07-currency-totals").textContent(),
    /1 \/ 1/,
  );
  assert.equal(
    await page.locator(".b07-currency-totals [data-b07-sum=none]").count(),
    0,
  );
  assert.equal(
    await page.locator(".b07-currency-totals .source-state").count(),
    0,
  );
  await page.unroute(endpoint);
  await degrade("negative");
  await fresh("en", 1440);
  assert.equal(await text("td.money bdi"), "USD -2.00");
  assert.equal(
    await text(`${visibleDetail} [data-b07-field=billed_cost] bdi`),
    "USD -2.00",
  );
  assert.equal(
    await text(".b07-currency-totals [data-b07-sum=value]"),
    "USD -2.00",
  );
  await page.unroute(endpoint);
  const layouts = [];
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 1251, 1250, 1024, 768, 767, 390]) {
      await fresh(lang, width);
      const shape = await fit();
      assert.equal(shape.overflow, false, `${lang} ${width}`);
      assert.deepEqual(shape.wide, [], `${lang} ${width}`);
      assert.deepEqual(shape.broken, [], `${lang} ${width}`);
      assert.equal(shape.isolatedState, 0);
      assert.ok(shape.values > 0);
      assert.equal(
        await page
          .locator(`${visibleDetail} .b07-source-context`)
          .evaluate((el) => el.open),
        true,
      );
      if (lang === "ar") {
        const edge = await page
          .locator(`${visibleDetail} [data-b07-field=id]`)
          .evaluate((el) => {
            const right = (x) => {
              const range = document.createRange();
              range.selectNodeContents(x);
              return Math.max(
                ...[...range.getClientRects()].map((r) => r.right),
              );
            };
            return [
              right(el.querySelector("dt")),
              right(el.querySelector(".b07-code")),
            ];
          });
        assert.ok(Math.abs(edge[0] - edge[1]) <= 2, `${width} ${edge}`);
      }
      if ([1251, 1250, 1024, 768, 767].includes(width))
        await page.screenshot({
          path: `${out}/${lang}-${width}-detail.png`,
          fullPage: true,
        });
      layouts.push({ lang, width });
    }
  assert.equal(layouts.length, 21);
  assert.deepEqual(faults, []);
  passed = true;
} finally {
  await writeFile(
    `${out}/result.json`,
    JSON.stringify({ passed, captures, faults }, null, 2),
  );
  await browser.close();
}
console.log(JSON.stringify({ passed, captures: captures.length, faults }));
