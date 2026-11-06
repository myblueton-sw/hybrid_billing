import { Organizations, OrganizationChange } from "../../modules/organizations/pages/organizations/Screen";
import Contracts from "../../modules/organizations/pages/contracts/Screen";
import Inbox from "../../modules/operations/pages/tasks/Screen";
import Findings from "../../modules/operations/pages/billing-findings/Screen";
import Imports from "../../modules/sources/pages/import-review/Screen";
import Assistant from "../../modules/assistant/pages/conversation/Screen";
import type { Props } from "../../shared/domain-screen-ui/work/contracts";
import "../../shared/domain-screen-ui/work/screen.css";
export default function WorkRemaining(p: Props) {
  const leaf =
    p.screenId === "O01" ? (
      <Organizations {...p} />
    ) : p.screenId === "O02" ? (
      <OrganizationChange {...p} />
    ) : p.screenId === "O03" ? (
      <Contracts {...p} />
    ) : p.screenId === "U03" ? (
      <Inbox {...p} />
    ) : p.screenId === "U04" ? (
      <Findings {...p} />
    ) : p.screenId === "S03" ? (
      <Imports {...p} />
    ) : p.screenId === "A01" ? (
      <Assistant {...p} />
    ) : null;
  return (
    <div
      className="work-domain"
      data-screen-id={p.screenId}
      data-source-state="unconnected"
      dir={p.locale === "ar" ? "rtl" : "ltr"}
      key={`${p.screenId}:${p.locale}:${p.view}:${p.audience ?? "restricted"}:${p.params.toString()}`}
    >
      {leaf}
    </div>
  );
}
