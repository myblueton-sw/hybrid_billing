import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import type { FormEvent, ReactNode } from "react";
import type { Locale, ScreenProps } from "../../shared/types";
import { useLocation, routeUrl } from "../../shared/navigation";
import { ob } from "./copy";
import { profiles, profileNotes } from "./profiles";
import {
  store,
  draftKey,
  setupRoutes,
  stepKeys,
  canStep,
  resumeTarget,
  profileOf,
  blockedProfile,
  requiredFields,
} from "./store";
import "./onboarding.css";

type Option = [string, string];

function go(route: string) {
  location.hash = routeUrl(route);
}

function Field({
  name,
  label,
  value,
  onChange,
  type = "text",
  required = true,
  dir = "ltr",
  l,
}: {
  name: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  required?: boolean;
  dir?: "ltr" | "auto";
  l: Locale;
}) {
  return (
    <label>
      <span>
        {label}
        {required && <span className="setup-required"> *</span>}
      </span>
      <input
        name={name}
        type={type}
        value={value}
        maxLength={200}
        dir={dir}
        aria-required={required}
        onChange={(e) => onChange(e.target.value)}
      />
    </label>
  );
}

function Choice({
  name,
  label,
  value,
  options,
  onChange,
  required = false,
}: {
  name: string;
  label: string;
  value: string;
  options: Option[];
  onChange: (value: string) => void;
  required?: boolean;
}) {
  return (
    <label>
      <span>
        {label}
        {required && <span className="setup-required"> *</span>}
      </span>
      <select
        aria-required={required}
        name={name}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      >
        {options.map(([v, text]) => (
          <option key={v} value={v}>
            {text}
          </option>
        ))}
      </select>
    </label>
  );
}

function Check({
  name,
  label,
  checked,
  onChange,
}: {
  name: string;
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="inline setup-check">
      <input
        name={name}
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
      />
      {label}
    </label>
  );
}

function Result({ tone, children }: { tone: string; children: ReactNode }) {
  return <span className={`status ${tone}`}>{children}</span>;
}

export function makeSetupScreen(id: string) {
  return function SetupRoute(props: ScreenProps) {
    return <Setup id={id} {...props} />;
  };
}

