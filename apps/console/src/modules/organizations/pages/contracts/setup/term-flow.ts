import type { Attempt } from "../../../../../service/admin";
import type { Session } from "../../../../../service/api";
import { SetupError } from "./context";
import type { ProtectedScope } from "./protected";
import { termWriteIntent, type PaymentTerms, type TermBinding, type TermWriteIntent } from "./terms";

export type TermOffer = "offer" | "own" | "none";
export type TermOffers = Readonly<{ record: boolean; approveTerm: TermOffer; apply: boolean; bindings: Readonly<Record<string, TermOffer>> }>;

export function termOffers(page: PaymentTerms, user: string): TermOffers {
  const t = page.term, by = (author: string): TermOffer => author === user ? "own" : "offer";
  const bindings: Record<string, TermOffer> = {};
  for (const r of page.rows) bindings[r.binding] = r.state === "pending" && r.observation === "pending" && r.origin === "configuration" && r.configuration === page.scope.configuration ? by(r.author) : "none";
  return { record: !t, approveTerm: t && t.state === "pending" && t.observation === "pending" ? by(t.author) : "none", apply: !!t && t.state === "approved" && t.observation === "approved", bindings };
}

type Owner = Pick<Session, "user" | "workspace" | "csrf">;
const base = (session: Owner, scope: ProtectedScope, revision: string, attempt: Attempt) => ({ scope, actor: session.user, workspace: session.workspace, sessionCSRF: session.csrf, revision, attempt });
const refused = (): never => { throw new SetupError("invalid_setup_request"); };

export function netDaysInput(value: string): string {
  const text = value.trim();
  return /^[0-9]{1,3}$/.test(text) && Number(text) <= 365 ? String(Number(text)) : refused();
}

export function recordTermIntent(session: Owner, page: PaymentTerms, term: string, draft: Readonly<Record<string, string>>, attempt: Attempt): TermWriteIntent {
  if (!termOffers(page, session.user).record) return refused();
  return termWriteIntent({ ...base(session, page.scope, page.revision, attempt), operation: "record_term", args: { term, net_days: netDaysInput(draft.days ?? "") }, rationale: (draft.reason ?? "").trim() });
}

export function approveTermIntent(session: Owner, page: PaymentTerms, attempt: Attempt): TermWriteIntent {
  if (!page.term || termOffers(page, session.user).approveTerm !== "offer") return refused();
  return termWriteIntent({ ...base(session, page.scope, page.revision, attempt), operation: "approve_term", args: { term: page.term.id, term_digest: page.term.digest }, rationale: "" });
}

export function bindTermIntent(session: Owner, page: PaymentTerms, binding: string, draft: Readonly<Record<string, string>>, attempt: Attempt): TermWriteIntent {
  if (!page.term || !termOffers(page, session.user).apply || !["date", "open"].includes(draft.end ?? "") || draft.end === "open" && (draft.to ?? "") !== "") return refused();
  const args: Record<string, string> = { binding, term: page.term.id, term_digest: page.term.digest, from: draft.from ?? "", reason: draft.reason ?? "" };
  if (draft.end === "date") args.to = draft.to ?? "";
  return termWriteIntent({ ...base(session, page.scope, page.revision, attempt), operation: "bind_term", args, rationale: "" });
}

export function approveBindingIntent(session: Owner, page: PaymentTerms, row: TermBinding, attempt: Attempt): TermWriteIntent {
  if (termOffers(page, session.user).bindings[row.binding] !== "offer" || !page.rows.includes(row)) return refused();
  return termWriteIntent({ ...base(session, page.scope, page.revision, attempt), operation: "approve_term_binding", args: { binding: row.binding, binding_digest: row.digest }, rationale: "" });
}
