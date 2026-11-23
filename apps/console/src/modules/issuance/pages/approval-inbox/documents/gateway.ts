import type { Session } from "../../../../../service/api";
import type { DocumentRow, PreparationRow, RequestIdentity } from "./contract";
import { DocumentError } from "./contract";
import { adviceOf, inspectionOf, preparationInspectionOf } from "./inspection";
import { loadContext, loadHistory } from "./reads";
import { documentTransport } from "./transport";
export function documentGateway(session: Session, f: typeof fetch = fetch.bind(globalThis)) {
 const client = documentTransport(session, f);
 return {
  context: (scope: RequestIdentity) => loadContext(client.read, scope),
  history: (scope: RequestIdentity, kind: "documents" | "preparations") => loadHistory(client.read, scope, kind),
  async document(scope: RequestIdentity, row: DocumentRow) {
   const inspection = inspectionOf(await client.read("document_evidence.inspect", row.document, { digest: row.digest }), scope, row);
   const advice = adviceOf(await client.read("document_evidence.advice", row.document, { digest: row.digest }), inspection.row);
   if (inspection.revision !== advice.revision) throw new DocumentError("revision_conflict");
   return { inspection, advice };
  },
  async preparation(row: PreparationRow) { return preparationInspectionOf(await client.read("erp_preparation.inspect", row.preparation, {}), row); },
  submit: client.submit,
 };
}
export type DocumentGateway = ReturnType<typeof documentGateway>;
