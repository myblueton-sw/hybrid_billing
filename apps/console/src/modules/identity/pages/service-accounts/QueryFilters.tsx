import { useState, type FormEvent } from "react";
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
  limit: "size",
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
  const fields =
    area === "accounts"
      ? [
          "q",
          "organization",
          "owner",
          "audience",
          "scope",
          "expiry",
          "state",
          "declaration",
          "limit",
        ]
      : ["q", "service", "state", "collection", "limit"];
  const [draft, setDraft] = useState<Record<string, string>>(() =>
    Object.fromEntries(
      fields.map((key) => [
        key,
        String((query as Record<string, unknown>)[key] ?? ""),
      ]),
    ),
  );
  const options: Record<string, string[]> = {
    expiry: ["all", "none", "expired", "expiring", "valid"],
    state:
      area === "accounts"
        ? ["all", "active", "suspended", "retired"]
        : ["all", "active", "expired", "revoked"],
    declaration: ["all", "recorded", "missing"],
    collection: ["all", "uncollected"],
  };
  const submit = (event: FormEvent) => {
    event.preventDefault();
    const extra = Object.fromEntries(
      fields.map((key) => [prefix + key, draft[key] || null]),
    );
    onParams({ ...extra, [prefix + "after"]: null }, true);
  };
  return (
    <details className="svc-query-filters">
      <summary>{t(l, "apply")}</summary>
      <form onSubmit={submit} className="live-sections">
        {fields.map((key) => (
          <label key={key}>
            {t(l, labels[key])}
            {options[key] ? (
              <select
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
                value={draft[key]}
                dir={key === "q" ? undefined : "ltr"}
                inputMode={key === "limit" ? "numeric" : undefined}
                onChange={(e) => {
                  if (e.target.value.length <= (key === "q" ? 8192 : 128))
                    setDraft({ ...draft, [key]: e.target.value });
                }}
              />
            )}
          </label>
        ))}
        <div className="live-buttons">
          <button className="live-secondary" type="submit">
            {t(l, "apply")}
          </button>
          <button
            className="live-secondary"
            type="button"
            onClick={() =>
              onParams(
                Object.fromEntries(
                  [...fields, "after"].map((key) => [prefix + key, null]),
                ),
                true,
              )
            }
          >
            {t(l, "reset")}
          </button>
        </div>
      </form>
    </details>
  );
}
