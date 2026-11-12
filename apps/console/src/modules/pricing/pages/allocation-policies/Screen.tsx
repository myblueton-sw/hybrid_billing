import { useEffect, useRef, useState } from "react";
import SlowReadHint from "../../../../shared/SlowReadHint";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { setNavigationGuard } from "../../../../shared/navigation";
import {
  loadOverview,
  ServiceError,
} from "../../../sources/collected-focus/api";
import { reviewMoney as money } from "./values";
import {
  prepare,
  emptyDraft,
  restricted,
  issues,
  simulate,
  type Prepared,
  type Draft,
  type Result,
} from "./model";
import { save, restore } from "./store";
import { basisLabel, issueLabel, totalLabels } from "./copy";
import PolicyEditor from "./PolicyEditor";
import SourceBasis, { SourceEvidence } from "./SourceBasis";
import Results from "./Results";
import "./screen.css";
type State = {
  key: string;
  source?: Prepared;
  draft?: Draft;
  result?: Result;
  baseline?: Result;
  error?: string;
  requestId?: string;
  notice?: string;
  saved?: boolean;
};
const itemIds = [
  "B03:scope:whole-source",
  "B03:form:fixed-percentages",
  "B03:calculation:source-precision-review",
  "B03:table:recipient-manifest",
  "B03:comparison:session-baseline",
  "B03:action:browser-draft",
  "B03:state:source-unavailable",
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
  onState: (s: string, items: string[]) => void;
  onDirty: (dirty: boolean) => void;
  canDiscard: () => boolean;
}) {
  const query = new URLSearchParams();
  for (const field of ["provider", "period", "basis", "snapshot"])
    if (params.has("focus-" + field))
      query.set(field, params.get("focus-" + field)!);
  const queryKey = query.toString(),
    currency = params.get("allocation-currency") ?? "",
    blocked = restricted(params),
    key = JSON.stringify([
      queryKey,
      currency,
      blocked,
      params.get("scope"),
      params.get("company"),
    ]);
  const [state, setState] = useState<State>({ key }),
    [dirty, setDirty] = useState(false),
    [attempted, setAttempted] = useState(false),
    [retry, setRetry] = useState(0),
    [focusRequest, setFocusRequest] = useState<{
      id: string;
      nonce: number;
    } | null>(null);
  const generation = useRef(0),
    errorSummary = useRef<HTMLDivElement>(null);
  const current = state.key === key && !blocked ? state : undefined,
    source = current?.source,
    draft = current?.draft;
  const errors =
    source && draft && attempted ? issues(draft, source, false) : [];
  const [saveErrors, setSaveErrors] = useState<string[]>([]);
  useEffect(() => {
    const controller = new AbortController(),
      g = ++generation.current;
    setState({ key });
    setDirty(false);
    setAttempted(false);
    setSaveErrors([]);
    setFocusRequest(null);
    if (blocked) return () => controller.abort();
    loadOverview(new URLSearchParams(queryKey), controller.signal)
      .then(async (data) => {
        const prepared = await prepare(data, currency);
        if (controller.signal.aborted || generation.current !== g) return;
        let recovered: Draft | null = null,
          notice: string | undefined;
        try {
          recovered = restore(window.localStorage, prepared);
          if (recovered) notice = "restored";
        } catch {
          notice = "restore_failed";
        }
        if (!controller.signal.aborted && generation.current === g)
          setState({
            key,
            source: prepared,
            draft: recovered ?? emptyDraft(prepared),
            saved: !!recovered,
            notice,
          });
      })
      .catch((error) => {
        if (!controller.signal.aborted && generation.current === g) {
          setState({
            key,
            error:
              error instanceof ServiceError
                ? error.code
                : error instanceof Error &&
                    [
                      "target_limit",
                      "target_duplicate",
                      "restricted_source",
                    ].includes(error.message)
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
  const reviewState = JSON.stringify({
    phase: "allocation",
    status: blocked
      ? "restricted"
      : (current?.error ?? (source ? "loaded" : "loading")),
    context: source?.context ?? null,
    view,
    draft: draft ?? null,
    candidate: !!current?.result,
    baseline:
      current?.baseline?.rows.map((r) => ({
        target: r.target,
        percentage: r.percentage,
      })) ?? null,
    saved: !!current?.saved,
    attempted,
    validation: [...new Set([...errors, ...saveErrors])],
    notice: current?.notice ?? null,
  });
  useEffect(() => onState(reviewState, itemIds), [onState, reviewState]);
  function update(next: Draft) {
    setState((old) =>
      old.key === key && old.source
        ? {
            ...old,
            draft: next,
            result: undefined,
            saved: false,
            notice: undefined,
          }
        : old,
    );
    setDirty(true);
    setSaveErrors([]);
  }
  function go(next: URLSearchParams) {
    setDirty(false);
    onDirty(false);
    setState({ key: "pending" });
    location.hash = `#/allocations?${next}`;
  }
  function change(field: string, value: string) {
    if (!canDiscard()) return;
    const next = new URLSearchParams(params);
    if (field === "currency") next.set("allocation-currency", value);
    else {
      next.set("focus-" + field, value);
      next.delete("focus-snapshot");
    }
    go(next);
  }
  function fullSource() {
    if (!canDiscard()) return;
    const next = new URLSearchParams(params);
    for (const field of ["account", "q", "after", "account_after", "snapshot"])
      next.delete("focus-" + field);
    if (next.toString() === params.toString()) {
      setState({ key });
      setRetry((n) => n + 1);
    } else go(next);
  }
  function refresh() {
    if (!canDiscard()) return;
    const next = new URLSearchParams(params);
    next.delete("focus-snapshot");
    setDirty(false);
    onDirty(false);
    setState({ key });
    if (next.toString() !== params.toString()) go(next);
    else setRetry((n) => n + 1);
  }
  function switchView(nextView: string) {
    if (nextView === view) return;
    const next = new URLSearchParams(params);
    next.set("view", nextView);
    setNavigationGuard(null);
    location.hash = `#/allocations?${next}`;
  }
  function focusErrors() {
    requestAnimationFrame(() => errorSummary.current?.focus());
  }
  function run() {
    if (!source || !draft) return;
    setAttempted(true);
    setSaveErrors([]);
    if (issues(draft, source).length) {
      focusErrors();
      return;
    }
    const result = simulate(draft, source);
    setState((old) => ({ ...old, result, notice: "preview_ready" }));
  }
  function persist() {
    if (!source || !draft) return;
    const invalid = issues(draft, source, true);
    setSaveErrors(invalid);
    setAttempted(true);
    if (invalid.length) {
      focusErrors();
      return;
    }
    try {
      save(window.localStorage, source, draft);
      setState((old) => ({ ...old, saved: true, notice: "saved" }));
      setDirty(false);
    } catch {
      setState((old) => ({ ...old, saved: false, notice: "save_failed" }));
    }
  }
  function keepBaseline() {
    if (!current?.result) return;
    const baseline = structuredClone(current.result);
    setState((old) => ({ ...old, baseline, notice: "baseline_kept" }));
  }
  function errorTarget(code: string) {
    if (code === "name_required" || code === "name_bound") {
      document.getElementById("allocation-name")?.focus();
      return;
    }
    if (code === "month_required") {
      document.getElementById("allocation-month")?.focus();
      return;
    }
    if (code.startsWith("amount_")) {
      document.getElementById("allocation-currency")?.focus();
      return;
    }
    const invalid = draft?.weights.find(
      (w) =>
        w.percentage === "" ||
        issues({ ...draft, weights: [w] }, source!).includes(
          "percentage_invalid",
        ),
    );
    const id = invalid?.target ?? draft?.weights[0]?.target;
    if (id) {
      if (view !== "policies") switchView("policies");
      setFocusRequest({ id, nonce: Date.now() });
    }
  }
  const notice = current?.notice;
  const noticeText =
    notice === "restored"
      ? t(
          l,
          "Loaded this browser's draft after a fresh authorized query.",
          "원천 조회 후 이 브라우저의 초안을 불러왔습니다.",
          "حُمّلت مسودة هذا المتصفح بعد استعلام حديث ومصرّح به.",
        )
      : notice === "restore_failed"
        ? t(
            l,
            "The browser draft could not be restored against the current source. A new draft is shown.",
            "브라우저 초안을 현재 원천에 맞춰 불러오지 못했습니다. 새 초안을 표시합니다.",
            "تعذّرت استعادة مسودة المتصفح مع المصدر الحالي. تُعرض مسودة جديدة.",
          )
        : notice === "saved"
          ? t(
              l,
              "Review draft saved in this browser only.",
              "검토 초안을 이 브라우저에만 저장했습니다.",
              "حُفظت مسودة المراجعة في هذا المتصفح فقط.",
            )
          : notice === "save_failed"
            ? t(
                l,
                "Browser save failed. The current draft remains unsaved; check storage and retry.",
                "브라우저 저장에 실패했습니다. 현재 초안은 저장되지 않았습니다. 저장 공간을 확인하고 다시 시도하세요.",
                "فشل الحفظ في المتصفح. المسودة الحالية غير محفوظة؛ تحقّق من التخزين وأعد المحاولة.",
              )
            : notice === "baseline_kept"
              ? t(
                  l,
                  "Current result kept as a baseline for this browser session.",
                  "현재 결과를 이 브라우저 세션의 비교 기준으로 유지합니다.",
                  "احتُفظ بالنتيجة الحالية أساساً للمقارنة في جلسة هذا المتصفح.",
                )
              : notice === "preview_ready"
                ? t(
                    l,
                    "Allocation review preview is ready. No financial policy was approved.",
                    "배분 검토 미리보기를 계산했습니다. 재무 정책을 승인한 결과가 아닙니다.",
                    "أصبحت معاينة مراجعة التوزيع جاهزة. لم تُعتمد أي سياسة مالية.",
                  )
                : undefined;
  const shownErrors = [...new Set([...errors, ...saveErrors])],
    measure = source?.data.amounts.find(
      (a) => a.currency === source.currency,
    )?.[source.data.selection.basis];
  return (
    <div
      className="allocation-screen"
      onKeyDown={(event) => {
        if (event.key !== "Escape" || event.defaultPrevented) return;
        const target = event.target;
        if (!(target instanceof HTMLElement)) return;
        const detail = target.closest<HTMLDetailsElement>("details[open]");
        if (detail) {
          detail.open = false;
          detail.querySelector("summary")?.focus();
          event.preventDefault();
        }
      }}
      data-allocation-state={
        blocked
          ? "restricted"
          : source
            ? "loaded"
            : current?.error
              ? "error"
              : "loading"
      }
      data-allocation-query={source?.context ?? ""}
      data-allocation-view={view}
    >
      <div className="allocation-toolbar">
        <label htmlFor="allocation-provider">
          {t(l, "Collected source", "수집 원천", "المصدر المجمّع")}
          <select
            id="allocation-provider"
            disabled={!source}
            value={query.get("provider") ?? ""}
            onChange={(e) => change("provider", e.target.value)}
          >
            <option value="">
              {t(
                l,
                "All authorized providers",
                "전체 허용 공급사",
                "كل المزوّدين المصرّح بهم",
              )}
            </option>
            {(source?.data.options.providers ?? [query.get("provider") ?? ""])
              .filter(Boolean)
              .map((v) => (
                <option key={v} value={v}>
                  {v}
                </option>
              ))}
          </select>
        </label>
        <label htmlFor="allocation-month">
          {t(l, "Charge-start month", "과금 시작 월", "شهر بدء الرسوم")}
          <select
            id="allocation-month"
            disabled={!source}
            value={query.get("period") ?? "all"}
            onChange={(e) => change("period", e.target.value)}
          >
            <option value="all">
              {t(
                l,
                "All charge-start months",
                "전체 과금 시작 월",
                "كل أشهر بدء الرسوم",
              )}
            </option>
            {source?.data.options.periods.map((v) => (
              <option key={v} value={v}>
                {v}
              </option>
            ))}
          </select>
        </label>
        <label htmlFor="allocation-basis">
          {t(l, "Cost basis", "비용 기준", "أساس التكلفة")}
          <select
            id="allocation-basis"
            disabled={!source}
            value={query.get("basis") ?? "billed_cost"}
            onChange={(e) => change("basis", e.target.value)}
          >
            <option value="billed_cost">{basisLabel(l, "billed_cost")}</option>
            <option value="effective_cost">
              {basisLabel(l, "effective_cost")}
            </option>
          </select>
        </label>
        <label htmlFor="allocation-currency">
          {t(l, "Source currency", "원천 통화", "عملة المصدر")}
          <select
            id="allocation-currency"
            disabled={!source}
            value={currency}
            onChange={(e) => change("currency", e.target.value)}
          >
            <option value="">
              {t(l, "Choose a currency", "통화 선택", "اختر عملة")}
            </option>
            {source?.data.amounts.map((a) => (
              <option key={a.currency} value={a.currency}>
                {a.currency}
              </option>
            ))}
          </select>
        </label>
        <button onClick={refresh}>
          {t(l, "New query", "새 조회", "استعلام جديد")}
        </button>
      </div>
      {blocked ? (
        <section className="allocation-blocked" role="alert">
          <h2>
            {t(
              l,
              "Whole-source scope required",
              "전체 원천 범위 필요",
              "يلزم نطاق المصدر الكامل",
            )}
          </h2>
          <p>{issueLabel(l, "restricted_source")}</p>
          <button onClick={fullSource}>
            {t(
              l,
              "Open whole source scope",
              "전체 원천 범위 열기",
              "فتح نطاق المصدر الكامل",
            )}
          </button>
        </section>
      ) : current?.error ? (
        <section className="allocation-blocked" role="alert">
          <h2>
            {t(l, "Source unavailable", "원천 조회 불가", "المصدر غير متاح")}
          </h2>
          <p>{issueLabel(l, current.error)}</p>
          <p className="allocation-meta">
            {t(
              l,
              "Previous source, draft and results were cleared. No browser draft is used as a fallback.",
              "이전 원천·초안·결과는 제거되었습니다. 브라우저 초안으로 대체하지 않습니다.",
              "أُزيل المصدر والمسودة والنتائج السابقة. لا تُستخدم مسودة المتصفح بديلاً.",
            )}
          </p>
          <p className="allocation-meta">
            <bdi dir="ltr">{current.error}</bdi>
            {current.requestId && (
              <>
                {" "}
                / <bdi dir="ltr">{current.requestId}</bdi>
              </>
            )}
          </p>
        </section>
      ) : !source || !draft ? (
        <p role="status">
          {t(
            l,
            "Loading the authorized full-source review context…",
            "권한이 허용된 전체 원천 범위를 조회하고 있습니다…",
            "جارٍ تحميل نطاق مراجعة المصدر الكامل المصرّح به…",
          )}
          <SlowReadHint locale={l} />
        </p>
      ) : (
        <>
          <div className="allocation-pool-strip">
            <dl>
              <dt>{totalLabels(l).source}</dt>
              <dd>
                <bdi dir="ltr" data-allocation-total="source">
                  {!source.currency
                    ? t(l, "Choose a currency", "통화 선택", "اختر عملة")
                    : measure?.coefficient == null
                      ? t(l, "Unknown", "알 수 없음", "غير معروف")
                      : money(measure.coefficient, source.currency, l)}
                </bdi>
                {!!measure?.missing && (
                  <small>
                    {t(
                      l,
                      "Known subtotal only",
                      "확인된 금액의 부분 합계",
                      "مجموع فرعي للمبالغ المعلومة فقط",
                    )}
                  </small>
                )}
              </dd>
            </dl>
            <dl>
              <dt>
                {t(
                  l,
                  "Complete recipient set",
                  "전체 검토 대상 집합",
                  "مجموعة مستلمي المراجعة الكاملة",
                )}
              </dt>
              <dd>
                {source.targets.length}{" "}
                <small>
                  {t(
                    l,
                    "Tenant access scopes",
                    "테넌트 접근 범위",
                    "نطاقات وصول المستأجرين",
                  )}
                </small>
              </dd>
            </dl>
            <dl>
              <dt>
                {t(
                  l,
                  "Source amount coverage",
                  "원천 금액 확인 범위",
                  "تغطية مبالغ المصدر",
                )}
              </dt>
              <dd>
                {measure ? (
                  <bdi dir="ltr">
                    {measure.present} / {measure.present + measure.missing}
                  </bdi>
                ) : (
                  t(l, "Choose a currency", "통화 선택", "اختر عملة")
                )}
              </dd>
            </dl>
          </div>
          <div className="allocation-source-note">
            <p className="allocation-meta">
              {t(
                l,
                "Collected public/synthetic source totals, not approved infrastructure cost or sales. Legal-organization and contract ownership remain unconnected.",
                "수집한 공개·합성 원천의 합계이며 승인된 인프라 원가나 판매액이 아닙니다. 법인·계약 소유 관계는 미연결입니다.",
                "إجماليات مصادر عامة وتجريبية مجمّعة، وليست تكلفة بنية تحتية أو مبيعات معتمدة. ملكية الجهات القانونية والعقود غير متصلة.",
              )}
            </p>
            <SourceEvidence locale={l} source={source} />
          </div>
          {shownErrors.length > 0 && (
            <div
              className="allocation-error-summary"
              tabIndex={-1}
              ref={errorSummary}
              role="alert"
            >
              <h2>
                {t(
                  l,
                  "Check the policy inputs",
                  "배분 정책 입력 확인",
                  "تحقّق من مدخلات السياسة",
                )}
              </h2>
              <ul>
                {shownErrors.map((code) => (
                  <li key={code}>
                    <button
                      className="allocation-text-button"
                      onClick={() => errorTarget(code)}
                    >
                      {issueLabel(l, code)}
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          )}
          {noticeText && view !== "policies" && (
            <p
              role={
                notice === "save_failed" || notice === "restore_failed"
                  ? "alert"
                  : "status"
              }
              className="allocation-meta"
            >
              {noticeText}
            </p>
          )}
          {view === "policies" && (
            <PolicyEditor
              locale={l}
              source={source}
              draft={draft}
              result={current.result}
              attempted={attempted}
              update={update}
              run={run}
              persist={persist}
              saved={!!current.saved}
              notice={noticeText}
              noticeError={
                notice === "save_failed" || notice === "restore_failed"
              }
              switchView={switchView}
              focusRequest={focusRequest}
            />
          )}
          {view === "pools" && <SourceBasis locale={l} source={source} />}
          {(view === "simulation" || view === "comparison") && (
            <Results
              locale={l}
              source={source}
              result={current.result}
              baseline={current.baseline}
              comparison={view === "comparison"}
              keepBaseline={keepBaseline}
              switchView={switchView}
            />
          )}
        </>
      )}
    </div>
  );
}
