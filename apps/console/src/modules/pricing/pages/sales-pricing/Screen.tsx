import { useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { setNavigationGuard } from "../../../../shared/navigation";
import {
  loadOverview,
  ServiceError,
  type OverviewData,
} from "../../../sources/collected-focus/api";
import { focusQuery } from "../../../sources/collected-focus/query";
import {
  prepare,
  emptyDraft,
  issues,
  simulate,
  adjust,
  adjustmentIssues,
  type Prepared,
  type Draft,
  type Result,
  type Adjustment,
} from "./model";
import { save, restore } from "./store";
import { t, issueLabel, missing } from "./copy";
import { number } from "./values";
import RuleEditor from "./RuleEditor";
import SourcePicker, { Evidence } from "./SourcePicker";
import Results, { Summary } from "./Results";
import Adjustments from "./Adjustments";
import "./screen.css";
type State = {
  key: string;
  data?: OverviewData;
  source?: Prepared;
  draft?: Draft;
  result?: Result;
  baseline?: Result;
  adjustment?: Adjustment;
  error?: string;
  requestId?: string;
  notice?: string;
  saved?: boolean;
};
function pricingQuery(params: URLSearchParams) {
  const query = focusQuery(params);
  if (!params.has("focus-provider")) query.set("provider", "VMware");
  return query;
}
const items = [
  "B04:scope:source-row",
  "B04:form:unit-price-rule",
  "B04:calculation:ordered-review",
  "B04:adjustment:target-difference",
  "B04:comparison:session-baseline",
  "B04:action:browser-draft",
  "B04:state:source-unavailable",
];
export default function Screen({
  locale: l,
  params,
  view,
  onState,
  onDirty,
  canDiscard,
}: {
  locale: Locale;
  params: URLSearchParams;
  view: string;
  onState: (s: string, i: string[]) => void;
  onDirty: (d: boolean) => void;
  canDiscard: () => boolean;
}) {
  const query = pricingQuery(params);
  const row = params.get("pricing-row") ?? "",
    key = JSON.stringify([
      query.toString(),
      row,
      params.get("scope"),
      params.get("company"),
      params.get("tenant"),
    ]);
  const [state, setState] = useState<State>({ key: "" }),
    [retry, setRetry] = useState(0),
    [dirty, setDirty] = useState(false),
    [validation, setValidation] = useState<string[]>([]),
    [attempts, setAttempts] = useState({ preview: 0, save: 0, adjust: 0 }),
    [search, setSearch] = useState(query.get("q") ?? "");
  const generation = useRef(0),
    summary = useRef<HTMLDivElement>(null),
    current = state.key === key ? state : undefined,
    source = current?.source,
    draft = current?.draft;
  useEffect(() => {
    const controller = new AbortController(),
      g = ++generation.current;
    setState({ key });
    setDirty(false);
    setValidation([]);
    setAttempts({ preview: 0, save: 0, adjust: 0 });
    setSearch(query.get("q") ?? "");
    loadOverview(query, controller.signal)
      .then(async (data) => {
        if (controller.signal.aborted || g !== generation.current) return;
        if (new Set(data.rows.map((r) => r.id)).size !== data.rows.length)
          throw Error("source_duplicate");
        if (!row) {
          setState({ key, data });
          return;
        }
        const source = await prepare(data, row);
        if (controller.signal.aborted || g !== generation.current) return;
        let draft = emptyDraft(),
          notice: string | undefined;
        try {
          const restored = restore(window.localStorage, source);
          if (restored) {
            draft = restored;
            notice = "restored";
          }
        } catch {
          notice = "restore_failed";
        }
        if (!controller.signal.aborted && g === generation.current)
          setState({ key, data, source, draft, notice });
      })
      .catch((error) => {
        if (!controller.signal.aborted && g === generation.current) {
          setState({
            key,
            error:
              error instanceof ServiceError
                ? error.code
                : error instanceof Error
                  ? error.message
                  : "service_unavailable",
            requestId:
              error instanceof ServiceError ? error.requestId : undefined,
          });
          setDirty(false);
        }
      });
    return () => controller.abort();
  }, [key, retry]);
  useEffect(() => onDirty(dirty), [dirty, onDirty]);
  const review = JSON.stringify({
    phase: "sales-pricing",
    context: source?.context ?? null,
    sourceIdentity: current?.data
      ? [key, current.data.snapshot, current.data.source_snapshot]
      : null,
    view,
    status: current?.error ?? (current?.data ? "loaded" : "loading"),
    draft: draft ?? null,
    result: current?.result ?? null,
    baseline: current?.baseline ?? null,
    adjustment: current?.adjustment ?? null,
    validation,
    attempts,
    notice: current?.notice ?? null,
    saved: !!current?.saved,
  });
  useEffect(() => onState(review, items), [review, onState]);
  function update(next: Draft) {
    if (!draft) return;
    const core = (
      ["name", "unitPrice", "denominator", "discount", "fee"] as const
    ).some((k) => next[k] !== draft[k]);
    setState((old) => ({
      ...old,
      draft: next,
      result: core ? undefined : old.result,
      adjustment: undefined,
      saved: false,
      notice: undefined,
    }));
    setDirty(true);
    setValidation([]);
  }
  function go(next: URLSearchParams) {
    setDirty(false);
    onDirty(false);
    if (
      next.toString() === params.toString() ||
      JSON.stringify([
        pricingQuery(next).toString(),
        next.get("pricing-row") ?? "",
        next.get("scope"),
        next.get("company"),
        next.get("tenant"),
      ]) === key
    ) {
      setState({ key });
      setRetry((n) => n + 1);
    } else setState({ key: "pending" });
    setValidation([]);
    location.hash = `#/rates?${next}`;
  }
  function change(field: string, value: string) {
    if (!canDiscard()) return;
    const next = new URLSearchParams(params);
    next.set("focus-" + field, value);
    for (const k of [
      "pricing-row",
      "focus-after",
      "focus-account_after",
      "focus-snapshot",
    ])
      next.delete(k);
    go(next);
  }
  function refresh() {
    if (!canDiscard()) return;
    const next = new URLSearchParams(params);
    for (const k of [
      "pricing-row",
      "focus-after",
      "focus-account_after",
      "focus-snapshot",
    ])
      next.delete(k);
    if (next.toString() === params.toString()) {
      setState({ key });
      setDirty(false);
      onDirty(false);
      setRetry((n) => n + 1);
    } else go(next);
  }
  function select(id: string) {
    if (id === row || !current?.data || !canDiscard()) return;
    const next = new URLSearchParams(params);
    next.set("pricing-row", id);
    next.set("focus-snapshot", current.data.snapshot);
    go(next);
  }
  function page(after: number) {
    if (!current?.data || !canDiscard()) return;
    const next = new URLSearchParams(params);
    next.delete("pricing-row");
    next.set("focus-after", String(after));
    next.set("focus-snapshot", current.data.snapshot);
    go(next);
  }
  function switchView(nextView: string) {
    if (nextView === view) return;
    const next = new URLSearchParams(params);
    next.set("view", nextView);
    setNavigationGuard(null);
    location.hash = `#/rates?${next}`;
  }
  function invalid(codes: string[]) {
    setValidation(codes);
    requestAnimationFrame(() => summary.current?.focus());
  }
  function run() {
    setAttempts((a) => ({ ...a, preview: a.preview + 1 }));
    if (!source || !draft) {
      invalid(["row_unavailable"]);
      return;
    }
    const errors = issues(draft, source);
    if (errors.length) {
      invalid(errors);
      return;
    }
    try {
      const result = simulate(draft, source);
      setValidation([]);
      setState((old) => ({
        ...old,
        result,
        adjustment: undefined,
        notice: "preview_ready",
      }));
    } catch (e) {
      invalid([e instanceof Error ? e.message : "amount_overflow"]);
    }
  }
  function persist() {
    setAttempts((a) => ({ ...a, save: a.save + 1 }));
    if (!source || !draft) {
      invalid(["row_unavailable"]);
      return;
    }
    const errors = issues(draft, source, true);
    if (draft.target !== "") errors.push(...adjustmentIssues(draft));
    if (errors.length) {
      invalid(errors);
      return;
    }
    try {
      const candidate = simulate(draft, source);
      if (draft.target !== "") adjust(draft, source, candidate);
    } catch (e) {
      invalid([e instanceof Error ? e.message : "amount_overflow"]);
      return;
    }
    try {
      save(window.localStorage, source, draft);
      setState((old) => ({ ...old, saved: true, notice: "saved" }));
      setDirty(false);
      setValidation([]);
    } catch {
      setState((old) => ({ ...old, saved: false, notice: "save_failed" }));
      setDirty(true);
    }
  }
  function runAdjustment() {
    setAttempts((a) => ({ ...a, adjust: a.adjust + 1 }));
    if (!source || !draft || !current?.result) {
      invalid(["candidate_stale"]);
      return;
    }
    const errors = adjustmentIssues(draft);
    if (errors.length) {
      invalid(errors);
      return;
    }
    try {
      const adjustment = adjust(draft, source, current.result);
      setState((old) => ({ ...old, adjustment }));
      setValidation([]);
    } catch (e) {
      invalid([e instanceof Error ? e.message : "amount_overflow"]);
    }
  }
  const notice = current?.notice,
    noticeText =
      notice === "restored"
        ? t(
            l,
            "Browser draft restored after a fresh source query.",
            "새 원천 조회 후 브라우저 초안을 불러왔습니다.",
            "استُعيدت المسودة بعد استعلام مصدر حديث.",
          )
        : notice === "restore_failed"
          ? t(
              l,
              "Browser draft could not be restored for this context.",
              "현재 원천·범위에 맞춰 초안을 불러오지 못했습니다.",
              "تعذّرت استعادة المسودة لهذا السياق.",
            )
          : notice === "saved"
            ? t(
                l,
                "Review inputs saved in this browser.",
                "검토 입력값을 이 브라우저에 저장했습니다.",
                "حُفظت مدخلات المراجعة في هذا المتصفح.",
              )
            : notice === "save_failed"
              ? t(
                  l,
                  "Browser save failed. Changes remain unsaved.",
                  "브라우저 저장 실패. 변경 내용은 저장되지 않았습니다.",
                  "فشل الحفظ. التغييرات غير محفوظة.",
                )
              : notice === "preview_ready"
                ? t(
                    l,
                    "Review candidate calculated. No financial approval.",
                    "검토 후보를 계산했습니다. 재무 승인이 아닙니다.",
                    "حُسب المرشح للمراجعة دون اعتماد مالي.",
                  )
                : undefined;
  return (
    <div
      className="sales-pricing"
      data-pricing-state={
        current?.error ? "error" : current?.data ? "loaded" : "loading"
      }
      data-pricing-view={view}
      data-pricing-query={source?.context ?? current?.data?.snapshot ?? ""}
      data-pricing-selected-row={source?.row.id ?? ""}
      onKeyDown={(e) => {
        if (
          e.key !== "Escape" ||
          e.defaultPrevented ||
          !(e.target instanceof HTMLElement)
        )
          return;
        const d = e.target.closest<HTMLDetailsElement>("details[open]");
        if (d) {
          d.open = false;
          d.querySelector("summary")?.focus();
          e.preventDefault();
        }
      }}
    >
      <form
        className="pricing-toolbar"
        onSubmit={(e) => {
          e.preventDefault();
          change("q", search);
        }}
      >
        <label htmlFor="pricing-provider">
          {t(l, "Collected source", "수집 원천", "المصدر المجمّع")}
          <select
            id="pricing-provider"
            value={query.get("provider") ?? ""}
            onChange={(e) => change("provider", e.target.value)}
          >
            <option value="">
              {t(
                l,
                "All authorized providers",
                "전체 허용 공급자",
                "كل المزوّدين المصرّح بهم",
              )}
            </option>
            {Array.from(
              new Set([
                query.get("provider") ?? "",
                ...(current?.data?.options.providers ?? []),
              ]),
            )
              .filter(Boolean)
              .map((p) => (
                <option key={p}>{p}</option>
              ))}
          </select>
        </label>
        <label htmlFor="pricing-period">
          {t(l, "Charge-start month", "과금 시작 월", "شهر بدء الرسوم")}
          <select
            id="pricing-period"
            value={query.get("period") ?? "all"}
            onChange={(e) => change("period", e.target.value)}
          >
            <option value="all">
              {t(l, "All months", "전체 월", "كل الأشهر")}
            </option>
            {Array.from(
              new Set([
                query.get("period") ?? "",
                ...(current?.data?.options.periods ?? []),
              ]),
            )
              .filter((p) => p && p !== "all")
              .map((p) => (
                <option key={p}>{p}</option>
              ))}
          </select>
        </label>
        <label htmlFor="pricing-basis">
          {t(l, "Evidence cost basis", "근거 비용 기준", "أساس تكلفة الأدلة")}
          <select
            id="pricing-basis"
            value={query.get("basis") ?? "billed_cost"}
            onChange={(e) => change("basis", e.target.value)}
          >
            <option value="billed_cost">
              {t(l, "Billed cost", "청구 비용", "التكلفة المفوترة")}
            </option>
            <option value="effective_cost">
              {t(l, "Effective cost", "실효 비용", "التكلفة الفعلية")}
            </option>
          </select>
        </label>
        <label htmlFor="pricing-search">
          {t(l, "Source search", "원천 검색", "بحث المصدر")}
          <input
            id="pricing-search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </label>
        <button type="submit">{t(l, "Search", "검색", "بحث")}</button>
        <button type="button" onClick={refresh}>
          {t(l, "New query", "새 조회", "استعلام جديد")}
        </button>
      </form>
      {current?.error ? (
        <section role="alert">
          <h2>
            {t(l, "Source unavailable", "원천 조회 불가", "المصدر غير متاح")}
          </h2>
          <p>{issueLabel(l, current.error)}</p>
          <p className="pricing-meta">
            <bdi dir="ltr">
              {current.error} {current.requestId}
            </bdi>
          </p>
          <p>
            {t(
              l,
              "Previous source, draft, baseline and results were cleared.",
              "이전 원천·초안·비교 기준·결과를 제거했습니다.",
              "أُزيل المصدر والمسودة والأساس والنتائج السابقة.",
            )}
          </p>
          <button onClick={refresh}>
            {t(
              l,
              "Reset row and retry",
              "행 초기화 후 다시 조회",
              "إعادة تعيين الصف وإعادة الاستعلام",
            )}
          </button>
        </section>
      ) : !current?.data ? (
        <p role="status">
          {t(
            l,
            "Loading authorized source rows…",
            "권한이 허용된 원천 행을 조회하고 있습니다…",
            "جارٍ تحميل صفوف المصدر المصرّح بها…",
          )}
        </p>
      ) : (
        <>
          <div className="pricing-identity">
            <dl>
              <dt>
                {t(
                  l,
                  "Selected source row",
                  "선택 원천 행",
                  "صف المصدر المحدد",
                )}
              </dt>
              <dd>
                {source?.row.service ??
                  t(l, "No row selected", "선택한 행 없음", "لا يوجد صف محدد")}
              </dd>
            </dl>
            <dl>
              <dt>
                {t(
                  l,
                  "Pricing quantity / unit",
                  "요금 산정 수량 / 단위",
                  "كمية التسعير / الوحدة",
                )}
              </dt>
              <dd>
                <bdi dir="ltr">
                  {source
                    ? `${source.row.pricing_quantity === null ? missing(l) : number(source.row.pricing_quantity, l)} ${source.row.pricing_unit ?? missing(l)}`
                    : "—"}
                </bdi>
              </dd>
            </dl>
            <dl>
              <dt>{t(l, "Source currency", "원천 통화", "عملة المصدر")}</dt>
              <dd>
                <bdi dir="ltr">{source?.row.currency ?? "—"}</bdi>
              </dd>
            </dl>
            <dl>
              <dt>
                {t(
                  l,
                  "Source charge interval",
                  "원천 과금 구간",
                  "فترة رسوم المصدر",
                )}
              </dt>
              <dd>
                <bdi dir="ltr">
                  {source
                    ? `${source.row.charge_start} — ${source.row.charge_end}`
                    : "—"}
                </bdi>
              </dd>
            </dl>
          </div>
          <button
            className="pricing-source-jump"
            onClick={() =>
              document.getElementById("pricing-source-list")?.focus()
            }
          >
            {t(l, "Choose a source row", "원천 행 선택", "اختر صف مصدر")}
          </button>
          <div className="pricing-source-note">
            <p className="pricing-meta">
              {t(
                l,
                "Source rows are evidence, not customer contracts, SKUs or selling prices. The charge interval is not a policy effective period.",
                "수집 원천 행은 근거이며 고객 계약·SKU·판매 단가가 아닙니다. 과금 구간은 정책 적용 기간이 아닙니다.",
                "صفوف المصدر أدلة وليست عقود عملاء أو وحدات مخزون أو أسعار بيع. فترة الرسوم ليست فترة سريان سياسة.",
              )}
            </p>
            {source && <Evidence row={source.row} locale={l} />}
          </div>
          {validation.length > 0 && (
            <div
              ref={summary}
              tabIndex={-1}
              role="alert"
              className="pricing-validation"
            >
              <h2>
                {t(
                  l,
                  "Check review inputs",
                  "검토 입력 확인",
                  "تحقّق من مدخلات المراجعة",
                )}
              </h2>
              <ul>
                {Array.from(new Set(validation)).map((c) => (
                  <li key={c}>{issueLabel(l, c)}</li>
                ))}
              </ul>
            </div>
          )}
          {view === "rates" && (
            <div className="pricing-workspace pricing-rates">
              <RuleEditor
                errors={validation}
                locale={l}
                draft={draft ?? emptyDraft()}
                source={source}
                update={update}
              />
              <aside className="pricing-rail">
                <Summary locale={l} result={current.result} />
                <button
                  id="pricing-preview"
                  className="pricing-primary"
                  onClick={run}
                >
                  {t(
                    l,
                    "Preview candidate",
                    "후보 미리보기 실행",
                    "معاينة المرشح",
                  )}
                </button>
                {!source && (
                  <p className="pricing-meta">
                    {t(
                      l,
                      "Select a source row below.",
                      "아래에서 원천 행을 선택하세요.",
                      "اختر صف مصدر أدناه.",
                    )}
                  </p>
                )}
                <section className="pricing-draft">
                  <h3>
                    {t(
                      l,
                      "Browser review draft",
                      "브라우저 검토 초안",
                      "مسودة مراجعة المتصفح",
                    )}
                  </h3>
                  <p className="pricing-meta">
                    {dirty
                      ? t(
                          l,
                          "Unsaved changes",
                          "저장되지 않은 변경 있음",
                          "تغييرات غير محفوظة",
                        )
                      : current.saved
                        ? t(l, "Saved", "저장됨", "محفوظ")
                        : t(l, "Not saved", "저장 전", "غير محفوظ")}
                  </p>
                  <button id="pricing-save" onClick={persist}>
                    {t(
                      l,
                      "Save browser review draft",
                      "브라우저 검토 초안 저장",
                      "حفظ مسودة المراجعة في المتصفح",
                    )}
                  </button>
                  <p className="pricing-meta">
                    {t(
                      l,
                      "Entered prices, fees, targets and review text are saved in this browser. Storage is not private or per-user. Public/synthetic data only; do not enter real customer information.",
                      "입력한 단가·추가 요금·목표 금액·검토 문구를 이 브라우저에 저장합니다. 비공개·사용자별 저장소가 아닙니다. 공개·합성 데이터만 사용하고 실제 고객 정보를 입력하지 마세요.",
                      "تُحفظ الأسعار والرسوم والأهداف ونص المراجعة في هذا المتصفح. التخزين ليس خاصاً أو منفصلاً لكل مستخدم. استخدم بيانات عامة أو تجريبية فقط ولا تدخل معلومات عملاء حقيقية.",
                    )}
                  </p>
                </section>
                {noticeText && (
                  <p
                    className="pricing-meta"
                    role={
                      notice === "save_failed" || notice === "restore_failed"
                        ? "alert"
                        : "status"
                    }
                  >
                    {noticeText}
                  </p>
                )}
              </aside>
              <SourcePicker
                locale={l}
                data={current.data}
                selected={row}
                select={select}
                page={page}
              />
            </div>
          )}
          {view !== "rates" &&
            source &&
            draft &&
            (view === "adjustments" ? (
              <Adjustments
                locale={l}
                draft={draft}
                source={source}
                result={current.result}
                adjustment={current.adjustment}
                update={update}
                run={runAdjustment}
                back={() => switchView("rates")}
              />
            ) : (
              <Results
                name={draft.name}
                locale={l}
                source={source}
                result={current.result}
                baseline={current.baseline}
                view={view}
                keep={() => {
                  if (current.result)
                    setState((old) => ({
                      ...old,
                      baseline: structuredClone(current.result),
                    }));
                }}
                clear={() =>
                  setState((old) => ({ ...old, baseline: undefined }))
                }
                back={() => switchView("rates")}
              />
            ))}
          {view !== "rates" && !source && (
            <>
              <p>
                {t(
                  l,
                  "Select a source row to start this review.",
                  "검토를 시작할 원천 행을 선택하세요.",
                  "اختر صف مصدر لبدء المراجعة.",
                )}
              </p>
              <SourcePicker
                locale={l}
                data={current.data}
                select={select}
                page={page}
              />
            </>
          )}
        </>
      )}
    </div>
  );
}
