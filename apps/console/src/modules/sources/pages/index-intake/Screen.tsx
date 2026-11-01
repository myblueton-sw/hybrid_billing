import { useEffect, useRef, useState } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { useLocation, routeUrl } from "../../../../shared/navigation";
import {
  readSyntheticEvidence,
  latestRequest,
  type EvidenceRow,
} from "../../api";
import { screens } from "../../../../shared/catalog";
const spec = screens.find((s) => s.id === "index-intake")!;
export default function Screen({
  locale: l,
  mode,
  profile,
  contributions,
  onState,
}: ScreenProps) {
  const hash = useLocation(),
    params = new URLSearchParams(hash.split("?")[1]);
  const scope = params.get("scope") === "sample-b" ? "sample-b" : "sample-a";
  const filter = ["ready", "missing"].includes(params.get("filter") ?? "")
    ? params.get("filter")!
    : "all";
  const provider = ["AWS", "Azure", "GCP"].includes(
    params.get("provider") ?? "",
  )
    ? params.get("provider")!
    : "";
  const sort = params.get("sort") === "desc" ? "desc" : "asc",
    size = [25, 50, 100].includes(Number(params.get("size")))
      ? Number(params.get("size"))
      : 25;
  const n = Number(params.get("page")),
    page = Number.isInteger(n) && n > 0 && n < 100 ? n : 1;
  const search =
    mode === "synthetic" ? (params.get("q") ?? "").slice(0, 60) : "";
  const [input, setInput] = useState(search),
    [scenario, setScenario] = useState("normal"),
    [retry, setRetry] = useState(0),
    [view, setView] = useState("table");
  const [result, setResult] = useState({ rows: [] as EvidenceRow[], total: 0 }),
    [phase, setPhase] = useState("loading"),
    [resultKey, setResultKey] = useState("");
  const [selected, setSelected] = useState<string[]>([]),
    [all, setAll] = useState(false),
    [excluded, setExcluded] = useState<string[]>([]),
    [hidden, setHidden] = useState<string[]>([]),
    [detail, setDetail] = useState<EvidenceRow | null>(null),
    [fileNotice, setFileNotice] = useState("");
  const [dialogKind, setDialogKind] = useState("");
  const dialog = useRef<HTMLDialogElement>(null),
    trigger = useRef<HTMLElement | null>(null),
    requests = useRef(latestRequest());
  const queryKey = JSON.stringify([
    scope,
    filter,
    provider,
    sort,
    size,
    page,
    search,
    scenario,
    retry,
    mode,
    profile,
  ]);
  const ready =
    phase === "ready" && resultKey === queryKey && scenario !== "expired";
  useEffect(() => setInput(search), [search]);
  useEffect(() => {
    const epoch = requests.current.next(),
      controller = new AbortController();
    setSelected([]);
    setAll(false);
    setExcluded([]);
    setDetail(null);
    setFileNotice("");
    dialog.current?.close();
    setDialogKind("");
    setPhase(mode === "unconnected" ? "unconnected" : "loading");
    setResult({ rows: [], total: 0 });
    if (mode === "synthetic")
      readSyntheticEvidence(
        { scope, filter, provider, sort, size, page, search, scenario },
        controller.signal,
      )
        .then((data) => {
          if (requests.current.isCurrent(epoch)) {
            setResult(data);
            setResultKey(queryKey);
            setPhase("ready");
          }
        })
        .catch((error) => {
          if (requests.current.isCurrent(epoch) && error.name !== "AbortError")
            setPhase("error");
        });
    return () => {
      requests.current.invalidate();
      controller.abort();
    };
  }, [queryKey]);
  const columns = spec.columns.filter((c) => !hidden.includes(c.key));
  const activeItems = [
    "region:layout",
    "section:purpose",
    "field:record-search",
    "field:provider-filter",
    "field:status-filter",
    "field:sort",
    "action:refresh",
    "action:table-view",
    "action:card-view",
    "action:column-picker",
    "action:reset-filters",
    "action:intake",
    "field:page-size",
    "action:prev",
    "action:next",
    "action:scenario-normal",
    "action:scenario-error",
    "action:scenario-expire",
    "action:scenario-empty",
    "action:review-selection",
    ...contributions.map((e) => `extension:${e.id}`),
    ...(dialogKind === "columns"
      ? spec.columns.map((c) => `field:column-${c.key}`)
      : []),
    ...(dialogKind === "intake" ? ["field:local-file"] : []),
    ...(ready && result.rows.length
      ? [
          "field:row-selection",
          "action:record-detail",
          "field:page-select",
          "action:all-select",
          "action:clear-selection",
          ...columns.map((c) => `data:${c.key}`),
        ]
      : []),
    ...(phase === "error" ? ["action:retry-load"] : []),
    ...(detail ? ["action:close-detail"] : []),
  ].map((i) => `index-intake:${i}`);
  const visibleState = JSON.stringify({
    mode,
    scope,
    filter,
    provider,
    sort,
    size,
    page,
    search,
    input,
    scenario,
    phase:
      mode === "unconnected"
        ? "unconnected"
        : ready
          ? "ready"
          : scenario === "expired"
            ? "expired"
            : phase === "error"
              ? "error"
              : "loading",
    view,
    hidden,
    selected: ready ? selected : [],
    all: ready && all,
    excluded: ready ? excluded : [],
    detail: ready ? detail?.id : null,
    dialogKind: dialog.current?.open ? dialogKind : "",
    fileNotice,
    rows: ready ? result.rows.map((r) => r.id) : [],
  });
  useEffect(
    () => onState(visibleState, activeItems),
    [visibleState, activeItems.join("|"), onState],
  );
  function change(values: Record<string, string>) {
    location.hash = routeUrl("index-intake", {
      ...values,
      page: values.page ?? "1",
    });
  }
  function checked(id: string) {
    return all ? !excluded.includes(id) : selected.includes(id);
  }
  function toggle(id: string) {
    if (all)
      setExcluded(
        excluded.includes(id)
          ? excluded.filter((x) => x !== id)
          : [...excluded, id],
      );
    else
      setSelected(
        selected.includes(id)
          ? selected.filter((x) => x !== id)
          : [...selected, id],
      );
  }
  function open(kind: string, row?: EvidenceRow) {
    trigger.current = document.activeElement as HTMLElement;
    setDialogKind(kind);
    setDetail(row ?? null);
    dialog.current?.showModal();
  }
  function close() {
    dialog.current?.close();
    setDialogKind("");
    setDetail(null);
    trigger.current?.focus();
  }
  function value(row: EvidenceRow, key: string) {
    if (key === "source") return `SRC-${row.id}`;
    if (key === "evidence")
      return row.state === "missing"
        ? t(l, "Not available", "확인 불가", "غير متاح")
        : `EVD-${row.id}`;
    if (key === "status")
      return row.state === "ready"
        ? t(l, "Ready", "확인 가능", "جاهز")
        : t(l, "Missing", "원본 누락", "مفقود");
    return t(l, "Not available", "확인 불가", "غير متاح");
  }
  return (
    <>
      <section
        className="sd-list-context"
        data-design-id="index-intake:section:purpose"
      >
        <h2>
          {t(
            l,
            "Source and evidence context",
            "원천·근거 문맥",
            "سياق المصدر والأدلة",
          )}
        </h2>
        <p>
          {t(
            l,
            "Keep the source reference, billing period and revision together. Receipt, schema validity and financial reconciliation remain separate.",
            "원천 참조·청구 기간·리비전을 함께 확인합니다. 수신·규격 검증·재무 대사는 별개입니다.",
            "احتفظ بمرجع المصدر وفترة الفوترة والمراجعة معاً. الاستلام وصحة المخطط والمطابقة المالية حالات منفصلة.",
          )}
        </p>
      </section>
      <section className="panel">
        <div className="toolbar">
          <label className="grow">
            {t(l, "Search", "검색", "بحث")}
            <input
              id="record-search"
              type="search"
              value={input}
              placeholder="REC-0000001"
              maxLength={60}
              disabled={mode === "unconnected"}
              onChange={(e) => {
                setInput(e.target.value);
                change({ q: e.target.value });
              }}
            />
          </label>
          <label>
            {t(l, "Provider", "공급자", "المورد")}
            <select
              id="provider-filter"
              value={provider}
              onChange={(e) => change({ provider: e.target.value })}
            >
              <option value="">{t(l, "All", "전체", "الكل")}</option>
              {["AWS", "Azure", "GCP"].map((p) => (
                <option key={p}>{p}</option>
              ))}
            </select>
          </label>
          <label>
            {t(l, "Status", "상태", "الحالة")}
            <select
              id="status-filter"
              value={filter}
              onChange={(e) => change({ filter: e.target.value })}
            >
              <option value="all">{t(l, "All", "전체", "الكل")}</option>
              <option value="ready">
                {t(l, "Ready", "확인 가능", "جاهز")}
              </option>
              <option value="missing">
                {t(l, "Missing", "원본 누락", "مفقود")}
              </option>
            </select>
          </label>
          <label>
            {t(l, "Sort", "정렬", "الترتيب")}
            <select
              id="sort"
              value={sort}
              onChange={(e) => change({ sort: e.target.value })}
            >
              <option value="asc">
                {t(l, "Ascending", "오름차순", "تصاعدي")}
              </option>
              <option value="desc">
                {t(l, "Descending", "내림차순", "تنازلي")}
              </option>
            </select>
          </label>
          <button
            id="refresh"
            onClick={() => {
              setScenario("normal");
              setRetry(retry + 1);
            }}
          >
            {t(l, "Refresh", "새로고침", "تحديث")}
          </button>
        </div>
        <div className="toolbar">
          <button
            id="table-view"
            aria-pressed={view === "table"}
            onClick={() => setView("table")}
          >
            {t(l, "Table", "표", "جدول")}
          </button>
          <button
            id="card-view"
            aria-pressed={view === "cards"}
            onClick={() => setView("cards")}
          >
            {t(l, "Cards", "카드", "بطاقات")}
          </button>
          <button id="column-picker" onClick={() => open("columns")}>
            {t(l, "Columns", "열 선택", "الأعمدة")}
          </button>
          <button
            id="reset-filters"
            onClick={() =>
              change({ q: "", provider: "", filter: "all", sort: "asc" })
            }
          >
            {t(l, "Reset filters", "필터 초기화", "إعادة ضبط المرشحات")}
          </button>
          <button
            id="intake"
            className="primary"
            onClick={() => open("intake")}
          >
            {t(l, "Import files", "파일 수집", "استيراد الملفات")}
          </button>
        </div>
        <div id="bulk-controls">
          {ready && result.rows.length > 0 && (
            <div className="bulkbar">
              <div>
                <label className="inline">
                  <input
                    id="page-select"
                    type="checkbox"
                    checked={result.rows.every((r) => checked(r.id))}
                    ref={(node) => {
                      if (node)
                        node.indeterminate =
                          result.rows.some((r) => checked(r.id)) &&
                          !result.rows.every((r) => checked(r.id));
                    }}
                    onChange={(e) => {
                      const ids = result.rows.map((r) => r.id);
                      if (all)
                        setExcluded(
                          e.target.checked
                            ? excluded.filter((id) => !ids.includes(id))
                            : [...new Set([...excluded, ...ids])],
                        );
                      else
                        setSelected(
                          e.target.checked
                            ? [...new Set([...selected, ...ids])]
                            : selected.filter((id) => !ids.includes(id)),
                        );
                    }}
                  />
                  {t(
                    l,
                    "Select this page",
                    "현재 페이지 선택",
                    "تحديد هذه الصفحة",
                  )}
                </label>{" "}
                <span>
                  {t(l, "Selected", "선택", "المحدد")}:{" "}
                  {all ? result.total - excluded.length : selected.length}
                  {all ? ` / ${result.total}` : ""}
                </span>
              </div>
              <div>
                <button
                  id="all-select"
                  onClick={() => {
                    setAll(true);
                    setExcluded([]);
                    setSelected([]);
                  }}
                >
                  {t(
                    l,
                    "Select all matching records",
                    "조건에 맞는 전체 선택",
                    "تحديد كل النتائج",
                  )}
                </button>
                <button
                  id="clear-selection"
                  onClick={() => {
                    setSelected([]);
                    setAll(false);
                    setExcluded([]);
                  }}
                >
                  {t(l, "Clear selection", "선택 해제", "إلغاء التحديد")}
                </button>
                <button
                  id="review-selection"
                  className="primary"
                  disabled={!all && !selected.length}
                  onClick={() => open("selection")}
                >
                  {t(l, "Review selection", "선택 검토", "مراجعة التحديد")}
                </button>
              </div>
            </div>
          )}
        </div>
        <div id="results" aria-live="polite">
          {mode === "unconnected" ? (
            <div className="notice">
              {t(
                l,
                "Connect a source service",
                "원본 서비스를 연결하세요",
                "اربط خدمة المصادر",
              )}
            </div>
          ) : scenario === "expired" ? (
            <div className="notice">
              {t(
                l,
                "Snapshot expired. Refresh before continuing.",
                "스냅샷이 만료되었습니다. 새로고침 후 계속하세요.",
                "انتهت صلاحية اللقطة. حدّث قبل المتابعة.",
              )}
            </div>
          ) : phase === "error" ? (
            <div role="alert">
              <p>
                {t(
                  l,
                  "Source request failed. Retry.",
                  "원본 조회에 실패했습니다. 다시 시도하세요.",
                  "فشل طلب المصدر. أعد المحاولة.",
                )}
              </p>
              <button
                id="retry-load"
                onClick={() => {
                  setScenario("normal");
                  setRetry(retry + 1);
                }}
              >
                {t(l, "Retry", "다시 시도", "إعادة المحاولة")}
              </button>
            </div>
          ) : !ready ? (
            <p>{t(l, "Loading…", "불러오는 중…", "جار التحميل…")}</p>
          ) : !result.rows.length ? (
            <p>
              {t(
                l,
                "No matching evidence. Adjust the filters.",
                "조건에 맞는 원본이 없습니다. 필터를 조정하세요.",
                "لا توجد أدلة مطابقة. عدّل المرشحات.",
              )}
            </p>
          ) : view === "table" ? (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>{t(l, "Selection", "선택", "تحديد")}</th>
                    <th>ID</th>
                    {columns.map((c) => (
                      <th key={c.key}>{c.label[l]}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {result.rows.map((row) => (
                    <tr key={row.id}>
                      <td>
                        <input
                          type="checkbox"
                          aria-label={row.id}
                          checked={checked(row.id)}
                          onChange={() => toggle(row.id)}
                        />
                      </td>
                      <td>
                        <button
                          className="row-link"
                          onClick={() => open("detail", row)}
                        >
                          <bdi>{row.id}</bdi>
                        </button>
                      </td>
                      {columns.map((c) => (
                        <td key={c.key}>
                          <bdi>{value(row, c.key)}</bdi>
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="cards">
              {result.rows.map((row) => (
                <article className="record-card" key={row.id}>
                  <input
                    type="checkbox"
                    aria-label={row.id}
                    checked={checked(row.id)}
                    onChange={() => toggle(row.id)}
                  />
                  <button onClick={() => open("detail", row)}>
                    <bdi>{row.id}</bdi>
                  </button>
                  <dl className="facts">
                    {columns.map((c) => (
                      <div key={c.key}>
                        <dt>{c.label[l]}</dt>
                        <dd>
                          <bdi>{value(row, c.key)}</bdi>
                        </dd>
                      </div>
                    ))}
                  </dl>
                </article>
              ))}
            </div>
          )}
        </div>
        <div className="pager">
          <div>
            <label>
              {t(l, "Page size", "페이지 크기", "حجم الصفحة")}
              <select
                id="page-size"
                value={size}
                onChange={(e) => change({ size: e.target.value })}
              >
                {[25, 50, 100].map((n) => (
                  <option key={n}>{n}</option>
                ))}
              </select>
            </label>
            <span id="record-count">
              {ready && scenario !== "expired"
                ? `${result.total} ${t(l, "records", "건", "سجلات")}`
                : t(l, "Count unavailable", "건수 확인 불가", "العدد غير متاح")}
            </span>
          </div>
          <div>
            <button
              id="prev"
              disabled={!ready || page === 1}
              onClick={() => change({ page: String(page - 1) })}
            >
              {t(l, "Previous", "이전", "السابق")}
            </button>
            <span>
              {page} /{" "}
              {ready ? Math.max(1, Math.ceil(result.total / size)) : "—"}
            </span>
            <button
              id="next"
              disabled={!ready || page * size >= result.total}
              onClick={() => change({ page: String(page + 1) })}
            >
              {t(l, "Next", "다음", "التالي")}
            </button>
          </div>
        </div>
        <details className="scenario">
          <summary>
            {t(l, "Review scenarios", "검수 시나리오", "سيناريوهات المراجعة")}
          </summary>
          <div>
            {(["normal", "error", "expired", "empty"] as const).map((s) => (
              <button
                key={s}
                id={`scenario-${s === "expired" ? "expire" : s}`}
                onClick={() => setScenario(s)}
              >
                {s === "normal"
                  ? t(l, "Normal", "정상", "عادي")
                  : s === "error"
                    ? t(l, "Load error", "조회 실패", "خطأ تحميل")
                    : s === "empty"
                      ? t(l, "Empty", "빈 결과", "فارغ")
                      : t(l, "Expire snapshot", "스냅샷 만료", "انتهاء اللقطة")}
              </button>
            ))}
          </div>
        </details>
      </section>
      {contributions
        .filter((e) => e.slot === "sources/evidence/v1")
        .map((e) => (
          <e.render key={e.id} locale={l} />
        ))}
      <dialog
        ref={dialog}
        onCancel={close}
        aria-label={t(l, "Source review", "원본 검토", "مراجعة المصدر")}
      >
        {dialogKind === "selection" ? (
          <>
            <h2>{t(l, "Review selection", "선택 검토", "مراجعة التحديد")}</h2>
            <p>
              {t(
                l,
                "Local preview only. No approval or financial operation.",
                "로컬 미리보기입니다. 승인·금전 처리는 실행되지 않습니다.",
                "معاينة محلية فقط. لا موافقة أو عملية مالية.",
              )}
            </p>
            <p>{all ? result.total - excluded.length : selected.length}</p>
          </>
        ) : dialogKind === "columns" ? (
          <>
            <h2>{t(l, "Columns", "열 선택", "الأعمدة")}</h2>
            {spec.columns.map((c) => (
              <label className="checkbox" key={c.key}>
                <input
                  type="checkbox"
                  id={`column-${c.key}`}
                  checked={!hidden.includes(c.key)}
                  onChange={() =>
                    setHidden(
                      hidden.includes(c.key)
                        ? hidden.filter((k) => k !== c.key)
                        : [...hidden, c.key],
                    )
                  }
                />
                {c.label[l]}
              </label>
            ))}
          </>
        ) : dialogKind === "intake" ? (
          <>
            <h2>{t(l, "Import files", "파일 수집", "استيراد الملفات")}</h2>
            <p>
              {t(
                l,
                "Local preview only. Files are not uploaded.",
                "로컬 미리보기입니다. 파일은 업로드되지 않습니다.",
                "معاينة محلية فقط. لا يتم رفع الملفات.",
              )}
            </p>
            <input
              id="local-file"
              type="file"
              accept=".csv,.json"
              onChange={(e) =>
                setFileNotice(
                  e.target.files?.length
                    ? t(
                        l,
                        "File selected locally. No server receipt or validation has occurred.",
                        "로컬 파일을 선택했습니다. 서버 수신·검증은 수행되지 않았습니다.",
                        "تم اختيار ملف محلياً. لم يحدث استلام أو تحقق على الخادم.",
                      )
                    : "",
                )
              }
            />
            <p role="status">{fileNotice}</p>
          </>
        ) : (
          detail && (
            <>
              <h2>
                <bdi>{detail.id}</bdi>
              </h2>
              <dl className="facts">
                {spec.columns.map((c) => (
                  <div key={c.key}>
                    <dt>{c.label[l]}</dt>
                    <dd>
                      <bdi>{value(detail, c.key)}</bdi>
                    </dd>
                  </div>
                ))}
              </dl>
            </>
          )
        )}
        <button id="close-detail" onClick={close}>
          {t(l, "Close", "닫기", "إغلاق")}
        </button>
      </dialog>
    </>
  );
}
