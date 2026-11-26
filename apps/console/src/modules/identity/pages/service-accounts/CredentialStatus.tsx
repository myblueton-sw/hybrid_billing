import type { Locale } from "../../../../shared/types";
import type { ServiceCredential } from "../../../../service/service-query-types";
import { v } from "./live-copy";
import { r } from "./rotate-copy";
import { shownTime } from "./live-model";
import { filterValue } from "./QueryFilters";

export function CredentialStatus({
  l,
  row,
  select,
}: {
  l: Locale;
  row: ServiceCredential;
  select?: (id: string) => void;
}) {
  const other = (id: string) => select
    ? <button type="button" className="live-id-link" onClick={() => select(id)}>{"\u2066" + id + "\u2069"}</button>
    : <span className="live-mono live-nowrap">{"\u2066" + id + "\u2069"}</span>;
  return (
    <>
      <span>{filterValue(l, row.current_state)}</span>
      {row.current_state === "active" && !row.usable && (
        <p>
          {row.service_state === "retired"
            ? v.cRevokeRequired(l)
            : v.cOnHold(l)}
        </p>
      )}
      <p>
        {v.colService(l)}: <span>{filterValue(l, row.service_state)}</span>
      </p>
      {row.rotation === "ending" && <p className="live-rotation">{r.ending(l, "\u2068" + shownTime(l, row.not_after) + "\u2069")}</p>}
      {row.replaced_by && <p>{r.replacedBy(l)[0]}{other(row.replaced_by)}{r.replacedBy(l)[1]}</p>}
      {row.replaces && <p>{r.replaces(l)[0]}{other(row.replaces)}{r.replaces(l)[1]}</p>}
    </>
  );
}

export function CredentialPeriod({
  l,
  row,
}: {
  l: Locale;
  row: ServiceCredential;
}) {
  return (
    <>
      <p>{v.from(l, "\u2068" + shownTime(l, row.not_before) + "\u2069")}</p>
      <p>{v.until(l, "\u2068" + shownTime(l, row.not_after) + "\u2069")}</p>
    </>
  );
}
