import { useEffect, useState } from "react";
import type { ScreenProps } from "../../shared/types";
import { text as t } from "../../shared/copy";
import { screens } from "../../shared/catalog";
import { useLocation, routeUrl } from "../../shared/navigation";
import { workRecords, type WorkRecord } from "./index";
export default function WorkQueue({
  screenId,
  locale: l,
  mode,
  onState,
}: { screenId: string } & ScreenProps) {
  const hash = useLocation(),
    params = new URLSearchParams(hash.split("?")[1]),
    scope = params.get("scope") === "sample-b" ? "sample-b" : "sample-a";
  const prefix = screenId + "-",
    search = mode === "synthetic" ? (params.get(prefix + "q") ?? "") : "",
    status = params.get(prefix + "status") ?? "",
    period = params.get(prefix + "period") ?? "2026-10",
    provider = params.get(prefix + "provider") ?? "",
    size = [25, 50, 100].includes(Number(params.get(prefix + "size")))
      ? Number(params.get(prefix + "size"))
      : 25;
  const requested = Number(params.get(prefix + "page") ?? 1),
    page = Number.isInteger(requested) && requested > 0 ? requested : 1;
  const [refresh, setRefresh] = useState(0);
  const [selected, setSelected] = useState<string[]>([]),
    [scenario, setScenario] = useState("normal"),
    [phase, setPhase] = useState("loading"),
    [loaded, setLoaded] = useState("");
  const spec = screens.find((s) => s.id === screenId)!;
  const queryKey = JSON.stringify([
    scope,
    mode,
    search,
    status,
    period,
    provider,
    size,
    page,
    scenario,
    refresh,
  ]);
  useEffect(() => {
    setSelected([]);
    setPhase(mode === "unconnected" ? "unconnected" : "loading");
    if (mode === "unconnected") return;
    const timer = setTimeout(
      () => {
        setPhase(scenario === "error" ? "error" : "ready");
        setLoaded(queryKey);
      },
      scope === "sample-a" ? 160 : 40,
    );
    return () => clearTimeout(timer);
  }, [queryKey]);
  const ready = phase === "ready" && loaded === queryKey;
  const matches =
    mode === "synthetic" && scenario !== "empty"
      ? workRecords(scope).filter(
          (r) =>
            (!search ||
              `${r.id} ${r.customer} ${r.owner}`
                .toLowerCase()
                .includes(search.toLowerCase())) &&
            (!status || r.status === status) &&
            r.period === period &&
            (!provider || r.provider === provider) &&
            (screenId !== "my-tasks" || r.owner === "DEMO-PERSON-A"),
        )
      : [];
  const rows = ready ? matches.slice((page - 1) * size, page * size) : [];
  const items = [
    "region:layout",
    "section:purpose",
    "field:period",
    "field:record-search",
    "field:provider-filter",
    "field:status-filter",
    "field:page-size",
    "action:refresh",
    "action:prev",
    "action:next",
    "action:review-selection",
    "action:clear-selection",
    "action:scenario-normal",
    "action:scenario-error",
    "action:scenario-empty",
    ...(phase === "error" ? ["action:retry-load"] : []),
    ...(ready
      ? [
          "field:row-selection",
          "action:record-detail",
          ...spec.columns.map((c) => "data:" + c.key),
        ]
      : []),
  ].map((x) => screenId + ":" + x);
  useEffect(
    () =>
      onState(
        JSON.stringify({
          scope,
          search,
          status,
          period,
          provider,
          size,
          page,
          scenario,
          phase: ready ? "ready" : phase,
          selected: ready ? selected : [],
          rows: rows.map((r) => r.id),
        }),
        items,
      ),
    [
      queryKey,
      ready,
      phase,
      selected.join("|"),
      rows.map((r) => r.id).join("|"),
      onState,
    ],
  );
  function change(values: Record<string, string>) {
    location.hash = routeUrl(
      screenId,
      Object.fromEntries(
        Object.entries({ page: "1", ...values }).map(([k, v]) => [
          prefix + k,
          v,
        ]),
      ),
    );
  }
  function value(row: WorkRecord, key: string) {
    if (key === "status")
      return row.status === "missing"
        ? t(l, "Source missing", "원본 누락", "المصدر مفقود")
        : row.status === "processing"
          ? t(l, "Review pending", "검수 대기", "بانتظار المراجعة")
          : t(l, "Evidence ready", "근거 확인 가능", "الأدلة جاهزة");
    if (key === "reason")
      return row.status === "missing"
        ? t(
            l,
            "Supplier source reference is missing",
            "공급자 원본 참조 누락",
            "مرجع مصدر المورد مفقود",
          )
        : row.status === "processing"
          ? t(
              l,
              "Reviewer assignment is pending",
              "검수 담당자 지정 대기",
              "تعيين المراجع معلق",
            )
          : t(
              l,
              "Issuance prerequisites must be reviewed",
              "발행 선행 조건 검토 필요",
              "مراجعة متطلبات الإصدار مطلوبة",
            );
    return (
      (row as unknown as Record<string, string>)[key] ??
      t(l, "Not available", "확인 불가", "غير متاح")
    );
  }
  return (
    <>
      <section
        className="sd-list-context"
        data-design-id={`${screenId}:section:purpose`}
      >
        <h2>
          {t(l, "Billing work queue", "청구 업무 현황", "قائمة أعمال الفوترة")}
        </h2>
        <p>
          {t(
            l,
            "Review the customer, period, responsible person and unresolved reason before moving work forward. A missing amount or source is not zero.",
            "고객·기간·담당자·미해결 사유를 확인한 뒤 진행합니다. 미확인 금액이나 원천 누락을 0으로 표시하지 않습니다.",
            "راجع العميل والفترة والمسؤول والسبب غير المحسوم قبل المتابعة. المبلغ أو المصدر المفقود ليس صفراً.",
          )}
        </p>
      </section>
      <section className="panel">
        <div className="toolbar">
          <label>
            {t(l, "Billing period", "청구 기간", "فترة الفوترة")}
            <input
              id="period"
              type="month"
              value={period}
              onChange={(e) => change({ period: e.target.value })}
            />
          </label>
          <label className="grow">
            {t(
              l,
              "Find customer or record",
              "고객·업무 검색",
              "البحث عن عميل أو سجل",
            )}
            <input
              id="record-search"
              disabled={mode === "unconnected"}
              type="search"
              value={search}
              onChange={(e) => change({ q: e.target.value })}
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
              value={status}
              onChange={(e) => change({ status: e.target.value })}
            >
              <option value="">{t(l, "All", "전체", "الكل")}</option>
              <option value="missing">
                {t(l, "Source missing", "원본 누락", "المصدر مفقود")}
              </option>
              <option value="processing">
                {t(l, "Review pending", "검수 대기", "بانتظار المراجعة")}
              </option>
              <option value="ready">
                {t(l, "Evidence ready", "근거 확인 가능", "الأدلة جاهزة")}
              </option>
            </select>
          </label>
          <button
            id="refresh"
            onClick={() => {
              setScenario("normal");
              setRefresh((n) => n + 1);
              change({});
            }}
          >
            {t(l, "Refresh", "새로고침", "تحديث")}
          </button>
        </div>
        <div className="bulkbar">
          <span>
            {t(l, "Selected", "선택", "المحدد")}: {ready ? selected.length : 0}
          </span>
          <div>
            <button
              disabled={!ready || selected.length !== 1}
              id="review-selection"
              onClick={() => {
                location.hash = routeUrl("review-issue", {
                  case: selected[0],
                  period,
                });
              }}
            >
              {t(
                l,
                "Review prerequisites",
                "발행 선행 조건 검토",
                "مراجعة المتطلبات",
              )}
            </button>
            <button disabled={!selected.length} onClick={() => setSelected([])}>
              {t(l, "Clear selection", "선택 해제", "إلغاء التحديد")}
            </button>
          </div>
        </div>
        {mode === "unconnected" ? (
          <div className="state">
            <h2>
              {t(
                l,
                "Billing service not connected",
                "청구 서비스 미연결",
                "خدمة الفوترة غير متصلة",
              )}
            </h2>
          </div>
        ) : phase === "error" ? (
          <div className="state error" role="alert">
            <h2>
              {t(
                l,
                "Unable to load billing work",
                "청구 업무를 불러오지 못했습니다",
                "تعذر تحميل أعمال الفوترة",
              )}
            </h2>
            <button id="retry-load" onClick={() => setScenario("normal")}>
              {t(l, "Retry", "다시 시도", "إعادة المحاولة")}
            </button>
          </div>
        ) : !ready ? (
          <div className="state" role="status">
            {t(l, "Loading…", "불러오는 중…", "جار التحميل…")}
          </div>
        ) : !rows.length ? (
          <div className="state">
            {t(
              l,
              "No work matches the selected scope.",
              "선택한 범위의 업무가 없습니다.",
              "لا توجد أعمال تطابق النطاق.",
            )}
          </div>
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>{t(l, "Selection", "선택", "تحديد")}</th>
                  <th>ID</th>
                  {spec.columns.map((c) => (
                    <th key={c.key}>{c.label[l]}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <tr key={row.id}>
                    <td>
                      <input
                        type="checkbox"
                        aria-label={row.id}
                        checked={selected.includes(row.id)}
                        onChange={() =>
                          setSelected(
                            selected.includes(row.id)
                              ? selected.filter((x) => x !== row.id)
                              : [...selected, row.id],
                          )
                        }
                      />
                    </td>
                    <td>
                      <a
                        className="row-link"
                        href={routeUrl("customer-detail", {
                          case: row.id,
                          period,
                          from: screenId,
                        })}
                      >
                        <bdi>{row.id}</bdi>
                      </a>
                    </td>
                    {spec.columns.map((c) => (
                      <td key={c.key}>
                        {c.key === "customer" ? (
                          <a
                            href={routeUrl("customer-detail", {
                              case: row.id,
                              period,
                              from: screenId,
                            })}
                          >
                            <bdi>{row.customer}</bdi>
                          </a>
                        ) : (
                          <bdi>{value(row, c.key)}</bdi>
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
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
            <span>
              {ready
                ? matches.length
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
              {ready ? Math.max(1, Math.ceil(matches.length / size)) : "—"}
            </span>
            <button
              id="next"
              disabled={!ready || page * size >= matches.length}
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
            {["normal", "empty", "error"].map((s) => (
              <button
                key={s}
                id={"scenario-" + s}
                onClick={() => setScenario(s)}
              >
                {s === "normal"
                  ? t(l, "Normal", "정상", "عادي")
                  : s === "empty"
                    ? t(l, "Empty", "빈 결과", "فارغ")
                    : t(l, "Load error", "조회 실패", "خطأ تحميل")}
              </button>
            ))}
          </div>
        </details>
      </section>
    </>
  );
}
