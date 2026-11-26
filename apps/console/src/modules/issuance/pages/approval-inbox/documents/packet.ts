import { bytes, decimal, digest, DocumentError, freeze, id, revision, type FormContext } from "./contract";
export type LineInput = Readonly<{ quantity: string; tax: string; evidence: string }>;
export type PacketDraft = Readonly<{ selectedERP: boolean; unit: string; evidence: string; net: string; tax: string; rationale: string; lines: Readonly<Record<string, LineInput>> }>;
export type PacketLine = Readonly<{ Line: string; Product: string; Quantity: string; Unit: string; TaxCode: string; Tax: string; Evidence: string }>;
export type Packet = Readonly<{ Mode: "synthetic"; Profile: "synthetic_individual_net_v1"; Evidence: string; Request: string; RequestDigest: string; Connection: string; ExpectedNet: string; ExpectedTax: string; Lines: readonly PacketLine[] }>;
export class PacketError extends DocumentError {
  constructor(readonly fields: readonly string[]) { super("invalid_document_packet"); }
}
export function sum(values: readonly string[]): string {
  if (values.some(v => !decimal(v))) throw new PacketError(["amount"]);
  const scale = Math.max(0, ...values.map(v => v.split(".")[1]?.length ?? 0));
  const amount = values.reduce((n, v) => { const [whole, part = ""] = v.split("."); return n + BigInt(whole + part.padEnd(scale, "0")); }, 0n);
  if (!scale) return String(amount);
  const text = String(amount).padStart(scale + 1, "0");
  return `${text.slice(0, -scale)}.${text.slice(-scale)}`.replace(/0+$/, "").replace(/\.$/, "");
}
export function rationaleValid(v: string): boolean {
  return !!v.trim() && bytes(v) <= 2000 && [...v].every(c => c.length !== 1 || c.charCodeAt(0) < 0xd800 || c.charCodeAt(0) > 0xdfff);
}
export function emptyDraft(context: FormContext): PacketDraft {
  return freeze({ selectedERP: false, unit: "", evidence: "", net: "", tax: "", rationale: "", lines: Object.fromEntries(context.lines.map(l => [l.line, { quantity: "", tax: "", evidence: "" }])) });
}
export function packetOf(context: FormContext, draft: PacketDraft): Readonly<{ packet: Packet; text: string; rationale: string }> {
  const errors: string[] = [], scale = context.header.erp.scale;
  if (context.permission.state !== "allowed") throw new DocumentError("permission_denied");
  if (!id(context.request) || !digest(context.digest) || !revision(context.revision) || !context.lines.length || context.lines.length > 100) throw new PacketError(["context"]);
  if (!draft.selectedERP) errors.push("erp");
  if (!id(draft.unit)) errors.push("unit");
  if (!id(draft.evidence)) errors.push("evidence");
  if (!rationaleValid(draft.rationale)) errors.push("rationale");
  if (!decimal(draft.net, scale, true) || draft.net !== context.header.total || sum(context.lines.map(l => l.amount)) !== context.header.total) errors.push("net");
  if (!decimal(draft.tax, scale)) errors.push("tax");
  if (Object.keys(draft.lines).length !== context.lines.length || Object.keys(draft.lines).some(k => !context.lines.some(l => l.line === k))) errors.push("lines");
  const lines: PacketLine[] = [];
  for (const l of context.lines) {
    const d = Object.hasOwn(draft.lines, l.line) ? draft.lines[l.line] : undefined;
    if (!d || !decimal(d.quantity, scale, true) || d.quantity.split(".")[0].length > 12) errors.push(`quantity:${l.line}`);
    if (!d || !decimal(d.tax, scale) || d.tax.split(".")[0].length > 12) errors.push(`tax:${l.line}`);
    if (!d || !id(d.evidence)) errors.push(`evidence:${l.line}`);
    if (d) lines.push({ Line: l.line, Product: context.header.erp.product, Quantity: d.quantity, Unit: draft.unit, TaxCode: context.header.erp.tax_code, Tax: d.tax, Evidence: d.evidence });
  }
  if (!errors.some(k => k.startsWith("tax:") || k === "lines") && decimal(draft.tax, scale) && sum(lines.map(l => l.Tax)) !== draft.tax) errors.push("tax");
  if (errors.length) throw new PacketError(errors);
  const packet: Packet = { Mode: "synthetic", Profile: "synthetic_individual_net_v1", Evidence: draft.evidence, Request: context.request, RequestDigest: context.digest, Connection: context.header.erp.connection, ExpectedNet: draft.net, ExpectedTax: draft.tax, Lines: lines };
  const text = JSON.stringify(packet);
  if (bytes(text) > 4096) throw new PacketError(["packet_size"]);
  return freeze({ packet, text, rationale: draft.rationale });
}
