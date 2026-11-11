import type { Locale } from "../../../../shared/types";
import type { FocusRow } from "../../../sources/collected-focus/api";
import { identityText } from "./identity";
import { SplitId } from "./RowFacts";
export default function RelationEnd({
  locale: l,
  rows,
  id,
}: {
  locale: Locale;
  rows: FocusRow[];
  id: string;
}) {
  const row = rows.find((candidate) => candidate.id === id);
  return (
    <>
      {row && (
        <span className="infra-relation-identity">{identityText(l, row)}</span>
      )}
      <SplitId id={id} />
    </>
  );
}
