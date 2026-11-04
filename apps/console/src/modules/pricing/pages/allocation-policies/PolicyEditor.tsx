import { useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { reviewMoney as money } from "./values";
import {
  percentage,
  denominator,
  type Prepared,
  type Draft,
  type Result,
} from "./model";
import { percent, totalLabels } from "./copy";
import PercentText from "./PercentText";
export default function PolicyEditor({
  locale: l,
  source,
  draft,
  result,
  attempted,
  update,
  run,
  persist,
  saved,
  notice,
  noticeError,
  switchView,
  focusRequest,
}: {
  locale: Locale;
  source: Prepared;
  draft: Draft;
  result?: Result;
  attempted: boolean;
  update: (d: Draft) => void;
  run: () => void;
  persist: () => void;
  saved: boolean;
  notice?: string;
  noticeError: boolean;
  switchView: (v: string) => void;
  focusRequest: { id: string; nonce: number } | null;
}) {
  const [search, setSearch] = useState(""),
    [page, setPage] = useState(0),
    [touched, setTouched] = useState<Set<string>>(new Set());
  const pending = useRef<string | null>(null),
    caption = useRef<HTMLHeadingElement>(null);
  const matches = source.targets.filter((target) =>
    `${target.label} ${target.id}`
      .toLowerCase()
      .includes(search.trim().toLowerCase()),
  );
  const visible = matches.slice(page * 25, page * 25 + 25),
    values = draft.weights.map((w) => percentage(w.percentage)),
    sum = values.reduce<bigint>((n, v) => n + (v ?? 0n), 0n),
    missing = values.filter((v) => v === null).length;
  useEffect(() => {
    if (!focusRequest) return;
    setSearch("");
    const index = source.targets.findIndex((t) => t.id === focusRequest.id);
    setPage(Math.max(0, Math.floor(index / 25)));
    pending.current = focusRequest.id;
  }, [focusRequest, source.targets]);
  useEffect(() => {
    if (!pending.current) return;
    const target = document.getElementById(
      "allocation-weight-" + pending.current,
    );
    if (target) {
      requestAnimationFrame(() => target.focus());
      pending.current = null;
    }
  }, [page, search, focusRequest]);
  function pageTo(next: number) {
    setPage(next);
    requestAnimationFrame(() => caption.current?.focus());
  }
  const amount = source.data.amounts.find(
    (a) => a.currency === source.currency,
  )?.[source.data.selection.basis];
  const sourceBlocked =
    !source.currency ||
    !amount ||
    amount.coefficient === null ||
    amount.present === 0 ||
    amount.missing > 0 ||
    source.data.selection.period === "all";
  function missingTarget() {
    const invalid = draft.weights.find(
      (w) => percentage(w.percentage) === null,
    );
    if (!invalid) return;
    setSearch("");
    setPage(
      Math.max(
        0,
        Math.floor(
          source.targets.findIndex((t) => t.id === invalid.target) / 25,
        ),
      ),
    );
    pending.current = invalid.target;
    requestAnimationFrame(() =>
      document.getElementById("allocation-weight-" + invalid.target)?.focus(),
    );
  }
  return (
    <div className="allocation-policy-layout">
      <div className="allocation-policy-fields">
        <label htmlFor="allocation-name">
          {t(l, "Allocation policy name", "배분 정책명", "اسم سياسة التوزيع")}
          <input
            id="allocation-name"
            value={draft.name}
            maxLength={120}
            onChange={(e) => update({ ...draft, name: e.target.value })}
          />
        </label>
        <fieldset>
          <legend>
            {t(l, "Allocation method", "배분 방식", "طريقة التوزيع")}
          </legend>
          <label className="allocation-method">
            <input type="radio" checked readOnly name="allocation-method" />
            {t(
              l,
              "Fixed percentage — manually entered",
              "고정 비율 — 직접 입력",
              "نسبة ثابتة — إدخال يدوي",
            )}
          </label>
          <details className="allocation-meta">
            <summary>
              {t(
                l,
                "Other allocation methods",
                "다른 배분 방식",
                "طرق توزيع أخرى",
              )}
            </summary>
            <p>
              {t(
                l,
                "Usage and reservation allocation are unavailable: verified metric denominators are not connected.",
                "사용량·예약 비례 배분은 검증된 지표와 분모가 연결되지 않아 사용할 수 없습니다.",
                "التوزيع حسب الاستخدام والحجز غير متاح لعدم ربط مؤشرات ومقامات متحقّق منها.",
              )}
            </p>
          </details>
        </fieldset>
      </div>
      <aside
        className="allocation-policy-summary"
        aria-label={t(
          l,
          "Percentage denominator and actions",
          "배분 분모와 동작",
          "مقام النسب والإجراءات",
        )}
      >
        <h2>
          <PercentText>
            {t(
              l,
              "Full denominator 100%",
              "전체 분모 100%",
              "المقام الكامل 100%",
            )}
          </PercentText>
        </h2>
        <dl className="allocation-input-totals">
          <dt>
            {missing > 0
              ? t(
                  l,
                  "Valid entered percentage subtotal",
                  "유효 입력 비율 부분 합계",
                  "المجموع الفرعي للنسب المدخلة الصالحة",
                )
              : t(
                  l,
                  "Entered percentage total",
                  "입력 비율 합계",
                  "مجموع النسب المدخلة",
                )}
          </dt>
          <dd>
            <bdi dir="ltr" data-allocation-percent-total>
              {percent(sum)}%
            </bdi>
          </dd>
          <dt>
            {sum > denominator
              ? t(l, "Excess percentage", "초과 비율", "النسبة الزائدة")
              : t(
                  l,
                  "Unallocated percentage",
                  "미배분 비율",
                  "النسبة غير الموزّعة",
                )}
          </dt>
          <dd>
            {missing > 0 ? (
              t(
                l,
                "Unknown until all inputs are valid",
                "입력 완료 전 산정 불가",
                "غير معروف حتى اكتمال المدخلات الصالحة",
              )
            ) : (
              <bdi dir="ltr">
                {percent(
                  sum > denominator ? sum - denominator : denominator - sum,
                )}
                %
              </bdi>
            )}
          </dd>
          <dt>
            {t(
              l,
              "Missing or invalid inputs",
              "미입력·잘못된 입력",
              "مدخلات مفقودة أو غير صالحة",
            )}
          </dt>
          <dd>{missing}</dd>
        </dl>
        {missing > 0 && (
          <button className="allocation-text-button" onClick={missingTarget}>
            {t(
              l,
              "Show recipients needing input",
              "입력 필요한 대상 보기",
              "عرض المستلمين المحتاجين إلى إدخال",
            )}
          </button>
        )}
        <p className="allocation-meta">
          <PercentText>
            {t(
              l,
              "Percentages apply to 100% of the full basis amount. They are never normalized to visible rows or the entered total.",
              "배분 비율은 전체 기준액의 100%를 기준으로 계산합니다. 표시 대상이나 입력 비율 합계에 맞춰 재조정하지 않습니다.",
              "تُحسب النسب من 100% من مبلغ الأساس الكامل. ولا تُعاد موازنتها حسب الصفوف الظاهرة أو مجموع النسب المدخلة.",
            )}
          </PercentText>
        </p>
        <button
          id="allocation-preview"
          className="allocation-primary"
          onClick={run}
          disabled={sourceBlocked}
          aria-describedby={
            sourceBlocked ? "allocation-source-block" : undefined
          }
        >
          {t(
            l,
            "Run allocation preview",
            "배분 미리보기 실행",
            "تشغيل معاينة التوزيع",
          )}
        </button>
        {sourceBlocked && (
          <p className="allocation-meta" id="allocation-source-block">
            {t(
              l,
              "A specific month, an available currency and complete source amounts are required.",
              "과금 시작 월·원천 통화·누락 없는 기준액이 필요합니다.",
              "يلزم شهر محدّد وعملة مصدر ومبالغ مصدر مكتملة.",
            )}
          </p>
        )}
        {result && (
          <div className="allocation-mini-result" data-allocation-candidate>
            <dl>
              {(["assigned", "unassigned", "remainder"] as const).map((key) => (
                <div key={key}>
                  <dt>{totalLabels(l)[key]}</dt>
                  <dd>
                    <bdi dir="ltr" data-allocation-total={key}>
                      {money(result[key], source.currency, l)}
                    </bdi>
                  </dd>
                </div>
              ))}
            </dl>
            <button
              className="allocation-text-button"
              onClick={() => switchView("simulation")}
            >
              {t(
                l,
                "View full allocation preview",
                "전체 배분 미리보기 보기",
                "عرض معاينة التوزيع الكاملة",
              )}
            </button>
          </div>
        )}
        <div className="allocation-browser-save">
          <h3>
            {t(
              l,
              "Browser review draft",
              "브라우저 검토 초안",
              "مسودة مراجعة المتصفح",
            )}
          </h3>
          <button id="allocation-save" onClick={persist}>
            {t(
              l,
              "Save browser review draft",
              "브라우저 검토 초안 저장",
              "حفظ مسودة المراجعة في المتصفح",
            )}
          </button>
          <p
            className="allocation-meta"
            role={noticeError ? "alert" : "status"}
          >
            {notice ??
              (saved
                ? t(
                    l,
                    "Saved in this browser",
                    "이 브라우저에 저장됨",
                    "محفوظة في هذا المتصفح",
                  )
                : t(l, "Unsaved", "저장 안 됨", "غير محفوظة"))}
          </p>
          <p className="allocation-meta">
            {t(
              l,
              "This browser only; not a private per-user store or server policy. Do not enter credentials or customer information.",
              "이 브라우저에만 저장되며 사용자별 비공개 저장소나 서버 정책이 아닙니다. 자격 증명·고객 정보는 입력하지 마세요.",
              "في هذا المتصفح فقط؛ ليس مخزناً خاصاً لكل مستخدم ولا سياسة خادم. لا تُدخل بيانات اعتماد أو معلومات عملاء.",
            )}
          </p>
        </div>
      </aside>
      <section className="allocation-editor">
        <div className="allocation-section-head">
          <h2 ref={caption} tabIndex={-1}>
            {t(l, "Review recipients", "검토 대상", "مستلمو المراجعة")}
          </h2>
          <span className="allocation-meta">
            {matches.length} / {source.targets.length}
          </span>
        </div>
        <div className="allocation-target-toolbar">
          <label htmlFor="allocation-search">
            {t(
              l,
              "Find a displayed recipient",
              "표시 대상 검색",
              "البحث عن مستلم للعرض",
            )}
            <input
              id="allocation-search"
              type="search"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(0);
              }}
            />
          </label>
          <p className="allocation-meta">
            <PercentText>
              {t(
                l,
                "Search changes the display only. Every recipient remains in the 100% denominator.",
                "검색은 표시만 바꿉니다. 계산에는 전체 대상과 100% 분모를 유지합니다.",
                "يغيّر البحث العرض فقط. يبقى جميع المستلمين ضمن مقام 100%. ",
              )}
            </PercentText>
          </p>
        </div>
        <div
          className="allocation-table allocation-edit-table"
          tabIndex={0}
          role="region"
          aria-label={t(
            l,
            "Recipient percentage editor",
            "대상별 배분 비율 편집",
            "محرر نسب توزيع المستلمين",
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
                <th className="allocation-money">
                  <PercentText>
                    {t(
                      l,
                      "Allocation percentage (%)",
                      "배분 비율 (%)",
                      "نسبة التوزيع (%)",
                    )}
                  </PercentText>
                </th>
                <th>{t(l, "Input status", "입력 상태", "حالة الإدخال")}</th>
                <th className="allocation-money">
                  {t(
                    l,
                    "Preview allocation",
                    "미리보기 배분액",
                    "مبلغ معاينة التوزيع",
                  )}
                </th>
              </tr>
            </thead>
            <tbody>
              {visible.map((target) => {
                const weight = draft.weights.find(
                    (w) => w.target === target.id,
                  ),
                  value = weight?.percentage ?? "",
                  invalid = percentage(value) === null,
                  showError = invalid && (attempted || touched.has(target.id)),
                  preview = result?.rows.find((r) => r.target === target.id);
                return (
                  <tr key={target.id}>
                    <td
                      data-label={t(
                        l,
                        "Review recipient access scope",
                        "검토 대상 접근 범위",
                        "نطاق وصول مستلم المراجعة",
                      )}
                    >
                      <strong>
                        <bdi dir="ltr">{target.label}</bdi>
                      </strong>
                      <details>
                        <summary>
                          {t(
                            l,
                            "Recipient evidence",
                            "대상 근거",
                            "دليل المستلم",
                          )}
                        </summary>
                        <small>
                          <bdi dir="ltr">{target.id}</bdi>
                        </small>
                        <small>
                          <bdi dir="ltr">{target.timezone}</bdi>
                        </small>
                        <small>
                          {t(
                            l,
                            "Authorized sample access scope; organizational ownership unverified.",
                            "권한이 확인된 샘플 접근 범위이며 조직 소유 관계는 미확인입니다.",
                            "نطاق وصول عيّنة مصرّح به؛ الملكية التنظيمية غير متحقّقة.",
                          )}
                        </small>
                      </details>
                    </td>
                    <td
                      className="allocation-weight"
                      data-label={t(
                        l,
                        "Allocation percentage",
                        "배분 비율",
                        "نسبة التوزيع",
                      )}
                    >
                      <span className="allocation-percent-input">
                        <input
                          id={"allocation-weight-" + target.id}
                          data-allocation-target={target.id}
                          aria-label={`${t(l, "Allocation percentage", "배분 비율", "نسبة التوزيع")}: ${target.label}`}
                          aria-describedby={
                            "allocation-weight-help-" + target.id
                          }
                          aria-invalid={showError}
                          type="text"
                          inputMode="decimal"
                          dir="ltr"
                          value={value}
                          maxLength={8}
                          onBlur={() =>
                            setTouched((old) => new Set(old).add(target.id))
                          }
                          onChange={(e) =>
                            update({
                              ...draft,
                              weights: draft.weights.map((w) =>
                                w.target === target.id
                                  ? { ...w, percentage: e.target.value }
                                  : w,
                              ),
                            })
                          }
                        />
                        <span aria-hidden="true">%</span>
                      </span>
                      <small id={"allocation-weight-help-" + target.id}>
                        {showError
                          ? t(
                              l,
                              "Enter 0–100, up to 4 decimal places.",
                              "0~100, 소수점 4자리 이내로 입력하세요.",
                              "أدخل من 0 إلى 100، حتى 4 خانات عشرية.",
                            )
                          : t(
                              l,
                              "Up to 4 decimal places",
                              "소수점 4자리까지",
                              "حتى 4 خانات عشرية",
                            )}
                      </small>
                    </td>
                    <td
                      data-label={t(
                        l,
                        "Input status",
                        "입력 상태",
                        "حالة الإدخال",
                      )}
                    >
                      {invalid
                        ? value === ""
                          ? t(l, "Input required", "입력 필요", "الإدخال مطلوب")
                          : t(l, "Check input", "입력 확인", "تحقّق من الإدخال")
                        : t(l, "Entered", "입력됨", "مُدخل")}
                    </td>
                    <td
                      className="allocation-money"
                      data-label={t(
                        l,
                        "Preview allocation",
                        "미리보기 배분액",
                        "مبلغ معاينة التوزيع",
                      )}
                    >
                      <bdi dir="ltr">
                        {preview
                          ? money(preview.amount, source.currency, l)
                          : "—"}
                      </bdi>
                      {!preview && (
                        <small>
                          {t(
                            l,
                            "Run preview after editing",
                            "입력 후 미리보기 실행",
                            "شغّل المعاينة بعد التعديل",
                          )}
                        </small>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        {!matches.length && (
          <p>
            {t(
              l,
              "No recipients match this display search. The full recipient set is unchanged.",
              "표시 검색과 일치하는 대상이 없습니다. 전체 계산 대상은 유지됩니다.",
              "لا يوجد مستلمون يطابقون بحث العرض. مجموعة الحساب الكاملة لم تتغيّر.",
            )}
          </p>
        )}
        <nav
          className="allocation-pages"
          aria-label={t(l, "Recipient pages", "대상 페이지", "صفحات المستلمين")}
        >
          <button disabled={page === 0} onClick={() => pageTo(page - 1)}>
            {t(l, "Previous", "이전", "السابق")}
          </button>
          <span>
            <bdi dir="ltr">
              {matches.length ? page * 25 + 1 : 0}–
              {Math.min((page + 1) * 25, matches.length)} / {matches.length}
            </bdi>
          </span>
          <button
            disabled={(page + 1) * 25 >= matches.length}
            onClick={() => pageTo(page + 1)}
          >
            {t(l, "Next", "다음", "التالي")}
          </button>
        </nav>
      </section>
    </div>
  );
}
