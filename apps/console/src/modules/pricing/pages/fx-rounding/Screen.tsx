import { useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { setNavigationGuard } from "../../../../shared/navigation";
import { digest } from "../../../../shared/review-store";
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
  sourceAmount,
  type Prepared,
  type Draft,
  type Result,
} from "./model";
import { save, restore } from "./store";
import { t, issueLabel, errorFields } from "./copy";
import { money } from "./values";
import RuleEditor from "./RuleEditor";
import AccountScope from "./AccountScope";
import SourcePicker from "./SourcePicker";
import Results, { Summary, ManualEvidence } from "./Results";
import "./screen.css";
type State = {
  key: string;
  query?: string;
  data?: OverviewData;
  source?: Prepared;
  draft?: Draft;
  result?: Result;
  baseline?: Result;
  error?: string;
  requestId?: string;
  notice?: string;
  saved?: boolean;
  invalidated?: boolean;
};
const items = [
  "B05:scope:source-cost",
  "B05:form:manual-quote",
  "B05:calculation:quantum",
  "B05:comparison:session-baseline",
  "B05:action:browser-draft",
  "B05:state:authority-unconnected",
  "B05:state:source-unavailable",
];
function selection(params: URLSearchParams) {
  const q = focusQuery(params);
  if (!params.has("focus-provider")) q.set("provider", "VMware");
  if (!q.has("period")) q.set("period", "all");
  if (!q.has("basis")) q.set("basis", "billed_cost");
  return q;
}
function requestKey(params: URLSearchParams) {
  return JSON.stringify([
    selection(params).toString(),
    params.get("fx-row") ?? "",
    [...params]
      .filter(
        ([k]) =>
          !k.startsWith("focus-") && !["view", "lang", "fx-row"].includes(k),
      )
      .sort(),
  ]);
}
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
  const query = selection(params),
    row = params.get("fx-row") ?? "",
    key = requestKey(params);
  const [state, setState] = useState<State>({ key: "" }),
    [retry, setRetry] = useState(0),
    [dirty, setDirty] = useState(false),
    [errors, setErrors] = useState<string[]>([]),
    [attempts, setAttempts] = useState({ preview: 0, save: 0 }),
    [search, setSearch] = useState(query.get("q") ?? "");
  const generation = useRef(0),
    summary = useRef<HTMLDivElement>(null),
    current = state.key === key ? state : undefined,
    source = current?.source,
    draft = current?.draft ?? emptyDraft();
  useEffect(() => {
    const abort = new AbortController(),
      g = ++generation.current;
    setState({ key });
    setDirty(false);
    setErrors([]);
    setAttempts({ preview: 0, save: 0 });
    setSearch(query.get("q") ?? "");
    Promise.all([loadOverview(query, abort.signal), digest(key)])
      .then(async ([data, fingerprint]) => {
        if (abort.signal.aborted || generation.current !== g) return;
        if (new Set(data.rows.map((r) => r.id)).size !== data.rows.length)
          throw Error("source_duplicate");
        if (!row) {
          setState({ key, query: fingerprint, data, draft: emptyDraft() });
          return;
        }
        const source = await prepare(data, row);
        if (abort.signal.aborted || generation.current !== g) return;
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
        if (!abort.signal.aborted && generation.current === g)
          setState({ key, query: fingerprint, data, source, draft, notice });
      })
      .catch((error) => {
        if (!abort.signal.aborted && generation.current === g) {
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
    return () => abort.abort();
  }, [key, retry]);
  useEffect(() => onDirty(dirty), [dirty, onDirty]);
  const review = JSON.stringify({
    phase: "fx-rounding",
    query: current?.query ?? null,
    sourceSnapshot: current?.data?.source_snapshot ?? null,
    selectionSnapshot: current?.data?.snapshot ?? null,
    context: source?.context ?? null,
    row: source?.row.id ?? null,
    view,
    status: current?.error ?? (current?.data ? "loaded" : "loading"),
    draft: current?.draft ?? null,
    result: current?.result ?? null,
    baseline: current?.baseline ?? null,
    errors,
    attempts,
    notice: current?.notice ?? null,
    invalidated: !!current?.invalidated,
    saved: !!current?.saved,
  });
  useEffect(() => onState(review, items), [review, onState]);
  function update(draft: Draft) {
    setState((old) =>
      old.key === key
        ? {
            ...old,
            draft,
            result: undefined,
            saved: false,
            notice: undefined,
            invalidated: true,
          }
        : old,
    );
    setDirty(true);
    setErrors([]);
  }
  function go(next: URLSearchParams) {
    setDirty(false);
    onDirty(false);
    setErrors([]);
    setState({ key: "pending" });
    if (requestKey(next) === key) {
      setRetry((n) => n + 1);
      setState({ key });
    }
    if (next.toString() !== params.toString())
      location.hash = `#/exchange-rates?${next}`;
  }
  function change(field: string, value: string) {
    if (!canDiscard()) return;
    const next = new URLSearchParams(params);
    next.set("focus-" + field, value);
    for (const k of [
      "fx-row",
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
      "fx-row",
      "focus-after",
      "focus-account_after",
      "focus-snapshot",
    ])
      next.delete(k);
    go(next);
  }
  function retrySame() {
    if (!canDiscard()) return;
    setState({ key });
    setDirty(false);
    onDirty(false);
    setErrors([]);
    setRetry((n) => n + 1);
  }
  function select(id: string) {
    if (id === row || !current?.data || !canDiscard()) return;
    const next = new URLSearchParams(params);
    next.set("fx-row", id);
    next.set("focus-snapshot", current.data.snapshot);
    go(next);
  }
  function page(after: number) {
    if (!current?.data || !canDiscard()) return;
    const next = new URLSearchParams(params);
    next.delete("fx-row");
    next.set("focus-after", String(after));
    next.set("focus-snapshot", current.data.snapshot);
    go(next);
  }
  function accountPage(after: number) {
    if (!current?.data || !canDiscard()) return;
    const next = new URLSearchParams(params);
    next.delete("fx-row");
    next.delete("focus-after");
    next.set("focus-account_after", String(after));
    next.set("focus-snapshot", current.data.snapshot);
    go(next);
  }
  function switchView(nextView: string) {
    if (nextView === view) return;
    const next = new URLSearchParams(params);
    next.set("view", nextView);
    setNavigationGuard(null);
    location.hash = `#/exchange-rates?${next}`;
  }
  function invalid(codes: string[]) {
    setErrors(codes);
    requestAnimationFrame(() => summary.current?.focus());
  }
  function run() {
    setAttempts((a) => ({ ...a, preview: a.preview + 1 }));
    if (!source) {
      invalid(["row_unavailable"]);
      return;
    }
    const e = issues(draft, source);
    if (e.length) {
      invalid(e);
      return;
    }
    try {
      const result = simulate(draft, source);
      setErrors([]);
      setState((old) => ({
        ...old,
        result,
        notice: "preview_ready",
        invalidated: false,
      }));
    } catch (e) {
      invalid([e instanceof Error ? e.message : "amount_overflow"]);
    }
  }
  function persist() {
    setAttempts((a) => ({ ...a, save: a.save + 1 }));
    if (!source) {
      invalid(["row_unavailable"]);
      return;
    }
    const e = issues(draft, source, true);
    if (e.length) {
      invalid(e);
      return;
    }
    try {
      simulate(draft, source);
    } catch (e) {
      invalid([e instanceof Error ? e.message : "amount_overflow"]);
      return;
    }
    try {
      save(window.localStorage, source, draft);
      setState((old) => ({ ...old, saved: true, notice: "saved" }));
      setDirty(false);
      setErrors([]);
    } catch {
      setState((old) => ({ ...old, saved: false, notice: "save_failed" }));
      setDirty(true);
    }
  }
  const notices: Record<string, [string, string, string]> = {
    restored: [
      "Browser draft restored after a fresh authorized source query.",
      "새 권한 확인 조회 후 브라우저 초안을 불러왔습니다.",
      "استُعيدت مسودة المتصفح بعد استعلام مصدر حديث ومصرّح به.",
    ],
    restore_failed: [
      "The browser draft could not be restored for this exact source context.",
      "현재 원천·범위에 맞춰 브라우저 초안을 불러오지 못했습니다.",
      "تعذّرت استعادة المسودة لسياق المصدر المطابق.",
    ],
    saved: [
      "Review inputs saved in this browser.",
      "검토 입력값을 이 브라우저에 저장했습니다.",
      "حُفظت مدخلات المراجعة في هذا المتصفح.",
    ],
    save_failed: [
      "Browser save failed. Changes remain unsaved.",
      "브라우저 저장 실패. 변경 내용은 저장되지 않았습니다.",
      "فشل حفظ المتصفح. تبقى التغييرات غير محفوظة.",
    ],
    preview_ready: [
      "Review candidate calculated. No financial policy was approved.",
      "검토 후보를 계산했습니다. 재무 정책 승인 결과가 아닙니다.",
      "حُسب المرشح للمراجعة. لم تُعتمد سياسة مالية.",
    ],
  };
  function focusError(code: string) {
    const id = "fx-" + (errorFields[code] ?? "source-list"),
      target = document.getElementById(id);
    target?.focus();
    if (id === "fx-mode") target?.querySelector("input")?.focus();
  }
  const toolbar = (
    <form
      className="b05-toolbar"
      onSubmit={(e) => {
        e.preventDefault();
        change("q", search);
      }}
    >
      <label className="b05-field" htmlFor="fx-source-provider">
        {t(l, "Collected source", "수집 원천", "المصدر المجمّع")}
        <select
          id="fx-source-provider"
          data-fx-provider="true"
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
      <label className="b05-field" htmlFor="fx-period">
        {t(l, "Charge-start month", "과금 시작 월", "شهر بدء الرسوم")}
        <select
          id="fx-period"
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
      <label className="b05-field" htmlFor="fx-basis">
        {t(l, "Source cost basis", "원천 비용 기준", "أساس تكلفة المصدر")}
        <select
          id="fx-basis"
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
      <label className="b05-field" htmlFor="fx-search">
        {t(l, "Account search", "계정 검색", "بحث الحساب")}
        <input
          id="fx-search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <small>
          {t(
            l,
            "Find accounts, then select one to narrow source rows.",
            "계정을 검색한 뒤 선택하면 원천 행의 범위가 좁혀집니다.",
            "ابحث عن الحسابات ثم اختر حساباً لتضييق صفوف المصدر.",
          )}
        </small>
      </label>
      <button type="submit">{t(l, "Search", "검색", "بحث")}</button>
      <button type="button" onClick={refresh}>
        {t(l, "New query", "새 조회", "استعلام جديد")}
      </button>
    </form>
  );
  return (
    <div
      className="b05-screen"
      data-fx-state={
        current?.error ? "error" : current?.data ? "loaded" : "loading"
      }
      data-fx-view={view}
      data-fx-selected-row={source?.row.id ?? ""}
      data-fx-query={
        source?.context ?? current?.query ?? current?.data?.snapshot ?? ""
      }
      onKeyDown={(e) => {
        if (
          e.key !== "Escape" ||
          e.defaultPrevented ||
          !(e.target instanceof HTMLElement)
        )
          return;
        const detail = e.target.closest<HTMLDetailsElement>("details[open]");
        if (detail) {
          detail.open = false;
          detail.querySelector("summary")?.focus();
          e.preventDefault();
        }
      }}
    >
      {current?.error ? (
        <>
          <div className="b05-source-context">{toolbar}</div>
          <section role="alert">
            <h2>
              {t(l, "Source unavailable", "원천 조회 불가", "المصدر غير متاح")}
            </h2>
            <p>{issueLabel(l, current.error)}</p>
            <p className="b05-meta">
              <bdi dir="ltr">
                {current.error} {current.requestId}
              </bdi>
            </p>
            <p>
              {t(
                l,
                "Previous source values, draft, candidate and baseline were cleared. No offline draft replaces authorization.",
                "이전 원천·초안·후보·비교 기준을 제거했습니다. 오프라인 초안으로 권한 확인을 대체하지 않습니다.",
                "أُزيل المصدر والمسودة والمرشح والأساس. لا تحل مسودة دون اتصال محل التفويض.",
              )}
            </p>
            <div className="b05-actions">
              <button onClick={retrySame}>
                {t(
                  l,
                  "Retry same query",
                  "같은 조회 다시 시도",
                  "إعادة الاستعلام نفسه",
                )}
              </button>
              <button onClick={refresh}>
                {t(
                  l,
                  "Reset row and retry",
                  "행 초기화 후 다시 조회",
                  "إعادة تعيين الصف وإعادة المحاولة",
                )}
              </button>
            </div>
          </section>
        </>
      ) : !current?.data ? (
        <p role="status">
          {t(
            l,
            "Checking the selected scope and authorization; loading FX and rounding source evidence…",
            "선택한 범위와 권한을 확인하고 환율·반올림 원천 근거를 불러오는 중입니다…",
            "جارٍ التحقق من النطاق والتفويض وتحميل أدلة مصدر الصرف والتقريب…",
          )}
        </p>
      ) : (
        <>
          <div className="b05-source-context">
            {toolbar}
            <AccountScope
              locale={l}
              data={current.data}
              select={(id) => change("account", id)}
              page={accountPage}
            />
            <div className="b05-identity">
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
                    t(
                      l,
                      "No row selected",
                      "선택한 행 없음",
                      "لا يوجد صف محدد",
                    )}
                </dd>
              </dl>
              <dl>
                <dt>{t(l, "Source cost", "원천 비용", "تكلفة المصدر")}</dt>
                <dd>
                  <bdi dir="ltr">
                    {source && sourceAmount(source) !== null
                      ? money(sourceAmount(source)!, source.row.currency, l)
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
                  {source ? (
                    <>
                      <bdi className="b05-date" dir="ltr">
                        {source.row.charge_start}
                      </bdi>
                      {" — "}
                      <bdi className="b05-date" dir="ltr">
                        {source.row.charge_end}
                      </bdi>
                    </>
                  ) : (
                    "—"
                  )}
                </dd>
              </dl>
            </div>
            <div className="b05-source-note">
              <p className="b05-meta">
                {t(
                  l,
                  "Source cost, not a selling amount. The source charge interval is not a policy effective period.",
                  "원천 비용이며 판매 금액이 아닙니다. 과금 구간은 정책 적용 기간이 아닙니다.",
                  "تكلفة مصدر وليست مبلغ بيع. فترة رسوم المصدر ليست فترة سريان السياسة.",
                )}
              </p>
              <button
                onClick={() =>
                  document.getElementById("fx-source-list")?.focus()
                }
              >
                {t(l, "Choose a source row", "원천 행 선택", "اختر صف مصدر")}
              </button>
            </div>
          </div>
          <div className="b05-active">
            {errors.length > 0 && (
              <div
                className="b05-errors"
                ref={summary}
                tabIndex={-1}
                role="alert"
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
                  {Array.from(new Set(errors)).map((c) => (
                    <li key={c}>
                      <button
                        className="b05-text-button"
                        onClick={() => focusError(c)}
                      >
                        {issueLabel(l, c)}
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {view === "policies" && (
              <div className="b05-workspace">
                <RuleEditor
                  locale={l}
                  draft={draft}
                  source={source}
                  errors={errors}
                  update={update}
                />
                <aside className="b05-rail">
                  <Summary locale={l} result={current.result} />
                  {current.invalidated && (
                    <p role="status" className="b05-meta">
                      {t(
                        l,
                        "Inputs changed. Run a new preview.",
                        "입력이 바뀌었습니다. 다시 미리보기를 실행하세요.",
                        "تغيرت المدخلات. شغّل معاينة جديدة.",
                      )}
                    </p>
                  )}
                  <button className="b05-primary" id="fx-preview" onClick={run}>
                    {t(
                      l,
                      "Preview candidate",
                      "후보 미리보기 실행",
                      "معاينة المرشح",
                    )}
                  </button>
                  <section className="b05-draft">
                    <h3>
                      {t(
                        l,
                        "Browser review draft",
                        "브라우저 검토 초안",
                        "مسودة مراجعة المتصفح",
                      )}
                    </h3>
                    <p className="b05-meta">
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
                    <button id="fx-save" onClick={persist}>
                      {t(
                        l,
                        "Save browser review draft",
                        "브라우저 검토 초안 저장",
                        "حفظ مسودة المراجعة في المتصفح",
                      )}
                    </button>
                    <p className="b05-meta">
                      {t(
                        l,
                        "Entered financial parameters and evidence text are stored in this browser. Storage is not private or per-user. Public/synthetic exercise only; do not enter real customer information.",
                        "입력한 환율·가산율·수수료 등 검토 입력값과 증빙 문구를 이 브라우저에 저장합니다. 비공개·사용자별 저장소가 아닙니다. 공개·합성 검토만 사용하고 실제 고객 정보를 입력하지 마세요.",
                        "تُحفظ المعلمات المالية ونص الأدلة في هذا المتصفح. التخزين ليس خاصاً أو منفصلاً لكل مستخدم. للمراجعة العامة أو التجريبية فقط؛ لا تدخل معلومات عملاء حقيقية.",
                      )}
                    </p>
                  </section>
                </aside>
              </div>
            )}
            {view === "imports" && (
              <ManualEvidence
                locale={l}
                draft={draft}
                source={source}
                edit={() => switchView("policies")}
              />
            )}
            {view === "comparison" && (
              <Results
                locale={l}
                result={current.result}
                baseline={current.baseline}
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
              />
            )}
            {view === "approvals" && (
              <section>
                <h2>{t(l, "Approval history", "승인 이력", "سجل الاعتماد")}</h2>
                <p>
                  {t(
                    l,
                    "No authoritative approval history is connected. This screen does not request approval, apply policies, inherit customer → contract → MSP settings or activate financial policy. Browser draft save is not approval.",
                    "연결된 권한 있는 승인 이력이 없습니다. 이 화면은 승인 요청·적용·고객 예외 → 계약 → MSP 상속·재무 활성화를 수행하지 않습니다. 브라우저 초안 저장은 승인이 아닙니다.",
                    "لا يوجد سجل اعتماد موثوق متصل. لا تطلب هذه الشاشة اعتماداً ولا تطبق سياسات أو توريث إعدادات العميل ثم العقد ثم MSP أو تفعيل سياسة مالية. حفظ مسودة المتصفح ليس اعتماداً.",
                  )}
                </p>
                <p className="b05-meta">
                  {t(
                    l,
                    "Related service scopes: B08 conversion results · B09 FX approval. No authority or state is inferred.",
                    "관련 서비스 범위: B08 환산 결과 · B09 환율 승인. 권한·상태를 추정하지 않습니다.",
                    "نطاقات الخدمة ذات الصلة: B08 نتائج التحويل · B09 اعتماد سعر الصرف. لا تُستنتج صلاحيات أو حالات.",
                  )}
                </p>
              </section>
            )}
            {current.notice && notices[current.notice] && (
              <p
                className="b05-meta"
                role={
                  current.notice === "save_failed" ||
                  current.notice === "restore_failed"
                    ? "alert"
                    : "status"
                }
              >
                {t(l, ...notices[current.notice])}
                {current.notice === "save_failed" && (
                  <>
                    {" "}
                    <bdi dir="ltr">browser_storage_failure</bdi>
                  </>
                )}
              </p>
            )}
          </div>
          {view === "policies" && (
            <SourcePicker
              locale={l}
              data={current.data}
              selected={row}
              select={select}
              page={page}
            />
          )}
        </>
      )}
    </div>
  );
}
