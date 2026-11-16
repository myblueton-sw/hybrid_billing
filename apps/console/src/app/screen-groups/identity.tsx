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
import { useService } from "../../service/context";
import "../../modules/identity/screen-ui/security.css";
export type { Props } from "../../modules/identity/screen-ui/types";
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