function Setup({ id, locale: l, mode, onState }: ScreenProps & { id: string }) {
  const hash = useLocation(),
    scope = new URLSearchParams(hash.split("?")[1]).get("scope"),
    key = draftKey(scope, mode),
    version = useSyncExternalStore(store.subscribe, store.version);
  const [error, setError] = useState("");
  useEffect(() => () => store.dispose(), []);
  useEffect(() => setError(""), [id, key]);
  const focusAfter = useRef("");
  const nav = useRef<HTMLElement>(null);
  useEffect(() => {
    const box = nav.current,
      item = box?.querySelector<HTMLElement>("li.current");
    if (!box || !item || box.scrollWidth <= box.clientWidth) return;
    const a = box.getBoundingClientRect(),
      b = item.getBoundingClientRect();
    box.scrollLeft += b.left + b.width / 2 - (a.left + a.width / 2);
  }, [id]);
  const restore = (target: string) => {
    focusAfter.current = target;
  };
  const d = store.get(key),
    v = d.values,
    index = setupRoutes.indexOf(id as (typeof setupRoutes)[number]),
    p = profileOf(v),
    t = (k: string) => ob(k, l),
    s = (k: string) => (typeof v[k] === "string" ? (v[k] as string) : ""),
    unconnected = mode === "unconnected",
    locked = !canStep(d, index),
    blocked = blockedProfile(v),
    provider = s("provider");
  useEffect(() => {
    const target = focusAfter.current
      ? document.getElementById(focusAfter.current)
      : null;
    if (target && !(target as HTMLButtonElement).disabled) {
      target.focus();
      focusAfter.current = "";
    }
  }, [version]);
  const set = (name: string) => (value: string | boolean) => {
    setError("");
    store.setField(key, index, name, value);
  };
  const items: string[] = [
    "region:layout",
    "nav:setup-steps",
    "notice:draft-boundary",
  ];
  if (locked && index > 0) items.push("state:prerequisite", "action:resume");
  else if (index === 0)
    items.push(
      "section:readiness-summary",
      "notice:history-boundary",
      "action:start-or-resume",
    );
  else if ([1, 2, 3, 5].includes(index)) {
    if (index === 1)
      items.push(
        "field:legal-entity",
        "field:market",
        "field:currency",
        "field:timezone",
        "field:tax-policy",
      );
    if (index === 2)
      items.push(
        "field:customer-reference",
        "field:custodian-reference",
        "field:effective-from",
        "field:scope-ack",
      );
    if (index === 3) {
      items.push(
        "field:provider",
        "field:route",
        "field:billing-scope-id",
        "field:workload-identity",
      );
      if (provider === "azure") items.push("field:azure-agreement");
      if (["gcp", "alibaba"].includes(provider))
        items.push("field:preview-eligibility");
      if (provider === "ibm") items.push("field:ibm-route");
      items.push(
        ...requiredFields(v).map((f) => `field:provider-setting:${f.key}`),
      );
      items.push("notice:profile-limit", "action:review-plan");
      if (blocked) items.push("state:profile-blocked");
      if (!s("identity")) items.push("section:handoff");
      if (d.plan) {
        items.push("section:resource-plan", "field:plan-ack");
        if (v.route === "new")
          items.push("state:admin-readback", "action:admin-readback");
        items.push("action:save-continue");
      }
    } else items.push("action:save-continue");
    if (index === 5)
      items.push(
        "field:start-month",
        "field:collection-mode",
        "notice:history-boundary",
      );
    items.push("action:fill-example");
    if (error) items.push("state:validation-error");
  } else if (index === 4) {
    items.push(
      "data:connection-ref",
      "section:diagnostic-checks",
      "action:run-diagnostic",
      "action:save-continue",
    );
    items.push(
      unconnected
        ? "state:not-connected"
        : (
            {
              idle: "state:idle",
              running: "state:running",
              pass: "state:diagnostic-pass",
              failed: "state:read-denied",
              unknown: "state:outcome-unknown",
            } as const
          )[d.diagnostic],
    );
    items.push(
      "action:scenario-normal",
      "action:scenario-denied",
      "action:scenario-partial",
      "action:scenario-outcome",
    );
  } else if (index === 6) {
    items.push(
      "data:receipt-ref",
      "section:receipt-checks",
      "action:validate-receipt",
      "action:save-continue",
      "state:evidence-blocked",
    );
    if (unconnected) items.push("state:not-connected");
    else if (d.receipt === "running") items.push("state:running");
    else if (d.receipt === "partial") items.push("state:coverage-partial");
    else if (d.receipt === "pass") items.push("state:receipt-pass");
  } else
    items.push(
      "section:readiness-facts",
      "notice:financial-evidence",
      "action:publish-unavailable",
      "action:return-workspace",
    );
  const fingerprint = JSON.stringify({
    key,
    id,
    done: [...d.done],
    revision: d.revision,
    plan: d.plan,
    admin: d.admin,
    diagnostic: d.diagnostic,
    receipt: d.receipt,
    scenario: v.scenario,
    error: !!error,
  });
  const itemKey = items.join("|");
  useEffect(
    () => onState(fingerprint, itemKey.split("|")),
    [fingerprint, itemKey, onState],
  );

  function submit(e: FormEvent) {
    e.preventDefault();
    if (index === 3) {
      const result = store.reviewPlan(key);
      setError(
        result === "ok"
          ? ""
          : t(result === "blocked" ? "unverified" : "invalid"),
      );
      return;
    }
    const next = store.submit(key, index);
    if (next) go(next);
    else setError(t("invalid"));
  }

  const steps = (
    <nav ref={nav} className="setup-steps-nav" aria-label={t("steps")}>
      <ol>
        {setupRoutes.map((route, i) => {
          const status =
            i === index
              ? t("stepCurrent")
              : d.done.has(route)
                ? t("stepDone")
                : canStep(d, i)
                  ? t("stepOpen")
                  : t("stepLocked");
          return (
            <li
              key={route}
              className={
                i === index ? "current" : d.done.has(route) ? "done" : ""
              }
            >
              <button
                type="button"
                data-setup-route={route}
                disabled={!canStep(d, i)}
                aria-current={i === index ? "step" : undefined}
                onClick={() => go(route)}
              >
                <span className="setup-step-index" aria-hidden="true">
                  {i + 1}
                </span>
                <span className="setup-step-text">
                  <span>{t(stepKeys[i])}</span>
                  <small>{status}</small>
                </span>
              </button>
            </li>
          );
        })}
      </ol>
    </nav>
  );

  function check(label: string, tone: string, text: string) {
    return (
      <li className="setup-check-row" key={label}>
        <span>{label}</span>
        <Result tone={tone}>{text}</Result>
      </li>
    );
  }
  const diagnosticResult = (k: string): [string, string] => {
    if (unconnected) return ["", t("notConnected")];
    if (d.diagnostic === "pass") return ["ready", t("pass")];
    if (d.diagnostic === "running") return ["pending", t("running")];
    if (d.diagnostic === "failed")
      return k === "read" ? ["failed", t("fail")] : ["ready", t("pass")];
    if (d.diagnostic === "unknown") return ["failed", t("unknown")];
    return ["", t("idle")];
  };
  const receiptResult = (k: string): [string, string] => {
    if (k === "evidence") return ["failed", t("blocked")];
    if (unconnected) return ["", t("notConnected")];
    if (d.receipt === "pass") return ["ready", t("pass")];
    if (d.receipt === "running") return ["pending", t("running")];
    if (d.receipt === "partial") return ["failed", t("receiptPartial")];
    return ["", t("idle")];
  };
  const cadenceLabel = ["ibm", "file"].includes(provider)
    ? t("once")
    : s("cadence")
      ? t(s("cadence"))
      : "—";

  let body: ReactNode;
  if (index === 0)
    body = (
      <section className="panel">
        <h2>{t("start")}</h2>
        <p className="notice">{t("history")}</p>
        <div className="stats">
          <article className="stat">
            {t("technical")}
            <strong className="setup-stat-value">
              {d.diagnostic === "pass" ? t("pass") : t("idle")}
            </strong>
          </article>
          <article className="stat">
            {t("coverage")}
            <strong className="setup-stat-value">
              {d.receipt === "pass" ? t("pass") : t("idle")}
            </strong>
          </article>
          <article className="stat">
            {t("financial")}
            <strong className="setup-stat-value">{t("blocked")}</strong>
          </article>
        </div>
        <button
          id="setup-start"
          className="primary"
          onClick={() => go(resumeTarget(d))}
        >
          {d.done.size ? t("resume") : t("start")}
        </button>
      </section>
    );
  else if (locked)
    body = (
      <section className="panel" role="status">
        <h2>{t("prerequisite")}</h2>
        <button
          id="setup-start"
          className="primary"
          onClick={() => go(resumeTarget(d))}
        >
          {t("resume")}
        </button>
      </section>
    );
  else if ([1, 2, 3, 5].includes(index)) {
    let fields: ReactNode = null;
    if (index === 1)
      fields = (
        <>
          <Field
            l={l}
            name="entity"
            dir="auto"
            label={t("entity")}
            value={s("entity")}
            onChange={set("entity")}
          />
          <Choice
            name="country"
            label={t("country")}
            required
            value={s("country")}
            onChange={set("country")}
            options={[
              ["", "—"],
              ["US", "United States"],
              ["AE", "United Arab Emirates"],
              ["KR", "대한민국"],
            ]}
          />
          <Choice
            name="currency"
            label={t("currency")}
            required
            value={s("currency")}
            onChange={set("currency")}
            options={[
              ["", "—"],
              ["USD", "USD"],
              ["AED", "AED"],
              ["KRW", "KRW"],
            ]}
          />
          <Choice
            name="timezone"
            label={t("timezone")}
            required
            value={s("timezone")}
            onChange={set("timezone")}
            options={[
              ["", "—"],
              ["America/New_York", "America/New_York"],
              ["Asia/Dubai", "Asia/Dubai"],
              ["Asia/Seoul", "Asia/Seoul"],
            ]}
          />
          <Field
            l={l}
            name="tax"
            label={t("tax")}
            value={s("tax")}
            onChange={set("tax")}
          />
        </>
      );
    if (index === 2)
      fields = (
        <>
          <Field
            l={l}
            name="customer"
            label={t("customer")}
            value={s("customer")}
            onChange={set("customer")}
          />
          <Field
            l={l}
            name="custody"
            label={t("custody")}
            value={s("custody")}
            onChange={set("custody")}
          />
          <Field
            l={l}
            name="effective"
            type="date"
            label={t("effective")}
            value={s("effective")}
            onChange={set("effective")}
          />
          <div className="setup-span">
            <Check
              name="scopeAck"
              label={t("scopeAck")}
              checked={!!v.scopeAck}
              onChange={set("scopeAck")}
            />
          </div>
        </>
      );
    if (index === 3)
      fields = (
        <>
          <Choice
            name="provider"
            label={t("provider")}
            value={provider}
            onChange={set("provider")}
            options={Object.entries(profiles).map(([k, value]) => [
              k,
              value.name,
            ])}
          />
          <Choice
            name="route"
            label={t("route")}
            value={s("route")}
            onChange={set("route")}
            options={[
              ["existing", t("existing")],
              ["new", t("new")],
            ]}
          />
          <Field
            l={l}
            name="account"
            label={t("account")}
            value={s("account")}
            onChange={set("account")}
          />
          <Choice
            name="identity"
            label={t("identity")}
            value={s("identity")}
            onChange={set("identity")}
            options={[
              ["", t("missing")],
              ["DEMO-READ-IDENTITY", t("approved")],
            ]}
          />
          {provider === "azure" && (
            <Choice
              name="agreement"
              label="Azure billing agreement"
              value={s("agreement")}
              onChange={set("agreement")}
              options={[
                ["MCA", "MCA"],
                ["EA", "EA"],
                ["MOSP", "MOSP"],
                ["management", "Management group"],
              ]}
            />
          )}
          {provider === "ibm" && (
            <Choice
              name="ibmRoute"
              label="IBM report route"
              value={s("ibmRoute")}
              onChange={set("ibmRoute")}
              options={[
                ["csv", "Account CSV"],
                ["api", "Account API — unverified"],
                ["enterprise", "Enterprise API — unverified"],
              ]}
            />
          )}
          {requiredFields(v).map((f) => (
            <Field
              l={l}
              key={f.key}
              name={f.key}
              label={f.label[l]}
              value={s(f.key)}
              onChange={set(f.key)}
            />
          ))}
          {["gcp", "alibaba"].includes(provider) && (
            <div className="setup-span">
              <Check
                name="preview"
                label={
                  l === "ko"
                    ? "예시 Preview 사용 권한 확인"
                    : l === "ar"
                      ? "تأكيد أهلية Preview التجريبية"
                      : "Confirm example Preview eligibility"
                }
                checked={!!v.preview}
                onChange={set("preview")}
              />
            </div>
          )}
        </>
      );
    if (index === 5)
      fields = (
        <>
          <Field
            l={l}
            name="period"
            type="month"
            label={t("period")}
            value={s("period")}
            onChange={set("period")}
          />
          <Choice
            name="cadence"
            label={t("cadence")}
            required
            value={s("cadence")}
            onChange={set("cadence")}
            options={
              ["file", "ibm"].includes(provider)
                ? [
                    ["", "—"],
                    ["once", t("once")],
                  ]
                : [
                    ["", "—"],
                    ["daily", t("daily")],
                    ["monthly", t("monthly")],
                  ]
            }
          />
        </>
      );
    const note = profileNotes[provider];
    body = (
      <section className="panel">
        <h2>{t(stepKeys[index])}</h2>
        <form id="setup-form" noValidate onSubmit={submit}>
          <div className="form-grid">{fields}</div>
          {error && (
            <p id="setup-error" className="setup-error" role="alert">
              {error}
            </p>
          )}
          <div className="form-actions">
            <button
              type="button"
              id="setup-example"
              onClick={() => {
                setError("");
                store.example(key, index);
              }}
            >
              {t("demo")}
            </button>
            <button
              type="submit"
              id={index === 3 ? "setup-plan" : "setup-save"}
              className={index === 3 && d.plan ? "" : "primary"}
            >
              {index === 3 ? t("reviewPlan") : t("saveNext")}
            </button>
          </div>
        </form>
        {index === 5 && <p className="notice">{t("history")}</p>}
        {index === 3 && (
          <div className="setup-plan">
            <p className="notice">{note ? note[l] : t("unverified")}</p>
            {blocked && (
              <p className="setup-blocked" role="status">
                {t("unverified")}
              </p>
            )}
            {!s("identity") && (
              <section className="setup-subsection">
                <h3>{t("handoff")}</h3>
                <p>
                  {t("identity")} · {t("missing")}
                </p>
                <p>{t("handoffNote")}</p>
              </section>
            )}
            {d.plan && (
              <section
                className="setup-subsection"
                aria-labelledby="setup-plan-title"
              >
                <h3 id="setup-plan-title">{t("plan")}</h3>
                <p>
                  {provider === "aws" && v.route === "new"
                    ? t("awsPlan")
                    : v.route === "new"
                      ? t("planNew")
                      : t("planExisting")}
                </p>
                <p className="setup-ref">
                  <bdi>
                    {p.name} · {s("account")} · revision {d.revision}
                  </bdi>
                </p>
                <Check
                  name="planAck"
                  label={t("planAck")}
                  checked={!!v.planAck}
                  onChange={(c) => store.setPlanAck(key, c)}
                />
                {v.route === "new" && (
                  <div className="setup-handoff">
                    <h3>{t("handoff")}</h3>
                    <p>
                      {t("owner")}: {p.name} administrator
                    </p>
                    <p>{t("handoffNote")}</p>
                    <p>
                      <Result tone={d.admin ? "ready" : "pending"}>
                        {d.admin ? t("pass") : t("waiting")}
                      </Result>
                    </p>
                    <button
                      id="setup-admin"
                      disabled={!v.planAck || d.admin}
                      onClick={() => {
                        restore("setup-connection-next");
                        store.adminReadback(key);
                      }}
                    >
                      {t("resumeCheck")}
                    </button>
                  </div>
                )}
                <div className="form-actions">
                  <button
                    id="setup-connection-next"
                    className="primary"
                    disabled={
                      !v.planAck || (v.route !== "existing" && !d.admin)
                    }
                    onClick={() => {
                      const next = store.connectionNext(key);
                      if (next) go(next);
                    }}
                  >
                    {t("saveNext")}
                  </button>
                </div>
              </section>
            )}
          </div>
        )}
      </section>
    );
  } else if (index === 4)
    body = (
      <section className="panel">
        <h2>{t("diagnostics")}</h2>
        <p className="setup-ref">
          <bdi>
            {s("account")} · revision {d.revision}
          </bdi>
        </p>
        {unconnected && <p className="notice">{t("notConnected")}</p>}
        <ul className="setup-checks" aria-live="polite">
          {["network", "metadata", "read"].map((k) =>
            check(t(k), ...diagnosticResult(k)),
          )}
        </ul>
        <div className="form-actions">
          <button
            id="setup-diagnose"
            className={d.diagnostic === "pass" ? "" : "primary"}
            disabled={unconnected || d.diagnostic === "running"}
            onClick={() => {
              restore("setup-diagnose");
              store.runDiagnostic(key);
            }}
          >
            {t("run")}
          </button>
          <button
            id="setup-next"
            className={d.diagnostic === "pass" ? "primary" : ""}
            disabled={d.diagnostic !== "pass" || unconnected}
            onClick={() => {
              const next = store.next(key, 4);
              if (next) go(next);
            }}
          >
            {t("saveNext")}
          </button>
        </div>
        <details className="scenario">
          <summary>{t("scenarios")}</summary>
          <div role="group" aria-label={t("scenario")}>
            {(["normal", "denied", "partial", "outcome"] as const).map(
              (name) => (
                <button
                  key={name}
                  id={`scenario-${name}`}
                  aria-pressed={v.scenario === name}
                  onClick={() => {
                    focusAfter.current = "";
                    store.setScenario(key, name);
                  }}
                >
                  {t(name)}
                </button>
              ),
            )}
          </div>
        </details>
      </section>
    );
  else if (index === 6)
    body = (
      <section className="panel">
        <h2>{t("receipt")}</h2>
        <p className="setup-ref">
          <bdi>
            {s("account")} · {s("period")} · {s("cadence")}
          </bdi>
        </p>
        {unconnected && <p className="notice">{t("notConnected")}</p>}
        <ul className="setup-checks" aria-live="polite">
          {["complete", "schema", "evidence"].map((k) =>
            check(t(k), ...receiptResult(k)),
          )}
        </ul>
        <div className="form-actions">
          <button
            id="setup-validate"
            className={d.receipt === "pass" ? "" : "primary"}
            disabled={unconnected || d.receipt === "running"}
            onClick={() => {
              restore("setup-validate");
              store.validateReceipt(key);
            }}
          >
            {t("validate")}
          </button>
          <button
            id="setup-next"
            className={d.receipt === "pass" ? "primary" : ""}
            disabled={d.receipt !== "pass" || unconnected}
            onClick={() => {
              const next = store.next(key, 6);
              if (next) go(next);
            }}
          >
            {t("saveNext")}
          </button>
        </div>
      </section>
    );
  else
    body = (
      <section className="panel">
        <h2>{t("readiness")}</h2>
        <dl className="facts">
          <div>
            <dt>{t("technical")}</dt>
            <dd>{d.diagnostic === "pass" ? t("pass") : t("idle")}</dd>
          </div>
          <div>
            <dt>{t("coverage")}</dt>
            <dd>{d.receipt === "pass" ? t("pass") : t("idle")}</dd>
          </div>
          <div>
            <dt>{t("financial")}</dt>
            <dd>{t("blocked")}</dd>
          </div>
          <div>
            <dt>{t("cadence")}</dt>
            <dd>{cadenceLabel}</dd>
          </div>
        </dl>
        <p className="notice">{t("evidence")}</p>
        <div className="form-actions">
          <button disabled>{t("publish")}</button>
          <button
            id="setup-work"
            className="primary"
            onClick={() => go("operations")}
          >
            {t("continueWork")}
          </button>
        </div>
      </section>
    );

  return (
    <div className="setup-wizard" data-version={version}>
      {steps}
      <div className="setup-content">
        <p className="notice setup-draft">{t("draft")}</p>
        {body}
      </div>
    </div>
  );
}
