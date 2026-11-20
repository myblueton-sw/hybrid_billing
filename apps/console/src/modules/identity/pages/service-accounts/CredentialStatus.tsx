import type { Locale } from "../../../../shared/types";
import type { ServiceCredential } from "../../../../service/service-query-types";
import { v } from "./live-copy";
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
