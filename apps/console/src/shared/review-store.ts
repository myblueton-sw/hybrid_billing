export type ReviewRecord = {
  screen: string;
  revision: string;
  scope: string;
  items: string[];
  variants: {
    locale: string;
    width: number;
    profile: string;
    mode: string;
    extensions: string[];
    state: Record<string, string | number | boolean>;
  };
  confirmedAt: string;
  evidence: "local-owner-acknowledgement";
};
export type Store = Pick<Storage, "getItem" | "setItem">;
export const reviewKey = "hb-console-review-v1";
export function readRecords(storage: Store): ReviewRecord[] {
  const raw = storage.getItem(reviewKey);
  if (!raw) return [];
  const value: unknown = JSON.parse(raw);
  if (
    !Array.isArray(value) ||
    value.length > 500 ||
    value.some(
      (r) =>
        !r ||
        typeof r.screen !== "string" ||
        !/^[a-f0-9]{64}$/.test(r.revision) ||
        !/^[a-f0-9]{64}$/.test(r.scope) ||
        !r.variants ||
        !["ko", "en", "ar"].includes(r.variants.locale) ||
        !Number.isInteger(r.variants.width) ||
        r.variants.width <= 0 ||
        typeof r.variants.profile !== "string" ||
        !["synthetic", "unconnected"].includes(r.variants.mode) ||
        !Array.isArray(r.variants.extensions) ||
        !r.variants.state ||
        typeof r.variants.state !== "object" ||
        !Array.isArray(r.items) ||
        r.items.some((i: unknown) => typeof i !== "string") ||
        typeof r.confirmedAt !== "string" ||
        r.evidence !== "local-owner-acknowledgement",
    )
  )
    throw Error("invalid review records");
  return value;
}
export function saveRecord(storage: Store, record: ReviewRecord) {
  const records = readRecords(storage);
  const next = [
    ...records.filter(
      (r) =>
        !(
          r.screen === record.screen &&
          r.revision === record.revision &&
          r.scope === record.scope
        ),
    ),
    record,
  ];
  if (next.length > 500) throw Error("review capacity");
  const encoded = JSON.stringify(next);
  storage.setItem(reviewKey, encoded);
  if (storage.getItem(reviewKey) !== encoded) throw Error("review persistence");
  return next;
}
export async function digest(value: string) {
  const data = await crypto.subtle.digest(
    "SHA-256",
    new TextEncoder().encode(value),
  );
  return [...new Uint8Array(data)]
    .map((n) => n.toString(16).padStart(2, "0"))
    .join("");
}
