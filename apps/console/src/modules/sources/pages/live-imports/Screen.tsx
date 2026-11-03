import { useEffect, useRef, useState } from "react";
import "./service.css";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import {
  request,
  ServiceError,
  type Profile,
  type Receipts,
  type Receipt,
} from "./api";
export default function LiveImports({
  locale: l,
  onState,
  onDirty,
}: {
  locale: Locale;
  onDirty: (dirty: boolean) => void;
  onState: (state: string, items: string[]) => void;
}) {
  const [profile, setProfile] = useState<Profile | null>(null),
    [receipts, setReceipts] = useState<Receipts | null>(null);
  const [file, setFile] = useState<File | null>(null),
    [version, setVersion] = useState("1.3"),
    [revision, setRevision] = useState("1");
  const [busy, setBusy] = useState(false),
    [error, setError] = useState<ServiceError | null>(null),
    [notice, setNotice] = useState("");
  const [selected, setSelected] = useState<Receipt | null>(null);
  const controller = useRef<AbortController | null>(null),
    epoch = useRef(0);
  const copy = (en: string, ko: string, ar: string) => t(l, en, ko, ar);
  const statusLabel = (value: string) =>
    ({
      verified: copy(
        "Validated candidate",
        "검증된 후보",
        "مرشح تم التحقق منه",
      ),
      pending: copy("Pending", "대기", "قيد الانتظار"),
      sealed: copy("Published", "공개됨", "منشور"),
      terminal: copy("Storage complete", "저장 완료", "اكتمل الحفظ"),
      unknown: copy(
        "Outcome unknown — inspect before retry",
        "결과 미확정 — 재시도 전 확인",
        "النتيجة غير معروفة — افحص قبل إعادة المحاولة",
      ),
    })[value] ?? value;
  const run = async (upload = false) => {
    const sequence = ++epoch.current;
    controller.current?.abort();
    const c = new AbortController();
    controller.current = c;
    setBusy(true);
    setError(null);
    setNotice("");
    setSelected(null);
    setReceipts(null);
    try {
      const p = await request<Profile>("/profile", c.signal);
      if (sequence !== epoch.current) return;
      setProfile(p);
      if (upload && file) {
        if (file.size > p.max_bytes) throw new ServiceError("file_size", "");
        const result = await request<{
          state: string;
          duplicate: boolean;
          request_id: string;
        }>("", c.signal, {
          method: "POST",
          headers: {
            "Content-Type": "text/csv",
            "X-Focus-Version": version,
            "X-Source-Revision": revision,
          },
          body: file,
        });
        if (sequence !== epoch.current) return;
        onDirty(false);
        setNotice(
          result.state === "quarantined"
            ? copy(
                "Quarantined. Not eligible for billing.",
                "격리되었습니다. 청구에 사용할 수 없습니다.",
                "تم العزل. غير مؤهل للفوترة.",
              )
            : result.duplicate
              ? copy(
                  "Existing receipt returned; no duplicate import.",
                  "기존 수집 결과를 반환했습니다. 중복 수집하지 않았습니다.",
                  "تم إرجاع الاستلام السابق دون استيراد مكرر.",
                )
              : copy(
                  "Validation stored. Worker confirmation is still required.",
                  "검증 결과를 저장했습니다. 담당자 확인은 아직 필요합니다.",
                  "تم حفظ التحقق. لا يزال تأكيد المسؤول مطلوباً.",
                ),
        );
      }
      const r = await request<Receipts>("", c.signal);
      if (sequence === epoch.current) setReceipts(r);
    } catch (e) {
      if (sequence === epoch.current && !c.signal.aborted) {
        setError(
          e instanceof ServiceError
            ? e
            : new ServiceError("service_unavailable", ""),
        );
        setProfile(null);
        if (e instanceof ServiceError && e.code === "revision_conflict") {
          try {
            const freshProfile = await request<Profile>("/profile", c.signal);
            const freshReceipts = await request<Receipts>("", c.signal);
            if (sequence === epoch.current) {
              setProfile(freshProfile);
              setReceipts(freshReceipts);
            }
          } catch {
            if (sequence === epoch.current) {
              setProfile(null);
              setReceipts(null);
            }
          }
        }
      }
    } finally {
      if (sequence === epoch.current) setBusy(false);
    }
  };
  useEffect(() => {
    onDirty(false);
    void run();
    return () => {
      epoch.current++;
      controller.current?.abort();
    };
  }, []);
  useEffect(
    () =>
      onState(
        JSON.stringify({
          live: true,
          profile,
          busy,
          error: error?.code,
          selected,
          receipts,
        }),
        [
          "S03:service:receive",
          "S03:service:receipts",
          "S03:service:scope",
          "S03:service:errors",
        ],
      ),
    [profile, busy, error, selected, receipts, onState],
  );
  const inspect = async (id: string) => {
    const sequence = ++epoch.current;
    controller.current?.abort();
    const c = new AbortController();
    controller.current = c;
    setBusy(true);
    setError(null);
    setSelected(null);
    try {
      const r = await request<Receipts>("/" + encodeURIComponent(id), c.signal);
      if (sequence === epoch.current) setSelected(r.releases[0]);
    } catch (e) {
      if (sequence === epoch.current && !c.signal.aborted) {
        setError(
          e instanceof ServiceError
            ? e
            : new ServiceError("service_unavailable", ""),
        );
        setReceipts(null);
      }
    } finally {
      if (sequence === epoch.current) setBusy(false);
    }
  };
  return (
    <section className="planning-body live-imports">
      <header>
        <h2>
          {copy(
            "Source imports · connected local service",
            "원천 수집 · 로컬 서비스 연결",
            "استيراد المصادر · خدمة محلية متصلة",
          )}
        </h2>
        <p>
          {copy(
            "Actual validation and persistence using synthetic billing files. This local operator profile does not provide production login or input publication.",
            "합성 청구 파일을 실제로 검증하고 저장합니다. 이 로컬 작업자 환경은 운영 로그인이나 입력 확정·공개를 제공하지 않습니다.",
            "تحقق وحفظ فعليان لملفات فوترة تجريبية. لا يوفر هذا الملف المحلي تسجيل دخول إنتاجياً أو نشر المدخلات.",
          )}
        </p>
        <button disabled={busy} onClick={() => void run()}>
          {copy("Refresh from server", "서버에서 다시 조회", "تحديث من الخادم")}
        </button>
      </header>
      {busy && (
        <p role="status">
          {copy(
            "Processing server request…",
            "서버 요청 처리 중…",
            "جارٍ معالجة الطلب…",
          )}
        </p>
      )}
      {error && (
        <div role="alert">
          <strong>
            {copy(
              "Request not completed. Inspect saved receipts below, or refresh from server before retrying.",
              "요청이 완료되지 않았습니다. 아래 수집 결과를 확인하거나 서버에서 다시 조회한 후 재시도하세요.",
              "لم يكتمل الطلب. افحص الاستلامات أدناه أو حدّث من الخادم قبل إعادة المحاولة.",
            )}
          </strong>
          <p>
            <bdi>
              {error.code} {error.requestId}
            </bdi>
          </p>
        </div>
      )}
      {notice && <p role="status">{notice}</p>}
      {profile && (
        <section className="planning-panel">
          <h3>
            {copy(
              "Server-authorized source",
              "서버에서 허용한 원천",
              "المصدر المسموح به من الخادم",
            )}
          </h3>
          <dl className="live-scope">
            <div>
              <dt>{copy("Account", "계정", "الحساب")}</dt>
              <dd>
                <bdi dir="ltr">{profile.account}</bdi>
              </dd>
            </div>
            <div>
              <dt>{copy("Period", "기간", "الفترة")}</dt>
              <dd>
                <bdi dir="ltr" className="live-timestamp">
                  {profile.period_start}
                </bdi>
                <bdi dir="ltr" className="live-timestamp">
                  {profile.period_end}
                </bdi>
              </dd>
            </div>
            <div className="live-identity">
              <dt>
                {copy(
                  "Workspace / connection",
                  "워크스페이스 / 연결",
                  "مساحة العمل / الاتصال",
                )}
              </dt>
              <dd>
                <bdi dir="ltr">{profile.workspace}</bdi>
                <br />
                <bdi dir="ltr">{profile.connection}</bdi>
              </dd>
            </div>
          </dl>
        </section>
      )}
      <form
        className="planning-panel planning-form"
        onSubmit={(e) => {
          e.preventDefault();
          void run(true);
        }}
      >
        <h3>
          {copy(
            "Receive billing CSV",
            "청구 CSV 수집",
            "استلام ملف CSV للفوترة",
          )}
        </h3>
        <div className="live-fields">
          <label>
            {copy("File", "파일", "الملف")}
            <span className="live-file-picker">
              <span className="live-file-choice">
                {copy("Choose file", "파일 선택", "اختيار ملف")}
              </span>
              <span>
                <bdi className={file ? undefined : "live-empty-file"}>
                  {file?.name ??
                    copy(
                      "No file selected",
                      "선택된 파일 없음",
                      "لم يتم اختيار ملف",
                    )}
                </bdi>
              </span>
              <input
                type="file"
                accept=".csv,text/csv"
                required
                disabled={busy}
                onChange={(e) => {
                  setFile(e.target.files?.[0] ?? null);
                  onDirty(Boolean(e.target.files?.[0]));
                }}
              />
            </span>
          </label>
          <label>
            {copy("FOCUS version", "FOCUS 버전", "إصدار FOCUS")}
            <select
              value={version}
              disabled={busy}
              onChange={(e) => setVersion(e.target.value)}
            >
              {(profile?.versions ?? ["1.2", "1.3"]).map((v) => (
                <option key={v}>{v}</option>
              ))}
            </select>
          </label>
          <label>
            {copy("Source revision", "원천 리비전", "مراجعة المصدر")}
            <input
              inputMode="numeric"
              pattern="[1-9][0-9]{0,18}"
              required
              value={revision}
              disabled={busy}
              onChange={(e) => setRevision(e.target.value)}
            />
          </label>
        </div>
        <p>
          {copy(
            "Same revision and bytes return the saved receipt. Changed bytes require a new revision. Limit: 8 MiB.",
            "같은 리비전·파일은 저장된 결과를 반환합니다. 내용이 바뀌면 새 리비전이 필요합니다. 한도: 8 MiB.",
            "تعيد المراجعة والبيانات المتطابقتان الاستلام المحفوظ. تتطلب البيانات المعدلة مراجعة جديدة. الحد: 8 MiB.",
          )}
        </p>
        <p id="live-submit-reason">
          {busy
            ? copy(
                "Wait for the current request.",
                "현재 요청이 끝날 때까지 기다려 주세요.",
                "انتظر انتهاء الطلب الحالي.",
              )
            : !profile
              ? copy(
                  "Connect to the service before uploading.",
                  "서비스 연결 후 업로드할 수 있습니다.",
                  "اتصل بالخدمة قبل الرفع.",
                )
              : !file
                ? copy(
                    "Select a file to enable upload.",
                    "파일을 선택하면 수집할 수 있습니다.",
                    "اختر ملفاً لتفعيل الرفع.",
                  )
                : copy(
                    "Ready to receive the selected file.",
                    "선택한 파일을 수집할 수 있습니다.",
                    "جاهز لاستلام الملف المحدد.",
                  )}
        </p>
        <button
          type="submit"
          aria-describedby="live-submit-reason"
          disabled={busy || !profile || !file}
        >
          {copy("Receive and validate", "수집 및 검증", "استلام وتحقق")}
        </button>
      </form>
      {receipts && (
        <section className="planning-panel">
          <h3>
            {copy("Saved imports", "저장된 수집 결과", "الاستيرادات المحفوظة")}
          </h3>
          {!receipts.releases.length && (
            <p>
              {copy(
                "No saved imports.",
                "저장된 수집 결과가 없습니다.",
                "لا توجد استيرادات محفوظة.",
              )}
            </p>
          )}
          <div className="planning-table" tabIndex={0}>
            <table>
              <thead>
                <tr>
                  {[
                    copy("Receipt", "수집 ID", "الاستلام"),
                    copy("Revision", "리비전", "المراجعة"),
                    copy("Rows", "행 수", "الصفوف"),
                    copy(
                      "Validation / persistence",
                      "검증 / 저장 상태",
                      "التحقق / الحفظ",
                    ),
                  ].map((x) => (
                    <th key={x}>{x}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {receipts.releases.map((r) => (
                  <tr key={r.release_id}>
                    <td data-label={copy("Receipt", "수집 ID", "الاستلام")}>
                      <button
                        className="live-receipt-link"
                        disabled={busy}
                        onClick={() => void inspect(r.release_id)}
                      >
                        <bdi dir="ltr">{r.release_id}</bdi>
                      </button>
                    </td>
                    <td data-label={copy("Revision", "리비전", "المراجعة")}>
                      <bdi dir="ltr">{r.source_revision}</bdi>
                    </td>
                    <td data-label={copy("Rows", "행 수", "الصفوف")}>
                      <bdi dir="ltr">{r.row_count}</bdi>
                    </td>
                    <td
                      data-label={copy(
                        "Validation / persistence",
                        "검증 / 저장 상태",
                        "التحقق / الحفظ",
                      )}
                    >
                      {statusLabel(r.state)} / {statusLabel(r.operation_state)}
                      <small>
                        <bdi dir="ltr">
                          {r.state} / {r.operation_state}
                        </bdi>
                      </small>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {receipts.more_releases && (
            <p>
              {copy(
                "Showing the latest 25 receipts. More exist.",
                "최근 25건을 표시합니다. 이전 결과가 더 있습니다.",
                "عرض أحدث 25 استلاماً. توجد نتائج إضافية.",
              )}
            </p>
          )}
          <h3>{copy("Quarantine", "격리 결과", "العزل")}</h3>
          {!receipts.quarantine.length && (
            <p>
              {copy(
                "No quarantine receipts.",
                "격리 결과가 없습니다.",
                "لا توجد استلامات معزولة.",
              )}
            </p>
          )}
          {receipts.quarantine.map((r) => (
            <dl className="live-quarantine" key={r.receipt_id}>
              <dt>
                <strong>
                  {copy(
                    "Blocked — not eligible for billing",
                    "차단 — 청구에 사용할 수 없음",
                    "محظور — غير مؤهل للفوترة",
                  )}
                </strong>
              </dt>
              <dt>
                {copy(
                  "Quarantined receipt",
                  "격리된 수집 결과",
                  "استلام معزول",
                )}
              </dt>
              <dd>
                <bdi dir="ltr">{r.receipt_id}</bdi>
              </dd>
              <dt>{copy("Revision", "리비전", "المراجعة")}</dt>
              <dd>
                <bdi dir="ltr">{r.source_revision}</bdi>
              </dd>
              <dt>
                {copy("Validation failed", "입력 검증 실패", "فشل التحقق")}
              </dt>
              <dd>
                <bdi dir="ltr">{r.reason}</bdi>
              </dd>
              <dt>{copy("Persistence state", "저장 상태", "حالة الحفظ")}</dt>
              <dd>
                {statusLabel(r.operation_state)}{" "}
                <bdi dir="ltr">({r.operation_state})</bdi>
              </dd>
            </dl>
          ))}
          {receipts.more_quarantine && (
            <p>
              {copy(
                "Only the latest 25 quarantine receipts are shown.",
                "격리 결과는 최근 25건만 표시합니다.",
                "يتم عرض أحدث 25 استلاماً معزولاً فقط.",
              )}
            </p>
          )}
        </section>
      )}
      {selected && (
        <section className="planning-panel" aria-live="polite">
          <h3>
            {copy("Persisted evidence", "저장된 검증 근거", "الأدلة المحفوظة")}
          </h3>
          <dl>
            {Object.entries(selected).map(([k, v]) => (
              <div key={k}>
                <dt>
                  {(
                    {
                      release_id: copy("Receipt", "수집 ID", "الاستلام"),
                      source_revision: copy(
                        "Source revision",
                        "원천 리비전",
                        "مراجعة المصدر",
                      ),
                      state: copy(
                        "Validation state",
                        "검증 상태",
                        "حالة التحقق",
                      ),
                      operation_state: copy(
                        "Persistence state",
                        "저장 상태",
                        "حالة الحفظ",
                      ),
                      row_count: copy("Rows", "행 수", "الصفوف"),
                      version: copy(
                        "FOCUS version",
                        "FOCUS 버전",
                        "إصدار FOCUS",
                      ),
                      file_digest: copy(
                        "Original hash",
                        "원본 해시",
                        "بصمة الأصل",
                      ),
                      manifest_digest: copy(
                        "Manifest hash",
                        "매니페스트 해시",
                        "بصمة البيان",
                      ),
                      registry_digest: copy(
                        "Registry hash",
                        "레지스트리 해시",
                        "بصمة السجل",
                      ),
                      created_at: copy(
                        "Received at",
                        "수집 시각",
                        "وقت الاستلام",
                      ),
                      duplicate: copy(
                        "Duplicate replay",
                        "중복 요청",
                        "إعادة طلب مكرر",
                      ),
                    } as Record<string, string>
                  )[k] ?? k}
                </dt>
                <dd>
                  <bdi>{String(v)}</bdi>
                </dd>
              </div>
            ))}
          </dl>
        </section>
      )}
    </section>
  );
}
