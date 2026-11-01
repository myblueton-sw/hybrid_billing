import { profiles } from "./profiles";
import type { Mode } from "../../shared/types";

export const setupRoutes = [
  "setup-progress",
  "setup-profile",
  "setup-attribution",
  "setup-connectors",
  "setup-readtest",
  "setup-schedule",
  "setup-first",
  "setup-readiness",
] as const;
export const stepKeys = [
  "start",
  "profile",
  "attribution",
  "connector",
  "diagnostics",
  "schedule",
  "receipt",
  "readiness",
];
export type Check = "idle" | "running" | "pass" | "failed" | "unknown";
export type Receipt = "idle" | "running" | "pass" | "partial";
export type Values = Record<string, string | boolean | undefined>;
export type Draft = {
  values: Values;
  done: Set<string>;
  revision: number;
  plan: boolean;
  admin: boolean;
  diagnostic: Check;
  receipt: Receipt;
  run: number;
};
export type Timer = (fn: () => void, ms: number) => unknown;

export const draftKey = (scope: string | null, mode: Mode) =>
  `${scope === "sample-b" ? "sample-b" : "sample-a"}:${mode}`;

const str = (v: Values, key: string) =>
  typeof v[key] === "string" ? (v[key] as string) : "";

export function profileOf(v: Values) {
  return profiles[str(v, "provider")] ?? profiles.aws;
}
export function blockedProfile(v: Values) {
  const provider = str(v, "provider");
  return (
    !!profileOf(v).unverified ||
    (provider === "azure" && !["EA", "MCA"].includes(str(v, "agreement"))) ||
    (provider === "gcp" &&
      (!v.preview || !["US", "EU"].includes(str(v, "location")))) ||
    (provider === "alibaba" && !v.preview) ||
    (provider === "ibm" && str(v, "ibmRoute") !== "csv")
  );
}
export function requiredFields(v: Values) {
  return str(v, "provider") === "aws" && str(v, "route") === "new"
    ? []
    : profileOf(v).fields;
}
export function accountValid(v: Values) {
  return (
    !!str(v, "account") &&
    (str(v, "provider") !== "aws" || /^\d{12}$/.test(str(v, "account")))
  );
}
export function configurationReady(v: Values) {
  return (
    accountValid(v) &&
    requiredFields(v).every((f) => !!str(v, f.key)) &&
    !!str(v, "identity") &&
    !blockedProfile(v)
  );
}
export function canStep(d: Draft, i: number) {
  return i <= 1 || d.done.has(setupRoutes[i - 1]);
}
export function resumeTarget(d: Draft) {
  return (
    setupRoutes.find((route, i) => i > 0 && !d.done.has(route)) ??
    "setup-readiness"
  );
}
export function cadenceOptions(v: Values) {
  return ["file", "ibm"].includes(str(v, "provider"))
    ? ["once"]
    : ["daily", "monthly"];
}
const required: Record<number, string[]> = {
  1: ["entity", "country", "currency", "timezone", "tax"],
  2: ["customer", "custody", "effective"],
  5: ["period", "cadence"],
};
export function stepValid(v: Values, index: number) {
  if (!(required[index] ?? []).every((key) => !!str(v, key))) return false;
  if (index === 2) return !!v.scopeAck;
  if (index === 5)
    return (
      /^\d{4}-\d{2}$/.test(str(v, "period")) &&
      cadenceOptions(v).includes(str(v, "cadence"))
    );
  return true;
}
const providerKeys = Object.values(profiles).flatMap((p) =>
  p.fields.map((f) => f.key),
);

