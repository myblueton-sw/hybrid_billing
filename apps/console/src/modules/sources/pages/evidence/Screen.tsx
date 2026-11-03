import { useEffect, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { text } from "../../../../shared/copy";
import {
  context,
  inspect,
  sourceReturn,
  ServiceError,
  type Evidence,
  type Field,
} from "./api";
import "./style.css";
export default function EvidenceScreen({
  locale: l,
  objectId,
  onState,
}: {
  locale: Locale;
  objectId?: string;
  onState: (state: string, items: string[]) => void;
}) {
  const c = (en: string, ko: string, ar: string) => text(l, en, ko, ar);
  const query = location.hash.split("?")[1] ?? "";
  const [data, setData] = useState<Evidence | null>(null),
    [error, setError] = useState<ServiceError | null>(null),
    [loading, setLoading] = useState(false),
    [retry, setRetry] = useState(0);
  let selected: ReturnType<typeof context> = null,
    invalid = false;
  try {
    selected = context(objectId, new URLSearchParams(query));
  } catch {
    invalid = true;
  }
  const back = selected
    ? sourceReturn(selected, l)
    : "#/source-data?lang=" + l + "&service=local";
  useEffect(() => {
    let active = true;
    const controller = new AbortController();
    setData(null);
    setError(null);
    setLoading(false);
    if (invalid) {
      setError(new ServiceError("request_invalid", ""));
      return;
    }
    if (!selected) return;
    setLoading(true);
    void inspect(selected, controller.signal)
      .then((value) => {
        if (active) setData(value);
      })
      .catch((e) => {
        if (active) {
          setData(null);
          setError(
            e instanceof ServiceError
              ? e
              : new ServiceError("service_unavailable", ""),
          );
        }
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
      controller.abort();
    };
  }, [objectId, query, retry]);
  useEffect(() => {
    if (data) document.getElementById("evidence-title")?.focus();
  }, [data]);
  useEffect(() => {
    if (!selected) return;
    const close = (event: KeyboardEvent) => {
      if (
        event.key === "Escape" &&
        !event.defaultPrevented &&
        !document.querySelector("dialog[open]")
      ) {
        event.preventDefault();
        location.hash = back;
      }
    };
    window.addEventListener("keydown", close);
    return () => window.removeEventListener("keydown", close);
  }, [back]);
  useEffect(
    () =>
      onState(JSON.stringify({ data, error: error?.code, loading, invalid }), [
        "S05:scope:source-row",
        "S05:preview:safe-text",
        "S05:integrity:source",
        "S05:state:unavailable",
        "S05:action:return",
        "S05:state:retention",
      ]),
    [data, error, loading, invalid, onState],
  );
  const original = (f: Field) =>
    !f.source_present ? (
      <span>
        {c("Source column absent", "원본 열 없음", "عمود المصدر غير موجود")}
      </span>
    ) : f.source_text === "" ? (
      <span>{c("Empty text", "빈 문자열", "نص فارغ")}</span>
    ) : (
      <bdi dir="ltr" className="evidence-value">
        {f.source_text}
      </bdi>
    );
  const normalized = (f: Field) =>
    f.normalized === null ? (
      <span>{c("Missing (null)", "누락 (null)", "مفقود (null)")}</span>
    ) : (
      <bdi dir="ltr" className="evidence-value">
        {f.normalized}
      </bdi>
    );
  const labels = [
    c("Field", "필드", "الحقل"),
    c("Original text", "원본 문자열", "النص الأصلي"),
    c(
      "Recomputed normalization",
      "원본 재정규화 값",
      "القيمة المعيارية المعاد حسابها",
    ),
  ];
  return (
    <div className="evidence-screen">
      <nav>
        <a id="evidence-return" href={back}>
          {selected
            ? c(
                "Return to selected source row",
                "선택한 명세 행으로 복귀",
                "العودة إلى صف المصدر المحدد",
              )
            : c("Choose a source row", "명세 행 선택", "اختيار صف المصدر")}
        </a>
      </nav>
      <section>
        <h2 id="evidence-title" tabIndex={-1}>
          {c(
            "Received source evidence",
            "수신 원본 근거",
            "أدلة المصدر المستلم",
          )}
        </h2>
        <p>
          {c(
            "Local synthetic inspection · a text extract of one received CSV record. It is not a public or confirmed billing document.",
            "로컬 합성 자료 조회 · 수신 CSV의 한 레코드를 텍스트로 열람합니다. 고객 공개본이나 청구 확정 문서가 아닙니다.",
            "فحص محلي لبيانات تجريبية · مقتطف نصي لسجل CSV مستلم واحد. ليس مستنداً عاماً أو مستند فوترة مؤكداً.",
          )}
        </p>
      </section>
      <p role="status">
        {loading
          ? c(
              "Checking access and original integrity…",
              "권한과 원본 무결성을 확인하고 있습니다…",
              "جارٍ التحقق من الوصول وسلامة الأصل…",
            )
          : ""}
      </p>
      {!selected && !invalid && !error && (
        <section className="evidence-notice">
          <h2>
            {c(
              "Select evidence from source data",
              "명세에서 증빙을 선택하세요",
              "اختر الدليل من بيانات المصدر",
            )}
          </h2>
          <p>
            {c(
              "Open a received input and its row detail, then choose View source evidence.",
              "수신 입력의 행 상세에서 원본 근거 보기를 선택하세요.",
              "افتح تفاصيل صف المدخل المستلم ثم اختر عرض دليل المصدر.",
            )}
          </p>
        </section>
      )}
      {error && (
        <section role="alert" className="evidence-notice">
          <h2>
            {c(
              "Evidence unavailable",
              "증빙을 열람할 수 없습니다",
              "الدليل غير متاح",
            )}
          </h2>
          <p>
            {error.code === "scope_denied"
              ? c(
                  "Current access could not be verified. Previous evidence was cleared.",
                  "현재 열람 권한을 확인할 수 없어 이전 증빙을 지웠습니다.",
                  "تعذر التحقق من الوصول الحالي. تم مسح الأدلة السابقة.",
                )
              : c(
                  "The selected source or revision could not be verified. Retry or return to source data.",
                  "선택한 원본 또는 개정을 확인하지 못했습니다. 다시 조회하거나 명세로 돌아가세요.",
                  "تعذر التحقق من المصدر أو المراجعة المحددة. أعد المحاولة أو عد إلى بيانات المصدر.",
                )}
          </p>
          <p>
            <bdi>{error.code}</bdi>
            {error.requestId && (
              <>
                {" "}
                · <bdi>{error.requestId}</bdi>
              </>
            )}
          </p>
          <button onClick={() => setRetry((v) => v + 1)}>
            {c("Retry", "다시 조회", "إعادة المحاولة")}
          </button>
        </section>
      )}
      {data && (
        <>
          <section className="evidence-summary">
            <div>
              <h3>
                {c("Original verified", "원본 검증됨", "تم التحقق من الأصل")}
              </h3>
              <p>
                {c(
                  "File hash and pinned normalization match the received manifest. This does not verify current warehouse values or financial eligibility.",
                  "파일 해시와 고정된 정규화 결과가 수신 명세와 일치합니다. 현재 저장소 값이나 청구 적격성을 검증한 것은 아닙니다.",
                  "تطابق تجزئة الملف ونتيجة التطبيع المثبتة بيان الاستلام. لا يتحقق هذا من قيم المستودع الحالية أو الأهلية المالية.",
                )}
              </p>
            </div>
            <dl>
              <div>
                <dt>
                  {c(
                    "Input / logical record",
                    "입력 / 논리 레코드",
                    "المدخل / السجل المنطقي",
                  )}
                </dt>
                <dd>
                  <bdi>{data.release_id}</bdi>
                  <br />
                  {c("Record", "레코드", "السجل")}{" "}
                  <bdi dir="ltr">
                    {data.ordinal} / {data.row_count}
                  </bdi>
                </dd>
              </div>
              <div>
                <dt>{c("Account", "계정", "الحساب")}</dt>
                <dd>
                  <bdi>{data.scope.account}</bdi>
                </dd>
              </div>
              <div>
                <dt>
                  {c(
                    "Billing period · start inclusive, end exclusive",
                    "청구 기간 · 시작 포함, 끝 제외",
                    "فترة الفوترة · البداية مشمولة والنهاية مستثناة",
                  )}
                </dt>
                <dd>
                  <bdi dir="ltr" className="evidence-date">
                    {data.scope.period_start}
                  </bdi>
                  <bdi dir="ltr" className="evidence-date">
                    {data.scope.period_end}
                  </bdi>
                </dd>
              </div>
              <div>
                <dt>
                  {c(
                    "Source / normalization revision",
                    "원본 / 정규화 개정",
                    "مراجعة المصدر / التطبيع",
                  )}
                </dt>
                <dd>
                  <bdi dir="ltr">
                    {data.source_revision} / FOCUS {data.version}
                  </bdi>
                </dd>
              </div>
            </dl>
          </section>
          <section>
            <h2>
              {c(
                "Source record comparison",
                "원본 레코드 대조",
                "مقارنة سجل المصدر",
              )}
            </h2>
            <p>
              {c(
                "Only allowed fields are shown. Empty text, an absent source column and normalized null are distinct. Values are displayed without executing HTML, formulas or links.",
                "허용된 필드만 표시합니다. 빈 문자열·원본 열 없음·정규화 null을 구분하며 HTML·수식·링크를 실행하지 않습니다.",
                "تظهر الحقول المسموح بها فقط. يُميّز النص الفارغ والعمود الغائب وnull. تُعرض القيم دون تنفيذ HTML أو الصيغ أو الروابط.",
              )}
            </p>
            <div className="evidence-mobile">
              {data.fields.map((f) => (
                <article key={f.name}>
                  <h3>
                    <bdi>{f.source_name ?? f.name}</bdi>
                  </h3>
                  <dl>
                    <dt>{labels[1]}</dt>
                    <dd>{original(f)}</dd>
                    <dt>{labels[2]}</dt>
                    <dd>{normalized(f)}</dd>
                  </dl>
                </article>
              ))}
            </div>
            <details
              className="evidence-full-table"
              open={window.innerWidth >= 640}
            >
              <summary>
                {c(
                  "Full field comparison",
                  "전체 필드 대조표",
                  "مقارنة الحقول الكاملة",
                )}
              </summary>
              <div
                className="planning-table"
                tabIndex={0}
                role="region"
                aria-label={c(
                  "Evidence fields, horizontally scrollable",
                  "증빙 필드, 가로 스크롤 가능",
                  "حقول الأدلة، قابلة للتمرير أفقياً",
                )}
              >
                <table>
                  <thead>
                    <tr>
                      {labels.map((label) => (
                        <th key={label} scope="col">
                          {label}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {data.fields.map((f) => (
                      <tr key={f.name}>
                        <th scope="row">
                          <bdi>{f.source_name ?? f.name}</bdi>
                          <small>
                            <bdi>{f.name}</bdi>
                          </small>
                        </th>
                        <td>{original(f)}</td>
                        <td>{normalized(f)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </details>
          </section>
          <details className="evidence-integrity">
            <summary>
              {c(
                "Integrity and scope details",
                "무결성·범위 상세",
                "تفاصيل السلامة والنطاق",
              )}
            </summary>
            <dl>
              {[
                [
                  c("Original SHA-256", "원본 SHA-256", "SHA-256 للأصل"),
                  data.original_digest,
                ],
                [
                  c("Manifest SHA-256", "근거 명세 SHA-256", "SHA-256 للبيان"),
                  data.snapshot,
                ],
                [
                  c("Registry SHA-256", "변환 계약 SHA-256", "SHA-256 للعقد"),
                  data.registry_digest,
                ],
                [
                  c("Workspace", "업무 공간", "مساحة العمل"),
                  data.scope.workspace,
                ],
                [c("Observed at", "조회 시각", "وقت الفحص"), data.observed_at],
                [
                  c("Original bytes", "원본 바이트", "حجم الأصل بالبايت"),
                  String(data.original_bytes),
                ],
              ].map(([k, v]) => (
                <div key={k}>
                  <dt>{k}</dt>
                  <dd>
                    <bdi dir="ltr">{v}</bdi>
                  </dd>
                </div>
              ))}
            </dl>
          </details>
          <section className="evidence-notice">
            <h2>
              {c(
                "Availability of other document functions",
                "그 밖의 문서 기능 연결 상태",
                "توفر وظائف المستند الأخرى",
              )}
            </h2>
            <p>
              {c(
                "Public documents, OCR, original downloads and restore requests are not connected. Retention and archival state are unknown; this view does not infer that an object was deleted or archived.",
                "고객 공개 문서·OCR·원본 다운로드·복원 요청은 아직 연결되지 않았습니다. 보존·보관 계층 상태는 알 수 없으며 삭제나 아카이브 상태로 추정하지 않습니다.",
                "المستندات العامة وOCR وتنزيل الأصول وطلبات الاستعادة غير متصلة. حالة الاحتفاظ والأرشفة غير معروفة؛ لا يُستنتج أن الكائن حُذف أو أُرشف.",
              )}
            </p>
          </section>
        </>
      )}
    </div>
  );
}
