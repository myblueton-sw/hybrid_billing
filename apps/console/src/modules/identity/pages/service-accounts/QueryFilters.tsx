import { useEffect, useState, type FormEvent } from "react";
import { AppliedFilters, type Chip } from "../../../../shared/list/ListParts";
import { lc } from "../../../../shared/list/list-copy";
import type { Locale } from "../../../../shared/types";
import { v } from "./live-copy";
import { qcopy as t, type QueryTerm } from "./query-copy";
import type {
  AccountQuery,
  CredentialQuery,
} from "../../../../service/service-query-types";

type Change = (extra: Record<string, string | null>, push?: boolean) => void;
const labels: Record<string, QueryTerm> = {
  q: "search",
  organization: "organization",
  owner: "owner",
  audience: "audience",
  scope: "scopes",
  expiry: "expiry",
  state: "state",
  declaration: "declaration",
  collection: "collection",
  service: "service",
};
export function filterValue(l: Locale, value: string) {
  const terms: Record<string, [string, string, string]> = {
    active: ["Active", "활성", "نشط"],
    suspended: ["Suspended", "정지", "معلّق"],
    retired: ["Retired", "퇴역", "منتهي الخدمة"],
    revoked: ["Revoked", "폐기됨", "ملغى"],
    expired: ["Expired", "만료됨", "منتهي الصلاحية"],
    expiring: ["Expiring soon", "만료 임박", "قرب انتهاء الصلاحية"],
    valid: ["Unexpired", "미만료", "غير منتهي الصلاحية"],
    none: ["No credentials", "자격 없음", "لا توجد بيانات اعتماد"],
    recorded: ["Recorded", "기록됨", "مسجّل"],
    missing: ["Not recorded", "기록 없음", "غير مسجّل"],
    uncollected: ["Awaiting collection", "수령 대기", "بانتظار الاستلام"],
  };
  return value === "retired"
    ? v.stRetired(l)
    : value === "revoked"
      ? v.cRevoked(l)
      : value === "all"
        ? t(l, "all")
        : (terms[value]?.[l === "ko" ? 1 : l === "ar" ? 2 : 0] ?? value);
}
export function QueryFilters({
  l,
  area,
  query,
  onParams,
}: {
  l: Locale;
  area: "accounts" | "credentials";
  query: AccountQuery | CredentialQuery;
  onParams: Change;
}) {
  const prefix = area === "accounts" ? "svc_a_" : "svc_c_";
  const id = area === "accounts" ? "svc-a-filters" : "svc-c-filters";
  const fields =
    area === "accounts"
      ? ["q", "organization", "owner", "audience", "scope", "expiry", "state", "declaration"]
      : ["q", "service", "state", "collection"];
  const applied = query as Record<string, unknown>;
  const [draft, setDraft] = useState<Record<string, string>>(() =>
    Object.fromEntries(fields.map((key) => [key, String(applied[key] ?? "")])),
  );
  const [open, setOpen] = useState(false);
  const appliedKey = JSON.stringify([applied.workspace, ...fields.map((key) => String(applied[key] ?? ""))]);
  useEffect(() => {
    setDraft(Object.fromEntries(fields.map((key) => [key, String(applied[key] ?? "")])));
  }, [appliedKey]);
  const options: Record<string, string[]> = {
    expiry: ["all", "none", "expired", "expiring", "valid"],
    state:
      area === "accounts"
        ? ["all", "active", "suspended", "retired"]
        : ["all", "active", "expired", "revoked"],
    declaration: ["all", "recorded", "missing"],
    collection: ["all", "uncollected"],
  };
  const chips: Chip[] = fields
    .map((key) => [key, String(applied[key] ?? "")] as const)
    .filter(([, value]) => value !== "" && value !== "all")
    .map(([key, value]) => ({
      key,
      label: t(l, labels[key]),
      value: options[key] ? filterValue(l, value) : value,
      mono: !options[key] && key !== "q",
    }));
  const submit = (event: FormEvent) => {
    event.preventDefault();
    const extra = Object.fromEntries(
      fields.map((key) => [prefix + key, draft[key] || null]),
    );
    setOpen(false);
    onParams({ ...extra, [prefix + "after"]: null }, true);
  };
  const clear = () =>
    onParams(
      Object.fromEntries([...fields, "after"].map((key) => [prefix + key, null])),
      true,
    );
  const focusTarget = () => {
    const toggle = document.querySelector<HTMLElement>(`#${id} .list-filters-toggle`);
    return toggle && toggle.offsetParent ? toggle : document.getElementById(id + "-title");
  };
  return (
    <>
      <section
        id={id}
        className="svc-query-filters list-filters"
        data-open={open ? "true" : "false"}
        aria-labelledby={id + "-title"}
      >
        <h3 id={id + "-title"} className="list-filters-title" tabIndex={-1}>
          {lc.filters(l)}
        </h3>
        <button
          type="button"
          className="live-secondary list-filters-toggle"
          aria-expanded={open}
          aria-controls={id + "-form"}
          onClick={() => setOpen(!open)}
        >
          {lc.toggle(l, chips.length)}
        </button>
        <form id={id + "-form"} className="list-filter-form" onSubmit={submit} noValidate>
          <div className="list-filter-grid">
            {fields.map((key) => (
              <div className="service-field" key={key}>
                <label htmlFor={id + "-" + key}>{t(l, labels[key])}</label>
                {options[key] ? (
                  <select
                    id={id + "-" + key}
                    className="live-select"
                    value={draft[key] || "all"}
                    onChange={(e) => setDraft({ ...draft, [key]: e.target.value })}
                  >
                    {options[key].map((value) => (
                      <option key={value} value={value}>
                        {filterValue(l, value)}
                      </option>
                    ))}
                  </select>
                ) : (
                  <input
                    id={id + "-" + key}
                    value={draft[key]}
                    dir={key === "q" ? undefined : "ltr"}
                    autoComplete="off"
                    onChange={(e) => {
                      if (e.target.value.length <= (key === "q" ? 8192 : 128))
                        setDraft({ ...draft, [key]: e.target.value });
                    }}
                  />
                )}
              </div>
            ))}
          </div>
          <div className="live-buttons">
            <button className="live-secondary" type="submit">
              {t(l, "apply")}
            </button>
            <button className="live-secondary" type="button" onClick={clear}>
              {t(l, "reset")}
            </button>
          </div>
        </form>
      </section>
      <AppliedFilters
        l={l}
        id={id + "-applied"}
        chips={chips}
        onRemove={(key) => onParams({ [prefix + key]: null, ...(key === "audience" ? { [prefix + "scope"]: null } : {}), [prefix + "after"]: null }, true)}
        onClear={clear}
        fallback={focusTarget}
      />
    </>
  );
}
