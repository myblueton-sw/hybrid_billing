import type { Attempt } from "../../../../../service/admin";
import type { Session } from "../../../../../service/api";
import type { SetupReference } from "./attestation";
import type { Offer } from "./channel-flow";
import { SetupError } from "./context";
import type { ProtectedChange, ProtectedChannel, ProtectedField, ProtectedScope } from "./protected";
import { protectedWriteIntent, type ProtectedWriteIntent } from "./protected-writes";

type Owner = Pick<Session, "user" | "workspace" | "csrf">;
type Read = Readonly<{ scope: ProtectedScope; revision: string }>;
const refused = (): never => { throw new SetupError("invalid_setup_request"); };
export const subjectOf: Readonly<Record<ProtectedField, string>> = { document_recipient: "customer", document_template: "document_kind", payment_account: "entity" };

export function verifyOffer(row: ProtectedChange, user: string): Offer {
  return row.phase === "recorded" && row.state === "requested" && row.observation === "requested" ? (row.requester === user ? "own" : "offer") : "none";
}

export type DecideOffer = "offer" | "requester" | "verifier" | "decided" | "none";
export function decideOffer(row: ProtectedChange, user: string): DecideOffer {
  if (row.phase !== "recorded" || row.state !== "confirmed" || row.observation !== "confirmed") return "none";
  return row.requester === user ? "requester" : row.verifier === user ? "verifier" : row.decisions.some(d => d.approver === user) ? "decided" : "offer";
}

export function usableChannels(channels: readonly ProtectedChannel[], field: ProtectedField, user = ""): readonly ProtectedChannel[] {
  return channels.filter(c => c.subject_kind === subjectOf[field] && c.state === "admitted" && c.observation === "admitted" && c.registrar !== user && c.admitter !== user);
}

export function effectiveInput(value: string): string {
  const text = value.trim();
  return /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(text) && Number.isFinite(Date.parse(text + ":00Z")) && new Date(text + ":00Z").toISOString().slice(0, 16) === text ? text + ":00Z" : refused();
}

const base = (session: Owner, read: Read, revision: string, attempt: Attempt) => ({ scope: read.scope, actor: session.user, workspace: session.workspace, sessionCSRF: session.csrf, revision, attempt });

export function requestChangeIntent(session: Owner, read: Read, revision: string, change: string, attestation: string, field: ProtectedField, draft: Readonly<Record<string, string>>, references: readonly SetupReference[], channels: readonly ProtectedChannel[], attempt: Attempt): ProtectedWriteIntent {
  const usable = usableChannels(channels, field, session.user), channel = usable.find(c => c.channel === draft.channel), instruction = draft.instruction ?? "";
  if (!channel || !references.some(e => e.use === field && e.code === draft.reference)) return refused();
  const args: Record<string, string> = { change, field, reference: draft.reference, channel: channel.channel, channel_digest: channel.digest, provenance: draft.provenance ?? "", instruction, effective: effectiveInput(draft.effective ?? ""), attestation };
  if (instruction === "registered_channel") {
    if (!usableChannels(channels, field).some(c => c.channel === draft.instructionChannel && c.channel !== channel.channel)) return refused();
    args.instruction_channel = draft.instructionChannel;
  } else if ((draft.instructionChannel ?? "") !== "") return refused();
  return protectedWriteIntent({ ...base(session, read, revision, attempt), operation: "request_change", rationale: (draft.reason ?? "").trim(), args });
}

export function verifyChangeIntent(session: Owner, read: Read, revision: string, row: ProtectedChange, attestation: string, outcome: string, attempt: Attempt): ProtectedWriteIntent {
  if (verifyOffer(row, session.user) !== "offer") return refused();
  return protectedWriteIntent({ ...base(session, read, revision, attempt), operation: "verify_change", rationale: "", args: { change: row.change, change_digest: row.digest, outcome, attestation } });
}

export function decideChangeIntent(session: Owner, read: Read, row: ProtectedChange, draft: Readonly<Record<string, string>>, attempt: Attempt): ProtectedWriteIntent {
  if (decideOffer(row, session.user) !== "offer") return refused();
  const args: Record<string, string> = { change: row.change, change_digest: row.digest, verdict: draft.verdict ?? "" };
  if (draft.verdict === "reject") args.reason = draft.rejection ?? "";
  else if ((draft.rejection ?? "") !== "") return refused();
  return protectedWriteIntent({ ...base(session, read, read.revision, attempt), operation: "decide_change", rationale: "", args });
}
