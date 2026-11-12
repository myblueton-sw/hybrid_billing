import type { Locale } from "../../../../shared/types";
import {
  currencies,
  modes,
  dateDays,
  sourceAmount,
  type Draft,
  type Prepared,
} from "./model";
import { t, label, modeLabel, missing, issueLabel, errorFields } from "./copy";
import { fieldId } from "./values";
import { fieldLabel } from "../../../sources/collected-focus/labels";
import { SourceAmount } from "../../../sources/collected-focus/SourceValues";
export default function RuleEditor({
  locale: l,
  draft: d,
  source,
  errors,
  update,
}: {
  locale: Locale;
  draft: Draft;
  source?: Prepared;
  errors: string[];
  update: (d: Draft) => void;
}) {
  const quoteDay = dateDays(d.quoteDate),
    reviewDay = dateDays(d.reviewDate),
    age = quoteDay !== null && reviewDay !== null ? reviewDay - quoteDay : null;
  function edit(k: keyof Draft, value: string) {
    update({ ...d, [k]: value });
  }
  function field(k: keyof Draft, help?: string, suffix?: string) {
    const id = fieldId(k),
      codes = errors.filter((e) => errorFields[e] === id.slice(3));
    return (
      <label className="b05-field" htmlFor={id}>
        {label(l, k)}
        <span className="b05-input">
          <input
            id={id}
            type={k === "quoteDate" || k === "reviewDate" ? "date" : "text"}
            dir={
              ["name", "provider", "reference"].includes(k) ? undefined : "ltr"
            }
            inputMode={
              [
                "name",
                "provider",
                "reference",
                "quoteDate",
                "reviewDate",
              ].includes(k)
                ? undefined
                : k === "maxAge"
                  ? "numeric"
                  : "decimal"
            }
            value={d[k]}
            maxLength={
              k === "name"
                ? 120
                : ["provider", "reference"].includes(k)
                  ? 300
                  : 40
            }
            onChange={(e) => edit(k, e.target.value)}
            aria-invalid={codes.length > 0}
            aria-describedby={`${id}-help ${id}-error`}
          />
          {suffix && <bdi dir="ltr">{suffix}</bdi>}
        </span>
        <small id={`${id}-help`}>{help}</small>
        <span className="b05-field-error" id={`${id}-error`}>
          {codes.map((c) => (
            <span key={c}>{issueLabel(l, c)}</span>
          ))}
        </span>
      </label>
    );
  }
  function select(k: keyof Draft, options: [string, string][]) {
    const id = fieldId(k),
      codes = errors.filter((e) => errorFields[e] === id.slice(3));
    return (
      <label className="b05-field" htmlFor={id}>
        {label(l, k)}
        <select
          id={id}
          value={d[k]}
          onChange={(e) => edit(k, e.target.value)}
          aria-invalid={codes.length > 0}
          aria-describedby={`${id}-error`}
        >
          <option value="">
            {t(l, "Choose explicitly", "직접 선택", "اختر صراحةً")}
          </option>
          {options.map(([v, text]) => (
            <option key={v} value={v}>
              {text}
            </option>
          ))}
        </select>
        <span className="b05-field-error" id={`${id}-error`}>
          {codes.map((c) => (
            <span key={c}>{issueLabel(l, c)}</span>
          ))}
        </span>
      </label>
    );
  }
  return (
    <section className="b05-editor">
      <h2>
        {t(
          l,
          "Conversion rule inputs",
          "환산 규칙 입력",
          "مدخلات قاعدة التحويل",
        )}
      </h2>
      {field("name")}
      <fieldset>
        <legend>
          {t(
            l,
            "1. Declare the source cost",
            "1. 원천 비용 선언",
            "1. إقرار تكلفة المصدر",
          )}
        </legend>
        <p className="b05-source-cost" data-fx-editor-cost>
          {source ? (
            <SourceAmount
              value={sourceAmount(source)}
              currency={source.row.currency}
              note
              locale={l}
            />
          ) : (
            t(l, "Select a source row", "원천 행을 선택하세요", "اختر صف مصدر")
          )}
          {source && (
            <small>{fieldLabel(l, source.data.selection.basis)}</small>
          )}
        </p>
        {select("treatment", [
          [
            "unconverted",
            t(
              l,
              "Treat as unconverted",
              "미환산 금액으로 검토",
              "اعتباره غير محوّل",
            ),
          ],
          [
            "unknown",
            t(
              l,
              "Converted / unknown",
              "이미 환산 / 확인 불가",
              "محوّل / غير معروف",
            ),
          ],
        ])}
        {d.treatment && (
          <p className="b05-meta" data-fx-declaration="true">
            {d.treatment === "unconverted"
              ? t(
                  l,
                  "User declaration: treat this amount as unconverted source currency.",
                  "사용자 선언: 이 금액을 미환산 원천 통화 금액으로 검토합니다.",
                  "إقرار المستخدم: اعتبار هذا المبلغ بعملة المصدر غير محوّل.",
                )
              : t(
                  l,
                  "Already converted or conversion history unknown: conversion preview is blocked.",
                  "이미 환산되었거나 환산 이력을 알 수 없어 환산 미리보기를 차단합니다.",
                  "المبلغ محوّل أو سجل التحويل غير معروف: معاينة التحويل محظورة.",
                )}
          </p>
        )}
        <p className="b05-meta">
          {t(
            l,
            "User declaration only; supplier conversion history is unconnected. Source cost is not the B04 selling amount.",
            "사용자 선언이며 공급사 환산 이력은 미연결입니다. 원천 비용은 B04 판매 금액이 아닙니다.",
            "إقرار مستخدم فقط؛ سجل تحويل المزوّد غير متصل. تكلفة المصدر ليست مبلغ البيع في B04.",
          )}
        </p>
      </fieldset>
      <fieldset>
        <legend>
          {t(
            l,
            "2. Choose the target currency",
            "2. 대상 통화 선택",
            "2. اختيار العملة المستهدفة",
          )}
        </legend>
        {select(
          "target",
          currencies.map((c) => [c, c]),
        )}
        <p className="b05-meta">
          {t(
            l,
            "Review allowlist, not a live supported-currency catalog. Source and target must differ.",
            "검토용 허용 목록이며 실제 지원 통화 목록이 아닙니다. 원천과 대상 통화는 달라야 합니다.",
            "قائمة سماح للمراجعة وليست قائمة عملات مدعومة حية. يجب اختلاف المصدر والهدف.",
          )}
        </p>
      </fieldset>
      <fieldset>
        <legend>
          {t(
            l,
            "3. Enter manual quote evidence",
            "3. 수동 환율 근거 입력",
            "3. إدخال أدلة سعر الصرف يدوياً",
          )}
        </legend>
        <div className="b05-equation-fields">
          {field(
            "quoteUnit",
            t(
              l,
              "Positive source-currency quantity",
              "원천 통화의 양수 수량",
              "كمية موجبة بعملة المصدر",
            ),
            source?.row.currency ?? "—",
          )}
          <span aria-hidden="true">=</span>
          {field(
            "rate",
            t(
              l,
              "Positive target-currency quote",
              "대상 통화의 양수 환율",
              "سعر موجب بالعملة المستهدفة",
            ),
            d.target || "—",
          )}
        </div>
        <p className="b05-quote">
          <bdi dir="ltr">
            {d.quoteUnit || "—"} {source?.row.currency ?? "—"} = {d.rate || "—"}{" "}
            {d.target || "—"}
          </bdi>
        </p>
        <div className="b05-fields">
          {field(
            "provider",
            t(
              l,
              "Public/synthetic reference only",
              "공개·합성 참조만 입력",
              "مرجع عام أو تجريبي فقط",
            ),
          )}
          {field(
            "reference",
            t(
              l,
              "No credentials or real customer information",
              "인증정보·실제 고객 정보 입력 금지",
              "دون بيانات اعتماد أو معلومات عملاء حقيقية",
            ),
          )}
          {field("quoteDate")}
          {field("reviewDate")}
          {field(
            "maxAge",
            t(
              l,
              "Integer days · UTC calendar dates are user declared",
              "정수 일수 · UTC 날짜는 사용자 선언",
              "أيام صحيحة · تواريخ UTC بإقرار المستخدم",
            ),
          )}
        </div>
        {age !== null && (
          <p className="b05-meta">
            {t(
              l,
              "Declared age / maximum days",
              "선언 경과일 / 허용 일수",
              "العمر المصرّح / الحد الأقصى للأيام",
            )}
            :{" "}
            <bdi dir="ltr">
              {age} / {d.maxAge || "—"}
            </bdi>
          </p>
        )}
      </fieldset>
      <fieldset>
        <legend>
          {t(
            l,
            "4. Apply markup and target-currency fee",
            "4. 가산·대상 통화 수수료",
            "4. تطبيق الزيادة والرسوم بالعملة المستهدفة",
          )}
        </legend>
        <div className="b05-fields">
          {field(
            "markup",
            t(
              l,
              "Enter zero explicitly · up to 4 decimal places",
              "0도 직접 입력 · 소수점 4자리까지",
              "أدخل الصفر صراحةً · حتى 4 منازل عشرية",
            ),
            "%",
          )}
          {field(
            "fee",
            t(
              l,
              "Nonnegative · enter zero explicitly",
              "0 이상 · 0도 직접 입력",
              "غير سالب · أدخل الصفر صراحةً",
            ),
            d.target || "—",
          )}
        </div>
      </fieldset>
      <fieldset>
        <legend>
          {t(
            l,
            "5. Apply one rounding quantum",
            "5. 금액 처리 단위 한 번 적용",
            "5. تطبيق وحدة تقريب واحدة",
          )}
        </legend>
        {field(
          "quantum",
          t(
            l,
            "Positive quantum. Currency decimal places are not assumed.",
            "양수 단위. 통화 소수 자릿수를 가정하지 않습니다.",
            "وحدة موجبة. لا يُفترض عدد المنازل العشرية للعملة.",
          ),
          d.target || "—",
        )}
        <fieldset
          tabIndex={-1}
          className="b05-modes b05-field"
          id="fx-mode"
          role="radiogroup"
          aria-label={label(l, "mode")}
          aria-describedby="fx-mode-error"
        >
          <legend>{label(l, "mode")}</legend>
          {modes.map((m) => (
            <label key={m}>
              <input
                type="radio"
                name="fx-mode"
                value={m}
                checked={d.mode === m}
                onChange={() => edit("mode", m)}
              />
              <span>{modeLabel(l, m)}</span>
            </label>
          ))}
        </fieldset>
        <p className="b05-field-error" id="fx-mode-error">
          {errors.includes("mode_invalid") ? issueLabel(l, "mode_invalid") : ""}
        </p>
      </fieldset>
    </section>
  );
}
