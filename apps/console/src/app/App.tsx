import { useCallback, useEffect, useState } from "react";
import { assemble } from "./assemble";
import { getProfile } from "../profiles";
import { sources } from "../modules/sources/module";
import { designReview } from "../modules/design-review/module";
import { evidenceGuide } from "../extensions/evidence-guide";
import { text as t } from "../shared/copy";
import { useLocation, routeUrl } from "../shared/navigation";
import type { Locale, Mode } from "../shared/types";
import ReviewPanel from "../shared/ReviewPanel";
import logo from "../../../../packages/design-system/assets/logo/hb-lockup-color.svg";
export const modules = [designReview, sources];
export const extensions = [evidenceGuide];
function Workspace({
  id,
  l,
  mode,
  profile,
}: {
  id: string;
  l: Locale;
  mode: Mode;
  profile: string;
}) {
  const composition = assemble(modules, extensions, getProfile(profile));
  const route = composition.routes.find((r) => r.id === id);
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
  const extensionIds = composition.extensions.map((e) => e.id);
  if (!route)
    return (
      <section className="empty">
        <h1>
          {t(
            l,
            "Screen not implemented",
            "아직 구현되지 않은 화면입니다",
            "الشاشة غير منفذة بعد",
          )}
        </h1>
        <a href={routeUrl("contents")}>
          {t(l, "Implementation workspace", "구현 작업실", "مساحة التنفيذ")}
        </a>
      </section>
    );
  const Screen = route.component;
  return (
    <>
      <div className="screen-heading">
        <span className="eyebrow">
          <bdi>{id}</bdi>
        </span>
        <h1>{route.title[l]}</h1>
      </div>
      <Screen
        locale={l}
        mode={mode}
        profile={profile}
        extensionIds={extensionIds}
        contributions={composition.extensions}
        onState={onState}
      />
      <ReviewPanel
        screen={id}
        locale={l}
        context={JSON.stringify({
          profile,
          mode,
          extensions: extensionIds,
          state: view.state,
        })}
        items={view.items}
      />
    </>
  );
}
export default function App() {
  const hash = useLocation(),
    [routePart, query] = hash.slice(1).split("?"),
    params = new URLSearchParams(query);
  const id = routePart || "contents",
    l: Locale = ["en", "ar"].includes(params.get("lang") ?? "")
      ? (params.get("lang") as Locale)
      : "ko";
  const profile = params.get("profile") ?? "standard",
    mode: Mode =
      params.get("mode") === "synthetic" ? "synthetic" : "unconnected";
  useEffect(() => {
    document.documentElement.lang = l;
    document.documentElement.dir = l === "ar" ? "rtl" : "ltr";
  }, [l]);
  function update(values: Record<string, string>) {
    location.hash = routeUrl(id, {
      ...values,
      ...(values.mode ? { q: "", page: "1" } : {}),
    });
  }
  let valid = true;
  try {
    assemble(modules, extensions, getProfile(profile));
  } catch {
    valid = false;
  }
  return (
    <>
      <a
        className="skip"
        href="#main-content"
        onClick={(e) => {
          e.preventDefault();
          document.getElementById("main-content")?.focus();
        }}
      >
        {t(l, "Skip to content", "본문으로 이동", "انتقل إلى المحتوى")}
      </a>
      <header>
        <div className="brand">
          <img src={logo} alt="Fusion AI" />
        </div>
        <span className="product">Hybrid Billing</span>
        <label>
          {t(l, "Language", "언어", "اللغة")}
          <select
            aria-label="Language"
            value={l}
            onChange={(e) => update({ lang: e.target.value })}
          >
            <option value="ko">한국어</option>
            <option value="en">English</option>
            <option value="ar">العربية</option>
          </select>
        </label>
      </header>
      <div className="workspace">
        <aside className="rail">
          <p>{t(l, "BILLING WORKSPACE", "청구 업무 공간", "مساحة الفوترة")}</p>
          <nav>
            {modules
              .flatMap((m) => m.routes)
              .map((r) => (
                <a
                  key={r.id}
                  href={routeUrl(r.id)}
                  aria-current={id === r.id ? "page" : undefined}
                >
                  {r.title[l]}
                </a>
              ))}
          </nav>
          <div className="rail-note">
            {t(
              l,
              "First implementation increment. No production services connected.",
              "첫 구현 단계입니다. 운영 서비스는 연결되지 않았습니다.",
              "المرحلة الأولى من التنفيذ. خدمات الإنتاج غير متصلة.",
            )}
          </div>
        </aside>
        <main id="main-content" tabIndex={-1}>
          <section className="context">
            <label>
              {t(
                l,
                "Demo installation profile",
                "예제 설치 구성",
                "ملف التثبيت التجريبي",
              )}
              <select
                id="profile"
                value={valid ? profile : ""}
                onChange={(e) => update({ profile: e.target.value })}
              >
                {!valid && (
                  <option value="">
                    {t(
                      l,
                      "Invalid configuration",
                      "잘못된 구성",
                      "تكوين غير صالح",
                    )}
                  </option>
                )}
                <option value="standard">Standard</option>
                <option value="evidence">Evidence</option>
              </select>
            </label>
            <label>
              {t(l, "Data mode", "데이터 모드", "نمط البيانات")}
              <select
                id="mode"
                value={mode}
                onChange={(e) => update({ mode: e.target.value })}
              >
                <option value="unconnected">
                  {t(
                    l,
                    "Backend not connected",
                    "백엔드 미연결",
                    "الخدمة الخلفية غير متصلة",
                  )}
                </option>
                <option value="synthetic">
                  {t(l, "Synthetic demo", "합성 데이터 체험", "بيانات تجريبية")}
                </option>
              </select>
            </label>
            <span className="tag">
              {mode === "synthetic"
                ? t(l, "SYNTHETIC DATA", "합성 데이터", "بيانات تجريبية")
                : t(l, "NOT CONNECTED", "미연결", "غير متصل")}
            </span>
          </section>
          {valid ? (
            <Workspace
              key={`${id}:${l}:${profile}:${mode}`}
              id={id}
              l={l}
              mode={mode}
              profile={profile}
            />
          ) : (
            <section role="alert" className="empty">
              <h1>
                {t(
                  l,
                  "Installation profile rejected",
                  "설치 구성을 사용할 수 없습니다",
                  "تم رفض ملف التثبيت",
                )}
              </h1>
              <p>
                {t(
                  l,
                  "Choose a known profile. No unknown modules were loaded.",
                  "확인된 구성을 선택하세요. 알 수 없는 모듈은 불러오지 않았습니다.",
                  "اختر ملفًا معروفًا. لم يتم تحميل وحدات غير معروفة.",
                )}
              </p>
            </section>
          )}
        </main>
      </div>
    </>
  );
}
