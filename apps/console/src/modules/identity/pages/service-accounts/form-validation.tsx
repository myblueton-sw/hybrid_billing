import { useLayoutEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { c } from "../../../../service/copy";
import { w } from "../users/live-copy";
import { v } from "./live-copy";
import { r } from "./rotate-copy";
import { qcopy as t } from "./query-copy";
import { utf8Length } from "../../../../service/helpers";

export const fieldIds = {
  name: "svc-name",
  owner: "svc-owner",
  purpose: "svc-purpose",
  organization: "svc-organization",
  scopes: "svc-scopes",
  days: "svc-days",
  overlap: "svc-overlap",
  reason: "live-reason",
} as const;
export type FormField = keyof typeof fieldIds;
export type FieldErrors = Partial<Record<FormField, string>>;
export function fieldAttributes(
  field: FormField,
  error?: string,
  hint?: string,
) {
  return {
    id: fieldIds[field],
    "aria-invalid": error ? (true as const) : undefined,
    "aria-describedby":
      [hint, error ? fieldIds[field] + "-error" : ""]
        .filter(Boolean)
        .join(" ") || undefined,
  };
}
export function FieldMessage({
  field,
  error,
}: {
  field: FormField;
  error?: string;
}) {
  return error ? (
    <p className="service-field-error" id={fieldIds[field] + "-error"}>
      {error}
    </p>
  ) : null;
}
export const requiredField = (l: Locale) =>
  l === "en"
    ? "Complete this field."
    : l === "ko"
      ? "이 항목을 입력하거나 선택하세요."
      : "أكمل هذا الحقل.";
export const requiredScopes = (l: Locale) =>
  l === "en"
    ? "Add at least one declared scope."
    : l === "ko"
      ? "선언된 범위를 하나 이상 추가하세요."
      : "أضف نطاقاً مُعلناً واحداً على الأقل.";
export const fieldBytes = (l: Locale, limit: number) =>
  l === "en"
    ? `This value exceeds the current limit of ${limit} bytes.`
    : l === "ko"
      ? `현재 한도인 ${limit}바이트를 초과했습니다.`
      : `تتجاوز هذه القيمة الحد الحالي البالغ ${limit} بايت.`;
export const scopeCount = (l: Locale, limit: number) =>
  l === "en"
    ? `Keep at most ${limit} declared scopes.`
    : l === "ko"
      ? `선언된 범위는 최대 ${limit}개까지 선택하세요.`
      : `احتفظ بما لا يزيد على ${limit} من النطاقات المُعلنة.`;
export const requestBounds = (l: Locale) =>
  l === "en"
    ? "This request exceeds the current request limits. Review the field values."
    : l === "ko"
      ? "현재 요청 한도를 초과했습니다. 입력값을 검토하세요."
      : "يتجاوز هذا الطلب حدود الطلب الحالية. راجع قيم الحقول.";
export const rationaleError = (l: Locale, value: string, limit: number) =>
  !value.trim()
    ? w.enterReason(l)
    : utf8Length(value) > limit
      ? fieldBytes(l, limit)
      : "";

export function useFormValidation() {
  const [errors, setErrors] = useState<FieldErrors>({});
  const [problem, setProblem] = useState("");
  const [attempt, setAttempt] = useState(0);
  return {
    errors,
    problem,
    attempt,
    setProblem,
    fail(next: FieldErrors, message = "") {
      setErrors(next);
      setProblem(message);
      setAttempt((value) => value + 1);
    },
    clear(...fields: FormField[]) {
      setErrors((prior) => {
        const next = { ...prior };
        for (const field of fields) delete next[field];
        return next;
      });
      setProblem("");
    },
    reset() {
      setErrors({});
      setProblem("");
    },
  };
}
export function ValidationSummary({
  l,
  validation,
}: {
  l: Locale;
  validation: ReturnType<typeof useFormValidation>;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const { errors, problem, attempt } = validation;
  const entries = Object.entries(errors).filter(([, message]) => !!message) as [
    FormField,
    string,
  ][];
  const labels = {
    name: v.name(l),
    owner: t(l, "owner"),
    purpose: t(l, "purpose"),
    organization: t(l, "organization"),
    scopes: t(l, "scopes"),
    days: v.validFor(l),
    overlap: r.overlap(l),
    reason: w.reason(l),
  };
  useLayoutEffect(() => {
    if (attempt && (entries.length || problem)) ref.current?.focus();
  }, [attempt]);
  if (!entries.length && !problem) return null;
  return (
    <div
      id="svc-form-errors"
      className="service-summary"
      role="alert"
      tabIndex={-1}
      ref={ref}
    >
      <h2>{c.check(l)}</h2>
      {problem && <p>{problem}</p>}
      {!!entries.length && (
        <ul>
          {entries.map(([field, message]) => (
            <li key={field}>
              <a
                href={"#" + fieldIds[field]}
                onClick={(event) => {
                  event.preventDefault();
                  const target = event.currentTarget
                    .closest("form")
                    ?.querySelector<HTMLElement>("#" + fieldIds[field]);
                  const control =
                    target?.tagName === "FIELDSET"
                      ? target.querySelector<HTMLSelectElement>("select")
                      : null;
                  (control ?? target)?.focus();
                }}
              >
                {labels[field]}: {message}
              </a>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
