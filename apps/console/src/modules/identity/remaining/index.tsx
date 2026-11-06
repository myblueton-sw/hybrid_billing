import type { Props } from "./types";
import { say } from "./types";
import { allowedView, contextKey } from "./model";
import Entry from "./Entry";
import Users from "./Users";
import Authentication from "./Authentication";
import Access from "./Access";
import ServiceAccounts from "./ServiceAccounts";
import MySecurity from "./MySecurity";
import "./security.css";
export type { Props } from "./types";
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
