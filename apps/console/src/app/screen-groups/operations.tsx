import { fieldsByScreen } from "../../shared/domain-screen-ui/operations/contracts";
import { useEffect } from "react";
import type { Props } from "../../shared/domain-screen-ui/operations/types";
import Installation from "../../modules/administration/pages/installation/Screen";
import Integrations from "../../modules/administration/pages/integrations/Screen";
import Recovery from "../../modules/administration/pages/recovery/Screen";
import Retention from "../../modules/administration/pages/retention/Screen";
import ReleaseEvidence from "../../modules/administration/pages/release-evidence/Screen";
import Demo from "../../modules/administration/pages/demo/Screen";
import LiveLicenses from "../../modules/administration/pages/licenses/Live";
import Licenses from "../../modules/administration/pages/licenses/Screen";
import Portal from "../../modules/customer-portal/pages/workspace/Screen";
import LiveInstallation from "../../modules/administration/pages/installation/Live";
import LiveIntegrations from "../../modules/administration/pages/integrations/Live";
import LiveRecovery from "../../modules/administration/pages/recovery/Live";
import LiveRetention from "../../modules/administration/pages/retention/Live";
import LiveReleaseEvidence from "../../modules/administration/pages/release-evidence/Live";
import { useService } from "../../service/context";
import "../../shared/domain-screen-ui/operations/screen.css";
export default function Operations(p: Props) {
  const service = useService();
  const live = !!service && (p.screenId === "P01" || p.screenId === "P02" || p.screenId === "P03" || p.screenId === "P04" || p.screenId === "P05" || p.screenId === "P07");
  const context = JSON.stringify([
    p.screenId,
    p.view,
    p.locale,
    p.params.get("scope"),
    p.params.get("customer"),
    p.params.get("period"),
    p.params.get("le"),
    p.params.get("ccy"),
    p.params.get("contract"),
    p.params.get("asof"),
  ]);
  useEffect(() => {
    if (live) return;
    p.onState(
      JSON.stringify({
        screen: p.screenId,
        view: p.view,
        source: "unconnected",
      }),
      [
        `${p.screenId}:data:${p.view}`,
        `${p.screenId}:state:unconnected`,
        ...(fieldsByScreen[p.screenId] ?? []).map(
          (f) => `${p.screenId}:field:${f}`,
        ),
      ],
    );
  }, [p.screenId, p.view, p.onState, live]);
  if (live && service && p.screenId === "P07") return <div className="ops-screen" data-screen-id="P07" data-source-state="live" dir={p.locale === "ar" ? "rtl" : "ltr"} lang={p.locale}><LiveLicenses {...p} key={JSON.stringify([service.session.workspace,service.session.user,service.session.csrf])} onSignOut={service.signOut}/></div>;
  if (live && service && p.screenId === "P02")
    return (
      <div className="work-domain remaining-security" data-screen-id="P02" data-source-state="live" dir={p.locale === "ar" ? "rtl" : "ltr"} lang={p.locale} key={`P02:${p.locale}`}>
        <LiveIntegrations locale={p.locale} view={p.view} params={p.params} session={service.session} onSignOut={service.signOut} />
      </div>
    );
  if (live && service && p.screenId === "P03")
    return (
      <div className="work-domain remaining-security" data-screen-id="P03" data-source-state="live" dir={p.locale === "ar" ? "rtl" : "ltr"} lang={p.locale} key={`P03:${p.locale}`}>
        <LiveRecovery locale={p.locale} view={p.view} params={p.params} session={service.session} onSignOut={service.signOut} />
      </div>
    );
  if (live && service && p.screenId === "P04")
    return (
      <div className="work-domain remaining-security" data-screen-id="P04" data-source-state="live" dir={p.locale === "ar" ? "rtl" : "ltr"} lang={p.locale} key={`P04:${p.locale}`}>
        <LiveRetention locale={p.locale} view={p.view} params={p.params} session={service.session} onSignOut={service.signOut} />
      </div>
    );
  if (live && service && p.screenId === "P05")
    return (
      <div className="work-domain remaining-security" data-screen-id="P05" data-source-state="live" dir={p.locale === "ar" ? "rtl" : "ltr"} lang={p.locale} key={`P05:${p.locale}`}>
        <LiveReleaseEvidence locale={p.locale} view={p.view} params={p.params} session={service.session} onSignOut={service.signOut} />
      </div>
    );
  if (live && service && p.screenId === "P01")
    return (
      <div className="work-domain remaining-security" data-screen-id="P01" data-source-state="live" dir={p.locale === "ar" ? "rtl" : "ltr"} lang={p.locale} key={`P01:${p.locale}`}>
        <LiveInstallation locale={p.locale} view={p.view} params={p.params} session={service.session} onSignOut={service.signOut} />
      </div>
    );
  const Screen =
    p.screenId === "P01"
      ? Installation
      : p.screenId === "P02"
        ? Integrations
        : p.screenId === "P03"
          ? Recovery
          : p.screenId === "P04"
            ? Retention
            : p.screenId === "P05"
              ? ReleaseEvidence
              : p.screenId === "P06"
                ? Demo
                : p.screenId === "P07"
                  ? Licenses
                  : Portal;
  return (
    <div className="ops-screen" data-screen-id={p.screenId} key={context}>
      <Screen {...p} />
    </div>
  );
}
