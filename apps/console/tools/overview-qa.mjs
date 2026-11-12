import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import assert from "node:assert/strict";
const args = Object.fromEntries(
  process.argv
    .slice(2)
    .reduce(
      (a, v, i, all) =>
        v.startsWith("--") ? [...a, [v.slice(2), all[i + 1]]] : a,
      [],
    ),
);
const { chromium } = createRequire(import.meta.url)(
  args["playwright-module"] || "playwright",
);
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const context = await browser.newContext(),
  page = await context.newPage(),
  errors = [];
page.on("pageerror", (e) => errors.push(e.message));
const url = args.url || "http://127.0.0.1:8787/",
  output = args.output || "qa-output/overview";
await mkdir(output, { recursive: true });
let busyRetries = 0;
const settle = async () => {
  for (let attempt = 0; attempt < 8; attempt += 1) {
    const state = await page
      .waitForFunction(
        () => {
          const root = document.querySelector("[data-focus-state]");
          const value = root?.getAttribute("data-focus-state");
          if (value === "loaded") return "loaded";
          if (value === "error")
            return root.querySelector("[role=alert]")?.textContent ?? "error";
          return false;
        },
        null,
        { timeout: 60000 },
      )
      .then((handle) => handle.jsonValue())
      .catch(() => "timeout");
    if (!String(state).includes("source_busy")) return;
    busyRetries += 1;
    await page.waitForTimeout(3000);
    await page.locator("[role=alert] button").click();
  }
};
const loaded = async () => {
  await settle();
  await page
    .waitForFunction(
      () => {
        const current = new URLSearchParams(location.hash.split("?")[1]),
          expected = new URLSearchParams();
        for (const k of [
          "period",
          "provider",
          "account",
          "basis",
          "q",
          "after",
          "account_after",
          "snapshot",
        ])
          if (current.has(`focus-${k}`))
            expected.set(
              k,
              k === "q"
                ? current.get(`focus-${k}`).trim()
                : current.get(`focus-${k}`),
            );
        const root = document.querySelector("[data-focus-state=loaded]");
        return root?.getAttribute("data-focus-query") === expected.toString();
      },
      null,
      { timeout: 30000 },
    )
    .catch(async (e) => {
      console.error(
        await page.locator("[data-focus-state]").evaluate((el) => ({
          state: el.getAttribute("data-focus-state"),
          query: el.getAttribute("data-focus-query"),
          url: location.hash,
          error: el.querySelector("[role=alert]")?.textContent,
        })),
      );
      throw e;
    });
};
const open = async (query = "") => {
  await page.goto(url + "#/home?lang=en" + query);
  await loaded();
};
const results = [];
let passed = false;
try {
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + "#/home?lang=" + lang);
      await loaded();
      await page.evaluate(() => document.fonts.ready);
      await page.evaluate(() => window.scrollTo(0, 0));
      assert.equal(await page.locator("[data-focus-rows]").innerText(), "1121");
      assert.equal(await page.locator("#focus-provider option").count(), 7);
      assert.equal(
        await page
          .locator(".focus-amounts")
          .innerText()
          .then((t) => t.includes("27.37875272899")),
        true,
      );
      assert.equal(
        await page.locator(".focus-table").nth(1).locator("tbody tr").count(),
        25,
      );
      assert.equal(
        await page.locator(".focus-table").nth(2).locator("tbody tr").count(),
        25,
      );
      assert.equal(await page.locator("#confirm-review").count(), 1);
      assert.equal(
        await page.locator("html").getAttribute("dir"),
        lang === "ar" ? "rtl" : "ltr",
      );
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      assert(
        await page.evaluate(
          (lang) =>
            document.fonts.check(
              `600 16px "${lang === "ko" ? "IBM Plex Sans KR" : lang === "ar" ? "IBM Plex Sans Arabic" : "IBM Plex Sans"}"`,
            ),
          lang,
        ),
      );
      await page.screenshot({ path: `${output}/${lang}-${width}.png` });
      await page.locator(".focus-search").scrollIntoViewIfNeeded();
      await page.screenshot({
        path: `${output}/${lang}-${width}-accounts.png`,
      });
      results.push({
        lang,
        width,
        rows: 1121,
        accountsPage: 25,
        detailPage: 25,
        overflow: false,
      });
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await open();
  const all = await page.request.get(url + "api/focus-overview", {
    headers: { "X-HB-Local-Client": "1" },
  });
  assert.equal(all.status(), 200);
  const actual = await all.json();
  assert.equal(actual.account_count, 78);
  assert.equal(actual.row_count, 1121);
  assert.deepEqual(actual.options.periods, ["2024-03", "2024-09", "2024-10"]);
  assert.equal(actual.provenance["synthetic-contract-fixture"], 6);
  assert.equal(
    actual.provenance["official-public"] +
      actual.provenance["official-public-subset"],
    1115,
  );
  const hostile = await page.request.get(url + "api/focus-overview", {
    headers: {
      "X-HB-Local-Client": "1",
      Origin: "https://evil.example",
      "Sec-Fetch-Site": "cross-site",
    },
  });
  assert.equal(hostile.status(), 403);
  const missing = await page.request.get(url + "api/focus-overview");
  assert.equal(missing.status(), 403);
  const foreign = await page.request.get(
    url + "api/focus-overview?actor=foreign",
    { headers: { "X-HB-Local-Client": "1" } },
  );
  assert.equal(foreign.status(), 400);
  const forged = await page.request.get(
    url + "api/focus-overview?account=" + "f".repeat(64),
    { headers: { "X-HB-Local-Client": "1" } },
  );
  assert.equal(forged.status(), 400);
  const stale = await page.request.get(
    url +
      "api/focus-overview?after=25&basis=effective_cost&snapshot=" +
      actual.snapshot,
    { headers: { "X-HB-Local-Client": "1" } },
  );
  assert.equal(stale.status(), 409);
  await page.locator("input[value=effective_cost]").click();
  await page.waitForURL(/focus-basis=effective_cost/);
  await page.locator("input[value=effective_cost]:checked").waitFor();
  await loaded();
  assert.equal(
    await page.locator("input[value=effective_cost]").isChecked(),
    true,
  );
  assert(
    (await page.locator(".focus-amounts").innerText()).includes(
      "18.72651418586",
    ),
  );
  await page.locator("#focus-provider").selectOption("Microsoft");
  await page.waitForURL(/focus-provider=Microsoft/);
  await loaded();
  assert.equal(await page.locator("[data-focus-rows]").innerText(), "51");
  await page.locator("#focus-period").selectOption("2024-10");
  await page.waitForURL(/focus-period=2024-10/);
  await loaded();
  assert.equal(await page.locator("[data-focus-rows]").innerText(), "0");
  await open("&focus-q=%20AWS%20");
  assert.equal(await page.locator("#focus-account-search").inputValue(), "AWS");
  await page
    .locator("#focus-account-search")
    .fill("no-such-public-sample-account");
  await page.locator(".focus-search button").click();
  await page.waitForURL(/focus-q=no-such-public-sample-account/);
  await loaded();
  assert.equal(
    await page.locator(".focus-table").nth(1).locator("tbody tr").count(),
    0,
  );
  assert.equal(await page.locator("[data-focus-rows]").innerText(), "1121");
  await open();
  await page
    .locator(".overview-paging")
    .last()
    .getByRole("button", { name: "Next", exact: true })
    .click();
  await page.waitForURL(/focus-after=25/);
  await loaded();
  assert(page.url().includes("focus-after=25"));
  await page.goBack();
  await loaded();
  assert(!page.url().includes("focus-after=25"));
  await page
    .locator(".focus-table")
    .nth(1)
    .locator("tbody th button")
    .first()
    .click();
  await page.waitForURL(/focus-account=/);
  await loaded();
  assert((await page.locator("[data-focus-rows]").innerText()) !== "1121");
  await open();
  const replay = {};
  for (const provider of ["AWS", "Microsoft"]) {
    const response = await page.request.get(
      url + "api/focus-overview?provider=" + provider,
      { headers: { "X-HB-Local-Client": "1" } },
    );
    assert.equal(response.status(), 200);
    replay[provider] = await response.json();
  }
  await page.route("**/api/focus-overview?**", async (route) => {
    const provider = new URL(route.request().url()).searchParams.get(
      "provider",
    );
    if (provider === "AWS") await new Promise((r) => setTimeout(r, 1500));
    if (replay[provider])
      await route
        .fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(replay[provider]),
        })
        .catch(() => {});
    else await route.continue();
  });
  await page.locator("#focus-provider").selectOption("AWS");
  await page.waitForTimeout(100);
  await page.evaluate(() => {
    const p = new URLSearchParams(location.hash.split("?")[1]);
    p.set("focus-provider", "Microsoft");
    location.hash = "#/home?" + p;
  });
  await loaded();
  assert.equal(await page.locator("[data-focus-rows]").innerText(), "51");
  await page.waitForTimeout(1700);
  assert.equal(await page.locator("[data-focus-rows]").innerText(), "51");
  await page.unroute("**/api/focus-overview?**");
  await page.route("**/api/focus-overview?**", (route) =>
    route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({ error: "collected_unavailable" }),
    }),
  );
  await page.locator("#focus-provider").selectOption("Oracle");
  await page.locator("[data-focus-state=error]").waitFor();
  assert.equal(await page.locator(".focus-amounts").count(), 0);
  assert.equal(await page.locator(".focus-table").count(), 0);
  assert.equal(
    await page
      .locator(
        ".focus-overview :is([data-source-value],[data-u01-sum],[data-u01-field],[data-u01-row])",
      )
      .count(),
    0,
  );
  await page.unroute("**/api/focus-overview?**");
  await page
    .locator("[role=alert]")
    .getByRole("button", { name: "Refresh query", exact: true })
    .click();
  await loaded();
  assert.equal(await page.locator("[data-focus-rows]").innerText(), "7");
  await page.goto(url + "#/home?lang=en&focus-account=" + "f".repeat(64));
  await page.locator("[data-focus-state=error]").waitFor();
  assert.equal(await page.locator(".focus-amounts").count(), 0);
  const words = {
    en: {
      missing: "Missing",
      none: "No values present",
      allMissing: "All values missing",
      count: (n) => `${n} missing`,
    },
    ko: {
      missing: "누락",
      none: "값 없음",
      allMissing: "모든 값 누락",
      count: (n) => `누락 ${n}개`,
    },
    ar: {
      missing: "مفقود",
      none: "لا قيم موجودة",
      allMissing: "كل القيم مفقودة",
      count: (n) => `المفقود: ${n}`,
    },
  };
  const fresh = async (lang, width, query = "") => {
    await page.setViewportSize({ width, height: 1000 });
    await page.goto("about:blank");
    await page.goto(url + `#/home?lang=${lang}${query}`);
    await loaded();
    await page.evaluate(() => document.fonts.ready);
    await page
      .locator(".focus-trend details")
      .evaluate((el) => (el.open = true));
  };
  const text = (selector) => page.locator(selector).first().textContent();
  const regions =
    ".focus-amounts,.focus-providers,.focus-accounts,.focus-source";
  const shape = () =>
    page.locator(".focus-overview").evaluate((el) => ({
      overflow: document.documentElement.scrollWidth > innerWidth + 1,
      isolated: [
        ...el.querySelectorAll(
          ".source-state,[data-u01-missing],[data-u01-sum=none],[data-u01-field=evidence_class],[data-u01-month]",
        ),
      ].filter((x) => x.closest("bdi")).length,
      aggregateStates: [
        ...el.querySelectorAll(
          ":is(.focus-amounts,.focus-providers,.focus-trend,.focus-account-money) :is([data-source-value],.source-state)",
        ),
      ].length,
      values: el.querySelectorAll("[data-source-value]").length,
      sums: el.querySelectorAll("[data-u01-sum]").length,
      broken: [
        ...el.querySelectorAll(
          ".focus-source .source-instant,.focus-source [data-source-instant=raw],.focus-exact",
        ),
      ]
        .filter((x) => x.getClientRects().length > 1)
        .map((x) => x.textContent),
      outside: [...el.querySelectorAll(".focus-table td,.focus-table th")]
        .filter((x) => x.getClientRects().length)
        .filter((x) => {
          const box = x.getBoundingClientRect();
          return [...x.querySelectorAll("*")]
            .filter((c) => c.getClientRects().length)
            .some((c) => {
              const r = c.getBoundingClientRect();
              return r.left < box.left - 1 || r.right > box.right + 1;
            });
        })
        .map((x) => x.textContent?.slice(0, 40)),
      clipped: [
        ...el.querySelectorAll(
          ".focus-providers .focus-table,.focus-trend .table-wrap",
        ),
      ]
        .filter(
          (x) =>
            x.scrollHeight > x.clientHeight + 1 ||
            x.scrollWidth > x.clientWidth + 1,
        )
        .map((x) => x.className),
      detached: [
        ...el.querySelectorAll(".focus-table, .focus-trend .table-wrap"),
      ]
        .filter((x) => x.getClientRects().length)
        .filter((x) => {
          const box = x.getBoundingClientRect(),
            head = x.querySelector("thead").getBoundingClientRect();
          return (
            x.scrollTop === 0 &&
            (head.top < box.top - 1 || head.top > box.top + 1)
          );
        })
        .map((x) => x.className),
      targets: [...el.querySelectorAll(".focus-evidence-toggle")]
        .map((x) => Math.round(x.getBoundingClientRect().height))
        .filter((h) => h < (innerWidth <= 767 ? 44 : 32)),
      evidence: [...el.querySelectorAll(".focus-evidence dl")].map((x) => {
        const r = x.getBoundingClientRect();
        return r.left >= -1 && r.right <= innerWidth + 1;
      }),
      notes: el.querySelectorAll(".source-state-note").length,
    }));
  const sound = (s, label) => {
    assert.equal(s.overflow, false, label);
    assert.equal(s.isolated, 0, label);
    assert.equal(s.aggregateStates, 0, label);
    assert.ok(s.values > 0 && s.sums > 0, label);
    assert.deepEqual(s.broken, [], label);
    assert.deepEqual(s.outside, [], label);
    assert.deepEqual(s.clipped, [], label);
    assert.deepEqual(s.detached, [], label);
    assert.deepEqual(s.targets, [], label);
    assert.deepEqual(s.evidence, [true], label);
    assert.equal(s.notes, 0, label);
  };
  const instant = (v) => `${v.slice(0, 10)} ${v.slice(11, 19)} UTC`;
  const real = actual.rows[0],
    rowSelector = `[data-u01-row="${real.id}"]`,
    toggle = `[data-u01-evidence-toggle="${real.id}"]`;
  await fresh("en", 1440);
  assert.deepEqual(
    await page
      .locator(".focus-source thead th")
      .evaluateAll((els) => els.map((el) => el.textContent)),
    [
      "Charge-start month",
      "ProviderService",
      "Billing account IDSubaccount ID",
      "Billed cost",
      "Effective cost",
      "Sample provenance",
    ],
  );
  assert.deepEqual(
    await page
      .locator(".focus-accounts thead th")
      .evaluateAll((els) => els.slice(2, 4).map((el) => el.textContent)),
    ["Billing account ID", "Subaccount ID"],
  );
  assert.equal(
    await text("[data-u01-provenance]"),
    [
      ["official-public", "Public sample"],
      ["official-public-subset", "Public sample subset"],
      ["synthetic-contract-fixture", "Synthetic test data"],
    ]
      .filter(([k]) => actual.provenance[k] > 0)
      .map(([k, label]) => `${label} ${actual.provenance[k]}`)
      .join(" · "),
  );
  assert.equal(
    Object.keys(actual.provenance).every((k) =>
      [
        "official-public",
        "official-public-subset",
        "synthetic-contract-fixture",
      ].includes(k),
    ),
    true,
  );
  assert.equal(
    await text(`${rowSelector} [data-u01-field=charge_month]`),
    real.charge_month,
  );
  assert.equal(
    await text(`${rowSelector} [data-u01-field=billing_account]`),
    real.billing_account,
  );
  assert.equal(
    await text(`${rowSelector} [data-u01-field=evidence_class]`),
    {
      "official-public": "Public sample",
      "official-public-subset": "Public sample subset",
      "synthetic-contract-fixture": "Synthetic test data",
    }[real.evidence_class],
  );
  assert.equal(await page.locator(".focus-source details").count(), 0);
  const widths = () =>
    page
      .locator(`${rowSelector} > td`)
      .evaluateAll((els) =>
        els.map((el) => Math.round(el.getBoundingClientRect().width)),
      );
  const before = await widths();
  await page.locator(toggle).click();
  assert.equal(
    await page.locator(toggle).getAttribute("aria-expanded"),
    "true",
  );
  assert.deepEqual(await widths(), before);
  const evidence = `#u01-evidence-${real.id}`;
  assert.equal(
    await text(`${evidence} [data-u01-field=charge_start] dd`),
    instant(real.charge_start),
  );
  assert.equal(
    await text(`${evidence} [data-u01-field=charge_end] dd`),
    instant(real.charge_end),
  );
  assert.equal(await text(`${evidence} [data-u01-field=batch] dd`), real.batch);
  assert.equal(
    await text(`${evidence} [data-u01-field=ordinal] dd`),
    real.ordinal,
  );
  await page
    .locator(`[data-u01-evidence-toggle="${actual.rows[1].id}"]`)
    .click();
  assert.equal(await page.locator(".focus-evidence").count(), 1);
  assert.equal(
    await page.locator(toggle).getAttribute("aria-expanded"),
    "false",
  );
  await page.locator(toggle).click();
  const superseded =
    /official-public|synthetic-contract|BillingAccountId|SubAccountId|Not provided|제공 없음|غير مقدم|Unknown|미확인|غير معلوم|Partial:|부분:|جزئي:|No collected value|수집 값 없음|لا قيمة مجمعة|—/;
  assert.doesNotMatch(
    (await page.locator(regions).allTextContents()).join(" "),
    superseded,
  );
  assert.doesNotMatch(
    await page.locator(".focus-trend tbody").textContent(),
    superseded,
  );
  sound(await shape(), "real en 1440");
  const drop = (measure, value) => {
    measure.present -= 1;
    measure.missing += 1;
    measure.coefficient = measure.present
      ? (BigInt(measure.coefficient) - BigInt(value)).toString()
      : null;
  };
  const empty = (amount) => {
    amount.billed_cost = {
      coefficient: null,
      present: 0,
      missing: amount.rows,
    };
  };
  const patch = (d) => {
    const r = d.rows[0],
      of = (list) => list?.find((a) => a.currency === r.currency);
    assert.notEqual(r.currency, "CNY");
    if (r.billed_cost !== null) {
      for (const amount of [
        of(d.amounts),
        of(d.providers.find((p) => p.provider === r.provider)?.amounts),
        of(d.months.find((m) => m.period === r.charge_month)?.amounts),
        of(d.accounts.find((a) => a.id === r.account)?.amounts),
      ])
        if (amount) drop(amount.billed_cost, r.billed_cost);
      r.billed_cost = null;
    }
    r.service = null;
    r.sub_account = null;
    r.charge_start = r.charge_start.replace("+00:00", "Z");
    d.accounts[0].sub_account = null;
    for (const list of [
      d.amounts,
      ...d.providers.map((p) => p.amounts),
      ...d.months.map((m) => m.amounts),
      ...d.accounts.map((a) => a.amounts),
    ])
      for (const amount of list) if (amount.currency === "CNY") empty(amount);
    for (const row of d.rows)
      if (row.currency === "CNY") row.billed_cost = null;
    return d;
  };
  const overviewRequest = (u) => u.pathname.endsWith("/api/focus-overview");
  await page.route(overviewRequest, async (route) => {
    const q = new URL(route.request().url()).searchParams;
    const live = await route.fetch();
    if (
      !live.ok() ||
      ["period", "provider", "account", "q", "after", "account_after"].some(
        (k) => q.get(k),
      ) ||
      (q.get("basis") ?? "billed_cost") !== "billed_cost"
    )
      return route.fulfill({ response: live });
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(patch(await live.json())),
    });
  });
  const degraded = async (lang, width) => {
    const w = words[lang];
    await fresh(lang, width, "&focus-currency=CNY");
    assert.equal(
      await page.locator(".focus-overview").getAttribute("data-focus-state"),
      "loaded",
    );
    assert.equal(
      await text(`${rowSelector} [data-u01-field=billed_cost]`),
      w.missing,
    );
    assert.equal(
      await text(`${rowSelector} [data-u01-field=service]`),
      w.missing,
    );
    assert.equal(
      await text(`${rowSelector} [data-u01-field=sub_account]`),
      w.missing,
    );
    assert.equal(
      await text(".focus-accounts [data-u01-field=account-sub_account]"),
      w.missing,
    );
    assert.equal(
      await page.locator(".focus-amounts [data-u01-sum=none]").count(),
      1,
    );
    assert.equal(await text(".focus-amounts [data-u01-sum=none]"), w.none);
    assert.deepEqual(
      (
        await page
          .locator(".focus-amounts [data-u01-missing]")
          .allTextContents()
      ).sort(),
      actual.amounts
        .map((a) =>
          a.currency === "CNY"
            ? a.rows
            : a.billed_cost.missing +
              (a.currency === real.currency && real.billed_cost !== null
                ? 1
                : 0),
        )
        .filter((n) => n > 0)
        .map(w.count)
        .sort(),
    );
    assert.ok(
      (await page.locator(".focus-providers [data-u01-sum=none]").count()) >= 2,
    );
    assert.ok(
      (await page.locator(".focus-providers [data-u01-missing]").count()) >= 2,
    );
    if (actual.accounts.some((a) => a.id === real.account))
      assert.ok(
        (await page.locator(".focus-accounts [data-u01-missing]").count()) > 0,
      );
    const months = await page
      .locator(".focus-trend tbody [data-u01-month-state]")
      .allTextContents();
    assert.ok(months.length > 0 && months.every((m) => m === w.allMissing));
    assert.equal(
      await page.locator(".focus-trend tbody [data-u01-sum=none]").count(),
      months.length,
    );
    assert.ok(
      (await page
        .locator(".focus-trend tbody [data-u01-month=absent]")
        .count()) > 0,
    );
    assert.doesNotMatch(
      (await page.locator(regions).allTextContents()).join(" "),
      superseded,
    );
    assert.doesNotMatch(
      await page.locator(".focus-trend tbody").textContent(),
      superseded,
    );
    await page.locator(toggle).click();
    assert.equal(
      await text(
        `${evidence} [data-u01-field=charge_start] [data-source-instant=raw]`,
      ),
      real.charge_start.replace("+00:00", "Z"),
    );
    sound(await shape(), `degraded ${lang} ${width}`);
    await page.locator(".focus-source").scrollIntoViewIfNeeded();
    await page.screenshot({
      path: `${output}/${lang}-${width}-missing.png`,
      fullPage: true,
    });
  };
  for (const lang of ["en", "ko", "ar"]) await degraded(lang, 1440);
  for (const lang of ["ko", "ar"]) await degraded(lang, 390);
  await page.unroute(overviewRequest);
  const layouts = [];
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 1200, 1199, 1024, 768, 767, 390]) {
      await fresh(lang, width);
      assert.equal(
        await page
          .locator(
            `${rowSelector} [data-u01-field=billed_cost] [data-source-value]`,
          )
          .getAttribute("data-source-value"),
        real.billed_cost === null
          ? "missing"
          : BigInt(real.billed_cost) === 0n
            ? "zero"
            : "value",
        `real data ${lang} ${width}`,
      );
      assert.equal(
        await page.locator(`${evidence} [data-source-instant=raw]`).count(),
        0,
      );
      await page.locator(toggle).click();
      const s = await shape();
      sound(s, `${lang} ${width}`);
      layouts.push({
        lang,
        width,
        scroll: await page
          .locator(".focus-source .focus-table")
          .evaluate((el) => el.scrollWidth - el.clientWidth),
      });
      await page.screenshot({
        path: `${output}/${lang}-${width}-detail.png`,
        fullPage: true,
      });
    }
  assert.equal(layouts.length, 21);
  console.log(JSON.stringify(layouts.filter((x) => x.scroll > 1)));
  passed = true;
} finally {
  await writeFile(
    `${output}/report.json`,
    JSON.stringify(
      {
        verdict: passed ? "PASS" : "FAIL",
        variants: results,
        errors,
        busyRetries,
      },
      null,
      2,
    ),
  );
  await browser.close();
}
assert.equal(results.length, 6);
assert.deepEqual(errors, []);
console.log(JSON.stringify({ busyRetries }));
console.log(
  "PASS: six locale/layout variants; retained-source readback; precise basis switch; provider/month/empty/account/search/pages/history; hostile proxy; stale continuation; late response; failure clearing/recovery; foreign scope; source-field terms on real rows, routed missing row and empty-currency aggregates in five variants, reference row and layout at seven widths in three languages. Owner review untouched.",
);