export function createStore(
  timer: Timer = (fn, ms) => setTimeout(fn, ms),
  delay = 450,
) {
  const drafts = new Map<string, Draft>(),
    listeners = new Set<() => void>();
  let version = 0;
  const emit = () => {
    version++;
    listeners.forEach((l) => l());
  };
  function get(key: string) {
    if (!drafts.has(key))
      drafts.set(key, {
        values: {
          provider: "aws",
          route: "existing",
          identity: "",
          agreement: "MCA",
          ibmRoute: "csv",
          scenario: "normal",
        },
        done: new Set(),
        revision: 1,
        plan: false,
        admin: false,
        diagnostic: "idle",
        receipt: "idle",
        run: 0,
      });
    return drafts.get(key)!;
  }
  function invalidate(d: Draft, from: number) {
    for (let i = from; i < setupRoutes.length; i++)
      d.done.delete(setupRoutes[i]);
    d.revision++;
    d.run++;
    if (from <= 3) {
      d.plan = false;
      d.admin = false;
      d.values.planAck = false;
    }
    if (from <= 4) d.diagnostic = "idle";
    if (from <= 6) d.receipt = "idle";
  }
  function complete(d: Draft, index: number) {
    d.done.add(setupRoutes[index]);
    emit();
    return setupRoutes[Math.min(index + 1, 7)];
  }
  return {
    subscribe(listener: () => void) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    version: () => version,
    get,
    setField(
      key: string,
      index: number,
      name: string,
      value: string | boolean,
    ) {
      const d = get(key),
        v = d.values;
      if (name === "provider") {
        for (const k of providerKeys) delete v[k];
        Object.assign(v, {
          account: "",
          preview: false,
          identity: "",
          route: "existing",
          ibmRoute: "csv",
        });
      }
      v[name] = value;
      invalidate(d, index);
      emit();
    },
    example(key: string, index: number) {
      const d = get(key),
        v = d.values;
      if (index === 1)
        Object.assign(v, {
          entity: "Example Billing LLC",
          country: "AE",
          currency: "AED",
          timezone: "Asia/Dubai",
          tax: "DRAFT-TAX-POLICY",
        });
      if (index === 2)
        Object.assign(v, {
          customer: "CUSTOMER-DEMO-001",
          custody: "CUSTODIAN-DEMO-001",
          effective: "2026-10-01",
          scopeAck: true,
        });
      if (index === 3) {
        const p = profileOf(v);
        v.account = p.sample;
        v.identity = "DEMO-READ-IDENTITY";
        for (const f of p.fields) v[f.key] = f.sample;
        v.preview = true;
      }
      if (index === 5)
        Object.assign(v, {
          period: "2026-10",
          cadence: ["ibm", "file"].includes(str(v, "provider"))
            ? "once"
            : "daily",
        });
      invalidate(d, index);
      emit();
    },
    submit(key: string, index: number): string | null {
      const d = get(key);
      if (!canStep(d, index) || !stepValid(d.values, index)) return null;
      return complete(d, index);
    },
    reviewPlan(key: string): "ok" | "blocked" | "invalid" {
      const d = get(key);
      if (!canStep(d, 3) || !configurationReady(d.values))
        return blockedProfile(d.values) ? "blocked" : "invalid";
      d.plan = true;
      emit();
      return "ok";
    },
    setPlanAck(key: string, checked: boolean) {
      const d = get(key);
      invalidate(d, 3);
      d.plan = true;
      d.values.planAck = checked;
      emit();
    },
    adminReadback(key: string) {
      const d = get(key);
      if (
        canStep(d, 3) &&
        d.plan &&
        d.values.planAck &&
        configurationReady(d.values)
      ) {
        d.admin = true;
        emit();
      }
    },
    connectionNext(key: string): string | null {
      const d = get(key),
        v = d.values;
      if (
        canStep(d, 3) &&
        d.plan &&
        v.planAck &&
        configurationReady(v) &&
        (v.route === "existing" || d.admin)
      )
        return complete(d, 3);
      return null;
    },
    setScenario(key: string, scenario: string) {
      const d = get(key);
      d.values.scenario = scenario;
      invalidate(d, 4);
      emit();
    },
    runDiagnostic(key: string) {
      const d = get(key),
        v = d.values;
      if (!canStep(d, 4) || !configurationReady(v) || !d.plan || !v.planAck)
        return;
      invalidate(d, 4);
      const run = ++d.run;
      d.diagnostic = "running";
      emit();
      timer(() => {
        if (d.run !== run) return;
        d.diagnostic =
          v.scenario === "denied"
            ? "failed"
            : v.scenario === "outcome"
              ? "unknown"
              : "pass";
        emit();
      }, delay);
    },
    validateReceipt(key: string) {
      const d = get(key);
      if (!canStep(d, 6) || d.diagnostic !== "pass") return;
      invalidate(d, 6);
      const run = ++d.run;
      d.receipt = "running";
      emit();
      timer(() => {
        if (d.run !== run) return;
        d.receipt = d.values.scenario === "partial" ? "partial" : "pass";
        emit();
      }, delay);
    },
    next(key: string, index: number): string | null {
      const d = get(key);
      if (
        canStep(d, index) &&
        ((index === 4 && d.diagnostic === "pass") ||
          (index === 6 && d.receipt === "pass"))
      )
        return complete(d, index);
      return null;
    },
    dispose() {
      for (const d of drafts.values()) {
        d.run++;
        if (d.diagnostic === "running") d.diagnostic = "idle";
        if (d.receipt === "running") d.receipt = "idle";
      }
      emit();
    },
  };
}
export const store = createStore();
