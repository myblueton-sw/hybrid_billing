import EntryShell from "./EntryShell";
import { identity } from "../modules/identity/module";
import { operations } from "../modules/operations/module";
import { organizations } from "../modules/organizations/module";
import { issuance } from "../modules/issuance/module";
import { onboarding } from "../modules/onboarding/module";
import HeaderSelect from "./HeaderSelect";
import AccountPicker from "./AccountPicker";
import { useCallback, useEffect, useRef, useState } from "react";
import { assemble } from "./assemble";
import { getProfile } from "../profiles";
import { sources } from "../modules/sources/module";
import { designReview } from "../modules/design-review/module";
import { evidenceGuide } from "../extensions/evidence-guide";
import { text as t } from "../shared/copy";
import { useLocation, routeUrl } from "../shared/navigation";
import type { Locale, Mode } from "../shared/types";
import ReviewPanel from "../shared/ReviewPanel";
import shellCopy from "../shared/shell-copy.json";
import { screens, groups, domains } from "../shared/catalog";
import logo from "../../../../packages/design-system/assets/logo/hb-lockup-color.svg";
import chevron from "../../../../packages/design-system/assets/icons/chevron-down.svg";
export const modules = [
  identity,
  designReview,
  sources,
  operations,
  organizations,
  issuance,
  onboarding,
];
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
  const screen = screens.find((s) => s.id === id);
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
  if (!screen)
    return (
      <section className="panel">
        <h1>{t(l, "Unknown screen", "알 수 없는 화면", "شاشة غير معروفة")}</h1>
      </section>
    );
  const Screen = route?.component;
  const reviewPanel = (
    <ReviewPanel
      screen={id}
      locale={l}
      context={JSON.stringify({
        profile,
        account:
          new URLSearchParams(location.hash.split("?")[1]).get("scope") ??
          "sample-a",
        mode,
        extensions: extensionIds,
        state: view.state,
      })}
      items={view.items}
    />
  );
  return (
    <>
      <div className="crumb">
        {id === "overview"
          ? screen.title[l]
          : groups.find((g) => g.id === screen.group)?.label[l]}{" "}
        / <bdi>{id}</bdi>
      </div>
      <div className="heading">
        <div>
          <h1>{screen.title[l]}</h1>
          <p>{screen.description[l]}</p>
        </div>
        <span className="tag">{shellCopy.draft[l]}</span>
      </div>
      {id !== "overview" && (
        <div className="meta">
          <span>
            {shellCopy.account[l]}:{" "}
            <bdi>
              {new URLSearchParams(location.hash.split("?")[1]).get("scope") ===
              "sample-b"
                ? "ACC-000002"
                : "ACC-000001"}
            </bdi>
          </span>
          <span>
            {shellCopy.snapshot[l]}: <bdi>DEMO-S1</bdi>
          </span>
          <span>{shellCopy.notConnected[l]}</span>
        </div>
      )}
      {id !== "overview" && reviewPanel}
      <div id="content">
        {Screen ? (
          <Screen
            locale={l}
            mode={mode}
            profile={profile}
            extensionIds={extensionIds}
            contributions={composition.extensions}
            onState={onState}
          />
        ) : (
          <section className="panel">
            <h2>
              {t(
                l,
                "React implementation pending",
                "React 구현 대기",
                "تنفيذ React قيد الانتظار",
              )}
            </h2>
            <p>
              {t(
                l,
                "This screen retains its existing design identity. Its React implementation has not been completed.",
                "기존 시안의 화면 위치와 이름을 유지했습니다. 이 화면의 React 구현은 아직 완료되지 않았습니다.",
                "تحتفظ الشاشة بهويتها التصميمية الحالية. لم يكتمل تنفيذها بعد.",
              )}
            </p>
          </section>
        )}
      </div>
      {id === "overview" && reviewPanel}
      {id !== "overview" && (
        <section className="related-section">
          <h2>{shellCopy.related[l]}</h2>
          <div className="related">
            {screen.related
              .filter(
                (target) =>
                  screen.group !== "customer" ||
                  screens.find((s) => s.id === target)?.group === "customer",
              )
              .map((target) => (
                <a key={target} href={routeUrl(target)}>
                  {screens.find((s) => s.id === target)?.title[l]}
                </a>
              ))}
          </div>
        </section>
      )}
    </>
  );
}
export default function App() {
  const hash = useLocation(),
    [routePart, query] = hash.slice(1).split("?"),
    params = new URLSearchParams(query);
  const id = routePart === "/home" ? "overview" : routePart || "entry-start",
    l: Locale = ["en", "ar"].includes(params.get("lang") ?? "")
      ? (params.get("lang") as Locale)
      : "ko";
  const profile = params.get("profile") ?? "standard",
    mode: Mode =
      params.has("mode") && params.get("mode") !== "synthetic"
        ? "unconnected"
        : "synthetic";
  const [search, setSearch] = useState(""),
    [dialogKind, setDialogKind] = useState("");
  const dialog = useRef<HTMLDialogElement>(null),
    lastTrigger = useRef<HTMLElement | null>(null);
  const screen = screens.find((s) => s.id === id),
    customer = screen?.group === "customer";
  const domain = domains.find((d) =>
    d.groups.includes(screen?.group ?? "overview"),
  )!;
  const allowed = screens.filter((s) => !customer || s.group === "customer");
  const visible = allowed.filter((s) =>
    search
      ? `${s.title[l]} ${s.id} ${s.registryIds.join(" ")}`
          .toLowerCase()
          .includes(search.toLowerCase())
      : domain.groups.includes(s.group) && s.id !== "contents",
  );
  useEffect(() => {
    document.documentElement.lang = l;
    document.documentElement.dir = l === "ar" ? "rtl" : "ltr";
    document.title = `${screen?.title[l] ?? "Hybrid Billing"} · Hybrid Billing`;
  }, [l, id]);
  useEffect(() => {
    setSearch("");
    dialog.current?.close();
    document.getElementById("main")?.focus({ preventScroll: true });
  }, [id]);
  function update(values: Record<string, string>) {
    location.hash = routeUrl(id, {
      ...values,
      ...(values.mode ? { q: "", page: "1" } : {}),
    });
  }
  function open(kind: string) {
    lastTrigger.current = document.activeElement as HTMLElement;
    setDialogKind(kind);
    dialog.current?.showModal();
  }
  function close() {
    dialog.current?.close();
    setDialogKind("");
    lastTrigger.current?.focus();
  }
  let valid = true;
  try {
    assemble(modules, extensions, getProfile(profile));
  } catch {
    valid = false;
  }
  const entry = valid ? identity.routes.find((r) => r.id === id) : undefined;
  if (entry)
    return (
      <EntryShell
        key={`${id}:${l}:${mode}:${profile}`}
        route={entry}
        locale={l}
        mode={mode}
        profile={profile}
      />
    );
  return (
    <>
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
      <div className="preview">
        <span>{shellCopy.preview[l]}</span>
        <span>{shellCopy.review[l]}</span>
      </div>
      <header className="top">
        <a
          className="brand"
          dir="ltr"
          href={routeUrl(customer ? "core-customer-overview" : "overview")}
        >
          <img
            className="brand-logo"
            src={logo}
            width="155"
            height="32"
            alt="Hybrid Billing"
          />
        </a>
        <div className="top-actions">
          {!customer && (
            <a className="header-control" href={routeUrl("community")}>
              {t(l, "Contribute", "기여", "المساهمة")}
            </a>
          )}
          {id === "overview" ? (
            <label className="overview-workspace">
              <span className="sr-only">
                {t(
                  l,
                  "Example workspace",
                  "예제 업무 공간",
                  "مساحة عمل تجريبية",
                )}
              </span>
              <select
                id="overview-workspace"
                value={
                  params.get("scope") === "sample-b" ? "sample-b" : "sample-a"
                }
                onChange={(e) => update({ scope: e.target.value })}
              >
                <option value="sample-a">
                  {t(l, "Demo workspace A", "예제 공간 A", "مساحة تجريبية A")}
                </option>
                <option value="sample-b">
                  {t(l, "Demo workspace B", "예제 공간 B", "مساحة تجريبية B")}
                </option>
              </select>
            </label>
          ) : (
            <button
              id="choose-account"
              className="account-button"
              aria-label={`${shellCopy.account[l]} · ${params.get("scope") === "sample-b" ? "ACC-000002" : "ACC-000001"}`}
              disabled={customer}
              onClick={() => open("account")}
            >
              <span className="account-label">{shellCopy.account[l]} ·</span>
              <bdi>
                {params.get("scope") === "sample-b"
                  ? "ACC-000002"
                  : "ACC-000001"}
              </bdi>
              <img className="icon" src={chevron} alt="" />
            </button>
          )}
          <button id="open-jobs" onClick={() => open("jobs")}>
            {shellCopy.jobs[l]} <span className="tag">0</span>
          </button>
          <HeaderSelect
            label="Language"
            value={l}
            options={[
              { value: "en", label: "English (US)" },
              { value: "ar", label: "العربية (الإمارات)" },
              { value: "ko", label: "한국어" },
            ]}
            onChange={(lang) => update({ lang })}
          />
        </div>
      </header>
      <nav
        className="tnb"
        aria-label={t(l, "Workspace", "업무 영역", "مساحة العمل")}
      >
        {domains
          .filter((d) => !customer || d.id === "customer")
          .map((d) => (
            <button
              key={d.id}
              data-group={d.id}
              className={domain.id === d.id ? "active" : ""}
              aria-pressed={domain.id === d.id}
              onClick={() => {
                location.hash = routeUrl(
                  screens.find((s) => d.groups.includes(s.group))!.id,
                );
              }}
            >
              {d.label[l]}
            </button>
          ))}
      </nav>
      <div className="layout">
        <aside className="rail">
          <div className="rail-inner">
            <label>
              <span className="sr-only">{shellCopy.screens[l]}</span>
              <input
                id="screen-search"
                type="search"
                value={search}
                placeholder={shellCopy.screens[l]}
                onChange={(e) => setSearch(e.target.value)}
              />
            </label>
            <div className="rail-heading">
              <h2>{search ? shellCopy.screens[l] : domain.label[l]}</h2>
              <small>
                {allowed.length} {t(l, "screens", "전체 화면", "شاشة")}
              </small>
            </div>
            <nav
              id="screen-links"
              aria-label={t(l, "Screens", "전체 화면", "الشاشات")}
            >
              {visible.map((s) => (
                <a
                  key={s.id}
                  className={id === s.id ? "active" : ""}
                  href={routeUrl(s.id)}
                  aria-current={id === s.id ? "page" : undefined}
                >
                  {s.title[l]}
                </a>
              ))}
            </nav>
          </div>
        </aside>
        <main className="main" id="main" tabIndex={-1}>
          {valid ? (
            <Workspace
              key={`${id}:${l}:${profile}:${mode}:${params.get("scope") ?? "sample-a"}`}
              id={id}
              l={l}
              mode={mode}
              profile={profile}
            />
          ) : (
            <section className="panel" role="alert">
              {t(
                l,
                "Unknown installation profile",
                "알 수 없는 설치 구성",
                "ملف تثبيت غير معروف",
              )}
            </section>
          )}
          <details className="review-tools">
            <summary>
              {t(l, "Review tools", "검수 도구", "أدوات المراجعة")}
            </summary>
            <div className="toolbar">
              <a href={routeUrl("contents")}>
                {t(l, "Screen directory", "전체 화면 찾아보기", "دليل الشاشات")}
              </a>
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
                  {!valid && <option value="">—</option>}
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
                    {t(l, "Unconnected", "미연결", "غير متصل")}
                  </option>
                  <option value="synthetic">
                    {t(
                      l,
                      "Synthetic demo",
                      "합성 데이터 체험",
                      "بيانات تجريبية",
                    )}
                  </option>
                </select>
              </label>
            </div>
          </details>
        </main>
      </div>
      <dialog
        ref={dialog}
        onKeyDownCapture={(e) => {
          if (e.key === "Escape") {
            e.preventDefault();
            e.stopPropagation();
            close();
          }
        }}
        onCancel={close}
        aria-label={
          dialogKind === "account"
            ? t(l, "Choose account", "계정 선택", "اختيار الحساب")
            : shellCopy.jobs[l]
        }
      >
        <h2>
          {dialogKind === "account"
            ? t(l, "Choose account", "계정 선택", "اختيار الحساب")
            : shellCopy.jobs[l]}
        </h2>
        <p>
          {t(
            l,
            "Synthetic preview only. Production service not connected.",
            "합성 시안입니다. 운영 서비스는 연결되지 않았습니다.",
            "معاينة اصطناعية فقط. خدمة الإنتاج غير متصلة.",
          )}
        </p>
        {dialogKind === "account" ? (
          <AccountPicker
            locale={l}
            value={params.get("scope") ?? "sample-a"}
            onChange={(scope) => {
              update({ scope, page: "1", q: "" });
              close();
            }}
          />
        ) : (
          <p>
            {t(
              l,
              "No local preview jobs",
              "로컬 시안 작업이 없습니다",
              "لا توجد مهام معاينة محلية",
            )}
          </p>
        )}
        <button onClick={close}>{t(l, "Close", "닫기", "إغلاق")}</button>
      </dialog>
    </>
  );
}
