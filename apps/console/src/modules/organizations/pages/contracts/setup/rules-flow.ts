import type { Attempt } from "../../../../../service/admin";
import type { Session } from "../../../../../service/api";
import { SetupError } from "./context";
import { protectedWriteIntent, type ProtectedWriteIntent } from "./protected-writes";
import type { ProtectedSummary } from "./protected";

export type RuleOffer = "offer" | "own" | "none";
export type RuleOffers = Readonly<{ propose: boolean; confirm: RuleOffer }>;
export function ruleOffers(summary: ProtectedSummary, user: string): RuleOffers {
  const r = summary.rules;
  return { propose: r.state === "none" && r.observation === "absent", confirm: r.state === "proposed" && r.observation === "pending" ? (r.proposer === user ? "own" : "offer") : "none" };
}

export const ruleBounds = { validity: [3600, 2592000], lead: [3600, 31622400], channelAge: [86400, 7776000] } as const;
export type RuleDuration = keyof typeof ruleBounds;
const refused = (): never => { throw new SetupError("invalid_setup_request"); };

export function secondsInput(value: string, unit: string, field: RuleDuration): string {
  const text = value.trim(), factor = unit === "hours" ? 3600 : unit === "days" ? 86400 : 0;
  if (!/^[0-9]{1,5}$/.test(text) || !factor) return refused();
  const seconds = Number(text) * factor, [least, most] = ruleBounds[field];
  return seconds >= least && seconds <= most ? String(seconds) : refused();
}

type Owner = Pick<Session, "user" | "workspace" | "csrf">;
const base = (session: Owner, summary: ProtectedSummary, attempt: Attempt) => ({ scope: summary.scope, actor: session.user, workspace: session.workspace, sessionCSRF: session.csrf, revision: summary.revision, attempt });

export function proposeRulesIntent(session: Owner, summary: ProtectedSummary, rules: string, draft: Readonly<Record<string, string>>, attempt: Attempt): ProtectedWriteIntent {
  if (!ruleOffers(summary, session.user).propose || !["1", "2", "3"].includes(draft.approvals ?? "")) return refused();
  const seconds = (field: RuleDuration) => secondsInput(draft[field] ?? "", draft[field + "Unit"] ?? "", field);
  return protectedWriteIntent({ ...base(session, summary, attempt), operation: "propose_protected_rules", rationale: (draft.reason ?? "").trim(),
    args: { rules, approvals: draft.approvals, validity_seconds: seconds("validity"), max_lead_seconds: seconds("lead"), channel_age_seconds: seconds("channelAge") } });
}

export function confirmRulesIntent(session: Owner, summary: ProtectedSummary, reason: string, attempt: Attempt): ProtectedWriteIntent {
  if (ruleOffers(summary, session.user).confirm !== "offer") return refused();
  return protectedWriteIntent({ ...base(session, summary, attempt), operation: "confirm_protected_rules", rationale: reason.trim(), args: { rules: summary.rules.id, rules_digest: summary.rules.digest } });
}
