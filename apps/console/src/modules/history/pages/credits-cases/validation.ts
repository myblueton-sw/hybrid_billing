import {
  views,
  type Data,
  type EvidenceRecord,
  type Authority,
  type Dimension,
  type Money,
  type Hold,
  type PaymentRecord,
  type View,
} from "./contract";
const hash = (v: unknown): v is string =>
  typeof v === "string" && /^[a-f0-9]{64}$/.test(v);
const text = (v: unknown, max = 4096): v is string =>
  typeof v === "string" &&
  v.length > 0 &&
  v.length <= max &&
  !/[\u0000-\u001f\u007f]/.test(v);
const nullable = (v: unknown, max = 4096) => v === null || text(v, max);
const id = (v: unknown): v is string =>
  typeof v === "string" && /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(v);
const cursor = (v: unknown): v is string =>
  typeof v === "string" && /^[A-Za-z0-9_-]{1,128}$/.test(v);
const count = (v: unknown): v is number =>
  Number.isSafeInteger(v) && Number(v) >= 0 && Number(v) <= 1000000000;
const exact = (v: unknown, names: string[]) =>
  !!v &&
  typeof v === "object" &&
  !Array.isArray(v) &&
  Object.keys(v).length === names.length &&
  names.every((k) => Object.hasOwn(v, k));
function iso(v: unknown): v is string {
  if (!text(v, 64)) return false;
  const m =
    /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?(Z|[+-](\d{2}):(\d{2}))$/.exec(
      v,
    );
  if (
    !m ||
    m[1].startsWith("0000") ||
    Number(m[2]) > 23 ||
    Number(m[3]) > 59 ||
    Number(m[4]) > 59 ||
    (m[5] !== "Z" && (Number(m[6]) > 23 || Number(m[7]) > 59))
  )
    return false;
  const day = Date.parse(m[1] + "T00:00:00Z");
  return (
    Number.isFinite(day) &&
    new Date(day).toISOString().slice(0, 10) === m[1] &&
    Number.isFinite(Date.parse(v))
  );
}
const optionalTime = (v: unknown) => v === null || iso(v);
export function validDecimal(v: unknown): v is string {
  if (
    typeof v !== "string" ||
    v.length > 80 ||
    !/^-?(?:0|[1-9]\d*)(?:\.\d+)?$/.test(v)
  )
    return false;
  const digits = v.replace(/[-.]/g, "");
  return digits.length <= 76 && (v.split(".")[1]?.length ?? 0) <= 38;
}

const plain = (v: unknown, max: number) =>
  v === null ||
  (typeof v === "string" &&
    v.length > 0 &&
    v.length <= max &&
    !/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/.test(v));
const decimal = (v: unknown) => v === null || validDecimal(v);
const reference = (v: unknown) => v === null || id(v);
const currency = (v: unknown): v is string =>
  typeof v === "string" && /^[A-Z]{3}$/.test(v);
