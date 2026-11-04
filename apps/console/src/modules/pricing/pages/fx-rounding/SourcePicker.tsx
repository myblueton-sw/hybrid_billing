import type { Locale } from "../../../../shared/types";
import type {
  OverviewData,
  FocusRow,
} from "../../../sources/collected-focus/api";
import { t } from "./copy";
import { money } from "./values";
export function Evidence({
  locale: l,
  row: r,
}: {
  locale: Locale;
  row: FocusRow;
}) {
  return (
    <details>
      <summary>{t(l, "Source evidence", "원천 근거", "أدلة المصدر")}</summary>
      <dl className="b05-evidence">
        {[
          [
            t(l, "Billed cost", "청구 비용", "التكلفة المفوترة"),
            r.billed_cost === null
              ? t(
                  l,
                  "Source value missing",
                  "원천 값 없음",
                  "قيمة المصدر مفقودة",
                )
              : money(r.billed_cost, r.currency, l),
          ],
          [
            t(l, "Effective cost", "실효 비용", "التكلفة الفعلية"),
            r.effective_cost === null
              ? t(
                  l,
                  "Source value missing",
                  "원천 값 없음",
                  "قيمة المصدر مفقودة",
                )
              : money(r.effective_cost, r.currency, l),
          ],
          [t(l, "Row ID", "행 ID", "معرّف الصف"), r.id],
          [
            t(l, "Evidence classification", "근거 분류", "تصنيف الأدلة"),
            r.evidence_class,
          ],
          [
            t(l, "Account reference", "계정 참조", "مرجع الحساب"),
            r.billing_account,
          ],
        ].map(([k, v]) => (
          <div key={k}>
            <dt>{k}</dt>
            <dd>
              <bdi dir="ltr">{v}</bdi>
            </dd>
          </div>
        ))}
        <div>
          <dt>
            {t(
              l,
              "Source charge interval",
              "원천 과금 구간",
              "فترة رسوم المصدر",
            )}
          </dt>
          <dd>
            <bdi dir="ltr" className="b05-date">
              {r.charge_start}
            </bdi>
            {" — "}
            <bdi dir="ltr" className="b05-date">
              {r.charge_end}
            </bdi>
          </dd>
        </div>
      </dl>
    </details>
  );
}
export default function SourcePicker({
  locale: l,
  data,
  selected,
  select,
  page,
}: {
  locale: Locale;
  data: OverviewData;
  selected?: string;
  select: (id: string) => void;
  page: (n: number) => void;
}) {
  const headings = [
    t(l, "Select", "선택", "اختيار"),
    t(l, "Service / provider", "서비스 / 공급자", "الخدمة / المزوّد"),
    t(l, "Source cost", "원천 비용", "تكلفة المصدر"),
    t(l, "Review availability", "검토 가능 여부", "إتاحة المراجعة"),
    t(l, "Evidence", "근거", "الأدلة"),
  ];
  return (
    <section id="fx-source-list" className="b05-picker" tabIndex={-1}>
      <h2>
        {t(l, "Collected source rows", "수집 원천 행", "صفوف المصدر المجمّع")}
      </h2>
      <p className="b05-meta">
        <bdi dir="ltr">{data.row_count}</bdi>{" "}
        {t(
          l,
          "matching rows · up to 25 per page",
          "조건에 맞는 행 · 페이지당 최대 25개",
          "صفوف مطابقة · حتى 25 لكل صفحة",
        )}
      </p>
      {!data.rows.length ? (
        <p>
          {t(
            l,
            "No matching source rows. Missing amounts are not replaced with zero.",
            "조건에 맞는 원천 행이 없습니다. 0으로 대체하지 않습니다.",
            "لا توجد صفوف مصدر مطابقة. لا تُستبدل المبالغ المفقودة بصفر.",
          )}
        </p>
      ) : (
        <div
          className="b05-source-table b05-scroll"
          role="region"
          tabIndex={0}
          aria-label={t(l, "Source rows", "원천 행", "صفوف المصدر")}
        >
          <table>
            <thead>
              <tr>
                {headings.map((h) => (
                  <th key={h}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.rows.map((r) => (
                <tr
                  key={r.id}
                  aria-current={selected === r.id ? "true" : undefined}
                >
                  <td>
                    <button
                      data-fx-row={r.id}
                      aria-pressed={selected === r.id}
                      onClick={() => select(r.id)}
                    >
                      {selected === r.id
                        ? t(l, "Selected", "선택됨", "محدد")
                        : t(l, "Select row", "이 행 선택", "اختر الصف")}
                    </button>
                  </td>
                  <td data-label={headings[1]}>
                    {r.service ??
                      t(
                        l,
                        "Service unavailable",
                        "서비스명 없음",
                        "الخدمة غير متاحة",
                      )}
                    <small>
                      <bdi dir="ltr">{r.provider}</bdi>
                    </small>
                  </td>
                  <td data-label={headings[2]}>
                    <bdi dir="ltr">
                      {r[data.selection.basis] === null
                        ? t(
                            l,
                            "Source value missing",
                            "원천 값 없음",
                            "قيمة المصدر مفقودة",
                          )
                        : money(r[data.selection.basis]!, r.currency, l)}
                    </bdi>
                    <small>
                      {data.selection.basis === "billed_cost"
                        ? t(l, "Billed cost", "청구 비용", "التكلفة المفوترة")
                        : t(
                            l,
                            "Effective cost",
                            "실효 비용",
                            "التكلفة الفعلية",
                          )}
                    </small>
                  </td>
                  <td data-label={headings[3]}>
                    {r[data.selection.basis] === null
                      ? t(l, "Amount missing", "금액 없음", "المبلغ مفقود")
                      : t(
                          l,
                          "Source cost available",
                          "원천 비용 확인 가능",
                          "تكلفة المصدر متاحة",
                        )}
                  </td>
                  <td>
                    <Evidence locale={l} row={r} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <details className="b05-mobile-columns">
        <summary>
          {t(
            l,
            "All source columns — scroll horizontally",
            "전체 원천 열 — 가로로 스크롤",
            "كل أعمدة المصدر — مرّر أفقياً",
          )}
        </summary>
        <div
          className="b05-scroll"
          role="region"
          tabIndex={0}
          aria-label={t(
            l,
            "Full source columns",
            "전체 원천 열",
            "كل أعمدة المصدر",
          )}
        >
          <table>
            <thead>
              <tr>
                {[
                  t(l, "Row ID", "행 ID", "معرّف الصف"),
                  t(l, "Service", "서비스", "الخدمة"),
                  t(l, "Provider", "공급자", "المزوّد"),
                  t(l, "Billed cost", "청구 비용", "التكلفة المفوترة"),
                  t(l, "Effective cost", "실효 비용", "التكلفة الفعلية"),
                  t(
                    l,
                    "Source charge interval",
                    "원천 과금 구간",
                    "فترة رسوم المصدر",
                  ),
                ].map((h) => (
                  <th key={h}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.rows.map((r) => (
                <tr key={r.id}>
                  <td>
                    <bdi dir="ltr">{r.id}</bdi>
                  </td>
                  <td>{r.service}</td>
                  <td>
                    <bdi dir="ltr">{r.provider}</bdi>
                  </td>
                  <td>
                    <bdi dir="ltr">
                      {r.billed_cost === null
                        ? "—"
                        : money(r.billed_cost, r.currency, l)}
                    </bdi>
                  </td>
                  <td>
                    <bdi dir="ltr">
                      {r.effective_cost === null
                        ? "—"
                        : money(r.effective_cost, r.currency, l)}
                    </bdi>
                  </td>
                  <td>
                    <bdi dir="ltr" className="b05-date">
                      {r.charge_start}
                    </bdi>
                    {" — "}
                    <bdi dir="ltr" className="b05-date">
                      {r.charge_end}
                    </bdi>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
      <div className="b05-pager">
        <button
          id="fx-prev"
          disabled={data.after === 0}
          onClick={() => page(Math.max(0, data.after - 25))}
        >
          {t(l, "Previous", "이전", "السابق")}
        </button>
        <bdi dir="ltr">
          {data.rows.length ? data.after + 1 : 0}–
          {data.after + data.rows.length} / {data.row_count}
        </bdi>
        <button
          id="fx-next"
          disabled={data.next_after === null}
          onClick={() => data.next_after !== null && page(data.next_after)}
        >
          {t(l, "Next", "다음", "التالي")}
        </button>
      </div>
    </section>
  );
}
