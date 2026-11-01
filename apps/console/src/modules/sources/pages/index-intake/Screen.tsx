import { useEffect, useRef, useState } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { useLocation, routeUrl } from "../../../../shared/navigation";
import {
  readSyntheticEvidence,
  latestRequest,
  type EvidenceRow,
} from "../../api";
const baseItems = [
  "index-intake:field:scope",
  "index-intake:field:filter",
  "index-intake:field:search",
  "index-intake:table:records",
  "index-intake:state:feedback",
];
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
  const rawPage = Number(params.get("page") ?? 1),
    page =
      Number.isInteger(rawPage) && rawPage > 0 && rawPage <= 6 ? rawPage : 1;
  const search =
    mode === "synthetic" ? (params.get("q") ?? "").slice(0, 60) : "";
  const [input, setInput] = useState(search),
    [scenario, setScenario] = useState("normal"),
    [retry, setRetry] = useState(0);
  const [result, setResult] = useState<{ rows: EvidenceRow[]; total: number }>({
    rows: [],
    total: 0,
  });
  const [phase, setPhase] = useState("loading"),
    [selected, setSelected] = useState<string[]>([]),
    [detail, setDetail] = useState<EvidenceRow | null>(null);
  const requests = useRef(latestRequest());
  const queryKey = JSON.stringify([
    profile,
    mode,
    scope,
    filter,
    search,
    page,
    scenario,
    retry,
  ]);
  const [resultKey, setResultKey] = useState("");
  const ready = phase === "ready" && resultKey === queryKey;
  useEffect(() => setInput(search), [search]);
  useEffect(() => {
    const id = requests.current.next(),
      controller = new AbortController();
    setSelected([]);
    setDetail(null);
    setPhase(mode === "unconnected" ? "unconnected" : "loading");
    setResult({ rows: [], total: 0 });
    if (mode === "synthetic")
      readSyntheticEvidence(
        { scope, filter, search, page, scenario },
        controller.signal,
      )
        .then((data) => {
          if (requests.current.isCurrent(id)) {
            setResult(data);
            setResultKey(queryKey);
            setPhase("ready");
          }
        })
        .catch((error) => {
          if (requests.current.isCurrent(id) && error.name !== "AbortError")
            setPhase("error");
        });
    return () => {
      requests.current.invalidate();
      controller.abort();
    };
  }, [queryKey]);
  const visibleState = JSON.stringify({
    mode,
    scope,
    filter,
    search,
    input,
    page,
    scenario,
    phase: ready ? "ready" : phase,
    rows: ready ? result.rows.map((r) => r.id) : [],
    selected: ready ? selected : [],
    detail: detail?.id ?? null,
  });
  const reviewItems =
    mode === "synthetic"
      ? [
          ...baseItems,
          "index-intake:action:search",
          "index-intake:field:scenario",
          "index-intake:action:page",
          ...(ready && result.rows.length
            ? ["index-intake:action:select", "index-intake:action:inspect"]
            : []),
          ...(phase === "error" ? ["index-intake:action:retry"] : []),
          ...(detail ? ["index-intake:action:close-detail"] : []),
        ]
      : ["index-intake:state:unconnected"];
  reviewItems.push(
    ...contributions
      .filter((e) => e.slot === "sources/evidence/v1")
      .map((e) => `index-intake:extension:${e.id}`),
  );
  useEffect(
    () => onState(visibleState, reviewItems),
    [visibleState, mode, reviewItems.join("|"), onState],
  );
  function change(values: Record<string, string>) {
    location.hash = routeUrl("index-intake", {
      ...values,
      page: values.page ?? "1",
    });
  }
  return (
    <>
      <p className="lead">
        {t(
          l,
          "Check source completeness before moving to calculation.",
          "계산으로 넘어가기 전, 수집한 원본의 누락 여부를 확인합니다.",
          "تحقق من اكتمال المصادر قبل الانتقال إلى الحساب.",
        )}
      </p>
      {mode === "unconnected" ? (
        <section className="empty" role="status">
          <h2>
            {t(
              l,
              "Connect a source service",
              "원본 서비스 연결이 필요합니다",
              "يلزم ربط خدمة المصادر",
            )}
          </h2>
          <p>
            {t(
              l,
              "No backend contract is connected. Select the synthetic demo above to inspect this screen safely.",
              "현재 백엔드가 연결되지 않았습니다. 상단에서 합성 데이터 체험을 선택하면 화면 동작을 확인할 수 있습니다.",
              "لم يتم ربط الخدمة الخلفية. اختر البيانات التجريبية أعلاه لاستعراض الشاشة.",
            )}
          </p>
        </section>
      ) : (
        <>
          <section
            className="panel filters"
            aria-label={t(l, "Evidence filters", "원본 필터", "مرشحات الأدلة")}
          >
            <label>
              {t(
                l,
                "Sample workspace",
                "예제 업무 공간",
                "مساحة العمل التجريبية",
              )}
              <select
                id="scope"
                value={scope}
                onChange={(e) => change({ scope: e.target.value, q: "" })}
              >
                <option value="sample-a">Sample A</option>
                <option value="sample-b">Sample B</option>
              </select>
            </label>
            <label>
              {t(l, "Evidence status", "원본 상태", "حالة الأدلة")}
              <select
                id="filter"
                value={filter}
                onChange={(e) => change({ filter: e.target.value })}
              >
                <option value="all">{t(l, "All", "전체", "الكل")}</option>
                <option value="missing">
                  {t(l, "Missing evidence", "원본 누락", "أدلة ناقصة")}
                </option>
                <option value="ready">
                  {t(l, "Ready to inspect", "확인 가능", "جاهز للفحص")}
                </option>
              </select>
            </label>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                change({ q: input });
              }}
            >
              <label>
                {t(
                  l,
                  "Synthetic record ID",
                  "합성 레코드 ID",
                  "معرّف السجل التجريبي",
                )}
                <input
                  id="record-search"
                  maxLength={60}
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                />
              </label>
              <button type="submit">{t(l, "Search", "검색", "بحث")}</button>
            </form>
          </section>
          <details className="scenario">
            <summary>
              {t(
                l,
                "Synthetic state controls",
                "합성 상태 체험",
                "حالات العرض التجريبية",
              )}
            </summary>
            <label>
              {t(l, "Scenario", "시나리오", "السيناريو")}
              <select
                id="scenario"
                value={scenario}
                onChange={(e) => setScenario(e.target.value)}
              >
                <option value="normal">{t(l, "Normal", "정상", "عادي")}</option>
                <option value="empty">
                  {t(l, "Empty", "빈 결과", "فارغ")}
                </option>
                <option value="error">{t(l, "Failure", "실패", "فشل")}</option>
              </select>
            </label>
            <p>
              {t(
                l,
                "Only synthetic record searches are written to this demo URL.",
                "이 체험 URL에는 합성 레코드 검색어만 기록합니다.",
                "يُحفظ البحث عن السجلات التجريبية فقط في هذا الرابط.",
              )}
            </p>
          </details>
          <div aria-live="polite" role="status" className="result-summary">
            {phase === "error"
              ? t(
                  l,
                  "The sample request failed. Change the scenario or retry.",
                  "예제 요청이 실패했습니다. 시나리오를 변경하거나 다시 시도하세요.",
                  "فشل الطلب التجريبي. غيّر السيناريو أو أعد المحاولة.",
                )
              : !ready
                ? t(
                    l,
                    "Loading evidence…",
                    "원본을 불러오는 중…",
                    "جارٍ تحميل الأدلة…",
                  )
                : t(
                    l,
                    `${result.total} synthetic records · ${selected.length} selected`,
                    `합성 레코드 ${result.total}건 · ${selected.length}건 선택`,
                    `${result.total} سجل تجريبي · ${selected.length} محدد`,
                  )}
          </div>
          {phase === "error" && (
            <button id="retry" onClick={() => setRetry((n) => n + 1)}>
              {t(l, "Retry", "다시 시도", "إعادة المحاولة")}
            </button>
          )}
          {ready && result.total === 0 && (
            <section className="empty">
              {t(
                l,
                "No matching evidence. Adjust the filters.",
                "조건에 맞는 원본이 없습니다. 필터를 조정하세요.",
                "لا توجد أدلة مطابقة. عدّل المرشحات.",
              )}
            </section>
          )}
          {ready && result.total > 0 && (
            <div className="table-wrap">
              <table>
                <caption className="sr-only">
                  {t(
                    l,
                    "Synthetic supplier records",
                    "합성 공급자 원본 목록",
                    "سجلات المورد التجريبية",
                  )}
                </caption>
                <thead>
                  <tr>
                    {[
                      t(l, "Select", "선택", "تحديد"),
                      "ID",
                      t(l, "Provider", "공급자", "المورد"),
                      t(l, "Period", "기간", "الفترة"),
                      t(l, "Evidence", "원본", "الأدلة"),
                    ].map((c) => (
                      <th key={c} scope="col">
                        {c}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {result.rows.map((r) => (
                    <tr key={r.id}>
                      <td>
                        <input
                          type="checkbox"
                          aria-label={r.id}
                          checked={selected.includes(r.id)}
                          onChange={(e) =>
                            setSelected((old) =>
                              e.target.checked
                                ? [...old, r.id]
                                : old.filter((id) => id !== r.id),
                            )
                          }
                        />
                      </td>
                      <td>
                        <button className="link" onClick={() => setDetail(r)}>
                          <bdi>{r.id}</bdi>
                        </button>
                      </td>
                      <td>{r.provider}</td>
                      <td>
                        <bdi>{r.period}</bdi>
                      </td>
                      <td>
                        <span
                          className={`tag ${r.state === "missing" ? "warning" : ""}`}
                        >
                          {r.state === "missing"
                            ? t(
                                l,
                                "Missing evidence",
                                "원본 누락",
                                "أدلة ناقصة",
                              )
                            : t(
                                l,
                                "Ready to inspect",
                                "확인 가능",
                                "جاهز للفحص",
                              )}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          <div className="pagination">
            <button
              disabled={!ready || page <= 1}
              onClick={() => change({ page: String(page - 1) })}
            >
              {t(l, "Previous", "이전", "السابق")}
            </button>
            <span>
              <bdi>
                {page} /{" "}
                {ready ? Math.max(1, Math.ceil(result.total / 10)) : "—"}
              </bdi>
            </span>
            <button
              disabled={!ready || page * 10 >= result.total}
              onClick={() => change({ page: String(page + 1) })}
            >
              {t(l, "Next", "다음", "التالي")}
            </button>
          </div>
          {detail && ready && (
            <section
              className="panel"
              aria-label={t(
                l,
                "Evidence details",
                "원본 상세",
                "تفاصيل الأدلة",
              )}
            >
              <h2>
                <bdi>{detail.id}</bdi>
              </h2>
              <p>
                {t(
                  l,
                  "Synthetic source reference only. No amount calculation, download, approval or issuance is connected.",
                  "합성 원본 참조입니다. 금액 계산·다운로드·승인·발행은 연결되지 않았습니다.",
                  "مرجع مصدر تجريبي فقط. الحساب والتنزيل والموافقة والإصدار غير متصلة.",
                )}
              </p>
              <button onClick={() => setDetail(null)}>
                {t(l, "Close details", "상세 닫기", "إغلاق التفاصيل")}
              </button>
            </section>
          )}
        </>
      )}
      {contributions
        .filter((e) => e.slot === "sources/evidence/v1")
        .map((e) => {
          const View = e.render;
          return <View key={e.id} locale={l} />;
        })}
    </>
  );
}
