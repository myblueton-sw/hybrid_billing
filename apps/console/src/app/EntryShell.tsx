import { useCallback, useState } from "react";
import type { Locale, Mode, Route } from "../shared/types";
import { text as t } from "../shared/copy";
import { entryUrl } from "../modules/identity/entry-ui";
import ReviewPanel from "../shared/ReviewPanel";
import HeaderSelect from "./HeaderSelect";
import logo from "../../../../packages/design-system/assets/logo/hb-lockup-color.svg";
export default function EntryShell({
  route,
  planningId,
  locale: l,
  mode,
  profile,
}: {
  route: Route;
  planningId?: string;
  locale: Locale;
  mode: Mode;
  profile: string;
}) {
  const [view, setView] = useState({ state: "", items: [] as string[] });
  const onState = useCallback(
    (state: string, items: string[]) =>
      setView((old) =>
        old.state === state && old.items.join("|") === items.join("|")
          ? old
          : { state, items },
      ),
    [],
  );
  const Screen = route.component;
  return (
    <div className="entry-shell">
      <a
        className="skip"
        href="#main"
        onClick={(e) => {
          e.preventDefault();
          document.getElementById("main")?.focus();
        }}
      >
        {t(l, "Skip to content", "본문 바로가기", "انتقل إلى المحتوى")}
      </a>
      <header className="entry-header">
        <a className="brand" dir="ltr" href={entryUrl("entry-start", l, mode)}>
          <img
            className="brand-logo"
            src={logo}
            width="155"
            height="32"
            alt="Hybrid Billing"
          />
        </a>
        <HeaderSelect
          label={t(l, "Language", "언어", "اللغة")}
          value={l}
          options={[
            { value: "en", label: "English (US)" },
            { value: "ko", label: "한국어" },
            { value: "ar", label: "العربية (الإمارات)" },
          ]}
          onChange={(lang) => {
            location.hash = entryUrl(route.id, lang as Locale, mode);
          }}
        />
      </header>
      <main
        id="main"
        className={`entry-main ${["entry-start", "entry-login"].includes(route.id) ? "entry-designed" : ""}`}
        tabIndex={-1}
      >
        {!["entry-start", "entry-login"].includes(route.id) && (
          <a className="entry-back" href={entryUrl("entry-start", l, mode)}>
            {t(l, "Back to start", "시작 화면으로", "العودة إلى البداية")}
          </a>
        )}
        <div id="content">
          <Screen
            locale={l}
            mode={mode}
            profile={profile}
            extensionIds={[]}
            contributions={[]}
            onState={onState}
          />
        </div>
      </main>
      <footer className="entry-footer">
        <span className="entry-footer-brand">Hybrid Billing</span>
        <span className="entry-environment">
          {t(
            l,
            "Interactive preview · identity service not connected",
            "인터랙티브 시안 · 인증 서비스 미연결",
            "معاينة تفاعلية · خدمة الهوية غير متصلة",
          )}
        </span>
        <span>
          {t(l, "Workspace access", "업무 공간 접속", "الوصول لمساحة العمل")}
        </span>
        <details className="entry-review-tools">
          <summary>
            {t(l, "Confirm screen review", "검수 확인", "تأكيد مراجعة الشاشة")}
          </summary>
          <ReviewPanel
            screen={planningId ?? route.id}
            locale={l}
            context={JSON.stringify({
              profile,
              mode,
              extensions: [],
              state: view.state,
            })}
            items={view.items}
          />
        </details>
      </footer>
    </div>
  );
}
