import {
  Organizations,
  OrganizationChange,
} from "../../modules/organizations/pages/organizations/Screen";
import Contracts from "../../modules/organizations/pages/contracts/Screen";
import Inbox from "../../modules/operations/pages/tasks/Screen";
import Findings from "../../modules/operations/pages/billing-findings/Screen";
import Imports from "../../modules/sources/pages/import-review/Screen";
import Assistant from "../../modules/assistant/pages/conversation/Screen";
import type { Props } from "../../shared/domain-screen-ui/work/contracts";
import "../../shared/domain-screen-ui/work/screen.css";
import LiveOrganizations from "../../modules/organizations/pages/organizations/Live";
import LiveChanges from "../../modules/organizations/pages/changes/Live";
import LiveContracts from "../../modules/organizations/pages/contracts/Live";
import { useService } from "../../service/context";

function toView(view: string, extra: Record<string, string> = {}) {
  const [path, query = ""] = location.hash.replace(/^#/, "").split("?");
  const q = new URLSearchParams(query);
  q.set("view", view);
  for (const [k, v] of Object.entries(extra)) q.set(k, v);
  location.hash = path + "?" + q.toString();
}

function toParams(extra: Record<string, string | null>) {
  const [path, query = ""] = location.hash.replace(/^#/, "").split("?");
  const q = new URLSearchParams(query);
  for (const [k, v] of Object.entries(extra)) {
    if (v === null) q.delete(k);
    else q.set(k, v);
  }
  const next = path + "?" + q.toString();
  if (location.hash.replace(/^#/, "") === next) return;
  history.replaceState(history.state, "", "#" + next);
  dispatchEvent(new HashChangeEvent("hashchange"));
}

export default function WorkRemaining(p: Props) {
  const service = useService();
  if (service && p.screenId === "O01")
    return (
      <div className="work-domain remaining-security" data-screen-id="O01" data-source-state="live" dir={p.locale === "ar" ? "rtl" : "ltr"} lang={p.locale} key={`O01:${p.locale}`}>
        <LiveOrganizations locale={p.locale} view={p.view} params={p.params} session={service.session} onSignOut={service.signOut} onView={toView} onParams={toParams} />
      </div>
    );
  if (service && p.screenId === "O02")
    return (
      <div className="work-domain remaining-security" data-screen-id="O02" data-source-state="live" dir={p.locale === "ar" ? "rtl" : "ltr"} lang={p.locale} key={`O02:${p.locale}`}>
        <LiveChanges locale={p.locale} view={p.view} params={p.params} session={service.session} onSignOut={service.signOut} onView={toView} onParams={toParams} />
      </div>
    );
  if (service && p.screenId === "O03")
    return (
      <div className="work-domain remaining-security" data-screen-id="O03" data-source-state="live" dir={p.locale === "ar" ? "rtl" : "ltr"} lang={p.locale} key={`O03:${p.locale}`}>
        <LiveContracts locale={p.locale} view={p.view} params={p.params} session={service.session} onSignOut={service.signOut} onView={toView} onParams={toParams} />
      </div>
    );
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
      data-source-state={
        p.screenId === "S03" ? "receipt-reader" : "unconnected"
      }
      dir={p.locale === "ar" ? "rtl" : "ltr"}
      key={
        p.screenId === "S03"
          ? `${p.screenId}:${p.locale}:${p.view}:${p.audience ?? "restricted"}`
          : `${p.screenId}:${p.locale}:${p.view}:${p.audience ?? "restricted"}:${p.params.toString()}`
      }
    >
      {leaf}
    </div>
  );
}
