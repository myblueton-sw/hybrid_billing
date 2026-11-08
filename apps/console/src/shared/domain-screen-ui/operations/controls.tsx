import TableRegion from "../../TableRegion";
import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import {
  t,
  unknown,
  unconnected,
  localOnly,
  type Field,
  type Props,
  type Words,
} from "./types";
export function Panel({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="ops-panel">
      <h2>{title}</h2>
      {children}
    </section>
  );
}
export function Evidence({
  locale: l,
  title,
  fields,
  reason,
}: {
  locale: Props["locale"];
  title: Words;
  fields: readonly Words[];
  reason: Words;
}) {
  return (
    <Panel title={t(l, title)}>
      <div className="ops-status">
        <strong>{t(l, unconnected)}</strong>
        <p>{t(l, reason)}</p>
      </div>
      <dl className="ops-facts">
        {fields.map((f, i) => (
          <div key={i}>
            <dt>{t(l, f)}</dt>
            <dd>{t(l, unknown)}</dd>
          </div>
        ))}
      </dl>
    </Panel>
  );
}
export function Draft({
  props: p,
  title,
  fields,
  notice = localOnly,
}: {
  props: Props;
  title: Words;
  fields: readonly Field[];
  notice?: Words;
}) {
  const l = p.locale;
  const [values, setValues] = useState<Record<string, string>>({});
  const [reviewed, setReviewed] = useState(false);
  const [invalid, setInvalid] = useState(false);
  const result = useRef<HTMLDivElement>(null);
  const edited = Object.values(values).some(Boolean);
  const ids = fields.map((f) => `${p.screenId}:field:${p.view}:${f.id}`);
  const reviewId = `${p.screenId}:action:${p.view}:review-local`;
  useEffect(() => {
    p.onDirty(edited);
    return () => p.onDirty(false);
  }, [edited, p.onDirty]);
  useEffect(
    () =>
      p.onState(
        JSON.stringify({
          screen: p.screenId,
          view: p.view,
          mode: "unconnected",
          edited,
          reviewed,
          invalid,
        }),
        [...ids, reviewId, `${p.screenId}:state:unconnected`],
      ),
    [p.screenId, p.view, edited, reviewed, invalid, p.onState],
  );
  return (
    <Panel title={t(l, title)}>
      <p className="ops-meta">{t(l, notice)}</p>
      <form
        noValidate
        onSubmit={(e) => {
          e.preventDefault();
          const missing = fields.find(
            (f) =>
              (f.required && !values[f.id]?.trim()) ||
              (values[f.id] && f.pattern && !f.pattern.test(values[f.id])),
          );
          setInvalid(!!missing);
          setReviewed(!missing);
          if (missing)
            document
              .getElementById(`${p.screenId}-${p.view}-${missing.id}`)
              ?.focus();
          else requestAnimationFrame(() => result.current?.focus());
        }}
      >
        <div className="ops-form">
          {fields.map((f) => (
            <label key={f.id} htmlFor={`${p.screenId}-${p.view}-${f.id}`}>
              <span>
                {t(l, f.label)}
                {f.required && t(l, [" · 필수", " · required", " · مطلوب"])}
              </span>
              {f.options ? (
                <select
                  id={`${p.screenId}-${p.view}-${f.id}`}
                  value={values[f.id] ?? ""}
                  required={f.required}
                  aria-invalid={
                    invalid && !!f.required && !values[f.id]?.trim()
                      ? true
                      : undefined
                  }
                  aria-describedby={
                    invalid ? `${p.screenId}-${p.view}-error` : undefined
                  }
                  onChange={(e) => {
                    setValues({ ...values, [f.id]: e.target.value });
                    setReviewed(false);
                    setInvalid(false);
                  }}
                >
                  <option value="">
                    {t(l, ["선택하세요", "Choose an option", "اختر خيارًا"])}
                  </option>
                  {f.options.map((o, i) => (
                    <option key={i} value={String(i)}>
                      {t(l, o)}
                    </option>
                  ))}
                </select>
              ) : f.kind === "textarea" ? (
                <textarea
                  id={`${p.screenId}-${p.view}-${f.id}`}
                  value={values[f.id] ?? ""}
                  maxLength={2000}
                  rows={4}
                  required={f.required}
                  aria-invalid={
                    invalid && !!f.required && !values[f.id]?.trim()
                      ? true
                      : undefined
                  }
                  aria-describedby={
                    invalid ? `${p.screenId}-${p.view}-error` : undefined
                  }
                  onChange={(e) => {
                    setValues({ ...values, [f.id]: e.target.value });
                    setReviewed(false);
                    setInvalid(false);
                  }}
                />
              ) : (
                <input
                  id={`${p.screenId}-${p.view}-${f.id}`}
                  type={f.kind ?? "text"}
                  value={values[f.id] ?? ""}
                  maxLength={200}
                  required={f.required}
                  aria-invalid={
                    invalid && !!f.required && !values[f.id]?.trim()
                      ? true
                      : undefined
                  }
                  aria-describedby={
                    invalid ? `${p.screenId}-${p.view}-error` : undefined
                  }
                  autoComplete="off"
                  onChange={(e) => {
                    setValues({ ...values, [f.id]: e.target.value });
                    setReviewed(false);
                    setInvalid(false);
                  }}
                />
              )}
              {f.hint && <small>{t(l, f.hint)}</small>}
            </label>
          ))}
        </div>
        {invalid && (
          <p role="alert" id={`${p.screenId}-${p.view}-error`}>
            {t(l, [
              "필수 입력을 확인하세요. 업무 요청은 전송되지 않았습니다.",
              "Check required fields. No work request was sent.",
              "تحقق من الحقول المطلوبة. لم يتم إرسال أي طلب.",
            ])}
          </p>
        )}
        <div className="ops-actions">
          <button type="submit">
            {t(l, ["입력 내용 검토", "Review input", "مراجعة المدخلات"])}
          </button>
          <button
            type="button"
            disabled={!edited && !reviewed}
            onClick={() => {
              setValues({});
              setReviewed(false);
              setInvalid(false);
            }}
          >
            {t(l, ["입력 지우기", "Clear input", "مسح المدخلات"])}
          </button>
        </div>
      </form>
      {reviewed && (
        <div className="ops-review" tabIndex={-1} ref={result} role="status">
          <h3>{t(l, ["로컬 검토 내용", "Local review", "مراجعة محلية"])}</h3>
          <dl className="ops-facts">
            {fields.map((f) => (
              <div key={f.id}>
                <dt>{t(l, f.label)}</dt>
                <dd>
                  {values[f.id]
                    ? f.options
                      ? t(l, f.options[Number(values[f.id])])
                      : values[f.id]
                    : t(l, unknown)}
                </dd>
              </div>
            ))}
          </dl>
          <p>
            {t(l, [
              "입력 확인만 완료했습니다. 권한·서명·정합성 검증과 업무 실행은 미연결입니다.",
              "Input reviewed only. Authority, signature and integrity verification and execution are not connected.",
              "تمت مراجعة المدخلات فقط. التحقق من الصلاحيات والتوقيع والسلامة والتنفيذ غير متصل.",
            ])}
          </p>
        </div>
      )}
    </Panel>
  );
}
export function Steps({
  locale: l,
  steps,
}: {
  locale: Props["locale"];
  steps: readonly Words[];
}) {
  return (
    <ol className="ops-stages">
      {steps.map((s, i) => (
        <li key={i}>
          <span className="ops-step-number" aria-hidden="true">
            {i + 1}
          </span>
          <div>
            <strong>{t(l, s)}</strong>
            <p>{t(l, ["근거 대기", "Awaiting evidence", "بانتظار الأدلة"])}</p>
          </div>
        </li>
      ))}
    </ol>
  );
}
export function Ledger({
  locale: l,
  title,
  columns,
  reason,
}: {
  locale: Props["locale"];
  title: Words;
  columns: readonly Words[];
  reason: Words;
}) {
  return (
    <Panel title={t(l, title)}>
      <div className="ops-table-state" role="status">
        <strong>{t(l, unconnected)}</strong>
        <p>{t(l, reason)}</p>
      </div>
      <TableRegion locale={l} label={t(l, title)} className="ops-table">
        <table>
          <caption>
            {t(l, [
              "조회 서비스 미연결 · 전체 건수 미확인",
              "Reader not connected · total unknown",
              "خدمة القراءة غير متصلة · الإجمالي غير معروف",
            ])}
          </caption>
          <thead>
            <tr>
              {columns.map((c, i) => (
                <th key={i} scope="col">
                  {t(l, c)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody />
        </table>
      </TableRegion>
    </Panel>
  );
}