function authority(a: Authority) {
  return (
    exact(a, ["kind", "reference", "observed_at", "freshness"]) &&
    ["platform", "external", "unknown"].includes(a.kind) &&
    reference(a.reference) &&
    optionalTime(a.observed_at) &&
    ["current", "stale", "unknown"].includes(a.freshness)
  );
}
function dimension(d: Dimension) {
  return (
    exact(d, ["kind", "currency", "unit"]) &&
    ((d.kind === "money" && currency(d.currency) && d.unit === null) ||
      (d.kind === "quantity" && d.currency === null && text(d.unit, 128)))
  );
}
function money(m: Money) {
  return (
    exact(m, ["currency", "value"]) && currency(m.currency) && decimal(m.value)
  );
}
function moneyArray(v: Money[]) {
  return (
    Array.isArray(v) &&
    v.length <= 16 &&
    new Set(v.map((m) => m?.currency)).size === v.length &&
    v.every(money)
  );
}
function holds(v: Hold[]) {
  return (
    Array.isArray(v) &&
    v.length <= 16 &&
    v.every(
      (h) =>
        exact(h, [
          "type",
          "effect",
          "reference",
          "revision",
          "target_ref",
          "observed_at",
          "external_effect",
        ]) &&
        ["billing", "delivery", "collections"].includes(h.type) &&
        ["requested", "effective", "released", "pending", "unknown"].includes(
          h.effect,
        ) &&
        reference(h.reference) &&
        nullable(h.revision, 256) &&
        reference(h.target_ref) &&
        optionalTime(h.observed_at) &&
        ["confirmed", "pending", "unknown", "not_applicable"].includes(
          h.external_effect,
        ) &&
        (!["effective", "released"].includes(h.effect) ||
          (h.reference !== null &&
            h.revision !== null &&
            h.target_ref !== null &&
            h.observed_at !== null &&
            ["confirmed", "not_applicable"].includes(h.external_effect))),
    )
  );
}
export function validDate(v: unknown): v is string {
  if (
    typeof v !== "string" ||
    !/^[0-9]{4}-[0-9]{2}-[0-9]{2}$/.test(v) ||
    v.startsWith("0000")
  )
    return false;
  const value = Date.parse(v + "T00:00:00Z");
  return (
    Number.isFinite(value) && new Date(value).toISOString().slice(0, 10) === v
  );
}
export function validTimeZone(v: unknown): v is string {
  if (
    !text(v, 128) ||
    !(v === "UTC" || /^[A-Za-z][A-Za-z0-9_+-]*(?:\/[A-Za-z0-9_+-]+)+$/.test(v))
  )
    return false;
  try {
    new Intl.DateTimeFormat("en", { timeZone: v });
    return true;
  } catch {
    return false;
  }
}
function payment(r: PaymentRecord) {
  if (
    !reference(r.original_document_ref) ||
    !["issued", "not_issued", "internal", "unknown"].includes(r.issuance) ||
    !reference(r.term_ref) ||
    !nullable(r.term_label, 256) ||
    !(r.time_zone === null || validTimeZone(r.time_zone)) ||
    ![r.issue_date, r.original_due_date, r.effective_due_date].every(
      (d) => d === null || validDate(d),
    ) ||
    !money(r.receivable) ||
    !exact(r.receipt, ["state", "reference", "observed_at", "freshness"]) ||
    !["confirmed", "unknown", "not_provided"].includes(r.receipt.state) ||
    !reference(r.receipt.reference) ||
    !optionalTime(r.receipt.observed_at) ||
    !["current", "stale", "unknown"].includes(r.receipt.freshness) ||
    ![
      "on_time",
      "due_today",
      "overdue",
      "fully_paid",
      "unknown",
      "not_applicable",
    ].includes(r.overdue) ||
    !["open", "none", "unknown"].includes(r.dispute) ||
    !holds(r.holds)
  )
    return false;
  if (r.overdue === "not_applicable")
    return ["internal", "not_issued"].includes(r.issuance);
  if (r.overdue === "unknown") return true;
  const value = r.receivable.value;
  if (
    r.issuance !== "issued" ||
    r.original_document_ref === null ||
    r.term_ref === null ||
    r.term_label === null ||
    r.time_zone === null ||
    r.issue_date === null ||
    r.original_due_date === null ||
    r.effective_due_date === null ||
    r.authority.kind === "unknown" ||
    r.authority.freshness !== "current" ||
    r.authority.reference === null ||
    r.authority.observed_at === null ||
    r.receipt.state !== "confirmed" ||
    r.receipt.freshness !== "current" ||
    r.receipt.reference === null ||
    r.receipt.observed_at === null ||
    value === null ||
    (value.startsWith("-") && !/^-0(?:\.0+)?$/.test(value))
  )
    return false;
  const zero = /^-?0(?:\.0+)?$/.test(value);
  return r.overdue === "fully_paid" ? zero : !zero;
}
const common = [
  "id",
  "revision",
  "label",
  "target",
  "updated_at",
  "authority",
  "blockers",
];
const fields = {
  balance: [
    "dimension",
    "opening",
    "increases",
    "applied",
    "expired",
    "reversed",
    "closing",
    "available",
    "reserved",
    "migration_ref",
    "included_history_ref",
  ],
  reservation: [
    "dimension",
    "value",
    "credit_ref",
    "document_ref",
    "original_ref",
    "state",
  ],
  case: [
    "state",
    "original_document_ref",
    "amounts",
    "owner_label",
    "received_at",
    "deadline",
    "internal_note",
    "public_reply",
    "resolution",
    "resolution_ref",
    "consent",
    "holds",
  ],
  adjustment: [
    "original_document_ref",
    "original_claim_ref",
    "correction_ref",
    "reason",
    "evidence_ref",
    "approval_ref",
    "adjustment_kind",
    "state",
    "amounts",
  ],
  payment: [
    "original_document_ref",
    "issuance",
    "term_ref",
    "term_label",
    "time_zone",
    "issue_date",
    "original_due_date",
    "effective_due_date",
    "receivable",
    "receipt",
    "overdue",
    "dispute",
    "holds",
  ],
};
function record(r: EvidenceRecord) {
  if (
    !r ||
    !Object.hasOwn(fields, r.kind) ||
    !exact(r, [...common, "kind", ...fields[r.kind]]) ||
    !id(r.id) ||
    !nullable(r.revision, 256) ||
    !text(r.label, 512) ||
    !exact(r.target, ["entity", "customer", "contract", "period"]) ||
    ![r.target.entity, r.target.customer, r.target.contract].every((v) =>
      nullable(v, 256),
    ) ||
    !nullable(r.target.period, 128) ||
    !iso(r.updated_at) ||
    !authority(r.authority) ||
    !Array.isArray(r.blockers) ||
    r.blockers.length > 16 ||
    new Set(r.blockers.map((b) => b?.code)).size !== r.blockers.length ||
    !r.blockers.every(
      (b) => exact(b, ["code", "text"]) && id(b.code) && text(b.text, 1024),
    )
  )
    return false;
  switch (r.kind) {
    case "balance":
      return (
        dimension(r.dimension) &&
        [
          r.opening,
          r.increases,
          r.applied,
          r.expired,
          r.reversed,
          r.closing,
          r.available,
          r.reserved,
        ].every(decimal) &&
        reference(r.migration_ref) &&
        reference(r.included_history_ref)
      );
    case "reservation":
      return (
        dimension(r.dimension) &&
        decimal(r.value) &&
        [r.credit_ref, r.document_ref, r.original_ref].every(reference) &&
        [
          "reserved",
          "applied",
          "released",
          "outcome_unknown",
          "reversed",
          "unknown",
        ].includes(r.state)
      );
    case "case":
      return (
        ["open", "in_review", "resolved", "closed", "unknown"].includes(
          r.state,
        ) &&
        reference(r.original_document_ref) &&
        moneyArray(r.amounts) &&
        nullable(r.owner_label, 256) &&
        optionalTime(r.received_at) &&
        optionalTime(r.deadline) &&
        plain(r.internal_note, 4096) &&
        plain(r.public_reply, 4096) &&
        [
          "confirmed_normal",
          "correction_requested",
          "corrected",
          "unresolved",
          "unknown",
        ].includes(r.resolution) &&
        reference(r.resolution_ref) &&
        exact(r.consent, ["state", "evidence_ref"]) &&
        [
          "recorded",
          "declined",
          "not_recorded",
          "not_applicable",
          "unknown",
        ].includes(r.consent.state) &&
        reference(r.consent.evidence_ref) &&
        (r.consent.state !== "recorded" || r.consent.evidence_ref !== null) &&
        holds(r.holds)
      );
    case "adjustment":
      return (
        [
          r.original_document_ref,
          r.original_claim_ref,
          r.correction_ref,
          r.evidence_ref,
          r.approval_ref,
        ].every(reference) &&
        plain(r.reason, 1024) &&
        ["credit", "refund", "quantity_reversal", "other", "unknown"].includes(
          r.adjustment_kind,
        ) &&
        ["draft", "pending", "applied", "rejected", "unknown"].includes(
          r.state,
        ) &&
        moneyArray(r.amounts) &&
        (r.state !== "applied" ||
          r.original_document_ref !== null ||
          r.original_claim_ref !== null)
      );
    case "payment":
      return payment(r);
  }
}
export function validQuery(q: URLSearchParams) {
  return (
    ![...q.keys()].some(
      (k) =>
        !["view", "q", "cursor", "snapshot", "permission_revision"].includes(k),
    ) &&
    new Set(q.keys()).size === [...q.keys()].length &&
    views.includes(q.get("view") as View) &&
    q.has("q") &&
    (q.get("q") ?? "").length <= 128 &&
    !/[\u0000-\u001f\u007f]/.test(q.get("q") ?? "") &&
    (!q.has("snapshot") || hash(q.get("snapshot"))) &&
    (!q.has("permission_revision") || hash(q.get("permission_revision"))) &&
    (!q.has("cursor") ||
      (cursor(q.get("cursor")) &&
        hash(q.get("snapshot")) &&
        hash(q.get("permission_revision"))))
  );
}
const kindByView = {
  balances: "balance",
  reservations: "reservation",
  cases: "case",
  adjustments: "adjustment",
  "payment-terms": "payment",
};
export function valid(d: Data, q: URLSearchParams): boolean {
  try {
    if (
      !validQuery(q) ||
      !exact(d, [
        "schema",
        "numeric_format",
        "read_only",
        "commands",
        "source_mode",
        "auth_scope",
        "snapshot",
        "permission_revision",
        "view",
        "selection",
        "cursor",
        "next_cursor",
        "total",
        "observed_at",
        "records",
      ]) ||
      d.schema !== 1 ||
      d.numeric_format !== "decimal-strings-read-v1" ||
      d.read_only !== true ||
      d.commands !== "unconnected" ||
      d.source_mode !== "synthetic" ||
      !hash(d.auth_scope) ||
      !hash(d.snapshot) ||
      !hash(d.permission_revision) ||
      !iso(d.observed_at) ||
      d.view !== q.get("view") ||
      !exact(d.selection, ["q"]) ||
      d.selection.q !== q.get("q") ||
      d.cursor !== (q.get("cursor") ?? null) ||
      !(d.next_cursor === null || cursor(d.next_cursor)) ||
      (d.next_cursor !== null && d.next_cursor === d.cursor) ||
      (q.has("snapshot") && d.snapshot !== q.get("snapshot")) ||
      (q.has("permission_revision") &&
        d.permission_revision !== q.get("permission_revision"))
    )
      return false;
    return (
      (d.total === null || count(d.total)) &&
      Array.isArray(d.records) &&
      d.records.length <= 25 &&
      new Set(d.records.map((r) => r?.id)).size === d.records.length &&
      d.records.every((r) => record(r) && r.kind === kindByView[d.view]) &&
      (d.total === null || d.total >= d.records.length) &&
      (d.next_cursor === null ||
        (d.records.length === 25 && (d.total === null || d.total > 25))) &&
      !(
        d.cursor === null &&
        d.next_cursor === null &&
        d.total !== null &&
        d.total !== d.records.length
      )
    );
  } catch {
    return false;
  }
}
