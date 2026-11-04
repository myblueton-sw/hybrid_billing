import {
  draftKeys,
  profile,
  issues,
  simulate,
  type Prepared,
  type Draft,
} from "./model";
type Store = Pick<Storage, "getItem" | "setItem">;
const keys = (v: unknown, names: readonly string[]) =>
  !!v &&
  typeof v === "object" &&
  !Array.isArray(v) &&
  Object.keys(v).length === names.length &&
  names.every((k) => Object.hasOwn(v, k));
export const storageKey = (s: Prepared) => `hb-b05-draft-v1:${s.context}`;
function valid(d: Draft, s: Prepared) {
  if (issues(d, s, true).length) throw Error("draft_invalid");
  simulate(d, s);
}
export function restore(store: Store, s: Prepared): Draft | null {
  const raw = store.getItem(storageKey(s));
  if (raw === null) return null;
  if (raw.length > 8192) throw Error("draft_invalid");
  const v = JSON.parse(raw);
  if (
    !keys(v, ["version", "profile", "context", "row", "draft"]) ||
    v.version !== 1 ||
    v.profile !== profile ||
    v.context !== s.context ||
    v.row !== s.row.id ||
    !keys(v.draft, draftKeys) ||
    draftKeys.some(
      (k) => typeof v.draft[k] !== "string" || v.draft[k].length > 300,
    )
  )
    throw Error("draft_invalid");
  valid(v.draft, s);
  return v.draft;
}
export function save(store: Store, s: Prepared, d: Draft) {
  valid(d, s);
  const text = JSON.stringify({
    version: 1,
    profile,
    context: s.context,
    row: s.row.id,
    draft: Object.fromEntries(draftKeys.map((k) => [k, d[k]])),
  });
  if (text.length > 8192) throw Error("draft_invalid");
  store.setItem(storageKey(s), text);
  if (store.getItem(storageKey(s)) !== text) throw Error("draft_save_failed");
}
