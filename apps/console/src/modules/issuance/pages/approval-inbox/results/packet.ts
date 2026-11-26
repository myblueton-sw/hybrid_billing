import { decimal } from "../documents/contract";
import { rationaleValid } from "../documents/packet";
import { bytes, freeze, id, ResultError, type Context } from "./contract";
export type RowDraft = Readonly<{ row: string; product: string; unit: string; quantity: string; net: string; tax: string }>;
export type PacketDraft = Readonly<{ evidence: string; external: string; revision: string; rationale: string; rows: Readonly<Record<string, RowDraft>> }>;
export type ResultPacket = Readonly<{ Mode: "synthetic"; Profile: "synthetic_result_one_to_one_v1"; Evidence: string; Preparation: string; PreparationDigest: string; Connection: string; External: string; Currency: string; Customer: string; Period: string; Revision: number; Rows: readonly Readonly<{ ID: string; Line: string; Product: string; Unit: string; Net: string; Tax: string; Quantity: string }>[] }>;
export class PacketError extends ResultError { constructor(readonly fields: readonly string[]) { super("invalid_result_packet"); } }
export function emptyDraft(context: Context): PacketDraft {
 return freeze({ evidence: "", external: "", revision: "", rationale: "", rows: Object.fromEntries(context.references.map(r => [r.line, { row: "", product: "", unit: "", quantity: "", net: "", tax: "" }])) });
}
export function packetOf(context: Context, draft: PacketDraft) {
 if (context.permission.state !== "allowed" || context.readiness.state !== "ready") throw new ResultError("prerequisite_failed");
 const errors: string[] = [];
 for (const k of ["evidence", "external"] as const) if (!id(draft[k])) errors.push(k);
 if (!/^[1-9][0-9]{0,9}$/.test(draft.revision) || Number(draft.revision) > 2147483647) errors.push("revision");
 if (!rationaleValid(draft.rationale)) errors.push("rationale");
 if (!context.references.length || context.references.length > 100 || Object.keys(draft.rows).length !== context.references.length || Object.keys(draft.rows).some(k => !context.references.some(r => r.line === k))) errors.push("rows");
 const rows: ResultPacket["Rows"][number][] = [], seen = new Set<string>();
 for (const ref of context.references) {
  const row = Object.hasOwn(draft.rows, ref.line) ? draft.rows[ref.line] : undefined;
  for (const key of ["row", "product", "unit"] as const) if (!row || !id(row[key])) errors.push(`${key}:${ref.line}`);
  for (const key of ["quantity", "net", "tax"] as const) if (!row || !decimal(row[key], 4, key === "quantity") || row[key].split(".")[0].length > 12) errors.push(`${key}:${ref.line}`);
  if (row) {
   if (seen.has(row.row)) errors.push(`row:${ref.line}`); seen.add(row.row);
   rows.push({ ID: row.row, Line: ref.line, Product: row.product, Unit: row.unit, Net: row.net, Tax: row.tax, Quantity: row.quantity });
  }
 }
 if (errors.length) throw new PacketError(errors);
 const p = context.preparation;
 const packet: ResultPacket = { Mode: "synthetic", Profile: "synthetic_result_one_to_one_v1", Evidence: draft.evidence, Preparation: p.preparation, PreparationDigest: p.preparation_digest, Connection: p.connection, External: draft.external, Currency: p.currency, Customer: p.customer, Period: p.period, Revision: Number(draft.revision), Rows: rows };
 const text = JSON.stringify(packet);
 if (bytes(text) > 4096) throw new PacketError(["packet_size"]);
 return freeze({ packet, text, rationale: draft.rationale });
}
