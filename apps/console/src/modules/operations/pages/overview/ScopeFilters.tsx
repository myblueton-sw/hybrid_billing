import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { months, sources, organizations, type Currency } from "./model";
import { organizationName } from "./copy";
function Choices({
  title,
  options,
  selected,
  onChange,
  locale: l,
}: {
  title: string;
  options: { id: string; label: string }[];
  selected: string[];
  onChange: (ids: string[]) => void;
  locale: Locale;
}) {
  const all = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (all.current)
      all.current.indeterminate =
        selected.length > 0 && selected.length < options.length;
  }, [selected.length, options.length]);
  return (
    <details className="unified-picker">
      <summary>
        {title} ·{" "}
        <bdi dir="ltr">
          {selected.length}/{options.length}
        </bdi>
      </summary>
      <fieldset>
        <legend className="sr-only">{title}</legend>
        <label>
          <input
            ref={all}
            type="checkbox"
            checked={selected.length === options.length}
            onChange={(e) =>
              onChange(e.target.checked ? options.map((o) => o.id) : [])
            }
          />
          {t(l, "All", "전체", "الكل")}
        </label>
        {options.map((o) => (
          <label key={o.id}>
            <input
              type="checkbox"
              value={o.id}
              checked={selected.includes(o.id)}
              onChange={(e) =>
                onChange(
                  e.target.checked
                    ? [...selected, o.id]
                    : selected.filter((id) => id !== o.id),
                )
              }
            />
            {o.label}
          </label>
        ))}
      </fieldset>
    </details>
  );
}
export default function ScopeFilters({
  locale: l,
  period,
  currency,
  selectedSources,
  selectedOrganizations,
  change,
}: {
  locale: Locale;
  period: string;
  currency: Currency;
  selectedSources: string[];
  selectedOrganizations: string[];
  change: (values: Record<string, string>) => void;
}) {
  return (
    <section
      className="unified-filters"
      aria-label={t(
        l,
        "Integrated scope",
        "통합 조회 범위",
        "نطاق العرض الموحد",
      )}
    >
      <label>
        {t(l, "Reporting month", "보고월", "شهر التقرير")}
        <select
          id="overview-period"
          value={period}
          onChange={(e) => change({ "overview-period": e.target.value })}
        >
          {!months.includes(period) && (
            <option value={period}>{period || "—"}</option>
          )}
          {months.map((m) => (
            <option key={m}>{m}</option>
          ))}
        </select>
      </label>
      <label>
        {t(
          l,
          "Chart currency · no FX",
          "차트 통화 · 환산 없음",
          "عملة الرسم · دون تحويل",
        )}
        <select
          id="overview-currency"
          value={currency}
          onChange={(e) => change({ "ov-currency": e.target.value })}
        >
          <option>USD</option>
          <option>KRW</option>
        </select>
      </label>
      <Choices
        title={t(l, "Sources", "원천", "المصادر")}
        options={sources.map((id) => ({ id, label: id }))}
        selected={selectedSources}
        onChange={(ids) => change({ "ov-sources": ids.join("|") })}
        locale={l}
      />
      <Choices
        title={t(l, "Organizations", "조직", "المؤسسات")}
        options={organizations.map((id) => ({
          id,
          label: organizationName(l, id),
        }))}
        selected={selectedOrganizations}
        onChange={(ids) => change({ "ov-organizations": ids.join("|") })}
        locale={l}
      />
      <button
        id="overview-reset"
        onClick={() =>
          change({
            "ov-sources": sources.join("|"),
            "ov-organizations": organizations.join("|"),
          })
        }
      >
        {t(l, "Show all", "전체 통합 보기", "عرض الكل")}
      </button>
    </section>
  );
}
