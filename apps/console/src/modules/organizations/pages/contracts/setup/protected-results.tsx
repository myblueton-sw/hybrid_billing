import type { ReactNode } from "react";
import type { Locale } from "../../../../../shared/types";
import { ltr } from "../live-format";
import { actionStatementNote, statementRequestNote, type ProtectedEntry, type StatementEntry } from "./channel-flow";
import { channelError, channelText, type ChannelWord } from "./channels-copy";
import { text, type SetupWord } from "./copy";
import { rulesError, rulesText } from "./rules-copy";
import { ruleActionOf } from "./RuleWrites";
import { statementError, statementText, type StatementWord } from "./statements-copy";

const channelActionOf: Readonly<Record<string, ChannelWord>> = { register_channel: "actRegister", admit_channel: "actAccept" };
type Described = Readonly<{ action: string; target: ReactNode; state?: ReactNode; stateLabel?: string; note?: ReactNode }>;
const note = (l: Locale, key: StatementWord | undefined) => key && statementText(l, key);

export function describeProtected(l: Locale, entry: ProtectedEntry): Described {
 const operation = entry.intent.operation, tag = (label: string) => <span className="o03-tag">{label}</span>;
 if (ruleActionOf[operation]) return { action: rulesText(l, ruleActionOf[operation]), target: <>{rulesText(l, "targetRules")}: {ltr(entry.subject)}</>, state: entry.receipt && tag(text(l, entry.receipt.state as SetupWord)) };
 if (channelActionOf[operation]) return { action: channelText(l, channelActionOf[operation]), target: <>{channelText(l, "targetChannel")}: {ltr(entry.subject)}</>, state: entry.receipt && tag(channelText(l, entry.receipt.state as ChannelWord)),
  note: note(l, actionStatementNote(entry)) };
 return { action: operation, target: ltr(entry.subject) };
}

export function explainProtected(l: Locale, code: string, entry?: ProtectedEntry): string {
 return entry && channelActionOf[entry.intent.operation] ? channelError(l, code) : rulesError(l, code);
}

export function describeStatement(l: Locale, entry: StatementEntry, actions: readonly ProtectedEntry[]): Described {
 const withdraw = entry.intent.action === "withdraw", after = statementRequestNote(entry, actions);
 return { action: statementText(l, withdraw ? "actWithdraw" : "actAttest") + ": " + statementText(l, entry.intent.purpose as StatementWord), target: <>{statementText(l, "targetStatement")}: {ltr(entry.intent.attestation)}</>,
  stateLabel: statementText(l, after === "usedInSession" ? "state" : "stateWhenRecorded"), note: after && after !== "usedInSession" ? statementText(l, after) : undefined,
  state: entry.receipt && (after === "usedInSession" ? statementText(l, "usedInSession") : entry.receipt.state === "consumed" ? <span className="o03-tag">{statementText(l, "consumed")}</span> : statementText(l, entry.receipt.state as StatementWord)) };
}

export const explainStatement = (l: Locale, code: string) => statementError(l, code);
