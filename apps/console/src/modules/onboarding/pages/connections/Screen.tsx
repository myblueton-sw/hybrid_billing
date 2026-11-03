import { useEffect, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { text } from "../../../../shared/copy";
import { connection, actionUrl, ServiceError, type Connection } from "./api";
import "./style.css";
const displayTime = (value: string) =>
  !Number.isFinite(Date.parse(value)) ? "—" : new Date(value)
    .toISOString()
    .replace("T", " ")
    .replace(/\.\d{3}Z$/, " UTC");
export default function Connections({
  locale: l,
  objectId,
  onState,
}: {
  locale: Locale;
  objectId?: string;
  onState: (state: string, items: string[]) => void;
}) {
  const c = (en: string, ko: string, ar: string) => text(l, en, ko, ar);
  const [data, setData] = useState<Connection | null>(null),
    [error, setError] = useState<ServiceError | null>(null),
    [loading, setLoading] = useState(true),
    [refresh, setRefresh] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    let active = true;
    setData(null);
    setError(null);
    setLoading(true);
    void connection(objectId, controller.signal)
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
  }, [objectId, refresh]);
  useEffect(
    () =>
      onState(JSON.stringify({ data, error: error?.code, loading }), [
        "S01:scope:connection",
        "S01:data:input-support",
        "S01:data:last-reception",
        "S01:state:holds",
        "S01:action:next",
        "S01:action:refresh",
        "S01:state:unavailable",
      ]),
    [data, error, loading, onState],
  );
  const state = (value: string) =>
    ({
      pending: c("Pending input", "입력 대기", "مدخل قيد الانتظار"),
      verified: c("Validated input", "검증된 입력", "مدخل متحقق منه"),
      sealed: c(
        "Previously published input",
        "이전 공개 입력",
        "مدخل منشور سابقاً",
      ),
    })[value] ?? value;
  const actionLabel = (action: string) =>
    ({
      imports: c("Open intake", "수입 화면 열기", "فتح الاستيراد"),
      "source-data": c(
        "Inspect normalized rows",
        "정규화 명세 조회",
        "فحص الصفوف المعيارية",
      ),
      reconciliation: c(
        "Compare source and storage",
        "원본·저장 대사",
        "مقارنة المصدر والتخزين",
      ),
    })[action] ?? action;
  const r = data?.latest_candidate;
  return (
    <div className="connections-screen">
      <section className="connection-heading">
        <div>
          <h2>
            {c("Manual synthetic CSV", "수동 합성 CSV", "CSV تجريبي يدوي")}
          </h2>
          <p>
            {c(
              "Actual local connection metadata. Refresh does not test an external provider.",
              "실제 로컬 연결 메타데이터입니다. 새로고침은 외부 Provider 진단을 실행하지 않습니다.",
              "بيانات وصفية فعلية لاتصال محلي. التحديث لا يختبر مزوداً خارجياً.",
            )}
          </p>
        </div>
        <button
          id="connection-refresh"
          disabled={loading}
          onClick={() => setRefresh((v) => v + 1)}
        >
          {c("Refresh status", "현황 새로고침", "تحديث الحالة")}
        </button>
      </section>
      <p role="status">
        {loading
          ? c(
              "Checking current access and connection scope…",
              "현재 권한과 연결 범위를 확인하고 있습니다…",
              "جارٍ التحقق من الوصول الحالي ونطاق الاتصال…",
            )
          : ""}
      </p>
      {error && (
        <section role="alert" className="connection-notice">
          <h2>
            {c(
              "Connection details unavailable",
              "연결 상세를 조회할 수 없습니다",
              "تفاصيل الاتصال غير متاحة",
            )}
          </h2>
          <p>
            {error.code === "scope_denied"
              ? c(
                  "Access to this connection is unavailable. Previous metadata was cleared.",
                  "이 연결의 열람 권한을 확인할 수 없어 이전 정보를 지웠습니다.",
                  "الوصول إلى هذا الاتصال غير متاح. تم مسح المعلومات السابقة.",
                )
              : c(
                  "The service or response could not be verified. Retry with Refresh status.",
                  "서비스 또는 응답을 확인하지 못했습니다. 현황 새로고침으로 다시 조회하세요.",
                  "تعذر التحقق من الخدمة أو الاستجابة. أعد المحاولة بتحديث الحالة.",
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
        </section>
      )}
      {data && (
        <>
          <section className="connection-scope">
            <h2>{c("Configured scope", "설정된 범위", "النطاق المهيأ")}</h2>
            <p>
              {c(
                "This interval is allowed for input; it does not prove complete data coverage.",
                "입력이 허용된 기간입니다. 해당 기간의 자료가 모두 확보됐다는 뜻은 아닙니다.",
                "هذه الفترة مسموحة للإدخال؛ ولا تثبت اكتمال تغطية البيانات.",
              )}
            </p>
            <dl>
              <div>
                <dt>{c("Connection ID", "연결 ID", "معرّف الاتصال")}</dt>
                <dd>
                  <a
                    href={
                      "#/connections/" +
                      data.connection +
                      "?" +
                      new URLSearchParams({ lang: l, service: "local" })
                    }
                  >
                    <bdi>{data.connection}</bdi>
                  </a>
                </dd>
              </div>
              <div>
                <dt>{c("Workspace", "업무 공간", "مساحة العمل")}</dt>
                <dd>
                  <bdi>{data.workspace}</bdi>
                </dd>
              </div>
              <div>
                <dt>{c("Billing account", "청구 계정", "حساب الفوترة")}</dt>
                <dd>
                  <bdi>{data.account}</bdi>
                </dd>
              </div>
              <div>
                <dt>{c("Subaccount", "하위 계정", "الحساب الفرعي")}</dt>
                <dd>
                  {data.subaccount ? (
                    <bdi>{data.subaccount}</bdi>
                  ) : (
                    c("Not configured", "설정 없음", "غير مهيأ")
                  )}
                </dd>
              </div>
              <div>
                <dt>
                  {c(
                    "Period · start inclusive, end exclusive",
                    "기간 · 시작 포함, 끝 제외",
                    "الفترة · البداية مشمولة والنهاية مستثناة",
                  )}
                </dt>
                <dd>
                  <bdi dir="ltr" className="connection-date">
                    {displayTime(data.period_start)}
                  </bdi>
                  <bdi dir="ltr" className="connection-date">
                    {displayTime(data.period_end)}
                  </bdi>
                </dd>
              </div>
              <div>
                <dt>
                  {c(
                    "Policy revision / observed at",
                    "정책 개정 / 조회 시각",
                    "مراجعة السياسة / وقت الفحص",
                  )}
                </dt>
                <dd>
                  <bdi>{data.policy_revision}</bdi>
                  <br />
                  <bdi dir="ltr">{displayTime(data.observed_at)}</bdi>
                </dd>
              </div>
            </dl>
          </section>
          <section>
            <h2>
              {c(
                "Installed CSV input contracts",
                "설치된 CSV 입력 계약",
                "عقود إدخال CSV المثبتة",
              )}
            </h2>
            <p>
              {c(
                "Parser and admission capabilities of this local service. These are not verified cloud-provider connections.",
                "로컬 서비스의 파서·입력 허용 범위입니다. 클라우드 Provider 연결을 검증한 결과가 아닙니다.",
                "قدرات التحليل وقبول الإدخال لهذه الخدمة المحلية. ليست اتصالات بمزودي السحابة تم التحقق منها.",
              )}
            </p>
            <dl className="connection-limits">
              <div>
                <dt>
                  {c(
                    "Maximum input size (bytes)",
                    "최대 입력 크기 (바이트)",
                    "الحد الأقصى لحجم الإدخال (بايت)",
                  )}
                </dt>
                <dd>
                  <bdi dir="ltr">{data.max_bytes.toLocaleString("en-US")}</bdi>
                </dd>
              </div>
              <div>
                <dt>
                  {c(
                    "Maximum logical records per input",
                    "입력당 최대 논리 레코드",
                    "الحد الأقصى للسجلات المنطقية لكل إدخال",
                  )}
                </dt>
                <dd>
                  <bdi dir="ltr">{data.max_rows.toLocaleString("en-US")}</bdi>
                </dd>
              </div>
            </dl>
            <div className="connection-contracts">
              {data.contracts.map((v) => (
                <article key={v.version}>
                  <h3>
                    <bdi dir="ltr">FOCUS {v.version}</bdi>
                  </h3>
                  <p>
                    {c("Registry revision", "입력 계약 개정", "مراجعة العقد")}{" "}
                    <bdi>{v.revision}</bdi>
                  </p>
                  <details>
                    <summary>
                      {c("Contract SHA-256", "계약 SHA-256", "SHA-256 للعقد")}
                    </summary>
                    <bdi>{v.digest}</bdi>
                  </details>
                </article>
              ))}
            </div>
          </section>
          <section className="connection-reception">
            <h2>
              {c(
                "Latest registered input candidate",
                "최근 등록 입력 후보",
                "أحدث مرشح إدخال مسجل",
              )}
            </h2>
            <p>
              {c(
                "This excludes quarantined attempts. It is not the latest successful provider collection.",
                "격리된 시도는 포함하지 않습니다. Provider 수집의 최종 성공 시각을 뜻하지 않습니다.",
                "لا يشمل المحاولات المعزولة. لا يمثل آخر جمع ناجح من المزود.",
              )}
            </p>
            {r ? (
              <dl>
                <div>
                  <dt>{c("Candidate ID", "후보 ID", "معرّف المرشح")}</dt>
                  <dd>
                    <bdi>{r.release_id}</bdi>
                  </dd>
                </div>
                <div>
                  <dt>
                    {c(
                      "Received / source revision",
                      "수신 시각 / 원본 개정",
                      "وقت الاستلام / مراجعة المصدر",
                    )}
                  </dt>
                  <dd>
                    <bdi dir="ltr">{displayTime(r.created_at)}</bdi>
                    <br />
                    <bdi>{r.source_revision}</bdi>
                  </dd>
                </div>
                <div>
                  <dt>{c("Input state", "입력 상태", "حالة المدخل")}</dt>
                  <dd>{state(r.state)}</dd>
                </div>
                <div>
                  <dt>{c("Input contract", "입력 계약", "عقد الإدخال")}</dt>
                  <dd>
                    <bdi dir="ltr">FOCUS {r.version}</bdi>
                  </dd>
                </div>
              </dl>
            ) : (
              <p>
                {c(
                  "No registered input candidate. This does not imply that no quarantined attempt exists.",
                  "등록된 입력 후보가 없습니다. 격리된 시도까지 없다는 뜻은 아닙니다.",
                  "لا يوجد مرشح إدخال مسجل. هذا لا يعني عدم وجود محاولة معزولة.",
                )}
              </p>
            )}
            <div className="connection-holds">
              <p>
                {data.holds.quarantine
                  ? c(
                      "Quarantined input exists. Review the intake result before financial confirmation.",
                      "격리된 입력이 있습니다. 재무 확정 전에 수입 결과를 검토해야 합니다.",
                      "توجد مدخلات معزولة. راجع نتيجة الاستيراد قبل التأكيد المالي.",
                    )
                  : c(
                      "No quarantine entry was found in this connection at observation time.",
                      "조회 시점에 이 연결의 격리 기록이 없습니다.",
                      "لم يُعثر على سجل عزل لهذا الاتصال وقت الفحص.",
                    )}
              </p>
              <p>
                {data.holds.unresolved
                  ? c(
                      "An operation outcome remains unresolved. Do not infer that processing completed.",
                      "결과가 미확정인 작업이 있습니다. 처리가 완료됐다고 판단하지 마세요.",
                      "توجد عملية نتيجتها غير محسومة. لا تفترض اكتمال المعالجة.",
                    )
                  : c(
                      "No unresolved operation was found at observation time.",
                      "조회 시점에 결과 미확정 작업이 없습니다.",
                      "لم يُعثر على عملية غير محسومة وقت الفحص.",
                    )}
              </p>
            </div>
          </section>
          <section>
            <h2>
              {c(
                "Continue with this connection",
                "이 연결에서 계속하기",
                "المتابعة مع هذا الاتصال",
              )}
            </h2>
            <p>
              {c(
                "Actions reflect the fixed service identity. Each destination checks current access again.",
                "서버의 고정 실행 신원에 허용된 경로입니다. 이동한 화면에서 현재 권한을 다시 확인합니다.",
                "تعكس الإجراءات هوية الخدمة الثابتة. تتحقق كل وجهة من الوصول الحالي مجدداً.",
              )}
            </p>
            <nav className="connection-actions">
              {data.actions.map((a) => (
                <a key={a} data-action={a} href={actionUrl(a, data, l)}>
                  {actionLabel(a)}
                </a>
              ))}
              {data.actions.includes("imports") && r && (
                <a
                  data-action="new-import"
                  href={
                    "#/imports?" +
                    new URLSearchParams({ lang: l, service: "local" })
                  }
                >
                  {c(
                    "Receive another input",
                    "새 입력 수신",
                    "استلام مدخل آخر",
                  )}
                </a>
              )}
            </nav>
          </section>
          <section className="connection-notice">
            <h2>
              {c(
                "Coverage not yet established",
                "아직 확인되지 않은 지원 범위",
                "التغطية التي لم تُثبت بعد",
              )}
            </h2>
            <dl>
              <div>
                <dt>
                  {c("Data completeness", "자료 완전성", "اكتمال البيانات")}
                </dt>
                <dd>{c("Unknown", "알 수 없음", "غير معروف")}</dd>
              </div>
              <div>
                <dt>
                  {c(
                    "Provider authentication / operational reads / invoices",
                    "Provider 인증 / 운영 읽기 / 인보이스",
                    "مصادقة المزود / القراءة التشغيلية / الفواتير",
                  )}
                </dt>
                <dd>{c("Not connected", "미연결", "غير متصل")}</dd>
              </div>
            </dl>
            <p>
              {c(
                "Configuration edits, credential replacement, revocation and provider diagnostic jobs are not connected. Onboarding and go-live decisions are handled separately.",
                "설정 편집·자격 교체·연결 회수·Provider 진단 작업은 미연결입니다. 온보딩과 개통 판단은 별도 단계에서 처리합니다.",
                "تعديل الإعدادات واستبدال بيانات الاعتماد وسحب الاتصال ومهام تشخيص المزود غير متصلة. تتم معالجة قرارات التهيئة والتشغيل بشكل منفصل.",
              )}
            </p>
          </section>
        </>
      )}
    </div>
  );
}
