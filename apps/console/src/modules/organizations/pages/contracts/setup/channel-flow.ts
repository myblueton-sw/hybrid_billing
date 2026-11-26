import type { Attempt } from "../../../../../service/admin";
import type { Session } from "../../../../../service/api";
import { attestationIntent, type Attestation, type AttestationIntent, type AttestationPurpose, type AttestationReceipt, type SetupReference } from "./attestation";
import { SetupError } from "./context";
import type { ProtectedChannel, ProtectedScope } from "./protected";
import { protectedWriteIntent, type ProtectedWriteIntent, type ProtectedWriteReceipt } from "./protected-writes";
import type { StepEntry } from "./step-store";

export type StatementStep = AttestationIntent & Readonly<{ operation: string }>;
export type StatementEntry = StepEntry<StatementStep, AttestationReceipt>;
export type ProtectedEntry = StepEntry<ProtectedWriteIntent, ProtectedWriteReceipt>;
export type Offer = "offer" | "own" | "none";
type Owner = Pick<Session, "user" | "workspace" | "csrf">;
type Read = Readonly<{ scope: ProtectedScope; revision: string }>;
const refused = (): never => { throw new SetupError("invalid_setup_request"); };
const settled = (phase: string) => phase === "recorded" || phase === "reconciled";

export function channelOffer(row: ProtectedChannel, user: string): Offer {
  return row.state === "registered" && row.observation === "registered" ? (row.registrar === user ? "own" : "offer") : "none";
}

export function statementIntent(session: Owner, read: Read, purpose: AttestationPurpose, attestation: string, target: string, targetDigest: string, draft: Readonly<Record<string, string>>, attempt: Attempt): StatementStep {
  if ((draft.statement ?? "").trim() === "") return refused();
  const intent = attestationIntent({ action: "attest", purpose, scope: read.scope, attestation, target, targetDigest, statement: draft.statement, locator: (draft.locator ?? "").trim(), actor: session.user, workspace: session.workspace, sessionCSRF: session.csrf, revision: read.revision, attempt });
  return Object.freeze({ ...intent, operation: "attest_" + purpose });
}

export function withdrawIntent(session: Owner, read: Read, row: Attestation, attempt: Attempt): StatementStep {
  if (row.actor !== session.user || row.state !== "unconsumed") return refused();
  const intent = attestationIntent({ action: "withdraw", purpose: row.purpose as AttestationPurpose, scope: read.scope, attestation: row.attestation, target: "", targetDigest: "", statement: "", locator: "", actor: session.user, workspace: session.workspace, sessionCSRF: session.csrf, revision: read.revision, attempt });
  return Object.freeze({ ...intent, operation: "withdraw_" + row.purpose });
}

export function reusableStatement(statements: readonly StatementEntry[], actions: readonly ProtectedEntry[], scope: ProtectedScope, user: string, purpose: AttestationPurpose, target = "", targetDigest = ""): StatementEntry | undefined {
  const gone = new Set(statements.filter(e => e.intent.action === "withdraw" && e.phase !== "refused").map(e => e.intent.attestation));
  const used = new Set(actions.filter(e => e.phase !== "refused").map(e => e.intent.args.attestation));
  return statements.find(e => e.intent.action === "attest" && e.intent.purpose === purpose && settled(e.phase) && e.receipt?.state === "unconsumed" && e.intent.actor === user
    && e.intent.scope.contract === scope.contract && e.intent.scope.configuration === scope.configuration && e.intent.scope.digest === scope.digest
    && (target === "" || e.intent.target === target) && e.intent.targetDigest === targetDigest && !gone.has(e.intent.attestation) && !used.has(e.intent.attestation));
}

export function recordedStatement(entry: StatementEntry | undefined): Readonly<{ attestation: string; revision: string }> | undefined {
  return entry && entry.phase === "recorded" && entry.receipt && entry.receipt.state === "unconsumed" ? { attestation: entry.receipt.attestation, revision: entry.receipt.revision } : undefined;
}

export function actionStatementNote(entry: Pick<ProtectedEntry, "phase" | "recovers">): "notUsedNote" | "unknownUseNote" | undefined {
  if (entry.phase === "refused") return entry.recovers ? "unknownUseNote" : "notUsedNote";
  return entry.phase === "unknown" || entry.phase === "pending" ? "unknownUseNote" : undefined;
}

export function statementRequestNote(entry: StatementEntry, actions: readonly ProtectedEntry[]): "stepNotSent" | "stepNotSentRefused" | "usedInSession" | undefined {
  if (entry.intent.action !== "attest") return undefined;
  if (entry.phase === "refused") return "stepNotSentRefused";
  if (entry.phase === "pending" || entry.phase === "unknown") return "stepNotSent";
  return actions.some(a => settled(a.phase) && a.intent.args.attestation === entry.intent.attestation) ? "usedInSession" : undefined;
}

export function channelReferences(entries: readonly SetupReference[], subject: string): readonly SetupReference[] {
  return entries.filter(e => e.use === "channel_" + subject);
}

const base = (session: Owner, read: Read, revision: string, attempt: Attempt) => ({ scope: read.scope, actor: session.user, workspace: session.workspace, sessionCSRF: session.csrf, revision, attempt });

export function registerChannelIntent(session: Owner, read: Read, revision: string, channel: string, attestation: string, draft: Readonly<Record<string, string>>, references: readonly SetupReference[], attempt: Attempt): ProtectedWriteIntent {
  const subject = draft.subject ?? "", reference = draft.reference ?? "";
  if (!channelReferences(references, subject).some(e => e.code === reference)) return refused();
  return protectedWriteIntent({ ...base(session, read, revision, attempt), operation: "register_channel", rationale: (draft.reason ?? "").trim(), args: { channel, subject_kind: subject, kind: draft.kind ?? "", reference, attestation } });
}

export function admitChannelIntent(session: Owner, read: Read, revision: string, row: ProtectedChannel, attestation: string, attempt: Attempt): ProtectedWriteIntent {
  if (channelOffer(row, session.user) !== "offer") return refused();
  return protectedWriteIntent({ ...base(session, read, revision, attempt), operation: "admit_channel", rationale: "", args: { channel: row.channel, channel_digest: row.digest, attestation } });
}
