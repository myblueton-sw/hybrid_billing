import { useState } from "react";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { reviewMoney as money } from "./values";
import { comparable, type Prepared, type Result } from "./model";
import { totalLabels, basisLabel } from "./copy";
export default function Results({
  locale: l,
  source,
  result,
  baseline,
  comparison,
  keepBaseline,
  switchView,
}: {
  locale: Locale;
  source: Prepared;
  result?: Result;
  baseline?: Result;
  comparison: boolean;
  keepBaseline: () => void;
  switchView: (v: string) => void;
}) {
  const [search, setSearch] = useState(""),
    [page, setPage] = useState(0),
    labels = totalLabels(l);
  const targets = source.targets.filter((target) =>
      `${target.label} ${target.id}`
        .toLowerCase()
        .includes(search.trim().toLowerCase()),
    ),
    visible = targets.slice(page * 25, page * 25 + 25);
  function delta(a: string, b: string) {
    const n = BigInt(a) - BigInt(b);
    return money(n.toString(), source.currency, l, true);
  }
  function changeLabel(a: string, b: string) {
    const n = BigInt(a) - BigInt(b);
    return n > 0n
      ? t(l, "Increase", "증가", "زيادة")
      : n < 0n
        ? t(l, "Decrease", "감소", "انخفاض")
        : t(l, "No change", "변동 없음", "دون تغيير");
  }
  if (!result)
    return (
      <section>
        <h2>
          {comparison
            ? t(l, "Result comparison", "결과 비교", "مقارنة النتائج")
            : t(l, "Allocation preview", "배분 미리보기", "معاينة التوزيع")}
        </h2>
        <p>
          {t(
            l,
            "No current preview. Editing invalidates the candidate result; run a new preview from the policy editor.",
            "현재 미리보기가 없습니다. 입력이 바뀌면 후보 결과는 무효화됩니다. 배분 정책에서 새 미리보기를 실행하세요.",
            "لا توجد معاينة حالية. يُبطل التعديل النتيجة المرشّحة؛ شغّل معاينة جديدة من محرر السياسة.",
          )}
        </p>
        {baseline && (
          <p className="allocation-meta">
            {t(
              l,
              "The comparison baseline remains in this session until the source scope changes.",
              "원천 범위가 바뀌기 전까지 이 세션의 비교 기준은 유지됩니다.",
              "يبقى أساس المقارنة في هذه الجلسة حتى يتغيّر نطاق المصدر.",
            )}
          </p>
        )}
        <button onClick={() => switchView("policies")}>
          {t(
            l,
            "Open allocation policy",
            "배분 정책 열기",
            "فتح سياسة التوزيع",
          )}
        </button>
      </section>
    );
  if (comparison && !baseline)
    return (
      <section>
        <h2>{t(l, "Result comparison", "결과 비교", "مقارنة النتائج")}</h2>
        <p>
          {t(
            l,
            "No comparison baseline. Keep a current preview as the comparison baseline first.",
            "비교 기준이 없습니다. 배분 미리보기에서 현재 결과를 비교 기준으로 유지하세요.",
            "لا يوجد أساس للمقارنة. احتفظ أولاً بمعاينة حالية أساساً للمقارنة.",
          )}
        </p>
        <button onClick={() => switchView("simulation")}>
          {t(
            l,
            "Open allocation preview",
            "배분 미리보기 열기",
            "فتح معاينة التوزيع",
          )}
        </button>
      </section>
    );
  if (comparison && baseline && !comparable(baseline, result))
    return (
      <p role="alert">
        {t(
          l,
          "The baseline and candidate contexts differ. Comparison amounts are unavailable.",
          "비교 기준과 후보의 범위가 달라 금액을 비교할 수 없습니다.",
          "يختلف نطاق الأساس والنتيجة المرشّحة. مبالغ المقارنة غير متاحة.",
        )}
      </p>
    );
  return (
    <section className="allocation-results" data-allocation-candidate>
      <div className="allocation-section-head">
        <h2>
          {comparison
            ? t(l, "Result comparison", "결과 비교", "مقارنة النتائج")
            : t(l, "Allocation preview", "배분 미리보기", "معاينة التوزيع")}
        </h2>
        <span className="allocation-meta">
          {t(
            l,
            "Review arithmetic only",
            "검토용 산술 계산",
            "حسابات مراجعة فقط",
          )}
        </span>
      </div>
      <p className="allocation-meta">
        {t(
          l,
          "Idle cost: unknown. This is arithmetic conservation, not approval of an allocation policy.",
          "유휴 원가: 확인 불가. 산술 합계의 보존이며 배분 정책 승인 검증이 아닙니다.",
          "تكلفة الموارد الخاملة: غير معروفة. هذا تحقق من حفظ المجموع الحسابي وليس اعتماد سياسة توزيع.",
        )}
      </p>
      {!comparison ? (
        <>
          <div className="allocation-conservation">
            {(["assigned", "unassigned", "remainder", "source"] as const).map(
              (key) => (
                <dl key={key}>
                  <dt>{labels[key]}</dt>
                  <dd>
                    <bdi
                      dir="ltr"
                      data-allocation-total={key === "source" ? undefined : key}
                    >
                      {money(result[key], source.currency, l)}
                    </bdi>
                  </dd>
                </dl>
              ),
            )}
          </div>
          <p className="allocation-meta">
            {t(
              l,
              "Recipient allocations + unallocated amount + calculation precision remainder = collected-source basis amount.",
              "대상별 배분액 합계 + 미배분 금액 + 계산 정밀도 잔액 = 수집 원천 기준액.",
              "مجموع توزيعات المستلمين + المبلغ غير الموزّع + متبقي دقة الحساب = مبلغ أساس المصدر المجمّع.",
            )}
          </p>
          <div className="allocation-result-actions">
            <button id="allocation-baseline" onClick={keepBaseline}>
              {t(
                l,
                "Keep current result as comparison baseline",
                "현재 결과를 비교 기준으로 유지",
                "الاحتفاظ بالنتيجة الحالية أساساً للمقارنة",
              )}
            </button>
            <span className="allocation-meta">
              {t(
                l,
                "This browser session only; no server result history or approved revision.",
                "현재 브라우저 세션에서만 유지합니다. 서버 결과 이력이나 승인된 개정이 아닙니다.",
                "للجلسة الحالية في هذا المتصفح فقط؛ ليس سجل نتائج خادم أو إصداراً معتمداً.",
              )}
            </span>
          </div>
        </>
      ) : (
        <>
          <p className="allocation-meta">
            {t(
              l,
              "Same source, company, provider, month, basis, currency, recipient set and arithmetic profile.",
              "원천·회사·공급사·월·비용 기준·통화·대상 집합·계산 방식이 같은 결과입니다.",
              "نتائج للمصدر والشركة والمزوّد والشهر وأساس التكلفة والعملة والمستلمين وملف الحساب نفسها.",
            )}
          </p>
          <div className="allocation-table">
            <table>
              <thead>
                <tr>
                  <th>{t(l, "Amount", "금액", "المبلغ")}</th>
                  <th className="allocation-money">
                    {t(
                      l,
                      "Session comparison baseline",
                      "현재 세션의 비교 기준",
                      "أساس المقارنة للجلسة الحالية",
                    )}
                  </th>
                  <th className="allocation-money">
                    {t(
                      l,
                      "Current preview",
                      "현재 미리보기",
                      "المعاينة الحالية",
                    )}
                  </th>
                  <th className="allocation-money">
                    {t(l, "Change", "변동액", "التغيّر")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {(["assigned", "unassigned", "remainder"] as const).map(
                  (key) => (
                    <tr key={key}>
                      <td data-label={t(l, "Amount", "금액", "المبلغ")}>
                        {labels[key]}
                      </td>
                      <td
                        className="allocation-money"
                        data-label={t(l, "Baseline", "비교 기준", "الأساس")}
                      >
                        <bdi dir="ltr">
                          {money(baseline![key], source.currency, l)}
                        </bdi>
                      </td>
                      <td
                        className="allocation-money"
                        data-label={t(
                          l,
                          "Current preview",
                          "현재 미리보기",
                          "المعاينة الحالية",
                        )}
                      >
                        <bdi dir="ltr">
                          {money(result[key], source.currency, l)}
                        </bdi>
                      </td>
                      <td
                        className="allocation-money"
                        data-label={t(l, "Change", "변동액", "التغيّر")}
                      >
                        <bdi dir="ltr">
                          {delta(result[key], baseline![key])}
                        </bdi>
                        <small>
                          {changeLabel(result[key], baseline![key])}
                        </small>
                      </td>
                    </tr>
                  ),
                )}
              </tbody>
            </table>
          </div>
        </>
      )}
      <div className="allocation-target-toolbar">
        <label htmlFor="allocation-result-search">
          {t(
            l,
            "Find a displayed recipient",
            "표시 대상 검색",
            "البحث عن مستلم للعرض",
          )}
          <input
            id="allocation-result-search"
            type="search"
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(0);
            }}
          />
        </label>
        <p className="allocation-meta">
          {t(
            l,
            "Display search does not change the denominator or results.",
            "표시 검색으로 분모나 계산 결과를 바꾸지 않습니다.",
            "لا يغيّر بحث العرض المقام أو النتائج.",
          )}
        </p>
      </div>
      <div
        className="allocation-table"
        tabIndex={0}
        role="region"
        aria-label={t(
          l,
          "Recipient allocation results",
          "대상별 배분 결과",
          "نتائج توزيع المستلمين",
        )}
      >
        <table>
          <thead>
            <tr>
              <th>
                {t(
                  l,
                  "Review recipient access scope",
                  "검토 대상 접근 범위",
                  "نطاق وصول مستلم المراجعة",
                )}
              </th>
              {comparison && (
                <th>
                  {t(l, "Baseline percentage", "기준 비율", "نسبة الأساس")}
                </th>
              )}
              <th>
                {t(
                  l,
                  comparison ? "Current percentage" : "Allocation percentage",
                  comparison ? "현재 비율" : "배분 비율",
                  comparison ? "النسبة الحالية" : "نسبة التوزيع",
                )}
              </th>
              {comparison && (
                <th className="allocation-money">
                  {t(l, "Baseline allocation", "기준 배분액", "توزيع الأساس")}
                </th>
              )}
              <th className="allocation-money">
                {t(
                  l,
                  comparison ? "Current allocation" : "Recipient allocation",
                  comparison ? "현재 배분액" : "대상 배분액",
                  comparison ? "التوزيع الحالي" : "توزيع المستلم",
                )}
              </th>
              {comparison && (
                <th className="allocation-money">
                  {t(l, "Change", "변동액", "التغيّر")}
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {visible.map((target) => {
              const row = result.rows.find((r) => r.target === target.id)!,
                previous = baseline?.rows.find((r) => r.target === target.id);
              return (
                <tr key={target.id}>
                  <td data-label={t(l, "Recipient", "검토 대상", "المستلم")}>
                    <strong>
                      <bdi dir="ltr">{target.label}</bdi>
                    </strong>
                    <details>
                      <summary>
                        {t(l, "Recipient ID", "대상 ID", "معرّف المستلم")}
                      </summary>
                      <small>
                        <bdi dir="ltr">{target.id}</bdi>
                      </small>
                    </details>
                  </td>
                  {comparison && (
                    <td
                      data-label={t(
                        l,
                        "Baseline percentage",
                        "기준 비율",
                        "نسبة الأساس",
                      )}
                    >
                      <bdi dir="ltr">{previous!.percentage}% / 100%</bdi>
                    </td>
                  )}
                  <td
                    data-label={t(
                      l,
                      comparison
                        ? "Current percentage"
                        : "Allocation percentage",
                      comparison ? "현재 비율" : "배분 비율",
                      comparison ? "النسبة الحالية" : "نسبة التوزيع",
                    )}
                  >
                    <bdi dir="ltr">{row.percentage}% / 100%</bdi>
                  </td>
                  {comparison && (
                    <td
                      className="allocation-money"
                      data-label={t(
                        l,
                        "Baseline allocation",
                        "기준 배분액",
                        "توزيع الأساس",
                      )}
                    >
                      <bdi dir="ltr">
                        {money(previous!.amount, source.currency, l)}
                      </bdi>
                    </td>
                  )}
                  <td
                    className="allocation-money"
                    data-label={t(
                      l,
                      comparison
                        ? "Current allocation"
                        : "Recipient allocation",
                      comparison ? "현재 배분액" : "대상 배분액",
                      comparison ? "التوزيع الحالي" : "توزيع المستلم",
                    )}
                  >
                    <bdi dir="ltr">{money(row.amount, source.currency, l)}</bdi>
                  </td>
                  {comparison && (
                    <td
                      className="allocation-money"
                      data-label={t(l, "Change", "변동액", "التغيّر")}
                    >
                      <bdi dir="ltr">{delta(row.amount, previous!.amount)}</bdi>
                      <small>{changeLabel(row.amount, previous!.amount)}</small>
                    </td>
                  )}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      {!visible.length && (
        <p>
          {t(
            l,
            "No displayed recipients match. All calculated recipients remain in the result.",
            "표시 검색과 일치하는 대상이 없습니다. 전체 대상의 계산 결과는 유지됩니다.",
            "لا يوجد مستلمون مطابقون للعرض. تبقى نتائج جميع المستلمين محفوظة.",
          )}
        </p>
      )}
      <nav
        className="allocation-pages"
        aria-label={t(l, "Result pages", "결과 페이지", "صفحات النتائج")}
      >
        <button disabled={page === 0} onClick={() => setPage(page - 1)}>
          {t(l, "Previous", "이전", "السابق")}
        </button>
        <span>
          <bdi dir="ltr">
            {targets.length ? page * 25 + 1 : 0}–
            {Math.min((page + 1) * 25, targets.length)} / {targets.length}
          </bdi>
        </span>
        <button
          disabled={(page + 1) * 25 >= targets.length}
          onClick={() => setPage(page + 1)}
        >
          {t(l, "Next", "다음", "التالي")}
        </button>
      </nav>
      <details className="allocation-evidence">
        <summary>{t(l, "Result context", "결과 범위", "نطاق النتيجة")}</summary>
        <dl>
          <dt>
            {t(
              l,
              "Review context reference",
              "검토 범위 참조",
              "مرجع نطاق المراجعة",
            )}
          </dt>
          <dd>
            <bdi dir="ltr">{result.context}</bdi>
          </dd>
          <dt>{t(l, "Arithmetic profile", "계산 방식", "ملف الحساب")}</dt>
          <dd>
            <bdi dir="ltr">{result.profile}</bdi>
          </dd>
          <dt>{t(l, "Cost basis", "비용 기준", "أساس التكلفة")}</dt>
          <dd>{basisLabel(l, source.data.selection.basis)}</dd>
        </dl>
      </details>
    </section>
  );
}
