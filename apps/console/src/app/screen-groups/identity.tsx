import type { Props } from "../../modules/identity/screen-ui/types";
import { say } from "../../modules/identity/screen-ui/types";
import { allowedView, contextKey } from "../../modules/identity/screen-ui/model";
import Entry from "../../modules/identity/pages/access-entry/Screen";
import Users from "../../modules/identity/pages/users/Screen";
import Authentication from "../../modules/identity/pages/authentication/Screen";
import Access from "../../modules/identity/pages/access/Screen";
import ServiceAccounts from "../../modules/identity/pages/service-accounts/Screen";
import MySecurity from "../../modules/identity/pages/my-security/Screen";
import LiveUsers from "../../modules/identity/pages/users/Live";
import LivePolicy from "../../modules/identity/pages/authentication/Live";
import LiveAccess from "../../modules/identity/pages/access/Live";
import LiveServices from "../../modules/identity/pages/service-accounts/Live";
import LiveMySecurity from "../../modules/identity/pages/my-security/Live";
import { useService } from "../../service/context";
import "../../modules/identity/screen-ui/security.css";
export type { Props } from "../../modules/identity/screen-ui/types";
function toParams(extra: Record<string, string | null>, push = false) {
  const [path, query = ""] = location.hash.replace(/^#/, "").split("?");
  const q = new URLSearchParams(query);
  for (const [k, v] of Object.entries(extra)) {
    if (v === null) q.delete(k);
    else q.set(k, v);
  }
  const next = path + "?" + q.toString();
  if (location.hash.replace(/^#/, "") === next) return;
  if (push) history.pushState(history.state, "", "#" + next);
  else history.replaceState(history.state, "", "#" + next);
  dispatchEvent(new HashChangeEvent("hashchange"));
}

export default function RemainingSecurity(props: Props) {
  const Screen = props.screenId.startsWith("L")
    ? Entry
    : props.screenId === "M01"
      ? Users
      : props.screenId === "M02"
        ? Authentication
        : props.screenId === "M03"
          ? Access
          : props.screenId === "M04"
            ? ServiceAccounts
            : MySecurity;
  const service = useService();
  if (!allowedView(props.screenId, props.view))
    return (
      <p role="alert">
        {say(props.locale, [
          "This security view is unavailable.",
          "이 보안 보기는 제공하지 않습니다.",
          "هذا العرض الأمني غير متاح.",
        ])}
      </p>
    );
  if (service && props.screenId === "M01")
    return (
      <div
        className="remaining-security"
        data-security-screen="M01"
        data-security-view={props.view}
        data-security-phase="live"
        dir={props.locale === "ar" ? "rtl" : "ltr"}
        lang={props.locale}
      >
        <LiveUsers
          locale={props.locale}
          view={props.view}
          params={props.params}
          onParams={toParams}
          session={service.session}
          onSignOut={service.signOut}
          onView={(view) => {
            const [path, query = ""] = location.hash.replace(/^#/, "").split("?");
            const q = new URLSearchParams(query);
            q.set("view", view);
            location.hash = path + "?" + q.toString();
          }}
        />
      </div>
    );
  if (service && props.screenId === "M04")
    return (
      <div
        className="remaining-security"
        data-security-screen="M04"
        data-security-view={props.view}
        data-security-phase="live"
        dir={props.locale === "ar" ? "rtl" : "ltr"}
        lang={props.locale}
      >
        <LiveServices
          params={props.params}
          onParams={toParams}
          locale={props.locale}
          view={props.view}
          session={service.session}
          onSignOut={service.signOut}
          onView={(view) => {
            const [path, query = ""] = location.hash.replace(/^#/, "").split("?");
            const q = new URLSearchParams(query);
            q.set("view", view);
            location.hash = path + "?" + q.toString();
          }}
        />
      </div>
    );
  if (service && props.screenId === "M05")
    return (
      <div
        className="remaining-security"
        data-security-screen="M05"
        data-security-view={props.view}
        data-security-phase="live"
        dir={props.locale === "ar" ? "rtl" : "ltr"}
        lang={props.locale}
      >
        <LiveMySecurity
          params={props.params}
          onParams={(extra) => toParams(extra,true)}
          locale={props.locale}
          view={props.view}
          session={service.session}
          onSignOut={service.signOut}
          onView={(view) => {
            const [path, query = ""] = location.hash.replace(/^#/, "").split("?");
            const q = new URLSearchParams(query);
            q.set("view", view);
            location.hash = path + "?" + q.toString();
          }}
        />
      </div>
    );
  if (service && props.screenId === "M03")
    return (
      <div
        className="remaining-security"
        data-security-screen="M03"
        data-security-view={props.view}
        data-security-phase="live"
        dir={props.locale === "ar" ? "rtl" : "ltr"}
        lang={props.locale}
      >
        <LiveAccess
          locale={props.locale}
          view={props.view}
          session={service.session}
          onSignOut={service.signOut}
          onView={(view) => {
            const [path, query = ""] = location.hash.replace(/^#/, "").split("?");
            const q = new URLSearchParams(query);
            q.set("view", view);
            location.hash = path + "?" + q.toString();
          }}
        />
      </div>
    );
  if (service && props.screenId === "M02")
    return (
      <div
        className="remaining-security"
        data-security-screen="M02"
        data-security-view={props.view}
        data-security-phase="live"
        dir={props.locale === "ar" ? "rtl" : "ltr"}
        lang={props.locale}
      >
        <LivePolicy
          locale={props.locale}
          view={props.view}
          session={service.session}
          onSignOut={service.signOut}
          onView={(view) => {
            const [path, query = ""] = location.hash.replace(/^#/, "").split("?");
            const q = new URLSearchParams(query);
            q.set("view", view);
            location.hash = path + "?" + q.toString();
          }}
          onUsers={() => {
            const q = new URLSearchParams(location.hash.split("?")[1] ?? "");
            const next = new URLSearchParams({ view: "users" });
            if (q.get("lang")) next.set("lang", q.get("lang")!);
            location.hash = "/users?" + next.toString();
          }}
        />
      </div>
    );
  return (
    <div
      className="remaining-security"
      data-security-screen={props.screenId}
      data-security-view={props.view}
      data-security-phase="unconnected"
      dir={props.locale === "ar" ? "rtl" : "ltr"}
      lang={props.locale}
    >
      <Screen
        key={contextKey(props.screenId, props.view, props.params)}
        {...props}
      />
    </div>
  );
}
