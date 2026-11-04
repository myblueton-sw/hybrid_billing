import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { Locale, Mode } from "../shared/types";
import { serviceQuery } from "./service-navigation";
import ServiceNavigation from "./ServiceNavigation";
import { sectionFor, relatedScreens, navigationLabels } from "./service-ia";
import "./service-shell.css";
import { useLocation, setNavigationGuard } from "../shared/navigation";
import {
  planningScreens,
  legacyReviewMap,
  type PlanningScreen,
} from "../planning/registry";
import {
  resolveRoute,
  planningUrl,
  visibleTo,
  type Audience,
} from "../planning/routing";
import { titles, menuNames, local, tabLabel } from "../planning/copy";
import Onboarding from "../modules/onboarding/pages/onboarding/Screen";
import Connections from "../modules/onboarding/pages/connections/Screen";
import EvidenceScreen from "../modules/sources/pages/evidence/Screen";
import SourceData from "../modules/sources/pages/source-data/Screen";
import Reconciliation from "../modules/sources/pages/reconciliation/Screen";
import LiveImports from "../modules/sources/pages/live-imports/Screen";
import CandidateWorkbench from "../planning/CandidateWorkbench";
import ReviewPanel from "../shared/ReviewPanel";
import HeaderSelect from "./HeaderSelect";
import EntryShell from "./EntryShell";
import { identity } from "../modules/identity/module";
import AllocationPolicies from "../modules/pricing/pages/allocation-policies/Screen";
import InfrastructureCost from "../modules/pricing/pages/infrastructure-cost/Screen";
import CollectedSource from "../modules/sources/pages/collected-source-data/Screen";
import Overview from "../modules/operations/pages/overview/Screen";
import Closing from "../modules/operations/pages/core-index-close/Screen";
import logo from "../../../../packages/design-system/assets/logo/hb-lockup-color.svg";
import "../planning/planning.css";
const personas = ["all", "OP", "AM", "PR", "BL", "AP", "CP", "AU", "CU", "IA"];
export default function App() {
  const [mobile, setMobile] = useState(() => window.innerWidth < 768);
  useEffect(() => {
    const media = window.matchMedia("(max-width: 767px)");
    const update = () => setMobile(media.matches);
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);
  const hash = useLocation(),
    params = new URLSearchParams(hash.split("?")[1]);
  const l: Locale =
    params.get("lang") === "en"
      ? "en"
      : params.get("lang") === "ar"
        ? "ar"
        : "ko";
  const developer = params.get("review") === "1";
  const mode: Mode =
    params.has("mode") && params.get("mode") !== "synthetic"
      ? "unconnected"
      : "synthetic";
  const [audiencePreview, setAudience] = useState<Audience>("admin"),
    [personaPreview, setPersona] = useState("all"),
    [search, setSearch] = useState(""),
    [dirty, setDirty] = useState(false),
    [scenarioPreview, setScenario] = useState("normal"),
    [view, setView] = useState({ key: "", state: "{}", items: [] as string[] });
  const menuDialog = useRef<HTMLDialogElement>(null),
    jobs = useRef<HTMLDialogElement>(null),
    menuTrigger = useRef<HTMLButtonElement>(null),
    jobTrigger = useRef<HTMLButtonElement>(null);
  const audience: Audience = developer ? audiencePreview : "admin";
  const persona = developer ? personaPreview : "all";
  const scenario = developer ? scenarioPreview : "normal";
  const context = useMemo(
    () => ({
      audience,
      viewable: new Set(
        planningScreens
          .filter(
            (s) =>
              persona === "all" ||
              s.exampleRoles.includes(persona) ||
              s.audience === "prelogin",
          )
          .map((s) => s.id),
      ),
      assistant:
        audience === "admin" &&
        (persona === "all" || ["BL", "PR", "AP", "CP", "OP"].includes(persona)),
      designTools: audience === "admin" && developer,
    }),
    [audience, persona, developer],
  );
  const route = resolveRoute(hash || "#/home", context),
    screen = route.screen;
  const scope = params.get("scope") === "sample-b" ? "sample-b" : "sample-a";
  const liveImports =
    screen?.id === "S03" &&
    audience === "admin" &&
    params.get("service") === "local";
  const liveReconciliation =
    screen?.id === "S06" &&
    audience === "admin" &&
    params.get("service") === "local";
  const liveSourceData =
    screen?.id === "S04" &&
    audience === "admin" &&
    params.get("service") === "local";
  const liveEvidence =
    screen?.id === "S05" &&
    audience === "admin" &&
    params.get("service") === "local";
  const liveConnections =
    screen?.id === "S01" &&
    audience === "admin" &&
    params.get("service") === "local";
  const liveOnboarding =
    screen?.id === "S02" &&
    audience === "admin" &&
    params.get("service") === "local";
  const liveCollectedSource =
    screen?.id === "S04" &&
    audience === "admin" &&
    params.get("service") !== "local";
  const liveAllocation = screen?.id === "B03" && audience === "admin";
  const livePricing =
    (screen?.id === "B01" || liveAllocation) && audience === "admin";
  const liveOverview = screen?.id === "U01" && audience === "admin";
  const liveService =
    livePricing ||
    liveCollectedSource ||
    liveOverview ||
    liveImports ||
    liveReconciliation ||
    liveSourceData ||
    liveEvidence ||
    liveConnections ||
    liveOnboarding;
  const screenKey = `${liveService}:${screen?.id ?? route.path}:${audience}:${scope}:${params.get("period")}:${params.get("le")}:${params.get("ccy")}:${route.objectId ?? ""}:${mode}:${scenario}:${persona}:${liveOverview || liveCollectedSource || livePricing || liveSourceData || liveEvidence || liveOnboarding ? params.toString() : ""}`;
  const tabs =
    screen?.id === "U02" ? screen.tabs.slice(0, 2) : (screen?.tabs ?? []);
  const requestedTab = params.get("view") ?? route.tab ?? route.step;
  const tab = tabs.some((t) => t.key === requestedTab)
    ? requestedTab!
    : (tabs[0]?.key ?? "");
  const onState = useCallback(
    (state: string, items: string[]) =>
      setView((old) =>
        old.key === screenKey &&
        old.state === state &&
        old.items.join("|") === items.join("|")
          ? old
          : { key: screenKey, state, items },
      ),
    [screenKey],
  );
  const canLeave = () => {
    const allowed =
      !dirty ||
      window.confirm(
        local(l, [
          "저장하지 않은 입력을 버릴까요?",
          "Discard unsaved input?",
          "هل تريد تجاهل المدخلات غير المحفوظة؟",
        ]),
      );
    if (allowed) setNavigationGuard(null);
    return allowed;
  };
  function navigate(path: string, changes: Record<string, string> = {}) {
    if (!canLeave()) return;
    setDirty(false);
    setSearch("");
    menuDialog.current?.close();
    const q = serviceQuery(params, screen?.id ?? "");
    location.hash = planningUrl(path, q, changes);
  }
  function changeScope(changes: Record<string, string>) {
    if (!canLeave()) return;
    setDirty(false);
    location.hash = planningUrl(route.path, params, { ...changes, view: tab });
  }
  function previewAudience(next: Audience, path?: string) {
    if (!canLeave()) return;
    setDirty(false);
    setAudience(next);
    setPersona("all");
    setScenario("normal");
    location.hash = planningUrl(
      path ??
        (next === "admin"
          ? "/home"
          : next === "customer"
            ? "/portal/overview"
            : "/login"),
      new URLSearchParams({ lang: l, ...(developer ? { review: "1" } : {}) }),
    );
  }
  useEffect(() => {
    document.documentElement.lang = l;
    document.documentElement.dir = l === "ar" ? "rtl" : "ltr";
    document.title = `${screen ? local(l, titles[screen.id]) : "Hybrid Billing"} · Hybrid Billing`;
  }, [l, screen?.id]);
  useEffect(() => {
    if (!livePricing) setDirty(false);
    document.getElementById("main")?.focus({ preventScroll: true });
  }, [screenKey]);
  useEffect(() => {
    const h = (e: BeforeUnloadEvent) => {
      if (dirty) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", h);
    return () => window.removeEventListener("beforeunload", h);
  }, [dirty]);
  useEffect(() => {
    const nextQuery = new URLSearchParams(params);
    if (route.legacyId?.startsWith("entry-"))
      nextQuery.set("entry-view", route.legacyId);
    const expected = planningUrl(route.path, nextQuery);
    if (
      route.legacyId ||
      (!route.screen && !route.tool && !hash.startsWith("#" + route.path))
    ) {
      history.replaceState(null, "", expected);
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    }
  }, [hash, route.path, route.legacyId]);
  useEffect(() => {
    setNavigationGuard(
      () =>
        !dirty ||
        window.confirm(
          local(l, [
            "저장하지 않은 입력을 버릴까요?",
            "Discard unsaved input?",
            "هل تريد تجاهل المدخلات غير المحفوظة؟",
          ]),
        ),
    );
    return () => setNavigationGuard(null);
  }, [dirty, l, screenKey]);
  const allowed = planningScreens.filter((s) => visibleTo(s, context));
  const filtered = allowed.filter((s) =>
    `${local(l, titles[s.id])} ${s.id}`
      .toLowerCase()
      .includes(search.toLowerCase()),
  );
  const previousDeveloper = useRef(developer);
  useEffect(() => {
    if (previousDeveloper.current && !developer) {
      setAudience("admin");
      setPersona("all");
      setScenario("normal");
      setSearch("");
    }
    previousDeveloper.current = developer;
  }, [developer]);
  const sidebar = (
    <>
      {developer && (
        <label className="planning-search">
          {local(l, ["화면 검색", "Find a screen", "البحث عن شاشة"])}
          <input
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </label>
      )}
      <ServiceNavigation
        locale={l}
        entries={developer ? filtered : allowed}
        active={screen?.id}
        query={serviceQuery(params, screen?.id ?? "")}
        onNavigate={navigate}
        developer={developer}
        activeView={tab}
      />
    </>
  );
  const portal = audience === "customer";
  const headerActions = (
    <div className="planning-header-actions">
      {developer && !portal && !liveService && (
        <label className="planning-workspace">
          <span className="sr-only">
            {local(l, ["예제 업무 공간", "Example workspace", "مساحة تجريبية"])}
          </span>
          <select
            value={scope}
            onChange={(e) => changeScope({ scope: e.target.value })}
          >
            <option value="sample-a">
              {local(l, [
                "예제 공간 A",
                "Example workspace A",
                "مساحة تجريبية A",
              ])}
            </option>
            <option value="sample-b">
              {local(l, [
                "예제 공간 B",
                "Example workspace B",
                "مساحة تجريبية B",
              ])}
            </option>
          </select>
        </label>
      )}
      {context.assistant && allowed.some((s) => s.id === "A01") && (
        <button onClick={() => navigate("/assistant")}>AI Assistant</button>
      )}
      <details className="planning-work">
        <summary>{local(l, ["내 작업", "My work", "أعمالي"])}</summary>
        {allowed.some((s) => s.id === "U03") && (
          <button onClick={() => navigate("/tasks")}>
            {local(l, [
              "내 작업·알림",
              "My work and notifications",
              "أعمالي والإشعارات",
            ])}
          </button>
        )}
        <button ref={jobTrigger} onClick={() => jobs.current?.showModal()}>
          {local(l, ["실행 중 작업", "Background jobs", "المهام الخلفية"])}
        </button>
      </details>
      <details className="planning-help">
        <summary>{local(l, ["도움말", "Help", "المساعدة"])}</summary>
        <button onClick={() => navigate("/help/contribute")}>
          {local(l, [
            "기여·지원",
            "Contribution and support",
            "المساهمة والدعم",
          ])}
        </button>
      </details>
      <HeaderSelect
        label="Language"
        value={l}
        options={[
          { value: "ko", label: "한국어" },
          { value: "en", label: "English (US)" },
          { value: "ar", label: "العربية (الإمارات)" },
        ]}
        onChange={(lang) => changeScope({ lang })}
      />
      <details className="planning-user">
        <summary>{local(l, ["내 계정", "My account", "حسابي"])}</summary>
        <button onClick={() => navigate("/my-security")}>
          {local(l, titles.M05)}
        </button>
        <button onClick={() => changeScope({ review: developer ? "" : "1" })}>
          {local(
            l,
            developer
              ? ["개발 모드 종료", "Exit developer mode", "إنهاء وضع المطور"]
              : [
                  "개발·검수 도구",
                  "Developer and review tools",
                  "أدوات التطوير والمراجعة",
                ],
          )}
        </button>
        <button
          hidden={!developer}
          onClick={() => previewAudience("unauthenticated")}
        >
          {local(l, [
            "로그인 전 시안으로",
            "Open sign-in preview",
            "فتح معاينة الدخول",
          ])}
        </button>
      </details>
    </div>
  );
  const reviewTools = (
    <details className="planning-review-tools">
      <summary>
        {local(l, [
          "시안 검토 도구",
          "Preview review tools",
          "أدوات مراجعة المعاينة",
        ])}
      </summary>
      <div className="planning-controls">
        <label>
          {local(l, [
            "사용자 구분 미리보기",
            "Audience preview",
            "معاينة فئة المستخدم",
          ])}
          <select
            value={audience}
            onChange={(e) => previewAudience(e.target.value as Audience)}
          >
            <option value="admin">
              {local(l, ["내부 사용자", "Internal user", "مستخدم داخلي"])}
            </option>
            <option value="customer">
              {local(l, ["고객", "Customer", "عميل"])}
            </option>
            <option value="unauthenticated">
              {local(l, ["로그인 전", "Before sign-in", "قبل الدخول"])}
            </option>
          </select>
        </label>
        {audience === "admin" && (
          <label>
            {local(l, [
              "예제 역할 · 권한 부여 아님",
              "Example persona · grants nothing",
              "شخصية تجريبية · لا تمنح صلاحيات",
            ])}
            <select
              value={persona}
              onChange={(e) => {
                if (canLeave()) {
                  setPersona(e.target.value);
                  setDirty(false);
                }
              }}
            >
              {personas
                .filter((p) => p !== "CU")
                .map((p) => (
                  <option key={p}>{p}</option>
                ))}
            </select>
          </label>
        )}
        <label>
          {local(l, ["데이터 모드", "Data mode", "نمط البيانات"])}
          <select
            value={mode}
            onChange={(e) => changeScope({ mode: e.target.value })}
          >
            <option value="synthetic">
              {local(l, ["합성 예제", "Synthetic examples", "أمثلة تجريبية"])}
            </option>
            <option value="unconnected">
              {local(l, ["미연결", "Unconnected", "غير متصل"])}
            </option>
          </select>
        </label>
        <label>
          {local(l, ["상태 예제", "State example", "مثال الحالة"])}
          <select
            value={scenario}
            onChange={(e) => {
              if (canLeave()) {
                setDirty(false);
                setScenario(e.target.value);
              }
            }}
          >
            {["normal", "loading", "empty", "partial", "error", "revoked"].map(
              (s) => (
                <option key={s}>{s}</option>
              ),
            )}
          </select>
        </label>
        {audience === "admin" && (
          <button onClick={() => navigate("/design/screens")}>
            {local(l, [
              "전체 화면 찾아보기",
              "Browse all screens",
              "استعراض الشاشات",
            ])}
          </button>
        )}
      </div>
      <p>
        {local(l, [
          "합성 시안 전용입니다. 역할·사용자 구분 선택은 실제 인증이나 권한을 부여하지 않습니다.",
          "Synthetic preview only. Persona and audience choices provide no real authentication or authority.",
          "معاينة تجريبية فقط. اختيار الفئة أو الدور لا يمنح مصادقة أو صلاحية فعلية.",
        ])}
      </p>
    </details>
  );
  if (audience === "unauthenticated" && screen) {
    const entries =
      screen.id === "L01"
        ? ["entry-start", "entry-login", "entry-scope"]
        : screen.id === "L02"
          ? ["entry-bootstrap", "entry-invite"]
          : ["entry-challenge"];
    const requested = params.get("entry-view") ?? route.legacyId ?? "";
    const id = entries.includes(requested)
      ? requested
      : screen.id === "L01"
        ? "entry-login"
        : screen.id === "L02"
          ? "entry-invite"
          : "entry-challenge";
    const entry = identity.routes.find((r) => r.id === id)!;
    return (
      <>
        <EntryShell
          route={entry}
          planningId={screen.id}
          locale={l}
          mode={mode}
          profile="standard"
        />
        <div className="planning-entry-tools">
          {developer && reviewTools}
          {developer && (
            <button onClick={() => navigate("/home", { review: "" })}>
              {local(l, [
                "서비스로 돌아가기",
                "Return to service",
                "العودة إلى الخدمة",
              ])}
            </button>
          )}
          <nav>
            {planningScreens
              .filter((s) => s.audience === "prelogin")
              .map((s) => (
                <button key={s.id} onClick={() => navigate(s.path)}>
                  {local(l, titles[s.id])}
                </button>
              ))}
          </nav>
        </div>
      </>
    );
  }
  return (
    <div
      className={`planning-app service-shell ${screen?.id === "U01" || liveCollectedSource || livePricing ? "overview-shell" : ""} ${portal ? "planning-portal" : ""} ${liveSourceData ? "live-source-screen" : ""} ${liveEvidence ? "live-evidence-screen" : ""} ${liveConnections ? "live-connections-screen" : ""} ${liveOnboarding ? "live-onboarding-screen" : ""}`}
    >
      <a
        className="skip"
        href="#main"
        onClick={(e) => {
          e.preventDefault();
          document.getElementById("main")?.focus();
        }}
      >
        {local(l, ["본문 바로가기", "Skip to content", "انتقل إلى المحتوى"])}
      </a>
      <div className="planning-preview" hidden={!developer}>
        <span>
          {liveOverview || liveCollectedSource || livePricing
            ? local(l, [
                "수집 FOCUS 샘플 · 공개·합성 출처 · 조회 전용",
                "Collected FOCUS samples · public/synthetic sources · read only",
                "عينات FOCUS مجمعة · مصادر عامة وتجريبية · قراءة فقط",
              ])
            : liveOnboarding
              ? local(l, [
                  "로컬 온보딩 근거 · 합성 입력 전용 · 개통 승인 미연결",
                  "Local onboarding evidence · synthetic input only · activation approval not connected",
                  "أدلة تهيئة محلية · مدخلات تجريبية فقط · اعتماد التشغيل غير متصل",
                ])
              : liveConnections
                ? local(l, [
                    "로컬 연결 현황 · 실제 설정·수신 메타데이터 · 합성 입력 전용",
                    "Local connection status · actual configuration and reception metadata · synthetic input only",
                    "حالة اتصال محلي · إعدادات وبيانات استقبال فعلية · مدخلات تجريبية فقط",
                  ])
                : liveEvidence
                  ? local(l, [
                      "로컬 증빙 열람 · 원본 검증·텍스트 추출 · 합성 입력 전용",
                      "Local evidence inspection · verified original text extract · synthetic input only",
                      "فحص أدلة محلي · مقتطف نصي من أصل متحقق منه · مدخلات تجريبية فقط",
                    ])
                  : liveSourceData
                    ? local(l, [
                        "로컬 명세 조회 · 실제 저장 데이터 · 합성 입력 전용",
                        "Local source inspection · actual stored rows · synthetic input only",
                        "فحص مصدر محلي · صفوف مخزنة فعلية · مدخلات تجريبية فقط",
                      ])
                    : liveReconciliation
                      ? local(l, [
                          "로컬 대사 · 실제 저장 근거 조회 · 합성 데이터 전용",
                          "Local reconciliation · actual stored evidence · synthetic data only",
                          "مطابقة محلية · أدلة مخزنة فعلية · بيانات تجريبية فقط",
                        ])
                      : liveImports
                        ? local(l, [
                            "로컬 서비스 연동 · 실제 검증·저장 · 합성 입력 전용",
                            "Local service integration · actual validation and persistence · synthetic input only",
                            "تكامل خدمة محلية · تحقق وحفظ فعليان · مدخلات تجريبية فقط",
                          ])
                        : local(l, [
                            "인터랙티브 시안 · 합성 데이터 · 실제 서비스 미연결",
                            "Interactive preview · synthetic data · services not connected",
                            "معاينة تفاعلية · بيانات تجريبية · الخدمات غير متصلة",
                          ])}
        </span>
        {audience === "admin" && (
          <button onClick={() => previewAudience("admin", "/design/screens")}>
            {local(l, [
              "시안 전체 보기",
              "All screen candidates",
              "جميع المعاينات",
            ])}
          </button>
        )}
      </div>
      <header className="planning-header">
        <button
          ref={menuTrigger}
          className="planning-menu-button"
          onClick={() => menuDialog.current?.showModal()}
        >
          {local(l, ["메뉴", "Menu", "القائمة"])}
        </button>
        <a
          className="brand"
          dir="ltr"
          href={planningUrl(portal ? "/portal/overview" : "/home", params)}
          onClick={(e) => {
            e.preventDefault();
            navigate(portal ? "/portal/overview" : "/home");
          }}
        >
          <img
            className="brand-logo"
            src={logo}
            width="155"
            height="32"
            alt="Hybrid Billing"
          />
        </a>
        <span className="planning-installation">
          <span className="service-environment">
            {local(
              l,
              liveOverview || liveCollectedSource || livePricing
                ? [
                    "수집 FOCUS 샘플 · 공개·합성 출처",
                    "Collected FOCUS samples · public and synthetic sources",
                    "عينات FOCUS مجمعة · مصادر عامة وتجريبية",
                  ]
                : liveService
                  ? [
                      "로컬 검증 환경 · 합성 데이터",
                      "Local validation · synthetic data",
                      "تحقق محلي · بيانات تجريبية",
                    ]
                  : mode === "synthetic"
                    ? [
                        "데모 환경 · 합성 데이터",
                        "Demo · synthetic data",
                        "عرض تجريبي · بيانات اصطناعية",
                      ]
                    : [
                        "데이터 미연결",
                        "Data not connected",
                        "البيانات غير متصلة",
                      ],
            )}
          </span>
        </span>
        {!mobile && headerActions}
        {mobile && (
          <button
            className="planning-mobile-account"
            onClick={() => navigate("/my-security")}
          >
            {local(l, ["내 계정", "My account", "حسابي"])}
          </button>
        )}
      </header>
      {developer && reviewTools}
      {portal && (
        <nav
          className="planning-portal-nav"
          aria-label={local(l, [
            "고객 메뉴",
            "Customer navigation",
            "تنقل العميل",
          ])}
        >
          {["C01", "C02", "C03", "U03"].map((id) => (
            <button
              key={id}
              aria-current={screen?.id === id ? "page" : undefined}
              onClick={() =>
                navigate(planningScreens.find((s) => s.id === id)!.path)
              }
            >
              {local(l, titles[id])}
            </button>
          ))}
        </nav>
      )}
      <div className="planning-layout">
        {!portal && <aside className="planning-sidebar">{sidebar}</aside>}
        <main id="main" tabIndex={-1} className="planning-main">
          {route.tool ? (
            <>
              <header className="planning-page-header">
                <h1>
                  {local(l, [
                    "전체 화면 시안",
                    "All screen candidates",
                    "جميع معاينات الشاشات",
                  ])}
                </h1>
                <p>
                  {local(l, [
                    "기획 기준 49개 화면 · 각 화면에서 개별 검수",
                    "49 planning screens · review each screen separately",
                    "49 شاشة حسب التخطيط · مراجعة كل شاشة منفردة",
                  ])}
                </p>
              </header>
              {route.tool === "/help/contribute" ? (
                <section className="planning-panel">
                  <h2>
                    {local(l, [
                      "기여·지원",
                      "Contribution and support",
                      "المساهمة والدعم",
                    ])}
                  </h2>
                  <a
                    href="https://github.com/myblueton-sw/hybrid_billing"
                    target="_blank"
                    rel="noreferrer"
                  >
                    GitHub
                  </a>
                </section>
              ) : (
                <div className="planning-directory">
                  {planningScreens.map((s) => (
                    <button
                      key={s.id}
                      onClick={() =>
                        previewAudience(
                          s.audience === "prelogin"
                            ? "unauthenticated"
                            : s.audience === "portal"
                              ? "customer"
                              : "admin",
                          s.path,
                        )
                      }
                    >
                      <small>
                        {s.id} · {local(l, menuNames[s.menu])}
                      </small>
                      <strong>{local(l, titles[s.id])}</strong>
                      <span>{s.path}</span>
                    </button>
                  ))}
                </div>
              )}
            </>
          ) : screen ? (
            <>
              <header className="planning-page-header">
                <p className="crumb">
                  {local(
                    l,
                    sectionFor(screen.id)?.label ?? menuNames[screen.menu],
                  )}
                  {developer && (
                    <>
                      {" "}
                      / <bdi>{screen.id}</bdi>
                    </>
                  )}
                </p>
                <div>
                  <h1>
                    {screen.id === "U01"
                      ? local(l, [
                          "통합 빌링 현황",
                          "Billing overview",
                          "نظرة عامة على الفوترة",
                        ])
                      : local(
                          l,
                          liveCollectedSource
                            ? navigationLabels.S04
                            : titles[screen.id],
                        )}
                  </h1>
                  <span className="tag" hidden={!developer}>
                    {local(l, [
                      "검토 시안",
                      "Review candidate",
                      "معاينة للمراجعة",
                    ])}
                  </span>
                </div>
              </header>
              {relatedScreens(screen.id).some((id) =>
                allowed.some((s) => s.id === id),
              ) && (
                <details className="service-related">
                  <summary>
                    {local(l, [
                      "관련 업무",
                      "Related work",
                      "الأعمال ذات الصلة",
                    ])}
                  </summary>
                  <nav
                    aria-label={local(l, [
                      "관련 업무",
                      "Related work",
                      "الأعمال ذات الصلة",
                    ])}
                  >
                    {[
                      ...relatedScreens(screen.id),
                      ...(screen.id === "I01" ? ["I03"] : []),
                    ]
                      .filter((id, index, all) => all.indexOf(id) === index)
                      .flatMap((id) => {
                        const target = allowed.find((s) => s.id === id);
                        return target
                          ? [
                              <a
                                key={id}
                                href={planningUrl(
                                  target.path,
                                  serviceQuery(params, screen.id),
                                )}
                                onClick={(e) => {
                                  e.preventDefault();
                                  navigate(target.path);
                                }}
                              >
                                {local(l, navigationLabels[id] ?? titles[id])}
                              </a>,
                            ]
                          : [];
                      })}
                  </nav>
                </details>
              )}
              <ReviewPanel
                screen={screen.id}
                locale={l}
                context={JSON.stringify({
                  shellMode: developer ? "developer" : "service",
                  profile:
                    liveOverview || liveCollectedSource || livePricing
                      ? "collected-focus-samples-v1"
                      : liveService
                        ? "local-service-synthetic-v1"
                        : "planning-preview",
                  mode,
                  extensions: [],
                  audience,
                  legacy: screen.reactBodies,
                  scope,
                  tab,
                  period:
                    liveOverview || liveCollectedSource || livePricing
                      ? (params.get("focus-period") ?? "all")
                      : (params.get("period") ?? "2026-10"),
                  organization:
                    liveOverview || liveCollectedSource || livePricing
                      ? "server-authorized-focus-accounts"
                      : (params.get("le") ?? "DEMO-ORG-01"),
                  currency:
                    liveOverview || liveCollectedSource || livePricing
                      ? "source-currencies"
                      : (params.get("ccy") ?? "USD"),
                  objectId: route.objectId ?? null,
                  persona,
                  scenario,
                  dirty,
                  state: view.key === screenKey ? view.state : "{}",
                })}
                items={
                  view.key === screenKey && view.items.length
                    ? view.items
                    : [`${screen.id}:region:${tab}`]
                }
              />
              {!["U01", "U02"].includes(screen.id) && !liveCollectedSource && (
                <details
                  className="planning-scope"
                  open={!mobile}
                  hidden={liveService}
                >
                  <summary>
                    {local(l, ["조회 범위", "Scope", "النطاق"])} ·{" "}
                    <bdi>{params.get("period") ?? "2026-10"}</bdi> ·{" "}
                    {portal ? "DEMO Own organization" : scope}
                  </summary>
                  <div className="planning-controls">
                    {screen.id !== "M05" && (
                      <>
                        <label>
                          {local(l, ["조직", "Organization", "المؤسسة"])}
                          <select
                            value={params.get("le") ?? "DEMO-ORG-01"}
                            onChange={(e) =>
                              changeScope({ le: e.target.value })
                            }
                          >
                            <option>DEMO-ORG-01</option>
                            {!portal && <option>DEMO-ORG-02</option>}
                          </select>
                        </label>
                        <label>
                          {local(l, ["기간", "Period", "الفترة"])}
                          <input
                            type="month"
                            value={params.get("period") ?? "2026-10"}
                            onChange={(e) =>
                              changeScope({ period: e.target.value })
                            }
                          />
                        </label>
                        <label>
                          {local(l, ["통화", "Currency", "العملة"])}
                          <select
                            value={params.get("ccy") ?? "USD"}
                            onChange={(e) =>
                              changeScope({ ccy: e.target.value })
                            }
                          >
                            {["USD", "KRW", "EUR"].map((c) => (
                              <option key={c}>{c}</option>
                            ))}
                          </select>
                        </label>
                      </>
                    )}
                    <span>
                      {local(l, [
                        "합성 기준시각",
                        "Synthetic as-of",
                        "وقت مرجعي تجريبي",
                      ])}{" "}
                      <bdi>2026-11-03T12:00Z</bdi>
                    </span>
                  </div>
                </details>
              )}
              <div
                className={
                  screen.archetype.includes("wizard")
                    ? "planning-wizard-layout"
                    : ""
                }
              >
                <nav
                  className="planning-tabs"
                  style={
                    liveService &&
                    !(liveOverview || liveCollectedSource || livePricing)
                      ? { display: "none" }
                      : undefined
                  }
                  aria-label={local(l, [
                    "화면 보기",
                    "Screen views",
                    "عروض الشاشة",
                  ])}
                >
                  {tabs.map((t, i) => (
                    <button
                      key={t.key}
                      aria-current={tab === t.key ? "page" : undefined}
                      onClick={() => {
                        if (livePricing && tab === t.key) return;
                        if (livePricing || canLeave()) {
                          if (livePricing) setNavigationGuard(null);
                          if (!livePricing) setDirty(false);
                          location.hash = planningUrl(screen.path, params, {
                            view: t.key,
                            ...(screen.id === "U02"
                              ? { "close-view": t.key }
                              : {}),
                          });
                        }
                      }}
                    >
                      {screen.archetype.includes("wizard") ? `${i + 1}. ` : ""}
                      {livePricing
                        ? local(
                            l,
                            (
                              {
                                policies: [
                                  "배분 정책",
                                  "Allocation policy",
                                  "سياسة التوزيع",
                                ],
                                pools: [
                                  "배분 기준액",
                                  "Allocation basis",
                                  "مبلغ أساس التوزيع",
                                ],
                                simulation: [
                                  "배분 미리보기",
                                  "Allocation preview",
                                  "معاينة التوزيع",
                                ],
                                comparison: [
                                  "결과 비교",
                                  "Result comparison",
                                  "مقارنة النتائج",
                                ],
                                summary: [
                                  "구성 요약",
                                  "Composition summary",
                                  "ملخص المكوّنات",
                                ],
                                costs: [
                                  "원가 항목",
                                  "Cost items",
                                  "بنود التكلفة",
                                ],
                                products: [
                                  "상품별 포함 항목",
                                  "Included items by product",
                                  "البنود المشمولة لكل منتج",
                                ],
                                approvals: [
                                  "승인 이력",
                                  "Approval history",
                                  "سجل الاعتماد",
                                ],
                              } as Record<string, [string, string, string]>
                            )[t.key],
                          )
                        : tabLabel(l, t.key, t.labelKo)}
                    </button>
                  ))}
                </nav>
                {developer &&
                  screen.id !== "S04" &&
                  ["S01", "S02", "S03", "S04", "S05", "S06"].includes(
                    screen.id,
                  ) && (
                    <button
                      className="live-mode-switch"
                      onClick={() =>
                        changeScope({
                          service: liveService ? "preview" : "local",
                        })
                      }
                    >
                      {local(
                        l,
                        liveService
                          ? [
                              "시안 데이터로 전환",
                              "Switch to preview data",
                              "التبديل إلى المعاينة",
                            ]
                          : [
                              "로컬 서비스에 연결",
                              "Connect local service",
                              "الاتصال بالخدمة المحلية",
                            ],
                      )}
                    </button>
                  )}
                <div
                  id="content"
                  key={
                    livePricing
                      ? screen.id + ":" + l
                      : screenKey + ":" + tab + (liveService ? ":" + l : "")
                  }
                >
                  {liveAllocation ? (
                    <AllocationPolicies
                      locale={l}
                      params={params}
                      view={tab}
                      onState={onState}
                      onDirty={setDirty}
                      canDiscard={canLeave}
                    />
                  ) : livePricing ? (
                    <InfrastructureCost
                      locale={l}
                      params={params}
                      view={tab}
                      onState={onState}
                      onDirty={setDirty}
                      canDiscard={canLeave}
                    />
                  ) : liveCollectedSource ? (
                    <CollectedSource
                      locale={l}
                      view={tab}
                      objectId={route.objectId}
                      onState={onState}
                    />
                  ) : liveOnboarding ? (
                    <Onboarding
                      locale={l}
                      stage={tab}
                      expected={params.get("connection") ?? undefined}
                      onState={onState}
                    />
                  ) : liveConnections ? (
                    <Connections
                      locale={l}
                      objectId={route.objectId}
                      onState={onState}
                    />
                  ) : liveEvidence ? (
                    <EvidenceScreen
                      locale={l}
                      objectId={route.objectId}
                      onState={onState}
                    />
                  ) : liveSourceData ? (
                    <SourceData
                      locale={l}
                      onState={onState}
                      objectId={route.objectId}
                    />
                  ) : liveReconciliation ? (
                    <Reconciliation
                      locale={l}
                      onState={onState}
                      objectId={route.objectId}
                    />
                  ) : liveImports ? (
                    <LiveImports
                      objectId={route.objectId}
                      locale={l}
                      onState={onState}
                      onDirty={setDirty}
                    />
                  ) : screen.id === "U01" ? (
                    <Overview
                      view={tab}
                      locale={l}
                      mode={mode}
                      profile="standard"
                      extensionIds={[]}
                      contributions={[]}
                      onState={onState}
                    />
                  ) : screen.id === "U02" && scenario === "normal" ? (
                    <Closing
                      locale={l}
                      mode={mode}
                      profile="standard"
                      extensionIds={[]}
                      contributions={[]}
                      onState={onState}
                    />
                  ) : (
                    <CandidateWorkbench
                      screen={screen}
                      locale={l}
                      mode={mode}
                      tab={tab}
                      objectId={route.objectId}
                      canDiscard={canLeave}
                      scope={scope}
                      period={params.get("period") ?? "2026-10"}
                      currency={params.get("ccy") ?? "USD"}
                      organization={params.get("le") ?? "DEMO-ORG-01"}
                      scenario={scenario}
                      customer={portal}
                      onDirty={setDirty}
                      onState={onState}
                    />
                  )}
                </div>
              </div>
              <p className="planning-legacy-map" hidden={!developer}>
                {local(l, [
                  "기존 화면 연결",
                  "Legacy screen mapping",
                  "ربط الشاشات السابقة",
                ])}
                :{" "}
                {Object.entries(legacyReviewMap)
                  .filter(([id]) => screen.reactBodies.includes(id))
                  .map(([id]) => id)
                  .join(", ") || "—"}
              </p>
            </>
          ) : (
            <section className="planning-panel" role="status">
              <h1>
                {local(l, [
                  "현재 예제 범위에서 열 수 있는 화면 없음",
                  "No screen available in this preview scope",
                  "لا توجد شاشة متاحة في نطاق المعاينة",
                ])}
              </h1>
            </section>
          )}
        </main>
      </div>
      <dialog
        className="planning-menu-dialog"
        ref={menuDialog}
        onCancel={() => menuTrigger.current?.focus()}
      >
        <button
          autoFocus
          onClick={() => {
            menuDialog.current?.close();
            menuTrigger.current?.focus();
          }}
        >
          {local(l, ["메뉴 닫기", "Close menu", "إغلاق القائمة"])}
        </button>
        {mobile && headerActions}
        {!portal && sidebar}
      </dialog>
      <dialog
        className="planning-jobs"
        ref={jobs}
        onCancel={() => jobTrigger.current?.focus()}
      >
        <h2>
          {local(l, ["백그라운드 작업", "Background jobs", "المهام الخلفية"])}
        </h2>
        <p>
          {local(l, [
            "실제 작업 서비스가 연결되지 않았습니다. 화면에서 수행한 초안 검토는 작업을 생성하지 않습니다.",
            "Job service is not connected. Reviewing a local draft does not create a job.",
            "خدمة المهام غير متصلة. مراجعة مسودة محلية لا تنشئ مهمة.",
          ])}
        </p>
        <button
          autoFocus
          onClick={() => {
            jobs.current?.close();
            jobTrigger.current?.focus();
          }}
        >
          {local(l, ["닫기", "Close", "إغلاق"])}
        </button>
      </dialog>
    </div>
  );
}
