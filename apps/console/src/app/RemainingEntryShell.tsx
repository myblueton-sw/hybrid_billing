import "./remaining-shell.css";
import { useCallback, useState } from "react";
import type { Locale } from "../shared/types";
import HeaderSelect from "./HeaderSelect";
import ReviewPanel from "../shared/ReviewPanel";
import RemainingScreens from "./RemainingScreens";
import { planningScreens } from "../planning/registry";
import ScreenTabLabel from "./ScreenTabLabel";
import { local, titles } from "../planning/copy";
import logo from "../../../../packages/design-system/assets/logo/hb-lockup-color.svg";
export default function RemainingEntryShell({
  screenId,
  locale: l,
  params,
  view,
  onDirty,
}: {
  screenId: string;
  locale: Locale;
  params: URLSearchParams;
  view: string;
  onDirty: (dirty: boolean) => void;
}) {
  const [state, setState] = useState({ state: "{}", items: [] as string[] });
  const onState = useCallback(
    (state: string, items: string[]) =>
      setState((old) =>
        old.state === state && old.items.join("|") === items.join("|")
          ? old
          : { state, items },
      ),
    [],
  );
  return (
    <div className="entry-shell remaining-entry-shell">
      <header className="entry-header">
        <a className="brand" dir="ltr" href={`#/login?lang=${l}`}>
          <img
            className="brand-logo"
            src={logo}
            width="155"
            height="32"
            alt="Hybrid Billing"
          />
        </a>
        <HeaderSelect
          label={local(l, ["언어", "Language", "اللغة"])}
          value={l}
          options={[
            { value: "ko", label: "한국어" },
            { value: "en", label: "English (US)" },
            { value: "ar", label: "العربية (الإمارات)" },
          ]}
          onChange={(lang) => {
            const p = new URLSearchParams(params);
            p.set("lang", lang);
            location.hash =
              "#" +
              ({ L01: "/login", L02: "/activate", L03: "/recover" }[
                screenId as "L01" | "L02" | "L03"
              ] ?? "/login") +
              "?" +
              p.toString();
          }}
        />
      </header>
      <main id="main" className="entry-main" tabIndex={-1}>
        <nav
          className="planning-tabs"
          aria-label={local(l, ["화면 보기", "Screen views", "عروض الشاشة"])}
        >
          {planningScreens
            .find((s) => s.id === screenId)
            ?.tabs.map((tab) => (
              <a
                key={tab.key}
                aria-current={view === tab.key ? "page" : undefined}
                href={`${"#" + ({ L01: "/login", L02: "/activate", L03: "/recover" }[screenId as "L01" | "L02" | "L03"] ?? "/login")}?${new URLSearchParams({ lang: l, view: tab.key })}`}
              >
                <ScreenTabLabel id={screenId} locale={l} tab={tab} />
              </a>
            ))}
        </nav>
        <div id="content">
          <RemainingScreens
            screenId={screenId}
            locale={l}
            params={params}
            view={view}
            audience="unauthenticated"
            onState={onState}
            onDirty={onDirty}
          />
        </div>
      </main>
      <footer className="entry-footer">
        <span>Hybrid Billing</span>
        <span>
          {local(l, [
            "인증 서비스 미연결",
            "Identity service not connected",
            "خدمة الهوية غير متصلة",
          ])}
        </span>
        <ReviewPanel
          screen={screenId}
          locale={l}
          context={JSON.stringify({
            mode: "unconnected",
            audience: "unauthenticated",
            view,
            state: state.state,
          })}
          items={state.items}
        />
        <nav>
          {["L01", "L02", "L03"].map((id, i) => (
            <a
              key={id}
              href={`#${["/login", "/activate", "/recover"][i]}?lang=${l}`}
            >
              {local(l, titles[id])}
            </a>
          ))}
        </nav>
      </footer>
    </div>
  );
}
