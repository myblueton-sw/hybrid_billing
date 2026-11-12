import { useEffect, useRef, useState } from "react";
import SlowReadHint from "../../../../shared/SlowReadHint";
import { setNavigationGuard } from "../../../../shared/navigation";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import {
  loadOverview,
  ServiceError,
  type OverviewData,
} from "../../../sources/collected-focus/api";
import { focusQuery, focusHref } from "../../../sources/collected-focus/query";
import { emptyDraft, issues, type Draft } from "./model";
import { context, restore, save, clear, type Context } from "./store";
import { basisLabel, issueLabel } from "./copy";
import CostItems from "./CostItems";
import Summary from "./Summary";
import RelationEnd from "./RelationEnd";
import "./screen.css";
type State = {
  key: string;
  data?: OverviewData;
  context?: Context;
  draft: Draft;
  error?: string;
  notice?:
    | "restored"
    | "restore_failed"
    | "saved"
    | "save_failed"
    | "cleared"
    | "clear_failed";
  saved: boolean;
};
const itemIds = [
  "B01:scope:collected-source",
  "B01:form:review-draft",
  "B01:table:cost-components",
  "B01:calculation:declared-inclusion",
  "B01:action:browser-save",
  "B01:state:approval-unconnected",
  "B01:state:source-unavailable",
  "B01:table:row-identity",
  "B01:state:value-missing-zero",
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
  const query = focusQuery(params);
  if (!query.has("provider")) query.set("provider", "VMware");
  const queryKey = query.toString();
  const key = `${params.get("scope") ?? ""}:${params.get("company") ?? ""}:${queryKey}`;
  const [state, setState] = useState<State>({
    key,
    draft: emptyDraft(),
    saved: false,
  });
  const [dirty, setDirty] = useState(false),
    [attempted, setAttempted] = useState(false),
    [retry, setRetry] = useState(0);
  const generation = useRef(0),
    errorSummary = useRef<HTMLDivElement>(null);
  const current = state.key === key ? state : undefined,
    data = current?.data,
    draft = current?.draft ?? emptyDraft();
  const errors = data ? issues(draft, data, attempted) : [];
  useEffect(() => {
    const controller = new AbortController(),
      g = ++generation.current;
    setState({ key, draft: emptyDraft(), saved: false });
    setDirty(false);
    setAttempted(false);
    loadOverview(new URLSearchParams(queryKey), controller.signal)
      .then(async (data) => {
        const ctx = await context(data);
        if (controller.signal.aborted || generation.current !== g) return;
        let recovered: Draft | null = null,
          notice: State["notice"];
        try {
          recovered = restore(window.localStorage, ctx, data);
          if (recovered) notice = "restored";
        } catch {
          notice = "restore_failed";
        }
        if (!controller.signal.aborted && generation.current === g)
          setState({
            key,
            data,
            context: ctx,
            draft: recovered ?? emptyDraft(),
            saved: !!recovered,
            notice,
          });
      })
      .catch((error) => {
        if (!controller.signal.aborted && generation.current === g) {
          setState({
            key,
            draft: emptyDraft(),
            saved: false,
            error:
              error instanceof ServiceError
                ? error.code
                : "service_unavailable",
          });
          setDirty(false);
        }
      });
    return () => controller.abort();
  }, [key, retry]);
  useEffect(() => onDirty(dirty), [dirty, onDirty]);
  const reviewState = JSON.stringify({
    source: "collected-focus-review-draft",
    selection: data
      ? {
          period: data.selection.period,
          provider: data.selection.provider,
          basis: data.selection.basis,
        }
      : null,
    query_reference: current?.context?.query ?? null,
    snapshot: data?.snapshot ?? null,
    source_snapshot: data?.source_snapshot ?? null,
    after: data?.after ?? null,
    view,
    draft: data ? draft : null,
    saved: current?.saved ?? false,
    status: current?.error ?? (data ? "loaded" : "loading"),
    notice: current?.notice ?? null,
  });
  useEffect(() => onState(reviewState, itemIds), [reviewState, onState]);
  useEffect(() => {
    if (attempted && errors.length) errorSummary.current?.focus();
  }, [attempted]);
  function update(next: Draft) {
    setState((old) =>
      old.key === key && old.data
        ? { ...old, draft: next, saved: false, notice: undefined }
        : old,
    );
    setDirty(true);
  }
  function change(values: Record<string, string>, continuation = false) {
    if (!canDiscard()) return;
    const next = new URLSearchParams(params);
    next.set("focus-provider", query.get("provider") ?? "");
    if (!continuation)
      for (const name of ["after", "account_after", "snapshot"])
        next.delete(`focus-${name}`);
    for (const [name, value] of Object.entries(values))
      next.set(`focus-${name}`, value);
    setDirty(false);
    onDirty(false);
    setState({ key: "pending", draft: emptyDraft(), saved: false });
    location.hash = `#/cost-pools?${next}`;
  }
  function refresh() {
    if (!canDiscard()) return;
    setDirty(false);
    onDirty(false);
    if (
      query.has("snapshot") ||
      query.has("after") ||
      query.has("account_after")
    ) {
      const next = new URLSearchParams(params);
      next.set("focus-provider", query.get("provider") ?? "");
      for (const k of ["snapshot", "after", "account_after"])
        next.delete(`focus-${k}`);
      setState({ key: "pending", draft: emptyDraft(), saved: false });
      location.hash = `#/cost-pools?${next}`;
    } else {
      setState({ key, draft: emptyDraft(), saved: false });
      setRetry((n) => n + 1);
    }
  }
  function persist() {
    if (!data || !current?.context) return;
    setAttempted(true);
    if (issues(draft, data, true).length) {
      requestAnimationFrame(() => errorSummary.current?.focus());
      return;
    }
    try {
      save(window.localStorage, current.context, draft, data);
      setState((old) => ({ ...old, saved: true, notice: "saved" }));
      setDirty(false);
    } catch {
      setState((old) => ({ ...old, saved: false, notice: "save_failed" }));
    }
  }
  function discard() {
    if (!data || !current?.context || !canDiscard()) return;
    try {
      clear(window.localStorage, current.context);
      setState((old) => ({
        ...old,
        draft: emptyDraft(),
        saved: false,
        notice: "cleared",
      }));
      setDirty(false);
      setAttempted(false);
    } catch {
      setState((old) => ({ ...old, notice: "clear_failed" }));
    }
  }
  function switchTab(nextView: string) {
    const next = new URLSearchParams(params);
    next.set("view", nextView);
    setNavigationGuard(null);
    location.hash = `#/cost-pools?${next}`;
  }
  const p = new URLSearchParams(params);
  p.set("focus-provider", query.get("provider") ?? "");
  const sourceHref = focusHref("/source-data", p, data?.snapshot);
  const notice = current?.notice;
  const noticeText =
    notice === "restored"
      ? t(
          l,
          "Loaded this browser's draft after a fresh authorized source query.",
          "원천 조회 후 이 브라우저의 초안을 불러왔습니다.",
          "حُمّلت مسودة هذا المتصفح بعد استعلام مصدر حديث ومصرّح به.",
        )
      : notice === "restore_failed"
        ? t(
            l,
            "The browser draft could not be restored or does not match the current source. A new empty draft is shown.",
            "브라우저 초안을 읽지 못했거나 현재 원천과 맞지 않습니다. 새 빈 초안을 표시합니다.",
            "تعذّر استعادة مسودة المتصفح أو أنها لا تطابق المصدر الحالي. تُعرض مسودة جديدة فارغة.",
          )
        : notice === "saved"
          ? t(
              l,
              "Review draft saved in this browser only.",
              "이 브라우저에 검토 초안을 저장했습니다.",
              "حُفظت مسودة المراجعة في هذا المتصفح فقط.",
            )
          : notice === "save_failed"
            ? t(
                l,
                "Could not save in this browser. The current draft remains unsaved; check browser storage and retry.",
                "이 브라우저에 저장하지 못했습니다. 현재 초안은 저장되지 않았습니다. 저장 공간을 확인하고 다시 시도하세요.",
                "تعذّر الحفظ في هذا المتصفح. المسودة الحالية غير محفوظة؛ تحقّق من مساحة التخزين وأعد المحاولة.",
              )
            : notice === "cleared"
              ? t(
                  l,
                  "This browser's draft was removed.",
                  "이 브라우저의 초안을 삭제했습니다.",
                  "أُزيلت مسودة هذا المتصفح.",
                )
              : notice === "clear_failed"
                ? t(
                    l,
                    "Could not remove the browser draft. The current draft remains on screen.",
                    "브라우저 초안을 삭제하지 못했습니다. 현재 초안을 유지합니다.",
                    "تعذّر إزالة مسودة المتصفح. تبقى المسودة الحالية على الشاشة.",
                  )
                : "";
  const draftForm = data && (
    <section className="infra-draft" aria-labelledby="infra-draft-title">
      <h2 id="infra-draft-title">
        {t(l, "Review draft", "검토 초안", "مسودة المراجعة")}
      </h2>
      <label htmlFor="infra-draft-name">
        {t(l, "Composition name", "원가 구성명", "اسم مكوّنات التكلفة")}
      </label>
      <input
        id="infra-draft-name"
        value={draft.name}
        maxLength={120}
        aria-required="true"
        aria-invalid={attempted && !draft.name.trim()}
        onChange={(e) => update({ ...draft, name: e.target.value })}
      />
      <p className="infra-meta">
        {t(
          l,
          "This browser only. Not a private user account store. Do not enter credentials or customer information. No server policy or approval is created.",
          "이 브라우저에만 저장되며 사용자별 비공개 저장소가 아닙니다. 자격 증명이나 고객 정보는 입력하지 마세요. 서버 정책이나 승인 기록은 생성되지 않습니다.",
          "الحفظ في هذا المتصفح فقط وليس مخزناً خاصاً لكل مستخدم. لا تُدخل بيانات اعتماد أو معلومات عملاء. لا تُنشأ سياسة خادم أو سجلات اعتماد.",
        )}
      </p>
      <div className="infra-actions">
        <button id="infra-save" className="infra-primary" onClick={persist}>
          {t(l, "Save review draft", "검토 초안 저장", "حفظ مسودة المراجعة")}
        </button>
        <button onClick={discard}>
          {t(
            l,
            "Remove browser draft",
            "브라우저 초안 삭제",
            "إزالة مسودة المتصفح",
          )}
        </button>
      </div>
      <p className="infra-meta">
        {current?.saved
          ? t(
              l,
              "Saved in this browser",
              "이 브라우저에 저장됨",
              "محفوظة في هذا المتصفح",
            )
          : t(l, "Unsaved", "저장 안 됨", "غير محفوظة")}
      </p>
    </section>
  );
  return (
    <div
      className="infrastructure-cost"
      data-infra-state={data ? "loaded" : current?.error ? "error" : "loading"}
      data-infra-query={current?.context?.query ?? ""}
      data-infra-view={view}
    >
      <div className="infra-toolbar">
        <label htmlFor="infra-provider">
          {t(l, "Collected source", "수집 원천", "المصدر المجمّع")}
          <select
            id="infra-provider"
            value={query.get("provider") ?? ""}
            disabled={!data}
            onChange={(e) => change({ provider: e.target.value })}
          >
            <option value="">
              {t(l, "All providers", "전체 공급사", "كل المزوّدين")}
            </option>
            {(data?.options.providers ?? [query.get("provider") ?? "VMware"])
              .filter(Boolean)
              .map((v) => (
                <option value={v} key={v}>
                  {v}
                </option>
              ))}
          </select>
        </label>
        <label htmlFor="infra-period">
          {t(l, "Charge-start month", "과금 시작 월", "شهر بدء الرسوم")}
          <select
            id="infra-period"
            value={query.get("period") ?? "all"}
            disabled={!data}
            onChange={(e) => change({ period: e.target.value })}
          >
            <option value="all">
              {t(
                l,
                "All charge-start months",
                "전체 과금 시작 월",
                "كل أشهر بدء الرسوم",
              )}
            </option>
            {data?.options.periods.map((v) => (
              <option key={v} value={v}>
                {v}
              </option>
            ))}
          </select>
        </label>
        <label htmlFor="infra-basis">
          {t(l, "Cost basis", "비용 기준", "أساس التكلفة")}
          <select
            id="infra-basis"
            value={query.get("basis") ?? "billed_cost"}
            disabled={!data}
            onChange={(e) => change({ basis: e.target.value })}
          >
            <option value="billed_cost">{basisLabel(l, "billed_cost")}</option>
            <option value="effective_cost">
              {basisLabel(l, "effective_cost")}
            </option>
          </select>
        </label>
        <button onClick={refresh}>
          {t(l, "New query", "새 조회", "استعلام جديد")}
        </button>
        {data && (
          <a href={sourceHref}>
            {t(
              l,
              "Inspect collected source",
              "수집 원천 확인",
              "فحص المصدر المجمّع",
            )}
          </a>
        )}
      </div>
      {!data && !current?.error && (
        <p role="status">
          {t(
            l,
            "Loading authorized collected source…",
            "수집 원천을 조회하고 있습니다…",
            "جارٍ تحميل المصدر المجمّع المصرّح به…",
          )}
          <SlowReadHint locale={l} />
        </p>
      )}
      {current?.error && (
        <div className="infra-feedback" role="alert">
          <h2>
            {t(l, "Source unavailable", "원천 조회 불가", "المصدر غير متاح")}
          </h2>
          <p>{issueLabel(l, current.error)}</p>
          <p className="infra-meta">
            {t(
              l,
              "Previous source rows and draft are hidden. No browser draft is used as a fallback.",
              "이전 원천과 초안은 표시하지 않습니다. 브라우저 초안으로 대체하지 않습니다.",
              "أُخفيت صفوف المصدر والمسودة السابقة. لا تُستخدم مسودة المتصفح بديلاً.",
            )}
          </p>
        </div>
      )}
      {data && (
        <>
          <aside className="infra-source-notice">
            <h2>
              {t(
                l,
                "Collected sample evidence",
                "수집 샘플 근거",
                "دليل عيّنة مجمّعة",
              )}
            </h2>
            <p>
              {t(
                l,
                "Provider labels do not establish on-premises ownership or a complete infrastructure cost composition.",
                "공급사 이름만으로 온프렘 소유나 전체 인프라 원가 구성을 확인할 수 없습니다.",
                "تسمية المزوّد لا تثبت ملكية محلية أو اكتمال مكوّنات تكلفة البنية التحتية.",
              )}
            </p>
            <dl className="infra-source-facts">
              <div>
                <dt>{t(l, "Source rows", "원천 행", "صفوف المصدر")}</dt>
                <dd>{data.row_count}</dd>
              </div>
              <div>
                <dt>
                  {t(
                    l,
                    "Account scopes",
                    "계정 범위 수",
                    "عدد نطاقات الحسابات",
                  )}
                </dt>
                <dd>{data.account_count}</dd>
              </div>
              <div>
                <dt>{t(l, "Synthetic rows", "합성 행", "صفوف اصطناعية")}</dt>
                <dd>{data.provenance["synthetic-contract-fixture"] ?? 0}</dd>
              </div>
              <div>
                <dt>{t(l, "Current page", "현재 페이지", "الصفحة الحالية")}</dt>
                <dd>
                  <bdi dir="ltr">
                    {data.row_count ? data.after + 1 : 0}–
                    {Math.min(data.after + 25, data.row_count)} /{" "}
                    {data.row_count}
                  </bdi>
                </dd>
              </div>
            </dl>
            <p className="infra-meta">
              {t(
                l,
                "Account scopes are source account combinations, not legal entities or a count of unique billing accounts.",
                "계정 범위 수는 원천의 계정 조합 수이며 법인 수나 고유 결제 계정 수가 아닙니다.",
                "نطاقات الحسابات هي تركيبات حسابات المصدر، وليست كيانات قانونية أو عدد حسابات فوترة فريدة.",
              )}
            </p>
            <p className="infra-meta">
              {t(
                l,
                "Months use charge-start attribution in the tenant timezone, without proration. A selected month or matching charge period does not establish infrastructure-cost eligibility, depreciation or monthly accrual.",
                "월은 테넌트 시간대의 과금 시작 시각으로 구분하며 월별로 안분하지 않습니다. 월 선택이나 동일한 과금 기간만으로 인프라 원가 해당 여부·감가상각·월별 발생액을 확정할 수 없습니다.",
                "تُنسب الأشهر إلى بدء الرسوم بتوقيت المستأجر دون توزيع نسبي. اختيار الشهر أو تطابق فترة الرسوم لا يثبت أهلية تكلفة البنية التحتية أو الإهلاك أو الاستحقاق الشهري.",
              )}
            </p>
            {(query.get("q") || query.get("account")) && (
              <p className="infra-meta">
                {t(
                  l,
                  "This query retains the account or search restriction from the source screen.",
                  "원천 화면에서 지정한 계정 또는 검색 조건이 적용되어 있습니다.",
                  "يحتفظ هذا الاستعلام بقيد الحساب أو البحث من شاشة المصدر.",
                )}
              </p>
            )}
          </aside>
          {errors.length > 0 && (
            <div
              className="infra-feedback"
              ref={errorSummary}
              tabIndex={-1}
              role="alert"
            >
              <h2>
                {t(l, "Check the draft", "초안 확인 필요", "تحقّق من المسودة")}
              </h2>
              <ul>
                {errors.map((e) => (
                  <li key={e}>{issueLabel(l, e)}</li>
                ))}
              </ul>
            </div>
          )}
          {notice && (
            <p
              className="infra-feedback"
              role={notice.endsWith("failed") ? "alert" : "status"}
            >
              {noticeText}
            </p>
          )}
          {(view === "summary" || view === "costs") && (
            <>
              <div className="infra-summary">
                <Summary locale={l} data={data} draft={draft} />
                {draftForm}
              </div>
              <CostItems
                locale={l}
                data={data}
                draft={draft}
                edit={view === "costs"}
                update={update}
              />
              {view === "summary" && (
                <button onClick={() => switchTab("costs")}>
                  {t(
                    l,
                    "Edit cost items",
                    "원가 항목 편집",
                    "تعديل بنود التكلفة",
                  )}
                </button>
              )}
              {view === "costs" && (
                <button onClick={() => switchTab("summary")}>
                  {t(
                    l,
                    "Select source rows",
                    "원천 행 선택",
                    "تحديد صفوف المصدر",
                  )}
                </button>
              )}
              <nav
                className="infra-pages"
                aria-label={t(l, "Source pages", "원천 페이지", "صفحات المصدر")}
              >
                <button
                  disabled={!data.after}
                  onClick={() =>
                    change(
                      {
                        after: String(Math.max(0, data.after - 25)),
                        snapshot: data.snapshot,
                      },
                      true,
                    )
                  }
                >
                  {t(l, "Previous", "이전", "السابق")}
                </button>
                <span>
                  <bdi dir="ltr">
                    {data.row_count ? data.after + 1 : 0}–
                    {Math.min(data.after + 25, data.row_count)} /{" "}
                    {data.row_count}
                  </bdi>
                </span>
                <button
                  disabled={data.next_after === null}
                  onClick={() =>
                    change(
                      {
                        after: String(data.next_after),
                        snapshot: data.snapshot,
                      },
                      true,
                    )
                  }
                >
                  {t(l, "Next", "다음", "التالي")}
                </button>
              </nav>
              <section className="infra-evidence">
                <h2>
                  {t(
                    l,
                    "Evidence still required",
                    "추가 원가 근거",
                    "الأدلة المطلوبة بعد",
                  )}
                </h2>
                <p>
                  {t(
                    l,
                    "Legal owner, asset and lease records, depreciation policy, power/facility/license contracts, product catalog and approval history are not connected. Manual categories and evidence references are unverified declarations.",
                    "원가 소유 법인·자산 및 리스 명세·감가상각 정책·전력/시설/라이선스 계약·상품 카탈로그·승인 이력은 연결되지 않았습니다. 수동 범주와 근거 참조는 검증되지 않은 선언입니다.",
                    "لم تُربط الجهة القانونية المالكة وسجلات الأصول والإيجار وسياسة الإهلاك وعقود الكهرباء والمرافق والتراخيص وكتالوج المنتجات وسجل الاعتماد. الفئات والمراجع اليدوية إقرارات غير متحقّق منها.",
                  )}
                </p>
              </section>
            </>
          )}
          {view === "products" && (
            <section>
              <h2>
                {t(
                  l,
                  "Included items by product",
                  "상품별 포함 항목",
                  "البنود المشمولة لكل منتج",
                )}
              </h2>
              <p>
                {t(
                  l,
                  "Product catalog and contract bindings are not connected. The list below contains only whole-row inclusion declared within this draft, not product relationships or certified duplicate-cost exclusions.",
                  "상품 카탈로그와 계약은 연결되지 않았습니다. 아래는 이 초안에서 선언한 행 전체의 포함 관계이며 상품 관계나 검증된 중복 원가 제외 내역이 아닙니다.",
                  "كتالوج المنتجات وروابط العقود غير متصلة. تعرض القائمة أدناه تضمين الصفوف كاملةً كما أُعلن داخل هذه المسودة فقط، وليس علاقات منتجات أو استبعادات تكلفة مكررة موثّقة.",
                )}
              </p>
              {!draft.entries.some((e) => e.parent) ? (
                <p>
                  {t(
                    l,
                    "No declared inclusion relationships.",
                    "선언한 포함 관계가 없습니다.",
                    "لا توجد علاقات تضمين مُعلنة.",
                  )}
                </p>
              ) : (
                <div className="infra-relations">
                  {draft.entries
                    .filter((e) => e.parent)
                    .map((e) => (
                      <dl key={e.row}>
                        <dt>
                          {t(
                            l,
                            "Included row ID",
                            "포함 선언 행 ID",
                            "معرّف الصف المُعلن مشمولاً",
                          )}
                        </dt>
                        <dd data-relation="included">
                          <RelationEnd locale={l} rows={data.rows} id={e.row} />
                        </dd>
                        <dt>
                          {t(
                            l,
                            "Target row ID",
                            "포함 대상 행 ID",
                            "معرّف الصف الهدف",
                          )}
                        </dt>
                        <dd data-relation="target">
                          <RelationEnd
                            locale={l}
                            rows={data.rows}
                            id={e.parent as string}
                          />
                        </dd>
                        <dt>
                          {t(
                            l,
                            "Evidence reference — unverified",
                            "근거 참조 — 미검증",
                            "مرجع الدليل — غير متحقّق منه",
                          )}
                        </dt>
                        <dd>
                          {e.reference ||
                            t(
                              l,
                              "No reference entered",
                              "근거 참조 미입력",
                              "لم يُدخل مرجع",
                            )}
                        </dd>
                      </dl>
                    ))}
                </div>
              )}
              <button onClick={() => switchTab("costs")}>
                {t(
                  l,
                  "Edit cost items",
                  "원가 항목 편집",
                  "تعديل بنود التكلفة",
                )}
              </button>
            </section>
          )}
          {view === "approvals" && (
            <section className="infra-unconnected">
              <h2>
                {t(
                  l,
                  "Approval history unavailable",
                  "승인 이력 미연결",
                  "سجل الاعتماد غير متصل",
                )}
              </h2>
              <p>
                {t(
                  l,
                  "An authoritative approval history is not connected. Saving a browser draft neither requests nor records financial approval.",
                  "공식 승인 이력은 연결되지 않았습니다. 브라우저 초안 저장으로 재무 승인을 요청하거나 기록하지 않습니다.",
                  "لم يُربط سجل اعتماد رسمي. حفظ مسودة المتصفح لا يطلب اعتماداً مالياً ولا يسجّله.",
                )}
              </p>
            </section>
          )}
        </>
      )}
    </div>
  );
}
