import TableRegion from "../../TableRegion";
import { useEffect, useRef, useState, type ReactNode } from "react";
import type { Locale } from "../../types";
import { fields, text, type Props, type Words, type WorkId } from "./contracts";
export const words = {
  unknown: ["미확인", "Unknown", "غير معروف"],
  unconnected: [
    "업무 서비스 미연결",
    "Business service not connected",
    "خدمة الأعمال غير متصلة",
  ],
  draft: [
    "브라우저 검토 초안",
    "Browser review draft",
    "مسودة مراجعة في المتصفح",
  ],
  review: ["입력 검토", "Review inputs", "مراجعة المدخلات"],
  clear: ["입력 지우기", "Clear inputs", "مسح المدخلات"],
  local: [
    "이 입력은 현재 화면에만 유지됩니다. 저장·실행 요청을 보내지 않습니다.",
    "These inputs remain in this screen. No save or execution request is sent.",
    "تبقى المدخلات في هذه الشاشة فقط. لا يتم إرسال طلب حفظ أو تنفيذ.",
  ],
  invalid: [
    "필수 입력과 형식을 확인해 주세요.",
    "Check required inputs and their format.",
    "تحقق من الحقول المطلوبة وتنسيقها.",
  ],
  ready: [
    "입력 형식을 검토했습니다. 서버 검증·저장·업무 실행은 미연결입니다.",
    "Input format reviewed. Server validation, saving and execution are not connected.",
    "تمت مراجعة تنسيق المدخلات. التحقق والحفظ والتنفيذ على الخادم غير متصل.",
  ],
} satisfies Record<string, Words>;
export function Boundary({ l }: { l: Locale }) {
  return (
    <aside className="work-boundary" aria-label={text(l, words.unconnected)}>
      <strong>{text(l, words.unconnected)}</strong>
      <p>
        {text(l, [
          "허용 범위와 최신 근거를 받지 못했습니다. 건수·금액·상태는 미확인입니다.",
          "Authorized scope and current evidence are unavailable. Counts, amounts and statuses are unknown.",
          "نطاق الوصول والأدلة الحالية غير متاحين. الأعداد والمبالغ والحالات غير معروفة.",
        ])}
      </p>
      <dl>
        <div>
          <dt>
            {text(l, [
              "인증된 범위",
              "Authenticated scope",
              "النطاق المصادق عليه",
            ])}
          </dt>
          <dd>{text(l, words.unknown)}</dd>
        </div>
        <div>
          <dt>
            {text(l, ["근거 기준시각", "Evidence as of", "وقت مرجع الأدلة"])}
          </dt>
          <dd>{text(l, words.unknown)}</dd>
        </div>
      </dl>
    </aside>
  );
}
export function Panel({
  title,
  children,
  id,
}: {
  title: string;
  children: ReactNode;
  id?: string;
}) {
  return (
    <section className="work-panel" data-item-id={id}>
      <h2>{title}</h2>
      {children}
    </section>
  );
}
export function Facts({ l, labels }: { l: Locale; labels: Words[] }) {
  return (
    <dl className="work-facts">
      {labels.map((label) => (
        <div key={label[1]}>
          <dt>{text(l, label)}</dt>
          <dd>{text(l, words.unknown)}</dd>
        </div>
      ))}
    </dl>
  );
}
export function EmptyList({
  l,
  columns,
  reason,
}: {
  l: Locale;
  columns: Words[];
  reason: Words;
}) {
  return (
    <div className="work-result">
      <p className="remaining-table-state" role="status">
        {text(l, reason)}
      </p>
      <TableRegion
        locale={l}
        className="work-table-scroll"
        label={text(l, [
          "업무 목록 열",
          "Business list columns",
          "أعمدة قائمة الأعمال",
        ])}
      >
        <table>
          <thead>
            <tr>
              {columns.map((c) => (
                <th key={c[1]} scope="col">
                  {text(l, c)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody />
        </table>
      </TableRegion>
      <p className="work-meta">
        {text(l, [
          "전체 건수 미확인 · 아직 조회된 페이지가 없습니다.",
          "Exact total unknown · no page has been retrieved.",
          "الإجمالي الدقيق غير معروف · لم يتم جلب أي صفحة.",
        ])}
      </p>
    </div>
  );
}
export function Label({
  label,
  children,
  hint,
}: {
  label: string;
  children: ReactNode;
  hint?: string;
}) {
  return (
    <label className="work-field">
      <span>{label}</span>
      {children}
      {hint && <small>{hint}</small>}
    </label>
  );
}
export function ReviewForm({
  l,
  children,
  onClear,
  onReviewed,
  validate,
  id,
}: {
  l: Locale;
  children: ReactNode;
  onClear: () => void;
  onReviewed?: () => void;
  validate?: (form: HTMLFormElement) => void;
  id: string;
}) {
  const [phase, setPhase] = useState<"draft" | "invalid" | "reviewed">("draft");
  const status = useRef<HTMLParagraphElement>(null);
  const [errors, setErrors] = useState<{ id: string; label: string }[]>([]);
  return (
    <form
      data-item-id={id}
      className="work-form"
      noValidate
      onChange={() => {
        setPhase("draft");
        setErrors([]);
      }}
      onSubmit={(e) => {
        e.preventDefault();
        const form = e.currentTarget;
        validate?.(form);
        if (!form.checkValidity()) {
          const invalid = Array.from(
            form.querySelectorAll<
              HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement
            >("input,textarea,select"),
          ).filter((el) => !el.validity.valid);
          setErrors(
            invalid.map((el, i) => {
              el.id ||= `${id}-invalid-${i}`;
              return {
                id: el.id,
                label:
                  el.closest("label")?.querySelector("span")?.textContent ??
                  text(l, words.invalid),
              };
            }),
          );
          setPhase("invalid");
          invalid[0]?.focus();
        } else {
          setPhase("reviewed");
          onReviewed?.();
          requestAnimationFrame(() => status.current?.focus());
        }
      }}
    >
      {errors.length > 0 && (
        <div role="alert" className="work-error-summary">
          <strong>{text(l, words.invalid)}</strong>
          <ul>
            {errors.map((error) => (
              <li key={error.id}>
                <a
                  href={`#${error.id}`}
                  onClick={(e) => {
                    e.preventDefault();
                    document.getElementById(error.id)?.focus();
                  }}
                >
                  {error.label}
                </a>
              </li>
            ))}
          </ul>
        </div>
      )}
      <p className="work-meta">{text(l, words.local)}</p>
      {children}
      <div className="work-actions">
        <button type="submit">{text(l, words.review)}</button>
        <button
          type="button"
          onClick={() => {
            onClear();
            setErrors([]);
            setPhase("draft");
          }}
        >
          {text(l, words.clear)}
        </button>
      </div>
      <p tabIndex={-1} ref={status} role="status" className="work-feedback">
        {phase === "invalid"
          ? text(l, words.invalid)
          : phase === "reviewed"
            ? text(l, words.ready)
            : text(l, words.draft)}
      </p>
    </form>
  );
}
export function useReview(p: Props, dirty: boolean, phase = "unconnected") {
  const id = p.screenId as WorkId;
  const map = fields[id] as Record<string, readonly string[]>;
  const fieldIds =
    id === "U03" && p.audience !== "admin"
      ? [
          "public_document_reference",
          "own_inquiry_reference",
          "read_state",
          "delivery_state",
        ]
      : (map[p.view] ?? []);
  const state = JSON.stringify({
    screen: id,
    view: p.view,
    phase,
    draft: dirty,
    audience: p.audience ?? "restricted",
    source: "unconnected",
    authority: "unknown",
  });
  useEffect(
    () =>
      p.onState(state, [
        ...fieldIds.map((f) => `${id}:field:${f}`),
        `${id}:state:unconnected`,
        `${id}:action:review-local-input`,
        `${id}:action:clear-local-input`,
      ]),
    [state, p.onState, id, p.view],
  );
  useEffect(() => {
    p.onDirty(dirty);
    return () => p.onDirty(false);
  }, [dirty, p.onDirty]);
}
export const refInput = {
  maxLength: 128,
  pattern: "[A-Za-z0-9][A-Za-z0-9._:\\-]{0,127}",
  autoComplete: "off",
  spellCheck: false,
} as const;
