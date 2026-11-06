import { Organizations, OrganizationChange } from "./Organizations";
import Contracts from "./ContractProfile";
import Inbox from "./Inbox";
import Findings from "./Findings";
import Imports from "./Imports";
import Assistant from "./Assistant";
import type { Props } from "./contracts";
import "./screen.css";
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
