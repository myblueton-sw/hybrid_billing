import { recordedStatement, type StatementStep } from "./channel-flow";
import type { ProtectedWriteIntent, ProtectedWriteReceipt } from "./protected-writes";
import type { AttestationReceipt } from "./attestation";
import type { StepWriteStore } from "./step-store";

export type ChainedStep = Readonly<{ subject: string; form: string; statement?: StatementStep; reuse?: Readonly<{ id: string; text: string; locator: string }>; build: (attestation: string, revision: string) => ProtectedWriteIntent }>;
export type ChainOutcome = "sent" | "statement_unsettled";

export async function sendChained(statements: StepWriteStore<StatementStep, AttestationReceipt>, actions: StepWriteStore<ProtectedWriteIntent, ProtectedWriteReceipt>, step: ChainedStep, revision: string): Promise<ChainOutcome> {
  let attestation = step.reuse?.id ?? "", current = revision;
  if (step.statement) {
    await statements.submit(step.statement, step.statement.attestation);
    const sent = recordedStatement(statements.snapshot().entries.find(e => e.intent === step.statement));
    if (!sent) return "statement_unsettled";
    attestation = sent.attestation; current = sent.revision;
  }
  await actions.submit(step.build(attestation, current), step.subject, step.form);
  return "sent";
}
