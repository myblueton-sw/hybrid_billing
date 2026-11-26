import type { Locale } from "../../../../shared/types";
import type { ServiceCredential } from "../../../../service/service-query-types";
import { v } from "./live-copy";
import { r } from "./rotate-copy";
import { utcTime } from "./live-model";
import { filterValue } from "./QueryFilters";

export function CredentialStatus({
  l,
  row,
}: {
  l: Locale;
  row: ServiceCredential;
}) {
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
      {row.rotation === "ending" && <p className="live-rotation">{r.ending(l, "\u2068" + utcTime(l, row.not_after).replace(/[\u061c\u200e\u200f]/g, "") + "\u2069")}</p>}
      {row.replaced_by && <p>{r.replacedBy(l)[0]}<span className="live-mono live-nowrap">{"\u2066" + row.replaced_by + "\u2069"}</span>{r.replacedBy(l)[1]}</p>}
      {row.replaces && <p>{r.replaces(l)[0]}<span className="live-mono live-nowrap">{"\u2066" + row.replaces + "\u2069"}</span>{r.replaces(l)[1]}</p>}
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
      <p>{v.from(l, "\u2068" + row.not_before + "\u2069")}</p>
      <p>{v.until(l, "\u2068" + row.not_after + "\u2069")}</p>
    </>
  );
}
