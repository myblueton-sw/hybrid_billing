import {
  issues,
  simulate,
  adjust,
  profile,
  type Prepared,
  type Draft,
} from "./model";
type Store = Pick<Storage, "getItem" | "setItem">;
const keys = (value: unknown, names: string[]) =>
  !!value &&
  typeof value === "object" &&
  !Array.isArray(value) &&
  Object.keys(value).length === names.length &&
  names.every((n) => Object.hasOwn(value, n));
const draftKeys = [
  "name",
  "unitPrice",
  "denominator",
  "discount",
  "fee",
  "target",
  "reason",
  "reference",
] as const;
export const storageKey = (source: Prepared) =>
  `hb-b04-draft-v1:${source.context}`;
function validate(draft: Draft, source: Prepared) {
  if (issues(draft, source, true).length) throw Error("draft_invalid");
  const result = simulate(draft, source);
  if (draft.target !== "") adjust(draft, source, result);
}
export function restore(store: Store, source: Prepared): Draft | null {
  const raw = store.getItem(storageKey(source));
  if (raw === null) return null;
  if (raw.length > 8192) throw Error("draft_invalid");
  const value = JSON.parse(raw);
  if (
    !keys(value, ["version", "profile", "context", "row", "draft"]) ||
    value.version !== 1 ||
    value.profile !== profile ||
    value.context !== source.context ||
    value.row !== source.row.id ||
    !keys(value.draft, [...draftKeys])
  )
    throw Error("draft_invalid");
  if (
    draftKeys.some(
      (k) => typeof value.draft[k] !== "string" || value.draft[k].length > 300,
    )
  )
    throw Error("draft_invalid");
  validate(value.draft, source);
  return value.draft;
}
export function save(store: Store, source: Prepared, draft: Draft) {
  validate(draft, source);
  const parameters = Object.fromEntries(draftKeys.map((k) => [k, draft[k]]));
  const text = JSON.stringify({
    version: 1,
    profile,
    context: source.context,
    row: source.row.id,
    draft: parameters,
  });
  if (text.length > 8192) throw Error("draft_invalid");
  store.setItem(storageKey(source), text);
  if (store.getItem(storageKey(source)) !== text)
    throw Error("draft_save_failed");
}
