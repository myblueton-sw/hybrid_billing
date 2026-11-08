import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type { OverviewData } from "../../../sources/collected-focus/api";
import { money } from "../../../sources/collected-focus/values";
import { categories, removeEntry, type Draft, type Entry } from "./model";
import { categoryLabel, basisLabel } from "./copy";
export default function CostItems({
  locale: l,
  data,
  draft,
  edit,
  update,
}: {
  locale: Locale;
  data: OverviewData;
  draft: Draft;
  edit: boolean;
  update: (d: Draft) => void;
}) {
  const rows = edit
    ? data.rows.filter((r) => draft.entries.some((e) => e.row === r.id))
    : data.rows;
  function patch(id: string, value: Partial<Entry>) {
    update({
      ...draft,
      entries: draft.entries.map((e) =>
        e.row === id ? { ...e, ...value } : e,
      ),
    });
  }
  return (
    <section className="infra-items" aria-labelledby="infra-items-title">
      <header className="infra-section-head">
        <h2 id="infra-items-title">
          {edit
            ? t(l, "Cost items", "원가 항목", "بنود التكلفة")
            : t(l, "Source candidates", "원천 후보", "صفوف المصدر المرشّحة")}
        </h2>
        <span className="infra-meta">
          {t(l, "Selected", "선택", "المحدّد")}:{" "}
          <bdi dir="ltr">{draft.entries.length} / 25</bdi>
        </span>
      </header>
      <p className="infra-meta" id="infra-selection-limit">
        {t(
          l,
          "Up to 25 rows on the current page. Changing source, month, cost basis or page resets the composition. Classifications and whole-row inclusion are user declarations, not verified evidence.",
          "현재 페이지에서 최대 25행을 선택합니다. 원천·월·비용 기준·페이지를 바꾸면 구성이 초기화됩니다. 범주와 행 전체의 포함 관계는 사용자 선언이며 검증된 근거가 아닙니다.",
          "يمكن تحديد 25 صفاً من الصفحة الحالية. تغيير المصدر أو الشهر أو أساس التكلفة أو الصفحة يعيد ضبط المكوّنات. التصنيفات وتضمين الصفوف كاملةً إقرارات من المستخدم وليست أدلة متحقّقاً منها.",
        )}
      </p>
      {!rows.length ? (
        <p>
          {t(
            l,
            "No source rows selected.",
            "선택한 원천 행이 없습니다.",
            "لا توجد صفوف مصدر محدّدة.",
          )}
        </p>
      ) : (
        <div
          className="infra-table"
          tabIndex={0}
          role="region"
          aria-label={t(
            l,
            "Source cost items",
            "원천 원가 항목",
            "بنود تكلفة المصدر",
          )}
        >
          <table>
            <thead>
              <tr>
                <th>{t(l, "Selection", "선택", "التحديد")}</th>
                <th>{t(l, "Source row", "원천 행", "صف المصدر")}</th>
                <th>{t(l, "Charge interval", "과금 구간", "فترة الرسوم")}</th>
                <th>{basisLabel(l, data.selection.basis)}</th>
                <th>{t(l, "Category", "범주", "الفئة")}</th>
                {edit && (
                  <>
                    <th>{t(l, "Included in", "포함 대상", "مشمولة ضمن")}</th>
                    <th>
                      {t(l, "Evidence reference", "근거 참조", "مرجع الدليل")}
                    </th>
                    <th>
                      {t(
                        l,
                        "Verification / approval",
                        "검증 / 승인",
                        "التحقّق / الاعتماد",
                      )}
                    </th>
                  </>
                )}
              </tr>
            </thead>
            <tbody>
              {rows.map((r, index) => {
                const entry = draft.entries.find((e) => e.row === r.id),
                  label = `${r.provider} ${r.service ?? ""} ${r.id.slice(0, 8)}`;
                return (
                  <tr key={r.id} data-selected={!!entry}>
                    <td data-label={t(l, "Selection", "선택", "التحديد")}>
                      <input
                        type="checkbox"
                        checked={!!entry}
                        data-source-row={r.id}
                        aria-label={`${t(l, "Select source row", "원천 행 선택", "تحديد صف المصدر")} ${index + 1}: ${label}`}
                        aria-describedby="infra-selection-limit"
                        disabled={!entry && draft.entries.length >= 25}
                        onChange={() =>
                          update(
                            entry
                              ? removeEntry(draft, r.id)
                              : {
                                  ...draft,
                                  entries: [
                                    ...draft.entries,
                                    {
                                      row: r.id,
                                      category: "unclassified",
                                      parent: null,
                                      reference: "",
                                    },
                                  ],
                                },
                          )
                        }
                      />
                    </td>
                    <td data-label={t(l, "Source row", "원천 행", "صف المصدر")}>
                      <strong>
                        {r.service ??
                          t(
                            l,
                            "Unspecified service",
                            "서비스 미지정",
                            "خدمة غير محدّدة",
                          )}
                      </strong>
                      <small>
                        <bdi dir="ltr">{r.provider}</bdi>
                      </small>
                      <details
                        onKeyDown={(event) => {
                          if (event.key === "Escape") {
                            event.currentTarget.open = false;
                            event.currentTarget
                              .querySelector("summary")
                              ?.focus();
                          }
                        }}
                      >
                        <summary>
                          {t(l, "Source evidence", "원천 근거", "دليل المصدر")}
                        </summary>
                        <dl>
                          <dt>{t(l, "Row ID", "행 ID", "معرّف الصف")}</dt>
                          <dd>
                            <bdi dir="ltr">{r.id}</bdi>
                          </dd>
                          <dt>
                            {t(l, "Account scope", "계정 범위", "نطاق الحساب")}
                          </dt>
                          <dd>
                            <bdi dir="ltr">{r.billing_account}</bdi>
                            {r.sub_account && (
                              <small>
                                <bdi dir="ltr">{r.sub_account}</bdi>
                              </small>
                            )}
                          </dd>
                          <dt>
                            {t(
                              l,
                              "Evidence class",
                              "근거 분류",
                              "تصنيف الدليل",
                            )}
                          </dt>
                          <dd>
                            <bdi dir="ltr">{r.evidence_class}</bdi>
                          </dd>
                          <dt>{basisLabel(l, "billed_cost")}</dt>
                          <dd>
                            <bdi dir="ltr">
                              {r.billed_cost === null
                                ? t(l, "No value", "값 없음", "لا توجد قيمة")
                                : money(r.billed_cost, r.currency, l)}
                            </bdi>
                          </dd>
                          <dt>{basisLabel(l, "effective_cost")}</dt>
                          <dd>
                            <bdi dir="ltr">
                              {r.effective_cost === null
                                ? t(l, "No value", "값 없음", "لا توجد قيمة")
                                : money(r.effective_cost, r.currency, l)}
                            </bdi>
                          </dd>
                        </dl>
                      </details>
                    </td>
                    <td
                      data-label={t(
                        l,
                        "Charge interval",
                        "과금 구간",
                        "فترة الرسوم",
                      )}
                    >
                      <bdi dir="ltr">{r.charge_month}</bdi>
                      <small>
                        <bdi dir="ltr">{r.charge_start}</bdi>
                      </small>
                      <small>
                        <bdi dir="ltr">{r.charge_end}</bdi>
                      </small>
                    </td>
                    <td
                      data-label={basisLabel(l, data.selection.basis)}
                      className="infra-money"
                    >
                      <bdi dir="ltr">
                        {r[data.selection.basis] === null
                          ? t(l, "No value", "값 없음", "لا توجد قيمة")
                          : money(r[data.selection.basis], r.currency, l)}
                      </bdi>
                      {entry?.parent && (
                        <small>
                          {t(
                            l,
                            "Declared included row",
                            "포함 선언 행",
                            "صف مُعلن مشمول",
                          )}
                        </small>
                      )}
                    </td>
                    <td data-label={t(l, "Category", "범주", "الفئة")}>
                      {edit && entry ? (
                        <select
                          aria-label={`${t(l, "Category", "범주", "الفئة")}: ${label}`}
                          value={entry.category}
                          onChange={(e) =>
                            patch(r.id, {
                              category: e.target.value as Entry["category"],
                            })
                          }
                        >
                          {categories.map((c) => (
                            <option key={c} value={c}>
                              {categoryLabel(l, c)}
                            </option>
                          ))}
                        </select>
                      ) : (
                        categoryLabel(l, entry?.category ?? "unclassified")
                      )}
                    </td>
                    {edit && entry && (
                      <>
                        <td
                          data-label={t(
                            l,
                            "Included in",
                            "포함 대상",
                            "مشمولة ضمن",
                          )}
                        >
                          <select
                            value={entry.parent ?? ""}
                            aria-label={`${t(l, "Included in", "포함 대상", "مشمولة ضمن")}: ${label}`}
                            onChange={(e) =>
                              patch(r.id, {
                                parent: e.target.value || null,
                                reference: e.target.value
                                  ? entry.reference
                                  : "",
                              })
                            }
                          >
                            <option value="">
                              {t(
                                l,
                                "No inclusion target",
                                "포함 대상 없음",
                                "بلا هدف تضمين",
                              )}
                            </option>
                            {data.rows
                              .filter(
                                (p) =>
                                  p.id !== r.id &&
                                  draft.entries.some((e) => e.row === p.id),
                              )
                              .map((p) => (
                                <option
                                  key={p.id}
                                  value={p.id}
                                  disabled={
                                    p.currency !== r.currency ||
                                    p.charge_start !== r.charge_start ||
                                    p.charge_end !== r.charge_end ||
                                    p.charge_month !== r.charge_month
                                  }
                                >
                                  {p.service ?? p.provider} ({p.id.slice(0, 8)})
                                  {p.currency !== r.currency ||
                                  p.charge_start !== r.charge_start ||
                                  p.charge_end !== r.charge_end ||
                                  p.charge_month !== r.charge_month
                                    ? ` — ${t(l, "Different currency or interval", "통화 또는 기간 다름", "عملة أو فترة مختلفة")}`
                                    : ""}
                                </option>
                              ))}
                          </select>
                          {entry.parent && (
                            <small className="infra-target-id">
                              <span>
                                {t(
                                  l,
                                  "Target row ID",
                                  "포함 대상 행 ID",
                                  "معرّف الصف الهدف",
                                )}
                              </span>
                              <bdi dir="ltr" data-inclusion-target>
                                {entry.parent}
                              </bdi>
                            </small>
                          )}
                        </td>
                        <td
                          data-label={t(
                            l,
                            "Evidence reference",
                            "근거 참조",
                            "مرجع الدليل",
                          )}
                        >
                          <input
                            aria-label={`${t(l, "Evidence reference", "근거 참조", "مرجع الدليل")}: ${label}`}
                            value={entry.reference}
                            maxLength={300}
                            required={!!entry.parent}
                            disabled={!entry.parent}
                            aria-invalid={
                              !!entry.parent && !entry.reference.trim()
                            }
                            onChange={(e) =>
                              patch(r.id, { reference: e.target.value })
                            }
                          />
                          {!!entry.parent && !entry.reference.trim() && (
                            <small>
                              {t(
                                l,
                                "Reference required",
                                "근거 참조 필요",
                                "المرجع مطلوب",
                              )}
                            </small>
                          )}
                        </td>
                        <td
                          data-label={t(
                            l,
                            "Verification / approval",
                            "검증 / 승인",
                            "التحقّق / الاعتماد",
                          )}
                        >
                          <small>
                            {t(
                              l,
                              "Unverified declaration",
                              "검증되지 않은 선언",
                              "إقرار غير متحقّق منه",
                            )}
                          </small>
                          <small>
                            {t(
                              l,
                              "No approval record",
                              "승인 기록 없음",
                              "لا يوجد سجل اعتماد",
                            )}
                          </small>
                        </td>
                      </>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
